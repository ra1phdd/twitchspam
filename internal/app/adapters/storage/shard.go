package storage

import (
	"hash/fnv"
	"sync"
)

type Shard struct {
	mu       sync.RWMutex
	userData map[string][]*Message
	expHeap  MsgHeap
}

func (s *Store) getShard(username string) *Shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(username))
	return &s.shards[h.Sum32()%uint32(len(s.shards))]
}
