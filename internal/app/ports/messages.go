package ports

import (
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
)

type CheckerPort interface {
	Check(msg *domain.ChatMessage) *CheckerAction
	CheckBanwords(textLower string, wordsOriginal []string) *CheckerAction
	CheckAds(text string, username string) *CheckerAction
	CheckMwords(msg *domain.ChatMessage) *CheckerAction
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
