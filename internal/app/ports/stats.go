package ports

import "time"

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
	GetStats() *AnswerType
	GetUserStats(username string) *AnswerType
	GetTopStats(count int) *AnswerType
}
