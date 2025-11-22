package stream

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/ports"
)

type Stats struct {
	channelName string
	stats       SessionStats
	fs          ports.FileServerPort
	cache       ports.CachePort[SessionStats]
	mu          sync.RWMutex
	lastActive  atomic.Int64
}

type SessionStats struct {
	StartStreamTime time.Time
	EndStreamTime   time.Time
	Online          struct {
		MaxViewers int
		SumViewers int64
		Count      int
	}
	CountMessages   map[string]int
	CountDeletes    map[string]int
	CountTimeouts   map[string]int
	CountWarns      map[string]int
	CountBans       map[string]int
	CategoryHistory []CategoryInterval
	StatIntervals   []TimeInterval
}

type CategoryInterval struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
}

type TimeInterval struct {
	Start time.Time
	End   time.Time
}

func newStats(channelName string, fs ports.FileServerPort, cache ports.CachePort[SessionStats]) *Stats {
	s := &Stats{
		channelName: channelName,
		fs:          fs,
		cache:       cache,
		stats: SessionStats{
			CountMessages:   make(map[string]int),
			CountDeletes:    make(map[string]int),
			CountTimeouts:   make(map[string]int),
			CountWarns:      make(map[string]int),
			CountBans:       make(map[string]int),
			CategoryHistory: make([]CategoryInterval, 0),
			StatIntervals:   make([]TimeInterval, 0),
		},
	}

	if stats, ok := cache.Get(channelName); ok {
		s.stats = stats

		var countBans, countWarns, countTimeouts, countDeletes, countMessages int
		for _, v := range stats.CountMessages {
			countMessages += v
		}
		for _, v := range stats.CountDeletes {
			countDeletes += v
		}
		for _, v := range stats.CountTimeouts {
			countTimeouts += v
		}
		for _, v := range stats.CountWarns {
			countWarns += v
		}
		for _, v := range stats.CountBans {
			countBans += v
		}

		var avgViewers float64
		if stats.Online.Count > 0 {
			avgViewers = math.Round(float64(stats.Online.SumViewers) / float64(stats.Online.Count))
		}

		metrics.StreamStartTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(stats.StartStreamTime.Unix()))
		metrics.StreamEndTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(stats.EndStreamTime.Unix()))
		metrics.OnlineViewers.With(prometheus.Labels{"channel": s.channelName}).Set(avgViewers)
		metrics.MessagesPerOnline.With(prometheus.Labels{"channel": s.channelName}).Add(float64(countMessages))
		metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "delete"}).Add(float64(countDeletes))
		metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "timeout"}).Add(float64(countTimeouts))
		metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "warn"}).Add(float64(countWarns))
		metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "ban"}).Add(float64(countBans))
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			s.cache.Set(s.channelName, s.getStatsCopy())
		}
	}()

	return s
}

func (s *Stats) Reset() {
	s.mu.Lock()

	s.stats.StartStreamTime = time.Time{}
	s.stats.EndStreamTime = time.Time{}
	s.stats.Online.MaxViewers = 0
	s.stats.Online.SumViewers = 0
	s.stats.Online.Count = 0
	s.stats.CountMessages = make(map[string]int)
	s.stats.CountDeletes = make(map[string]int)
	s.stats.CountTimeouts = make(map[string]int)
	s.stats.CountBans = make(map[string]int)
	s.stats.CategoryHistory = make([]CategoryInterval, 0)
	s.stats.StatIntervals = make([]TimeInterval, 0)

	s.cache.Set(s.channelName, s.stats)
	metrics.StreamStartTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(s.stats.StartStreamTime.Unix()))
	metrics.StreamEndTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(s.stats.EndStreamTime.Unix()))
	metrics.OnlineViewers.With(prometheus.Labels{"channel": s.channelName}).Set(0)
	metrics.MessagesPerOnline.Delete(prometheus.Labels{"channel": s.channelName})
	metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "delete"}).Set(0)
	metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "timeout"}).Set(0)
	metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "warn"}).Set(0)
	metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "ban"}).Set(0)
	metrics.UserCommands.Delete(prometheus.Labels{"channel": s.channelName})

	s.mu.Unlock()
}

func (s *Stats) SetStartTime(t time.Time) {
	s.mu.Lock()
	s.stats.StartStreamTime = t
	s.stats.EndStreamTime = t
	s.mu.Unlock()
}

func (s *Stats) GetStartTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stats.StartStreamTime
}

func (s *Stats) SetEndTime(t time.Time) {
	s.mu.Lock()
	s.stats.EndStreamTime = t
	metrics.StreamEndTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(s.stats.EndStreamTime.Unix()))
	s.mu.Unlock()
}

