package storage

import (
	"sync"
	"time"
)

type Store[T any] struct {
	mu       sync.RWMutex
	keys     map[string]*keyStore[T]
	capacity int
	ttl      time.Duration
}

type keyStore[T any] struct {
	mu    sync.Mutex
	items []*entry[T]
}

type entry[T any] struct {
	val       T
	expiresAt time.Time
}

func New[T any](capacity int, ttl time.Duration) *Store[T] {
	s := &Store[T]{
		keys:     make(map[string]*keyStore[T]),
		capacity: capacity,
		ttl:      ttl,
	}
	go s.startAutoCleanup(100 * time.Millisecond)

	return s
}

func (s *Store[T]) getOrCreateKeyStore(key string) *keyStore[T] {
	s.mu.Lock()
	defer s.mu.Unlock()

	ks, ok := s.keys[key]
	if !ok {
		ks = &keyStore[T]{items: make([]*entry[T], 0, s.capacity)}
		s.keys[key] = ks
	}
	return ks
}

func (s *Store[T]) Push(key string, val T) {
	ks := s.getOrCreateKeyStore(key)

	ks.mu.Lock()
	defer ks.mu.Unlock()

	if s.capacity > 0 && len(ks.items) >= s.capacity {
		over := len(ks.items) - s.capacity
		ks.items = ks.items[over:]
	}

	e := &entry[T]{val: val, expiresAt: time.Now().Add(s.ttl)}
	ks.items = append(ks.items, e)
}

func (s *Store[T]) Get(key string) ([]T, bool) {
	s.mu.RLock()
	ks, ok := s.keys[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()

	var items []T
	for _, e := range ks.items {
		items = append(items, e.val)
	}

	return items, true
}

func (s *Store[T]) Len(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ks, ok := s.keys[key]
	if !ok {
		return 0
	}

	return len(ks.items)
}

func (s *Store[T]) ForEach(key string, fn func(val *T)) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ks, ok := s.keys[key]
	if !ok {
		return
	}

	for _, item := range ks.items {
		fn(&item.val)
	}
}

func (s *Store[T]) startAutoCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:
			s.CleanAllExpired()
		}
	}
}

func (s *Store[T]) CleanAllExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for key, store := range s.keys {
		store.mu.Lock()
		newItems := store.items[:0]
		for _, item := range store.items {
			if item.expiresAt.After(now) {
				newItems = append(newItems, item)
			}
		}
		store.items = newItems
		store.mu.Unlock()

		if len(newItems) == 0 {
			delete(s.keys, key)
		}
	}
}

func (s *Store[T]) ClearKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.keys, key)
}

func (s *Store[T]) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys = make(map[string]*keyStore[T])
}

func (s *Store[T]) SetCapacity(capacity int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.capacity = capacity
	if capacity <= 0 {
		return
	}

	for _, ks := range s.keys {
		ks.mu.Lock()
		if len(ks.items) > capacity {
			ks.items = ks.items[len(ks.items)-capacity:]
		}
		ks.mu.Unlock()
	}
}

func (s *Store[T]) SetTTL(newTTL time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldTTL := s.ttl
	diff := newTTL - oldTTL
	s.ttl = newTTL

	for _, ks := range s.keys {
		ks.mu.Lock()
		for _, item := range ks.items {
			item.expiresAt = item.expiresAt.Add(diff)
		}
		ks.mu.Unlock()
	}
}
