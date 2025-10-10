package storage

import (
	"github.com/maypok86/otter/v2"
	"sync/atomic"
	"time"
)

type Cache[T any] struct {
	outer *otter.Cache[string, T]

	ttl atomic.Int64
	cap atomic.Int32
}

func NewCache[T any](capacity int32, ttl time.Duration) *Cache[T] {
	s := &Cache[T]{
		outer: otter.Must(&otter.Options[string, T]{
			InitialCapacity:  int(capacity),
			ExpiryCalculator: otter.ExpiryAccessing[string, T](ttl),
		}),
	}
	s.ttl.Store(ttl.Nanoseconds())
	s.cap.Store(capacity)

	return s
}

func (c *Cache[T]) Set(key string, val T) {
	c.outer.Set(key, val)
}

func (c *Cache[T]) Get(key string) (T, bool) {
	return c.outer.GetIfPresent(key)
}

func (c *Cache[T]) ClearKey(key string) {
	c.outer.Invalidate(key)
}

func (c *Cache[T]) ClearAll() {
	c.outer.InvalidateAll()
}
