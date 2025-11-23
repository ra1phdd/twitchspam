package ports

import (
	"sync"
	"time"
)

type StreamPort interface {
	Stats() StatsPort
	IsLive() bool
	SetIslive(isLive bool)
	ChannelID() string
	SetChannelID(channelID string)
	ChannelName() string
	SetCategory(category string)
	Category() string
	OnceStart() *sync.Once
}

type StatsPort interface {
	Reset()
	ResetOfflineMessages()
	SetIsOnline(isOnline bool)
	SetStartTime(t time.Time)
	GetStartTime() time.Time
	SetEndTime(t time.Time)
	GetEndTime() time.Time
	SetOnline(viewers int)
	AddMessage(username string)
	AddDeleted(username string)
	AddWarn(username string)
	AddBan(username string)
	AddTimeout(username string)
	AddCategoryChange(category string, t time.Time)
	GetStats() *AnswerType
	GetUserStats(username string) *AnswerType
	GetTopStats(count int) *AnswerType
}
