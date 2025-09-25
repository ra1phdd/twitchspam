package ports

import "time"

type StreamPort interface {
	Stats() StatsPort
	IsLive() bool
	SetIslive(isLive bool)
	StreamID() string
	SetStreamID(streamID string)
	ChannelID() string
	SetChannelID(channelID string)
	ChannelName() string
	SetCategory(category string)
	Category() string
}

type StatsPort interface {
	SetStartTime(t time.Time)
	GetStartTime() time.Time
	SetEndTime(t time.Time)
	GetEndTime() time.Time
	SetOnline(viewers int)
	AddMessage(username string)
	AddDeleted(username string)
	AddBan(username string)
	AddTimeout(username string)
	AddCategoryChange(category string, t time.Time)
	GetStats() *AnswerType
	GetUserStats(username string) *AnswerType
	GetTopStats(count int) *AnswerType
}
