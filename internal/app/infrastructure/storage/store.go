package storage

import (
	"container/heap"
	"time"
)

type Store[T any] struct {
	shards          []Shard[T]
	granularity     time.Duration
	getLimitPerUser func() int
}

type Item[T any] struct {
	Username string
	Data     T
	Time     int64
	TTL      int64
}

func New[T any](nShards int, granularity time.Duration, getLimitPerUser func() int) *Store[T] {
	s := &Store[T]{
		shards:          make([]Shard[T], nShards),
		granularity:     granularity,
		getLimitPerUser: getLimitPerUser,
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

	sh.userMu.RLock()
	defer sh.userMu.RUnlock()

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

	sh.userMu.Lock()
	q := sh.userData[username]
	q = append(q, item)
	messageLimitPerUser := s.getLimitPerUser()

	if len(q) > messageLimitPerUser {
		q = q[len(q)-messageLimitPerUser:]
	}
	sh.userData[username] = q
	sh.userMu.Unlock()

	sh.heapMu.Lock()
	heap.Push(&sh.expHeap, item)
	sh.heapMu.Unlock()
}

func (s *Store[T]) ForEach(username string, fn func(item *Item[T])) {
	sh := s.getShard(username)

	sh.userMu.RLock()
	msgs, ok := sh.userData[username]
	if !ok {
		sh.userMu.RUnlock()
		return
	}

	for _, msg := range msgs {
		fn(msg)
	}
	sh.userMu.RUnlock()
}

func (s *Store[T]) Cleanup() {
	now := time.Now().UnixNano()
	for i := range s.shards {
		sh := &s.shards[i]

		for {
			sh.userMu.Lock()
			sh.heapMu.Lock()
			if len(sh.expHeap) == 0 {
				sh.heapMu.Unlock()
				sh.userMu.Unlock()
				break
			}

			item := sh.expHeap[0]
			if item.TTL > now {
				sh.heapMu.Unlock()
				sh.userMu.Unlock()
				break
			}
			heap.Pop(&sh.expHeap)
			sh.heapMu.Unlock()

			userItems, ok := sh.userData[item.Username]
			if ok && userItems != nil {
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
			sh.userMu.Unlock()
		}
	}
}

func (s *Store[T]) CleanupUser(username string) {
	sh := s.getShard(username)

	sh.userMu.Lock()
	msgs, ok := sh.userData[username]
	if !ok {
		sh.userMu.Unlock()
		return
	}
	delete(sh.userData, username)
	sh.userMu.Unlock()

	if len(msgs) == 0 {
		return
	}

	sh.heapMu.Lock()
	if len(sh.expHeap) > 0 {
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
	}
	sh.heapMu.Unlock()
}
