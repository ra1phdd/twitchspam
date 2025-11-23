package storage

import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
	"twitchspam/internal/app/infrastructure/timers"
	"twitchspam/internal/app/ports"
)

type Store[T any] struct {
	cache sync.Map

	mu         sync.RWMutex
	lastAccess map[string]*atomic.Int64

	ttl, subTTL   atomic.Int64
	cap, subCap   atomic.Int32
	mode, subMode atomic.Int32

	timers ports.TimersPort
}

type entry[T any] struct {
	Value        map[string]*T
	OrderedValue list.List

	mu          sync.RWMutex
	lastAccess  map[string]*atomic.Int64
	orderedRefs map[string]*list.Element
}

type orderedEntry[T any] struct {
	Key   string
	Value *T
}

type ExpirationMode int32

const (
	ExpireAfterWrite ExpirationMode = iota
	ExpireAfterAccess
)

type StoreOption[T any] func(*Store[T])

func WithTTLV2[T any](ttl time.Duration) StoreOption[T] {
	return func(s *Store[T]) {
		s.ttl.Store(ttl.Nanoseconds())
	}
}

func WithCapacity[T any](newCap int32) StoreOption[T] {
	return func(s *Store[T]) {
		s.cap.Store(newCap)
	}
}

func WithSubTTL[T any](ttl time.Duration) StoreOption[T] {
	return func(s *Store[T]) {
		s.subTTL.Store(ttl.Nanoseconds())
	}
}

func WithSubCapacity[T any](newCap int32) StoreOption[T] {
	return func(s *Store[T]) {
		s.subCap.Store(newCap)
	}
}

func WithMode[T any](mode ExpirationMode) StoreOption[T] {
	return func(s *Store[T]) {
		s.mode.Store(int32(mode))
	}
}

func WithSubMode[T any](mode ExpirationMode) StoreOption[T] {
	return func(s *Store[T]) {
		s.subMode.Store(int32(mode))
	}
}

func New[T any](opts ...StoreOption[T]) *Store[T] {
	s := &Store[T]{
		lastAccess: make(map[string]*atomic.Int64),
		timers:     timers.NewTimingWheel(100*time.Millisecond, 600),
	}

	s.ttl.Store(0)
	s.subTTL.Store(0)
	s.cap.Store(0)
	s.subCap.Store(0)
	s.mode.Store(int32(ExpireAfterWrite))
	s.subMode.Store(int32(ExpireAfterWrite))

	for _, opt := range opts {
		opt(s)
	}

	return s
}

func (s *Store[T]) Push(key, subKey string, val T, ttl *time.Duration) {
	if ttl == nil {
		d := time.Duration(s.ttl.Load())
		ttl = &d
	}

	raw, ok := s.cache.Load(key)
	if !ok || raw == nil {
		ent := &entry[T]{
			Value:        make(map[string]*T),
			OrderedValue: list.List{},
			lastAccess:   make(map[string]*atomic.Int64),
			orderedRefs:  make(map[string]*list.Element),
		}
		s.cache.Store(key, ent)
		s.addTimer(key, *ttl, ExpirationMode(s.mode.Load()))
		raw = ent

		s.mu.Lock()
		if _, exists := s.lastAccess[key]; !exists {
			s.lastAccess[key] = &atomic.Int64{}
		}
		s.lastAccess[key].Store(time.Now().Unix())
		s.mu.Unlock()
	}

	el := raw.(*entry[T])
	el.mu.Lock()

	el.Value[subKey] = &val
	if elem, exists := el.orderedRefs[subKey]; exists {
		el.OrderedValue.MoveToBack(elem)
		elem.Value.(*orderedEntry[T]).Value = &val
	} else {
		elem = el.OrderedValue.PushBack(&orderedEntry[T]{Key: subKey, Value: &val})
		el.orderedRefs[subKey] = elem
	}
	if _, exists := el.lastAccess[subKey]; !exists {
		el.lastAccess[subKey] = &atomic.Int64{}
	}
	el.lastAccess[subKey].Store(time.Now().Unix())

	if el.OrderedValue.Len() > int(s.subCap.Load()) {
		front := el.OrderedValue.Front()
		if front != nil {
			fk := front.Value.(*orderedEntry[T]).Key
			delete(el.Value, fk)
			delete(el.lastAccess, fk)
			delete(el.orderedRefs, fk)
			el.OrderedValue.Remove(front)
		}
	}
	el.mu.Unlock()

	s.addSubTimer(key, subKey, *ttl, ExpirationMode(s.subMode.Load()))
}

func (s *Store[T]) Update(key string, subKey string, updateFn func(current *T, exists bool) *T) {
	raw, ok := s.cache.Load(key)
	if !ok || raw == nil {
		return
	}
	el := raw.(*entry[T])

	s.mu.Lock()
	if _, exists := s.lastAccess[key]; !exists {
		s.lastAccess[key] = &atomic.Int64{}
	}
	s.lastAccess[key].Store(time.Now().Unix())
	s.mu.Unlock()

	current, exists := el.Value[subKey]
	el.mu.Lock()

	newVal := updateFn(current, exists)
	if newVal == nil {
		if elem, ok := el.orderedRefs[subKey]; ok {
			el.OrderedValue.Remove(elem)
			delete(el.orderedRefs, subKey)
		}
		delete(el.Value, subKey)
		delete(el.lastAccess, subKey)
		return
	}

	el.Value[subKey] = newVal

	if elem, ok := el.orderedRefs[subKey]; ok {
		elem.Value.(*orderedEntry[T]).Value = newVal
		el.OrderedValue.MoveToBack(elem)
	} else {
		elem = el.OrderedValue.PushBack(&orderedEntry[T]{Key: subKey, Value: newVal})
		el.orderedRefs[subKey] = elem
	}

	if _, exists := el.lastAccess[subKey]; !exists {
		el.lastAccess[subKey] = &atomic.Int64{}
	}
	el.lastAccess[subKey].Store(time.Now().Unix())
	el.mu.Unlock()

	s.timers.RemoveTimer(key + subKey)
	s.addSubTimer(key, subKey, time.Duration(s.subTTL.Load()), ExpirationMode(s.subMode.Load()))
}

