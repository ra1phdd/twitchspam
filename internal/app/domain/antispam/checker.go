package antispam

import (
	"runtime"
	"strings"
	"time"
	"twitchspam/config"
	"twitchspam/internal/app/adapters/banwords"
	"twitchspam/internal/app/adapters/seventv"
	"twitchspam/internal/app/adapters/storage"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

const (
	None    ports.ActionType = "none"
	Ban     ports.ActionType = "ban"
	Timeout ports.ActionType = "timeout"
)

type Empty struct{}

type Checker struct {
	log logger.Logger
	cfg *config.Config

	stream   ports.StreamPort
	stats    ports.StatsPort
	timeouts ports.StorePort[Empty]
	messages ports.StorePort[string]
	bwords   ports.BanwordsPort
	sevenTV  ports.SevenTVPort
}

func NewCheck(log logger.Logger, cfg *config.Config, stream ports.StreamPort, stats ports.StatsPort) *Checker {
	return &Checker{
		log:    log,
		cfg:    cfg,
		stream: stream,
		stats:  stats,
		timeouts: storage.New[Empty](runtime.NumCPU(), 500*time.Millisecond, func() int {
			return 15
		}),
		messages: storage.New[string](runtime.NumCPU(), 500*time.Millisecond, func() int {
			defLimit := float64(cfg.Spam.SettingsDefault.MessageLimit*cfg.Spam.SettingsDefault.MinGapMessages) / cfg.Spam.SettingsDefault.SimilarityThreshold
			vipLimit := float64(cfg.Spam.SettingsVIP.MessageLimit*cfg.Spam.SettingsVIP.MinGapMessages) / cfg.Spam.SettingsVIP.SimilarityThreshold

			return int(max(defLimit, vipLimit))
		}),
		bwords:  banwords.New(cfg.Banwords),
		sevenTV: seventv.New(log, stream),
	}
}

func (c *Checker) Check(irc *ports.IRCMessage) *ports.CheckerAction {
	if c.stream.IsLive() {
		c.stats.AddMessage(irc.Username)
		if irc.IsFirst {
			c.stats.CountFirstMessages()
		}
	}

	if irc.IsMod || !c.cfg.Spam.SettingsDefault.Enabled || (c.cfg.Spam.Mode == "online" && !c.stream.IsLive()) || (!c.cfg.Spam.SettingsEmotes.Enabled && irc.EmoteOnly) {
		return &ports.CheckerAction{Type: None}
	}
	text := strings.ToLower(domain.NormalizeText(irc.Text))
	words := strings.Fields(text)

	if !c.cfg.Spam.SettingsEmotes.Enabled && c.sevenTV.IsOnlyEmotes(words) {
		return &ports.CheckerAction{Type: None}
	}

	if c.bwords.CheckMessage(words) {
		return &ports.CheckerAction{
			Type:     Ban,
			Reason:   "банворд",
			UserID:   irc.UserID,
			Username: irc.Username,
			Text:     irc.Text,
		}
	}

	for _, user := range c.cfg.Spam.WhitelistUsers {
		if strings.ToLower(user) == strings.ToLower(irc.Username) {
			return &ports.CheckerAction{Type: None}
		}
	}

	if c.cfg.PunishmentOnline && c.bwords.CheckOnline(text) {
		return &ports.CheckerAction{
			Type:     Ban,
			Reason:   "тупое",
			UserID:   irc.UserID,
			Username: irc.Username,
			Text:     irc.Text,
		}
	}

	for _, word := range words {
		if c.cfg.Spam.SettingsDefault.MaxWordLength == 0 || len(word) < c.cfg.Spam.SettingsDefault.MaxWordLength || c.cfg.Spam.SettingsDefault.MaxWordTimeoutTime == 0 {
			continue
		}

		return &ports.CheckerAction{
			Type:     Timeout,
			Reason:   "превышена максимальная длина слова",
			Duration: time.Duration(c.cfg.Spam.SettingsDefault.MaxWordTimeoutTime) * time.Second,
			UserID:   irc.UserID,
			Username: irc.Username,
			Text:     irc.Text,
		}
	}

	settings := c.cfg.Spam.SettingsDefault
	if irc.IsVIP {
		settings = c.cfg.Spam.SettingsVIP

		if !c.cfg.Spam.SettingsVIP.Enabled {
			return &ports.CheckerAction{Type: None}
		}
	}

	var countSpam, gap int
	c.messages.ForEach(irc.Username, func(item *storage.Item[string]) {
		similarity := domain.JaccardSimilarity(text, item.Data)

		if similarity >= settings.SimilarityThreshold {
			if gap < settings.MinGapMessages {
				countSpam++
			}
			gap = 0
		} else {
			gap++
		}
	})

	// кол-во спама -1 новый
	if countSpam >= settings.MessageLimit-1 {
		c.messages.CleanupUser(irc.Username)

		dur := time.Duration(c.cfg.Spam.SpamExceptions[text]) * time.Second
		if dur == 0 {
			sec := domain.GetByIndexOrLast(settings.Timeouts, c.timeouts.Len(irc.Username))
			dur = time.Duration(sec) * time.Second
		}

		c.timeouts.Push(irc.Username, Empty{}, time.Duration(settings.ResetTimeoutSeconds)*time.Second)
		return &ports.CheckerAction{
			Type:     Timeout,
			Reason:   "спам",
			Duration: dur,
			UserID:   irc.UserID,
			Username: irc.Username,
			Text:     irc.Text,
		}
	}

	c.messages.Push(irc.Username, text, time.Duration(c.cfg.Spam.CheckWindowSeconds)*time.Second)
	return &ports.CheckerAction{Type: None}
}
