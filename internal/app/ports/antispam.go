package ports

import (
	"time"
	"twitchspam/config"
)

type CheckerPort interface {
	Check(irc *IRCMessage, cfg *config.Config) *Action
}

type ActionType string

type Action struct {
	Type     ActionType
	Reason   string
	Duration time.Duration
	UserID   string
	Username string
	Text     string
}
