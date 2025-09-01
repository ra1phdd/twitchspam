package ports

import "time"

type StatsPort interface {
	SetStartTime(t time.Time)
	SetEndTime(t time.Time)
	SetOnline(viewers int)
	AddMessage(username string)
	AddDeleted(username string)
	AddBan(username string)
	AddTimeout(username string)
	GetStats() string
	GetUserStats(username string) string
}
