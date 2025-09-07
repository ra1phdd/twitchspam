package ports

import "time"

type AdminPort interface {
	FindMessages(msg *ChatMessage) *AnswerType
}

type UserPort interface {
	FindMessages(msg *ChatMessage) *AnswerType
}

type CheckerPort interface {
	Check(msg *ChatMessage) *CheckerAction
	CheckBanwords(text, textOriginal string) *CheckerAction
	CheckAds(text string, username string) *CheckerAction
	CheckMwords(text string) *CheckerAction
}

type AnswerType struct {
	Text    []string
	IsReply bool
}

type CheckerAction struct {
	Type     string
	Reason   string
	Duration time.Duration
}