func (s *Stats) GetEndTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.stats.EndStreamTime
}

func (s *Stats) SetOnline(viewers int) {
	if viewers <= 0 {
		return
	}

	s.mu.Lock()
	if viewers > s.stats.Online.MaxViewers {
		s.stats.Online.MaxViewers = viewers
	}

	s.stats.Online.SumViewers += int64(viewers)
	s.stats.Online.Count++

	s.markActive(time.Now())
	metrics.OnlineViewers.With(prometheus.Labels{"channel": s.channelName}).Set(float64(viewers))
	s.mu.Unlock()
}

func (s *Stats) AddMessage(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountMessages == nil {
		return
	}

	s.markActive(time.Now())
	s.stats.CountMessages[strings.ToLower(username)]++
	metrics.MessagesPerOnline.With(prometheus.Labels{"channel": s.channelName}).Inc()
}

func (s *Stats) AddDeleted(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountDeletes == nil {
		return
	}

	s.markActive(time.Now())
	s.stats.CountDeletes[strings.ToLower(username)]++
}

func (s *Stats) AddTimeout(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountTimeouts == nil {
		return
	}

	s.markActive(time.Now())
	s.stats.CountTimeouts[strings.ToLower(username)]++
}

func (s *Stats) AddWarn(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountWarns == nil {
		return
	}

	s.markActive(time.Now())
	s.stats.CountWarns[strings.ToLower(username)]++
}

func (s *Stats) AddBan(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountBans == nil {
		return
	}

	s.markActive(time.Now())
	s.stats.CountBans[strings.ToLower(username)]++
}

func (s *Stats) AddCategoryChange(category string, t time.Time) {
	s.mu.Lock()
	if n := len(s.stats.CategoryHistory); n > 0 {
		s.stats.CategoryHistory[n-1].EndTime = t
	}

	s.markActive(time.Now())
	s.stats.CategoryHistory = append(s.stats.CategoryHistory, CategoryInterval{
		Name:      category,
		StartTime: t,
	})
	s.mu.Unlock()
}

func (s *Stats) GetStats() *ports.AnswerType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats.StartStreamTime.IsZero() {
		return &ports.AnswerType{
			Text:    []string{"нет данных за последний стрим!"},
			IsReply: false,
		}
	}

	countMessages, countDeletes, countTimeouts, countBans, combined := s.aggregateStats()

	avgViewers := 0.0
	if s.stats.Online.Count > 0 {
		avgViewers = math.Round(float64(s.stats.Online.SumViewers) / float64(s.stats.Online.Count))
	}

	msg := fmt.Sprintf(
		"длительность стрима: %s • средний онлайн: %.0f • максимальный онлайн: %d • всего сообщений: %d • кол-во чаттеров: %d • скорость сообщений: %.1f/сек • кол-во банов: %d • кол-во мутов: %d • кол-во удаленных сообщений: %d • топ 3 модератора за стрим: %s • посмотреть свою стату - !stats",
		domain.FormatDuration(s.stats.EndStreamTime.Sub(s.stats.StartStreamTime)),
		avgViewers,
		s.stats.Online.MaxViewers,
		countMessages,
		len(s.stats.CountMessages),
		float64(countMessages)/s.stats.EndStreamTime.Sub(s.stats.StartStreamTime).Seconds(),
		countBans,
		countTimeouts,
		countDeletes,
		topN(combined, 3),
	)

	if gaps := s.formatGaps(); gaps != "" {
		msg += fmt.Sprintf(" (статистика отсутствовала %s)", gaps)
	}

	return &ports.AnswerType{Text: []string{msg}, IsReply: false}
}

func (s *Stats) GetUserStats(username string) *ports.AnswerType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats.StartStreamTime.IsZero() || len(s.stats.CountMessages) == 0 {
		return &ports.AnswerType{Text: []string{"нет данных за последний стрим!"}, IsReply: false}
	}

	username = strings.ToLower(strings.TrimPrefix(username, "@"))
	position := 1
	for _, v := range s.stats.CountMessages {
		if v > s.stats.CountMessages[username] {
			position++
		}
	}

	msg := fmt.Sprintf(
		"статистика %s - кол-во сообщений за стрим: %d • топ-%d чаттер",
		username, s.stats.CountMessages[username], position,
	)

	if s.stats.CountBans[username] > 0 || s.stats.CountTimeouts[username] > 0 || s.stats.CountDeletes[username] > 0 {
		msg += fmt.Sprintf(" • кол-во банов: %d • кол-во мутов: %d • кол-во удаленных сообщений: %d",
			s.stats.CountBans[username], s.stats.CountTimeouts[username], s.stats.CountDeletes[username])
	}

	if gaps := s.formatGaps(); gaps != "" {
		msg += fmt.Sprintf(" (статистика отсутствовала %s)", gaps)
	}

	return &ports.AnswerType{Text: []string{msg}, IsReply: false}
}

