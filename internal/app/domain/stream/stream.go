package stream

import (
	"github.com/prometheus/client_golang/prometheus"
	"sync"
	"sync/atomic"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/ports"
)

type Stream struct {
	mu        sync.RWMutex
	onceStart sync.Once

	channelID   string
	channelName string
	category    string
	isLive      atomic.Bool

	stats ports.StatsPort
}

func NewStream(channelName string, fs ports.FileServerPort, cache ports.CachePort[SessionStats]) *Stream {
	s := &Stream{
		stats: newStats(channelName, fs, cache),
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
	metrics.StreamActive.With(prometheus.Labels{"channel": s.channelName}).Set(map[bool]float64{true: 1, false: 0}[v])
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

func (s *Stream) OnceStart() *sync.Once {
	return &s.onceStart
}
