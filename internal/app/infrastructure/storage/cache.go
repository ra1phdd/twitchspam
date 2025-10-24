package storage

import (
	"encoding/json"
	"github.com/maypok86/otter/v2"
	"os"
	"sync/atomic"
	"time"
)

type Cache[T any] struct {
	outer *otter.Cache[string, T]

	ttl atomic.Int64
	cap atomic.Int32

	persist       bool
	flushOnChange bool
	filePath      string
	stopFlush     chan struct{}
}

func NewCache[T any](capacity int32, ttl time.Duration, persist bool, flushOnChange bool, filePath string, flushInterval time.Duration) *Cache[T] {
	c := &Cache[T]{
		ttl:           atomic.Int64{},
		cap:           atomic.Int32{},
		persist:       persist,
		flushOnChange: flushOnChange,
		filePath:      filePath,
		stopFlush:     make(chan struct{}),
	}
	c.outer = otter.Must(&otter.Options[string, T]{
		InitialCapacity:  int(capacity),
		ExpiryCalculator: otter.ExpiryAccessing[string, T](ttl),
		OnDeletion: func(e otter.DeletionEvent[string, T]) {
			if c.persist && c.flushOnChange {
				c.FlushToDisk()
			}
		},
	})

	c.ttl.Store(ttl.Nanoseconds())
	c.cap.Store(capacity)

	if c.persist && c.filePath != "" {
		_ = c.loadFromDisk()
	}

	if c.persist && !c.flushOnChange && flushInterval > 0 {
		go c.periodicFlush(flushInterval)
	}

	return c
}

func (c *Cache[T]) Set(key string, val T) {
	c.outer.Set(key, val)
	if c.persist && c.flushOnChange {
		go c.FlushToDisk()
	}
}

func (c *Cache[T]) Get(key string) (T, bool) {
	return c.outer.GetIfPresent(key)
}

func (c *Cache[T]) ClearKey(key string) {
	c.outer.Invalidate(key)
	if c.persist && c.flushOnChange {
		c.FlushToDisk()
	}
}

func (c *Cache[T]) ClearAll() {
	c.outer.InvalidateAll()
	if c.persist && c.flushOnChange {
		c.FlushToDisk()
	}
}

func (c *Cache[T]) FlushToDisk() {
	if !c.persist || c.filePath == "" {
		return
	}

	cacheData := make(map[string]T)
	for k, v := range c.outer.All() {
		cacheData[k] = v
	}

	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(c.filePath, data, 0600)
}

func (c *Cache[T]) periodicFlush(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.FlushToDisk()
		case <-c.stopFlush:
			return
		}
	}
}

func (c *Cache[T]) loadFromDisk() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		return err
	}

	var items map[string]T
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	for k, v := range items {
		c.outer.Set(k, v)
	}

	return nil
}

func (c *Cache[T]) Close() {
	close(c.stopFlush)
}
