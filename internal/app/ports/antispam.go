package ports

import (
	"time"
)

type CheckerPort interface {
	Check(msg *ChatMessage) *CheckerAction
}

type ActionType string

type CheckerAction struct {
	Type     ActionType
	Reason   string
	Duration time.Duration
	UserID   string
	Username string
	Text     string
}
