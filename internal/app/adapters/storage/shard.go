package storage

import (
	"hash/fnv"
	"sync"
)

type Shard[T any] struct {
	mu       sync.RWMutex
	userData map[string][]*Item[T]
	expHeap  ItemHeap[T]
}

func (s *Store[T]) getShard(username string) *Shard[T] {
	h := fnv.New32a()
	_, _ = h.Write([]byte(username))
	return &s.shards[h.Sum32()%uint32(len(s.shards))]
}
