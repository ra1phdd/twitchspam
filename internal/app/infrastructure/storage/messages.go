package storage

import "twitchspam/internal/app/domain"

type Message struct {
	UserID             string
	Text               domain.MessageText
	HashWordsLowerNorm []uint64
	IgnoreAntispam     bool
}
