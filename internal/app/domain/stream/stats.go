package stream

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"math"
	"sort"
	"strings"
	"sync"
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
	mu          sync.Mutex
}

type SessionStats struct {
	StartTime       time.Time
	StartStreamTime time.Time
	EndStreamTime   time.Time
	Online          struct {
		MaxViewers int
		SumViewers int64
		Count      int
	}
	CountMessages map[string]int
	CountDeletes  map[string]int
	CountTimeouts map[string]int
	CountBans     map[string]int

	CategoryHistory []CategoryInterval
}

type CategoryInterval struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
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
			CountBans:       make(map[string]int),
			CategoryHistory: make([]CategoryInterval, 0),
		},
	}

	if stats, ok := cache.Get(channelName); ok {
		s.stats = stats

		var countBans, countTimeouts, countDeletes, countMessages int
		for _, v := range s.stats.CountMessages {
			countMessages += v
		}
		for _, v := range s.stats.CountDeletes {
			countDeletes += v
		}
		for _, v := range s.stats.CountTimeouts {
			countTimeouts += v
		}
		for _, v := range s.stats.CountBans {
			countBans += v
		}

		var avgViewers float64
		if s.stats.Online.Count > 0 {
			avgViewers = math.Round(float64(s.stats.Online.SumViewers) / float64(s.stats.Online.Count))
		}

		metrics.StreamStartTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(stats.StartStreamTime.Unix()))
		metrics.StreamEndTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(stats.EndStreamTime.Unix()))
		metrics.OnlineViewers.With(prometheus.Labels{"channel": s.channelName}).Set(avgViewers)
		metrics.MessagesPerStream.With(prometheus.Labels{"channel": s.channelName}).Add(float64(countMessages))
		metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "delete"}).Add(float64(countDeletes))
		metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "timeout"}).Add(float64(countTimeouts))
		metrics.ModerationActions.With(prometheus.Labels{"channel": s.channelName, "action": "ban"}).Add(float64(countBans))
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			s.cache.Set(s.channelName, s.stats)
		}
	}()

	return s
}

func (s *Stats) SetStartTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.StartTime.IsZero() {
		s.stats.StartTime = time.Now()
	}
	s.stats.StartStreamTime = t
	s.stats.EndStreamTime = t
	s.stats.Online.MaxViewers = 0
	s.stats.Online.SumViewers = 0
	s.stats.Online.Count = 0
	s.stats.CountMessages = make(map[string]int)
	s.stats.CountDeletes = make(map[string]int)
	s.stats.CountTimeouts = make(map[string]int)
	s.stats.CountBans = make(map[string]int)
	s.stats.CategoryHistory = make([]CategoryInterval, 0)

	s.cache.Set(s.channelName, s.stats)
	metrics.StreamStartTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(s.stats.StartStreamTime.Unix()))
	metrics.StreamEndTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(s.stats.StartStreamTime.Unix()))
	metrics.OnlineViewers.With(prometheus.Labels{"channel": s.channelName}).Set(0)
	metrics.MessagesPerStream.Delete(prometheus.Labels{"channel": s.channelName})
	metrics.ModerationActions.Delete(prometheus.Labels{"channel": s.channelName})
	metrics.UserCommands.Delete(prometheus.Labels{"channel": s.channelName})
}

func (s *Stats) GetStartTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.stats.StartStreamTime
}

func (s *Stats) SetEndTime(t time.Time) {
	s.mu.Lock()
	s.stats.EndStreamTime = t
	s.mu.Unlock()

	metrics.StreamEndTime.With(prometheus.Labels{"channel": s.channelName}).Set(float64(s.stats.EndStreamTime.Unix()))
}

func (s *Stats) GetEndTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Unlock()

	metrics.OnlineViewers.With(prometheus.Labels{"channel": s.channelName}).Set(float64(viewers))
}

func (s *Stats) AddMessage(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountMessages == nil {
		return
	}

	s.stats.CountMessages[strings.ToLower(username)]++
	metrics.MessagesPerStream.With(prometheus.Labels{"channel": s.channelName}).Inc()
}

func (s *Stats) AddDeleted(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountDeletes == nil {
		return
	}

	s.stats.CountDeletes[strings.ToLower(username)]++
}

func (s *Stats) AddTimeout(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountTimeouts == nil {
		return
	}

	s.stats.CountTimeouts[strings.ToLower(username)]++
}

func (s *Stats) AddBan(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.CountBans == nil {
		return
	}

	s.stats.CountBans[strings.ToLower(username)]++
}

func (s *Stats) AddCategoryChange(category string, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n := len(s.stats.CategoryHistory); n > 0 {
		s.stats.CategoryHistory[n-1].EndTime = t
	}

	s.stats.CategoryHistory = append(s.stats.CategoryHistory, CategoryInterval{
		Name:      category,
		StartTime: t,
	})
}

