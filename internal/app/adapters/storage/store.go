package storage

import (
	"container/heap"
	"time"
)

type Store[T any] struct {
	shards      []Shard[T]
	granularity time.Duration
}

type Item[T any] struct {
	Username string
	Data     T
	Time     int64
	TTL      int64
}

func New[T any](nShards int, granularity time.Duration) *Store[T] {
	s := &Store[T]{
		shards:      make([]Shard[T], nShards),
		granularity: granularity,
	}
	for i := range s.shards {
		s.shards[i].userData = make(map[string][]*Item[T])
		s.shards[i].expHeap = make(ItemHeap[T], 0)
		heap.Init(&s.shards[i].expHeap)
	}
	go func() {
		ticker := time.NewTicker(granularity)
		defer ticker.Stop()
		for range ticker.C {
			s.Cleanup()
		}
	}()
	return s
}

func (s *Store[T]) Len(username string) int {
	sh := s.getShard(username)
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	return len(sh.userData[username])
}

func (s *Store[T]) Push(username string, data T, ttl time.Duration) {
	now := time.Now().UnixNano()
	expire := now + ttl.Nanoseconds()
	item := &Item[T]{
		Username: username,
		Data:     data,
		Time:     now,
		TTL:      expire,
	}

	sh := s.getShard(username)
	sh.mu.Lock()
	sh.userData[username] = append(sh.userData[username], item)
	heap.Push(&sh.expHeap, item)
	sh.mu.Unlock()
}

func (s *Store[T]) ForEach(username string, fn func(item *Item[T])) {
	sh := s.getShard(username)
	sh.mu.RLock()
	msgs, ok := sh.userData[username]
	if !ok {
		sh.mu.RUnlock()
		return
	}
	for _, msg := range msgs {
		fn(msg)
	}
	sh.mu.RUnlock()
}

func (s *Store[T]) Cleanup() {
	now := time.Now().UnixNano()
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.Lock()
		for len(sh.expHeap) > 0 {
			item := sh.expHeap[0]
			if item.TTL > now {
				break
			}
			heap.Pop(&sh.expHeap)

			userItems := sh.userData[item.Username]
			newItems := userItems[:0]
			for _, it := range userItems {
				if it != item {
					newItems = append(newItems, it)
				}
			}
			if len(newItems) == 0 {
				delete(sh.userData, item.Username)
			} else {
				sh.userData[item.Username] = newItems
			}
		}
		sh.mu.Unlock()
	}
}

func (s *Store[T]) CleanupUser(username string) {
	sh := s.getShard(username)
	sh.mu.Lock()
	defer sh.mu.Unlock()

	msgs, ok := sh.userData[username]
	if !ok {
		return
	}

	newHeap := sh.expHeap[:0]
	for _, m := range sh.expHeap {
		remove := false
		for _, um := range msgs {
			if m == um {
				remove = true
				break
			}
		}
		if !remove {
			newHeap = append(newHeap, m)
		}
	}
	sh.expHeap = newHeap
	heap.Init(&sh.expHeap)

	delete(sh.userData, username)
}
