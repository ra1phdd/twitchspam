package ports

import "time"

type TimersPort interface {
	AddTimer(id string, interval time.Duration, repeat bool, args map[string]any, task func(map[string]any))
	RemoveTimer(id string)
}
