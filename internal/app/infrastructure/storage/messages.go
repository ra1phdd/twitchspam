package storage

import (
	"time"
	"twitchspam/internal/app/domain"
)

type Message struct {
	Data               *domain.ChatMessage
	Time               time.Time
	HashWordsLowerNorm []uint64
	IgnoreAntispam     bool
}
