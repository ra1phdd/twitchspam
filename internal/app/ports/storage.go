package ports

import (
	"time"
	"twitchspam/internal/app/adapters/storage"
)

type StorePort interface {
	Push(username, text string, ttl time.Duration)
	ForEach(username string, fn func(msg *storage.Message))
	Cleanup()
	CleanupUser(username string)
}
