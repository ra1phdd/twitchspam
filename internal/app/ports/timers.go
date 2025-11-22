package ports

import "time"

type TimersPort interface {
	AddTimer(id string, interval time.Duration, repeat bool, args map[string]any, task func(map[string]any))
	ActiveTimers() map[string]time.Duration
	UpdateTimerTTL(id string, newInterval time.Duration) bool
	RemoveTimer(id string)
}
