package antispam

import (
	"runtime"
	"strings"
	"time"
	"twitchspam/config"
	"twitchspam/internal/app/adapters/banwords"
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

type Checker struct {
	log logger.Logger

	messages ports.StorePort
	bwords   ports.BanwordsPort
}

func NewCheck(log logger.Logger, cfg *config.Config) *Checker {
	return &Checker{
		log:      log,
		messages: storage.New(runtime.NumCPU(), 500*time.Millisecond),
		bwords:   banwords.New(cfg.Banwords),
	}
}

func (c *Checker) Check(irc *ports.IRCMessage, cfg *config.Config) *ports.Action {
	text := strings.ToLower(domain.NormalizeText(irc.Text))

	if irc.IsMod {
		return &ports.Action{Type: None}
	}

	if c.bwords.CheckMessage(text) {
		return &ports.Action{
			Type:     Ban,
			Reason:   "банворд",
			UserID:   irc.UserID,
			Username: irc.Username,
			Text:     irc.Text,
		}
	}

	if irc.IsVIP && !cfg.Spam.VIPEnabled {
		return &ports.Action{Type: None}
	}

	var countSpam int
	c.messages.ForEach(irc.Username, func(msg *storage.Message) {
		similarity := domain.JaccardSimilarity(text, msg.Text)

		if similarity >= cfg.Spam.SimilarityThreshold {
			countSpam++
		}
	})

	// кол-во спама -1 новый
	if countSpam >= cfg.Spam.MessageLimit-1 {
		c.messages.CleanupUser(irc.Username)

		return &ports.Action{
			Type:     Timeout,
			Reason:   "спам",
			Duration: 600 * time.Second,
			UserID:   irc.UserID,
			Username: irc.Username,
			Text:     irc.Text,
		}
	}

	c.messages.Push(irc.Username, text, time.Duration(cfg.Spam.CheckWindowSeconds)*time.Second)
	return &ports.Action{Type: None}
}
