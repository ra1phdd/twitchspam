package ports

import (
	"time"
)

type StorePort[T any] interface {
	Push(key string, val T)
	GetAll() map[string][]T
	Get(key string) ([]T, bool)
	Len(key string) int
	ForEach(key string, fn func(val *T))
	CleanAllExpired()
	ClearKey(key string)
	ClearAll()
	SetCapacity(capacity int)
	GetCapacity() int
	SetTTL(newTTL time.Duration)
	GetTTL() time.Duration
}
