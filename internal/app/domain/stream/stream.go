package stream

import (
	"sync"
	"sync/atomic"
	"time"
	"twitchspam/internal/app/ports"
)

type Stream struct {
	mu sync.RWMutex

	channelID   string
	channelName string
	streamID    string
	category    string
	isLive      atomic.Bool

	stats ports.StatsPort
}

func NewStream(channelName string) *Stream {
	s := &Stream{
		stats: NewStats(),
	}

	s.SetChannelName(channelName)
	return s
}

func (s *Stream) Stats() ports.StatsPort {
	return s.stats
}

func (s *Stream) IsLive() bool {
	return s.isLive.Load()
}

func (s *Stream) SetIslive(v bool) {
	s.isLive.Store(v)
}

func (s *Stream) StreamID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.streamID
}

func (s *Stream) SetStreamID(streamID string) {
	s.mu.Lock()
	s.streamID = streamID
	s.mu.Unlock()
}

func (s *Stream) ChannelID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.channelID
}

func (s *Stream) SetChannelID(channelID string) {
	s.mu.Lock()
	s.channelID = channelID
	s.mu.Unlock()
}

func (s *Stream) ChannelName() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.channelName
}

func (s *Stream) SetChannelName(channelName string) {
	s.mu.Lock()
	s.channelName = channelName
	s.mu.Unlock()
}

func (s *Stream) SetCategory(category string) {
	s.mu.Lock()
	if s.category == category {
		s.mu.Unlock()
		return
	}
	s.category = category
	s.mu.Unlock()

	if s.stats != nil {
		s.stats.AddCategoryChange(category, time.Now())
	}
}

func (s *Stream) Category() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.category
}
