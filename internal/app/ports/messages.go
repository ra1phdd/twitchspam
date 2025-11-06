package ports

import (
	"time"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/infrastructure/config"
)

type MessagePort interface {
	Check(msg *message.ChatMessage)
	CheckAutomod(msg *message.ChatMessage)
}

type CheckerPort interface {
	Check(msg *message.ChatMessage, checkSpam bool) *CheckerAction
}

type CommandPort interface {
	FindMessages(msg *message.ChatMessage) *AnswerType
}

type Command interface {
	Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *AnswerType
}

type AnswerType struct {
	Text          []string
	IsReply       bool
	ReplyUsername string
}

type CheckerAction struct {
	Type       string
	ReasonMod  string
	ReasonUser string
	Duration   time.Duration
}
