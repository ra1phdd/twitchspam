package ports

import (
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

type CheckerPort interface {
	Check(msg *ChatMessage) *CheckerAction
	CheckBanwords(textLower string, wordsOriginal []string) *CheckerAction
	CheckAds(text string, username string) *CheckerAction
	CheckMwords(msg *ChatMessage) *CheckerAction
}

type CommandPort interface {
	FindMessages(msg *ChatMessage) *AnswerType
}

type Command interface {
	Execute(cfg *config.Config, text *MessageText) *AnswerType
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
