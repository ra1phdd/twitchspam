package ports

import (
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
)

type MessagePort interface {
	Check(msg *domain.ChatMessage)
	CheckAutomod(msg *domain.ChatMessage)
}

type CheckerPort interface {
	Check(msg *domain.ChatMessage, checkSpam bool) *CheckerAction
}

type CommandPort interface {
	FindMessages(msg *domain.ChatMessage) *AnswerType
}

type Command interface {
	Execute(cfg *config.Config, text *domain.MessageText) *AnswerType
}

type AnswerType struct {
	Text          []string
	IsReply       bool
	ReplyUsername string
}

type CheckerAction struct {
	Type     string
	Reason   string
	Duration time.Duration
}
