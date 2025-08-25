package storage

import (
	"container/heap"
	"time"
)

type Store struct {
	shards      []Shard
	granularity time.Duration
}

func New(nShards int, granularity time.Duration) *Store {
	s := &Store{
		shards:      make([]Shard, nShards),
		granularity: granularity,
	}
	for i := range s.shards {
		s.shards[i].userData = make(map[string][]*Message)
		s.shards[i].expHeap = make(MsgHeap, 0)
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

func (s *Store) Push(username, text string, ttl time.Duration) {
	now := time.Now().UnixNano()
	expire := now + ttl.Nanoseconds()
	msg := &Message{
		Username: username,
		Text:     text,
		Time:     now,
		TTL:      expire,
	}

	sh := s.getShard(username)
	sh.mu.Lock()
	sh.userData[username] = append(sh.userData[username], msg)
	heap.Push(&sh.expHeap, msg)
	sh.mu.Unlock()
}

func (s *Store) ForEach(username string, fn func(msg *Message)) {
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

func (s *Store) Cleanup() {
	now := time.Now().UnixNano()
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.Lock()
		for len(sh.expHeap) > 0 {
			msg := sh.expHeap[0]
			if msg.TTL > now {
				break
			}
			heap.Pop(&sh.expHeap)

			userMsgs := sh.userData[msg.Username]
			n := userMsgs[:0]
			for _, m := range userMsgs {
				if m != msg {
					n = append(n, m)
				}
			}
			if len(n) == 0 {
				delete(sh.userData, msg.Username)
			} else {
				sh.userData[msg.Username] = n
			}
		}
		sh.mu.Unlock()
	}
}

func (s *Store) CleanupUser(username string) {
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
