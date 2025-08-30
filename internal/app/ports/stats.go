package ports

import "time"

type StatsPort interface {
	SetStartTime(t time.Time)
	SetEndTime(t time.Time)
	SetOnline(viewers int)
	CountFirstMessages()
	AddMessage(username string)
	AddBan(username string)
	AddTimeout(username string)
	GetStats() string
	GetModeratorStats(username string) string
	GetUserStats(username string) string
}
