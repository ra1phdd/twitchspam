package antispam

import (
	"runtime"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/seventv"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/banwords"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

const (
	None    ports.ActionType = "none"
	Ban     ports.ActionType = "ban"
	Timeout ports.ActionType = "timeout"
	Delete  ports.ActionType = "delete"
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

func (c *Checker) Check(msg *ports.ChatMessage) *ports.CheckerAction {
	if c.stream.IsLive() {
		c.stats.AddMessage(msg.Chatter.Username)
	}

	if msg.Chatter.IsBroadcaster || msg.Chatter.IsMod || !c.cfg.Spam.SettingsDefault.Enabled ||
		(c.cfg.Spam.Mode == "online" && !c.stream.IsLive()) || (!c.cfg.Spam.SettingsEmotes.Enabled && msg.Message.EmoteOnly) {
		return &ports.CheckerAction{Type: None}
	}
	text := strings.ToLower(domain.NormalizeText(msg.Message.Text))
	words := strings.Fields(text)

	if !c.cfg.Spam.SettingsEmotes.Enabled && c.sevenTV.IsOnlyEmotes(msg.Message.Text) {
		return &ports.CheckerAction{Type: None}
	}

	if c.bwords.CheckMessage(words) {
		return &ports.CheckerAction{
			Type:     Ban,
			Reason:   "банворд",
			UserID:   msg.Chatter.UserID,
			Username: msg.Chatter.Username,
			Text:     msg.Message.Text,
		}
	}

	for _, user := range c.cfg.Spam.WhitelistUsers {
		if strings.ToLower(user) == strings.ToLower(msg.Chatter.Username) {
			return &ports.CheckerAction{Type: None}
		}
	}

	for _, group := range c.cfg.MwordGroup {
		if !group.Enabled {
			continue
		}

		for _, word := range words {
			if !strings.Contains(text, word) {
				continue
			}

			return &ports.CheckerAction{
				Type:     ports.ActionType(group.Action),
				Reason:   "мворд",
				Duration: time.Duration(group.Duration) * time.Second,
				UserID:   msg.Chatter.UserID,
				Username: msg.Chatter.Username,
				Text:     msg.Message.Text,
			}
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
			UserID:   msg.Chatter.UserID,
			Username: msg.Chatter.Username,
			Text:     msg.Message.Text,
		}
	}

	settings := c.cfg.Spam.SettingsDefault
	if msg.Chatter.IsVip {
		settings = c.cfg.Spam.SettingsVIP

		if !c.cfg.Spam.SettingsVIP.Enabled {
			return &ports.CheckerAction{Type: None}
		}
	}

	var countSpam, gap int
	c.messages.ForEach(msg.Chatter.Username, func(item *storage.Item[string]) {
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
		c.messages.CleanupUser(msg.Chatter.Username)

		dur := time.Duration(c.cfg.Spam.SpamExceptions[text]) * time.Second
		if dur == 0 {
			sec := domain.GetByIndexOrLast(settings.Timeouts, c.timeouts.Len(msg.Chatter.Username))
			dur = time.Duration(sec) * time.Second
		}

		c.timeouts.Push(msg.Chatter.Username, Empty{}, time.Duration(settings.ResetTimeoutSeconds)*time.Second)
		return &ports.CheckerAction{
			Type:     Timeout,
			Reason:   "спам",
			Duration: dur,
			UserID:   msg.Chatter.UserID,
			Username: msg.Chatter.Username,
			Text:     msg.Message.Text,
		}
	}

	c.messages.Push(msg.Chatter.Username, text, time.Duration(c.cfg.Spam.CheckWindowSeconds)*time.Second)
	return &ports.CheckerAction{Type: None}
}
