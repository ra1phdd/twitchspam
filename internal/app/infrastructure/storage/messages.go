package storage

import (
	"time"
	"twitchspam/internal/app/domain"
)

type Message struct {
	Data           *domain.ChatMessage
	Time           time.Time
	IgnoreAntispam bool
	IgnoreNuke     bool
}
