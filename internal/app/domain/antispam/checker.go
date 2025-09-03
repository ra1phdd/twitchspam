package antispam

import (
	"fmt"
	"log/slog"
	"regexp"
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

	if action := c.checkBypass(msg); action != nil {
		return action
	}

	text := strings.ToLower(domain.NormalizeText(msg.Message.Text))
	words := strings.Fields(text)

	if !c.cfg.Spam.SettingsEmotes.Enabled && (msg.Message.EmoteOnly || c.sevenTV.IsOnlyEmotes(msg.Message.Text)) {
		return &ports.CheckerAction{Type: None}
	}

	if action := c.checkBanwords(msg, words); action != nil {
		return action
	}

	if action := c.checkMwords(msg, text); action != nil {
		return action
	}

	if action := c.checkWordLength(msg, words); action != nil {
		return action
	}

	if action := c.checkSpam(msg, text); action != nil {
		return action
	}

	return &ports.CheckerAction{Type: None}
}

func (c *Checker) checkBypass(msg *ports.ChatMessage) *ports.CheckerAction {
	if msg.Chatter.IsBroadcaster || msg.Chatter.IsMod || !c.cfg.Spam.SettingsDefault.Enabled ||
		(c.cfg.Spam.Mode == "online" && !c.stream.IsLive()) {
		return &ports.CheckerAction{Type: None}
	}

	for _, user := range c.cfg.Spam.WhitelistUsers {
		if user == msg.Chatter.Username {
			return &ports.CheckerAction{Type: None}
		}
	}

	return nil
}

func (c *Checker) checkBanwords(msg *ports.ChatMessage, words []string) *ports.CheckerAction {
	if !c.bwords.CheckMessage(words) {
		return nil
	}

	return &ports.CheckerAction{
		Type:     Ban,
		Reason:   "банворд",
		UserID:   msg.Chatter.UserID,
		Username: msg.Chatter.Username,
		Text:     msg.Message.Text,
	}
}

func (c *Checker) checkMwords(msg *ports.ChatMessage, text string) *ports.CheckerAction {
	for _, group := range c.cfg.MwordGroup {
		if !group.Enabled {
			continue
		}

		for _, word := range group.Words {
			if word == "" {
				continue
			}

			pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(word))
			if (strings.HasPrefix(word, `r"`) && strings.HasSuffix(word, `"`)) ||
				(strings.HasPrefix(word, `r'`) && strings.HasSuffix(word, `'`)) {
				pattern = word[2 : len(word)-1]
			}

			re, err := regexp.Compile(pattern)
			if err != nil {
				c.log.Error("Invalid regex", err, slog.String("pattern", pattern))
				continue
			}

			if re.MatchString(text) {
				return &ports.CheckerAction{
					Type:     ports.ActionType(group.Action),
					Reason:   fmt.Sprintf("мворд (%s)", word),
					Duration: time.Duration(group.Duration) * time.Second,
					UserID:   msg.Chatter.UserID,
					Username: msg.Chatter.Username,
					Text:     msg.Message.Text,
				}
			}
		}
	}

	for word, mw := range c.cfg.Mword {
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(word))
		if (strings.HasPrefix(word, `r"`) && strings.HasSuffix(word, `"`)) ||
			(strings.HasPrefix(word, `r'`) && strings.HasSuffix(word, `'`)) {
			pattern = word[2 : len(word)-1]
		}

		re, err := regexp.Compile(pattern)
		if err != nil {
			c.log.Error("Invalid regex", err, slog.String("pattern", pattern))
			continue
		}

		if re.MatchString(text) {
			return &ports.CheckerAction{
				Type:     ports.ActionType(mw.Action),
				Reason:   fmt.Sprintf("мворд (%s)", word),
				Duration: time.Duration(mw.Duration) * time.Second,
				UserID:   msg.Chatter.UserID,
				Username: msg.Chatter.Username,
				Text:     msg.Message.Text,
			}
		}
	}
	return nil
}

func (c *Checker) checkWordLength(msg *ports.ChatMessage, words []string) *ports.CheckerAction {
	settings := c.cfg.Spam.SettingsDefault
	if msg.Chatter.IsVip {
		settings = c.cfg.Spam.SettingsVIP
		if !settings.Enabled {
			return &ports.CheckerAction{Type: None}
		}
	}

	for _, word := range words {
		if settings.MaxWordLength > 0 && len(word) >= settings.MaxWordLength && settings.MaxWordTimeoutTime > 0 {
			return &ports.CheckerAction{
				Type:     Timeout,
				Reason:   "превышена максимальная длина слова",
				Duration: time.Duration(settings.MaxWordTimeoutTime) * time.Second,
				UserID:   msg.Chatter.UserID,
				Username: msg.Chatter.Username,
				Text:     msg.Message.Text,
			}
		}
	}
	return nil
}

func (c *Checker) checkSpam(msg *ports.ChatMessage, text string) *ports.CheckerAction {
	settings := c.cfg.Spam.SettingsDefault
	if msg.Chatter.IsVip {
		settings = c.cfg.Spam.SettingsVIP
		if !settings.Enabled {
			return nil
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

	if countSpam >= settings.MessageLimit-1 {
		var dur time.Duration
		if except, ok := c.cfg.Spam.Exceptions[text]; ok && except.Enabled {
			if countSpam < except.MessageLimit-1 {
				c.messages.Push(msg.Chatter.Username, text, time.Duration(c.cfg.Spam.CheckWindowSeconds)*time.Second)
				return nil
			}
			dur = time.Duration(except.Timeout) * time.Second
		} else {
			sec := domain.GetByIndexOrLast(settings.Timeouts, c.timeouts.Len(msg.Chatter.Username))
			dur = time.Duration(sec) * time.Second
			c.timeouts.Push(msg.Chatter.Username, Empty{}, time.Duration(settings.ResetTimeoutSeconds)*time.Second)
		}

		c.messages.CleanupUser(msg.Chatter.Username)
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
	return nil
}
