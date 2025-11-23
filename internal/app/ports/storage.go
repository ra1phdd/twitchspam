package ports

import (
	"time"
)

type StorePort[T any] interface {
	Push(key string, subKey string, val T, ttl *time.Duration)
	Update(key string, subKey string, updateFn func(current *T, exists bool) *T)
	GetAllData() map[string]map[string]T
	GetAll(key string) map[string]T
	Get(key string, subKey string) (T, bool)
	Len(key string) int
	ForEach(key string, fn func(val *T))
	ClearKey(key string)
	ClearAll()
	SetCapacity(capacity int32)
	GetCapacity() int32
	SetTTL(newTTL time.Duration)
	GetTTL() time.Duration
}

type CachePort[T any] interface {
	Set(key string, val T)
	Get(key string) (T, bool)
	ClearKey(key string)
	ClearAll()
	FlushToDisk()
}