func (s *Stats) GetTopStats(count int) *ports.AnswerType {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.stats.StartStreamTime.IsZero() || len(s.stats.CountMessages) == 0 {
		return &ports.AnswerType{Text: []string{"нет данных за последний стрим!"}, IsReply: false}
	}

	switch {
	case count <= 0:
		count = 10
	case count > 100:
		count = 100
	}

	pairs := make([]struct {
		Key string
		Val int
	}, 0, len(s.stats.CountMessages))
	for k, v := range s.stats.CountMessages {
		pairs = append(pairs, struct {
			Key string
			Val int
		}{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].Val > pairs[j].Val })

	sep := ", "
	if count > 10 {
		sep = "\n"
	}

	msg := fmt.Sprintf("топ-%d чаттеров по кол-ву сообщений за стрим: ", count)
	for i := 0; i < count && i < len(pairs); i++ {
		if i > 0 {
			msg += sep
		}
		msg += fmt.Sprintf("%s (%d)", pairs[i].Key, pairs[i].Val)
	}

	if gaps := s.formatGaps(); gaps != "" {
		msg += fmt.Sprintf(" (статистика отсутствовала %s)", gaps)
	}

	if count > 10 {
		key, err := s.fs.UploadToHaste(msg)
		if err != nil {
			return &ports.AnswerType{Text: []string{"неизвестная ошибка!"}, IsReply: true}
		}
		msg = s.fs.GetURL(key)
	}

	return &ports.AnswerType{Text: []string{msg}, IsReply: false}
}

func (s *Stats) aggregateStats() (countMessages, countDeletes, countTimeouts, countBans int, combined map[string]int) {
	combined = make(map[string]int)
	for _, v := range s.stats.CountMessages {
		countMessages += v
	}
	for k, v := range s.stats.CountDeletes {
		countDeletes += v
		combined[k] += v
	}
	for k, v := range s.stats.CountTimeouts {
		countTimeouts += v
		combined[k] += v
	}
	for k, v := range s.stats.CountBans {
		countBans += v
		combined[k] += v
	}
	return
}

func (s *Stats) formatGaps() string {
	if len(s.stats.StatIntervals) <= 1 {
		return ""
	}
	var gaps []string
	start := s.stats.StartStreamTime
	for _, interval := range s.stats.StatIntervals {
		if interval.Start.Sub(start) >= 5*time.Minute {
			gaps = append(gaps, fmt.Sprintf("с %s по %s", start.Format("15:04"), interval.Start.Format("15:04")))
		}
		start = interval.End
	}
	return strings.Join(gaps, ", ")
}

func topN(m map[string]int, n int) string {
	type kv struct {
		key   string
		value int
	}
	list := make([]kv, 0, len(m))
	for k, v := range m {
		list = append(list, kv{k, v})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].value > list[j].value })

	top := n
	if len(list) < n {
		top = len(list)
	}

	if top == 0 {
		return "не найдены"
	}

	var parts []string
	for i := 0; i < top; i++ {
		parts = append(parts, fmt.Sprintf("%s (%d)", list[i].key, list[i].value))
	}
	return strings.Join(parts, ", ")
}

func (s *Stats) getStatsCopy() SessionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	copyMap := func(orig map[string]int) map[string]int {
		newMap := make(map[string]int, len(orig))
		for k, v := range orig {
			newMap[k] = v
		}
		return newMap
	}

	statsCopy := s.stats
	statsCopy.CountMessages = copyMap(s.stats.CountMessages)
	statsCopy.CountDeletes = copyMap(s.stats.CountDeletes)
	statsCopy.CountTimeouts = copyMap(s.stats.CountTimeouts)
	statsCopy.CountWarns = copyMap(s.stats.CountWarns)
	statsCopy.CountBans = copyMap(s.stats.CountBans)

	statsCopy.CategoryHistory = append([]CategoryInterval(nil), s.stats.CategoryHistory...)
	statsCopy.StatIntervals = append([]TimeInterval(nil), s.stats.StatIntervals...)

	return statsCopy
}

func (s *Stats) markActive(t time.Time) {
	n := len(s.stats.StatIntervals)
	if n == 0 || t.Sub(s.stats.StatIntervals[n-1].End) > 5*time.Minute {
		s.stats.StatIntervals = append(s.stats.StatIntervals, TimeInterval{
			Start: t,
			End:   t,
		})
	} else {
		s.stats.StatIntervals[n-1].End = t
	}
}
