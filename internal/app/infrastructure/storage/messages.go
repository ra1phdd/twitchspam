package storage

import (
	"time"
	"twitchspam/internal/app/domain/message"
)

type Message struct {
	Data           *message.ChatMessage
	Time           time.Time
	IgnoreAntispam bool
	IgnoreNuke     bool
}
