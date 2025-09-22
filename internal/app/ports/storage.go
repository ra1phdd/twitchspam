package ports

import (
	"time"
)

type StorePort[T any] interface {
	Push(key string, val T)
	Get(key string) ([]T, bool)
	Len(key string) int
	ForEach(key string, fn func(val *T))
	CleanAllExpired()
	ClearKey(key string)
	ClearAll()
	SetCapacity(capacity int)
	SetTTL(newTTL time.Duration)
}