func (s *Stats) GetStats() *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.StartStreamTime.IsZero() {
		return &ports.AnswerType{
			Text:    []string{"нет данных за последний стрим!"},
			IsReply: false,
		}
	}

	combined := make(map[string]int)
	var countBans, countTimeouts, countDeletes, countMessages int
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

	type kv struct {
		key   string
		value int
	}

	list := make([]kv, 0, len(combined))
	for k, v := range combined {
		list = append(list, kv{k, v})
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].value > list[j].value
	})

	var avgViewers float64
	if s.stats.Online.Count > 0 {
		avgViewers = math.Round(float64(s.stats.Online.SumViewers) / float64(s.stats.Online.Count))
	}

	msg := fmt.Sprintf("длительность стрима: %s • средний онлайн: %.0f • максимальный онлайн: %d • всего сообщений: %d • кол-во чаттеров: %d • скорость сообщений: %.1f/сек • кол-во банов: %d • кол-во мутов: %d • кол-во удаленных сообщений: %d • топ 3 модератора за стрим: ",
		domain.FormatDuration(s.stats.EndStreamTime.Sub(s.stats.StartStreamTime)), math.Round(avgViewers), s.stats.Online.MaxViewers,
		countMessages, len(s.stats.CountMessages), float64(countMessages)/s.stats.EndStreamTime.Sub(s.stats.StartStreamTime).Seconds(), countBans, countTimeouts, countDeletes)

	top := 3
	if len(list) < 3 {
		top = len(list)
	}

	if top != 0 {
		for i, item := range list[:top] {
			if i > 0 {
				msg += ", "
			}
			msg += fmt.Sprintf("%s (%d)", item.key, item.value)
		}
	} else {
		msg += "не найдены"
	}

	msg += " • посмотреть свою стату - !stats"
	diff := s.stats.StartTime.Sub(s.stats.StartStreamTime)
	if diff >= 5*time.Minute {
		msg += fmt.Sprintf(" (статистика велась с %s)", s.stats.StartTime.Format("15:04:05"))
	}

	return &ports.AnswerType{
		Text:    []string{msg},
		IsReply: false,
	}
}

func (s *Stats) GetUserStats(username string) *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.StartStreamTime.IsZero() || len(s.stats.CountMessages) == 0 {
		return &ports.AnswerType{
			Text:    []string{"нет данных за последний стрим!"},
			IsReply: false,
		}
	}

	username = strings.TrimPrefix(username, "@")
	usernameLower := strings.ToLower(username)

	position := 1
	for _, v := range s.stats.CountMessages {
		if v > s.stats.CountMessages[usernameLower] {
			position++
		}
	}

	msg := fmt.Sprintf("статистика %s - кол-во сообщений за стрим: %d • топ-%d чаттер", username, s.stats.CountMessages[usernameLower], position)
	if s.stats.CountBans[usernameLower] > 0 || s.stats.CountTimeouts[usernameLower] > 0 || s.stats.CountDeletes[usernameLower] > 0 {
		msg += fmt.Sprintf(" • кол-во банов: %d • кол-во мутов: %d • кол-во удаленных сообщений: %d",
			s.stats.CountBans[usernameLower], s.stats.CountTimeouts[usernameLower], s.stats.CountDeletes[usernameLower])
	}

	diff := s.stats.StartTime.Sub(s.stats.StartStreamTime)
	if diff >= 5*time.Minute {
		msg += fmt.Sprintf(" (статистика велась с %s)", s.stats.StartTime.Format("15:04:05"))
	}

	return &ports.AnswerType{
		Text:    []string{msg},
		IsReply: false,
	}
}

func (s *Stats) GetTopStats(count int) *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stats.StartStreamTime.IsZero() || len(s.stats.CountMessages) == 0 {
		return &ports.AnswerType{
			Text:    []string{"нет данных за последний стрим!"},
			IsReply: false,
		}
	}

	switch {
	case count < 0:
		return &ports.AnswerType{
			Text:    []string{"не может быть меньше 1 записи!"},
			IsReply: false,
		}
	case count == 0:
		count = 10
	case count > 100:
		return &ports.AnswerType{
			Text:    []string{"максимум 100 записей!"},
			IsReply: false,
		}
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

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Val > pairs[j].Val
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("топ-%d чаттеров по кол-ву сообщений за стрим: ", count))

	sep := ", "
	if count > 10 {
		sep = "\n"
	}

	for i := 0; i < count && i < len(pairs); i++ {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(fmt.Sprintf("%s (%d)", pairs[i].Key, pairs[i].Val))
	}

	diff := s.stats.StartTime.Sub(s.stats.StartStreamTime)
	if diff >= 5*time.Minute {
		sb.WriteString(fmt.Sprintf(" (статистика велась с %s)", s.stats.StartTime.Format("15:04:05")))
	}

	msg := sb.String()
	if count > 10 {
		key, err := s.fs.UploadToHaste(msg)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неизвестная ошибка!"},
				IsReply: true,
			}
		}

		msg = s.fs.GetURL(key)
	}

	return &ports.AnswerType{
		Text:    []string{msg},
		IsReply: false,
	}
}
