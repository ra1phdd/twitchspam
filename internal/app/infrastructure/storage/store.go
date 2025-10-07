package storage

import (
	"github.com/maypok86/otter/v2"
	"sync/atomic"
	"time"
)

type Store[T any] struct {
	outer *otter.Cache[string, *otter.Cache[string, T]]

	ttl atomic.Int64
	cap atomic.Int32
}

func New[T any](cap int, ttl time.Duration) *Store[T] {
	s := &Store[T]{
		outer: otter.Must(&otter.Options[string, *otter.Cache[string, T]]{
			InitialCapacity:  128,
			ExpiryCalculator: otter.ExpiryAccessing[string, *otter.Cache[string, T]](24 * time.Hour),
		}),
	}
	s.ttl.Store(ttl.Nanoseconds())
	s.cap.Store(int32(cap))

	return s
}

func (s *Store[T]) getInner(key string) *otter.Cache[string, T] {
	inner, ok := s.outer.GetIfPresent(key)
	if ok {
		return inner
	}

	inner = otter.Must(&otter.Options[string, T]{
		MaximumSize:      int(s.cap.Load()),
		InitialCapacity:  int(s.cap.Load()),
		ExpiryCalculator: otter.ExpiryCreating[string, T](time.Duration(s.ttl.Load())),
	})
	s.outer.Set(key, inner)
	return inner
}

type PushOption func(*pushConfig)

type pushConfig struct {
	ttl *time.Duration
}

func WithTTL(ttl time.Duration) PushOption {
	return func(pc *pushConfig) {
		pc.ttl = &ttl
	}
}

func (s *Store[T]) Push(key string, subKey string, val T, opts ...PushOption) {
	inner := s.getInner(key)
	inner.Set(subKey, val)

	config := &pushConfig{}
	for _, opt := range opts {
		opt(config)
	}

	if config.ttl != nil && *config.ttl > 0 {
		inner.SetExpiresAfter(subKey, *config.ttl)
	}
}

func (s *Store[T]) Update(key string, subKey string, updateFn func(current T, exists bool) T) {
	inner := s.getInner(key)

	current, exists := inner.GetIfPresent(subKey)
	newValue := updateFn(current, exists)
	inner.Set(subKey, newValue)
}

func (s *Store[T]) GetAllData() map[string]map[string]T {
	result := make(map[string]map[string]T)

	for outerKey := range s.outer.All() {
		innerCache, ok := s.outer.GetIfPresent(outerKey)
		if !ok || innerCache == nil {
			continue
		}

		innerMap := make(map[string]T)
		for innerKey := range innerCache.All() {
			if val, ok := innerCache.GetIfPresent(innerKey); ok {
				innerMap[innerKey] = val
			}
		}

		if len(innerMap) > 0 {
			result[outerKey] = innerMap
		}
	}

	return result
}

func (s *Store[T]) GetAll(key string) map[string]T {
	inner, ok := s.outer.GetIfPresent(key)
	if !ok {
		return nil
	}
	values := make(map[string]T)
	for it := range inner.All() {
		if v, ok := inner.GetIfPresent(it); ok {
			values[it] = v
		}
	}
	return values
}

func (s *Store[T]) Get(key string, subKey string) (T, bool) {
	inner, ok := s.outer.GetIfPresent(key)
	if !ok {
		var zero T
		return zero, false
	}
	return inner.GetIfPresent(subKey)
}

func (s *Store[T]) Len(key string) int {
	inner := s.getInner(key)
	return inner.EstimatedSize()
}

func (s *Store[T]) ForEach(key string, fn func(val *T)) {
	inner := s.getInner(key)

	for innerKey := range inner.All() {
		v, ok := inner.GetIfPresent(innerKey)
		if !ok {
			continue
		}
		fn(&v)
	}
}

func (s *Store[T]) ClearKey(key string) {
	s.outer.Invalidate(key)
}

func (s *Store[T]) ClearAll() {
	s.outer.InvalidateAll()
}

func (s *Store[T]) SetCapacity(newCap int) {
	s.cap.Store(int32(newCap))

	for entry := range s.outer.All() {
		inner, ok := s.outer.GetIfPresent(entry)
		if !ok {
			continue
		}
		inner.SetMaximum(uint64(newCap))
	}
}

func (s *Store[T]) GetCapacity() int {
	return int(s.cap.Load())
}

func (s *Store[T]) SetTTL(newTTL time.Duration) {
	s.ttl.Store(newTTL.Nanoseconds())

	for entry := range s.outer.All() {
		inner, ok := s.outer.GetIfPresent(entry)
		if !ok {
			continue
		}

		for item := range inner.All() {
			inner.SetExpiresAfter(item, newTTL)
		}
	}
}

func (s *Store[T]) GetTTL() time.Duration {
	return time.Duration(s.ttl.Load())
}
