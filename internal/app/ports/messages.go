package ports

import "time"

type AdminPort interface {
	FindMessages(msg *ChatMessage) ActionType
}

type UserPort interface {
	FindMessages(msg *ChatMessage) ActionType
}

type CheckerPort interface {
	Check(msg *ChatMessage) *CheckerAction
	CheckBanwords(words []string) *CheckerAction
	CheckMwords(text string) *CheckerAction
}

type ActionType string

type CheckerAction struct {
	Type     ActionType
	Reason   string
	Duration time.Duration
}