func (s *Store[T]) GetAllData() map[string]map[string]T {
	result := make(map[string]map[string]T)

	s.cache.Range(func(k, v any) bool {
		key := k.(string)
		ent := v.(*entry[T])

		ent.mu.RLock()
		subMap := make(map[string]T)
		for sk, val := range ent.Value {
			if val != nil {
				subMap[sk] = *val
			}
		}
		ent.mu.RUnlock()

		result[key] = subMap
		return true
	})

	return result
}

func (s *Store[T]) GetAll(key string) map[string]T {
	raw, ok := s.cache.Load(key)
	if !ok || raw == nil {
		return nil
	}

	ent := raw.(*entry[T])
	ent.mu.RLock()

	result := make(map[string]T)
	for sk, val := range ent.Value {
		if val != nil {
			result[sk] = *val
		}
	}
	ent.mu.RUnlock()

	return result
}

func (s *Store[T]) Get(key, subKey string) (T, bool) {
	var zero T

	raw, ok := s.cache.Load(key)
	if !ok || raw == nil {
		return zero, false
	}
	ent := raw.(*entry[T])

	ent.mu.RLock()
	val, exists := ent.Value[subKey]
	if exists && val != nil {
		if la, ok := ent.lastAccess[subKey]; ok {
			la.Store(time.Now().Unix())
		}
	}
	ent.mu.RUnlock()

	if !exists || val == nil {
		return zero, false
	}
	return *val, true
}

func (s *Store[T]) Len(key string) int {
	raw, ok := s.cache.Load(key)
	if !ok || raw == nil {
		return 0
	}
	ent := raw.(*entry[T])

	ent.mu.RLock()
	defer ent.mu.RUnlock()
	return len(ent.Value)
}

func (s *Store[T]) ForEach(key string, fn func(val *T)) {
	raw, ok := s.cache.Load(key)
	if !ok || raw == nil {
		return
	}
	ent := raw.(*entry[T])

	ent.mu.RLock()
	for _, val := range ent.Value {
		if val != nil {
			fn(val)
		}
	}
	ent.mu.RUnlock()
}

func (s *Store[T]) ClearKey(key string) {
	raw, ok := s.cache.Load(key)
	if ok && raw != nil {
		s.cache.Delete(key)
	}

	s.mu.Lock()
	delete(s.lastAccess, key)
	s.mu.Unlock()
}

func (s *Store[T]) ClearAll() {
	s.cache.Clear()

	s.mu.Lock()
	s.lastAccess = make(map[string]*atomic.Int64)
	s.mu.Unlock()
}

func (s *Store[T]) SetCapacity(newCap int32) {
	s.cap.Store(newCap)
	s.subCap.Store(newCap)
}

func (s *Store[T]) GetCapacity() int32 {
	return s.cap.Load()
}

func (s *Store[T]) SetTTL(newTTL time.Duration) {
	oldTTL := time.Duration(s.ttl.Load())
	s.ttl.Store(newTTL.Nanoseconds())

	diff := newTTL - oldTTL
	for id, remaining := range s.timers.ActiveTimers() {
		newRemaining := remaining + diff
		s.timers.UpdateTimerTTL(id, newRemaining)
	}
}

func (s *Store[T]) GetTTL() time.Duration {
	return time.Duration(s.ttl.Load())
}

func (s *Store[T]) addTimer(key string, ttl time.Duration, mode ExpirationMode) {
	if ttl <= 0 {
		return
	}

	s.timers.AddTimer(key, ttl, false, map[string]any{
		"key": key,
	}, func(args map[string]any) {
		if mode == ExpireAfterAccess {
			raw, ok := s.cache.Load(key)
			if !ok || raw == nil {
				return
			}

			s.mu.RLock()
			la, exists := s.lastAccess[key]
			s.mu.RUnlock()
			if !exists || time.Since(time.Unix(0, la.Load())) < ttl {
				s.addTimer(key, ttl, mode)
				return
			}
		}

		s.cache.Delete(args["key"].(string))
		s.mu.Lock()
		delete(s.lastAccess, key)
		s.mu.Unlock()
	})
}

func (s *Store[T]) addSubTimer(key, subKey string, ttl time.Duration, mode ExpirationMode) {
	if ttl <= 0 {
		return
	}

	s.timers.AddTimer(key+subKey, ttl, false, map[string]any{
		"key":     key,
		"sub_key": subKey,
	}, func(args map[string]any) {
		raw, ok := s.cache.Load(args["key"].(string))
		if !ok || raw == nil {
			return
		}
		ent := raw.(*entry[T])

		ent.mu.Lock()
		defer ent.mu.Unlock()

		if mode == ExpireAfterAccess {
			la, exists := ent.lastAccess[args["sub_key"].(string)]
			if exists && time.Since(time.Unix(0, la.Load())) < ttl {
				s.addSubTimer(key, subKey, ttl, mode)
				return
			}
		}

		delete(ent.Value, args["sub_key"].(string))
		delete(ent.lastAccess, args["sub_key"].(string))
		if elem, ok := ent.orderedRefs[args["sub_key"].(string)]; ok {
			ent.OrderedValue.Remove(elem)
			delete(ent.orderedRefs, args["sub_key"].(string))
		}
	})
}
