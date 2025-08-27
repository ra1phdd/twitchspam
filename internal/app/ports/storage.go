package ports

import (
	"time"
	"twitchspam/internal/app/adapters/storage"
)

type StorePort[T any] interface {
	Len(username string) int
	Push(username string, data T, ttl time.Duration)
	ForEach(username string, fn func(item *storage.Item[T]))
	Cleanup()
	CleanupUser(username string)
}
