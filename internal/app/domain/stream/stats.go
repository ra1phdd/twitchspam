package stream

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/ports"
)

type Stats struct {
	fs ports.FileServerPort
	mu sync.Mutex

	startTime       time.Time
	startStreamTime time.Time
	endStreamTime   time.Time
	online          struct {
		maxViewers int
		sumViewers int64
		count      int
	}
	countMessages map[string]int
	countDeletes  map[string]int
	countTimeouts map[string]int
	countBans     map[string]int

	categoryHistory []CategoryInterval
}

type CategoryInterval struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
}

func newStats(fs ports.FileServerPort) *Stats {
	return &Stats{fs: fs}
}

func (s *Stats) SetStartTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.startTime = time.Now()
	s.startStreamTime = t
	s.endStreamTime = t
	s.online = struct {
		maxViewers int
		sumViewers int64
		count      int
	}{}
	s.countMessages = make(map[string]int)
	s.countDeletes = make(map[string]int)
	s.countTimeouts = make(map[string]int)
	s.countBans = make(map[string]int)
}

func (s *Stats) GetStartTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.startStreamTime
}

func (s *Stats) SetEndTime(t time.Time) {
	s.mu.Lock()
	s.endStreamTime = t
	s.mu.Unlock()
}

func (s *Stats) GetEndTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.endStreamTime
}

func (s *Stats) SetOnline(viewers int) {
	if viewers <= 0 {
		return
	}

	s.mu.Lock()
	if viewers > s.online.maxViewers {
		s.online.maxViewers = viewers
	}

	s.online.sumViewers += int64(viewers)
	s.online.count++
	s.mu.Unlock()
}

func (s *Stats) AddMessage(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.countMessages == nil {
		return
	}

	s.countMessages[strings.ToLower(username)]++
}

func (s *Stats) AddDeleted(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.countDeletes == nil {
		return
	}

	s.countDeletes[strings.ToLower(username)]++
}

func (s *Stats) AddTimeout(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.countTimeouts == nil {
		return
	}

	s.countTimeouts[strings.ToLower(username)]++
}

func (s *Stats) AddBan(username string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.countBans == nil {
		return
	}

	s.countBans[strings.ToLower(username)]++
}

func (s *Stats) AddCategoryChange(category string, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n := len(s.categoryHistory); n > 0 {
		s.categoryHistory[n-1].EndTime = t
	}

	s.categoryHistory = append(s.categoryHistory, CategoryInterval{
		Name:      category,
		StartTime: t,
	})
}

func (s *Stats) GetStats() *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.startStreamTime.IsZero() {
		return &ports.AnswerType{
			Text:    []string{"нет данных за последний стрим!"},
			IsReply: false,
		}
	}

	combined := make(map[string]int)
	var countBans, countTimeouts, countDeletes, countMessages int
	for _, v := range s.countMessages {
		countMessages += v
	}
	for k, v := range s.countDeletes {
		countDeletes += v
		combined[k] += v
	}
	for k, v := range s.countTimeouts {
		countTimeouts += v
		combined[k] += v
	}
	for k, v := range s.countBans {
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
	if s.online.count > 0 {
		avgViewers = math.Round(float64(s.online.sumViewers) / float64(s.online.count))
	}

	msg := fmt.Sprintf("длительность стрима: %s • средний онлайн: %.0f • максимальный онлайн: %d • всего сообщений: %d • кол-во чаттеров: %d • скорость сообщений: %.1f/сек • кол-во банов: %d • кол-во мутов: %d • кол-во удаленных сообщений: %d • топ 3 модератора за стрим: ",
		domain.FormatDuration(s.endStreamTime.Sub(s.startStreamTime)), math.Round(avgViewers), s.online.maxViewers,
		countMessages, len(s.countMessages), float64(countMessages)/s.endStreamTime.Sub(s.startStreamTime).Seconds(), countBans, countTimeouts, countDeletes)

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

	for i, item := range list[:top] {
		if i > 0 {
			msg += ", "
		}
		msg += fmt.Sprintf("%s (%d)", item.key, item.value)
	}

	msg += " • посмотреть свою стату - !stats"
	diff := s.startTime.Sub(s.startStreamTime)
	if diff >= 5*time.Minute {
		msg += fmt.Sprintf(" (статистика велась с %s)", s.startTime.Format("15:04:05"))
	}

	return &ports.AnswerType{
		Text:    []string{msg},
		IsReply: false,
	}
}

func (s *Stats) GetUserStats(username string) *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.startStreamTime.IsZero() || len(s.countMessages) == 0 {
		return &ports.AnswerType{
			Text:    []string{"нет данных за последний стрим!"},
			IsReply: false,
		}
	}

	username = strings.TrimPrefix(username, "@")
	usernameLower := strings.ToLower(username)

	position := 1
	for _, v := range s.countMessages {
		if v > s.countMessages[usernameLower] {
			position++
		}
	}

	msg := fmt.Sprintf("статистика %s - кол-во сообщений за стрим: %d • топ-%d чаттер", username, s.countMessages[usernameLower], position)
	if s.countBans[usernameLower] > 0 || s.countTimeouts[usernameLower] > 0 || s.countDeletes[usernameLower] > 0 {
		msg += fmt.Sprintf(" • кол-во банов: %d • кол-во мутов: %d • кол-во удаленных сообщений: %d",
			s.countBans[usernameLower], s.countTimeouts[usernameLower], s.countDeletes[usernameLower])
	}

	diff := s.startTime.Sub(s.startStreamTime)
	if diff >= 5*time.Minute {
		msg += fmt.Sprintf(" (статистика велась с %s)", s.startTime.Format("15:04:05"))
	}

	return &ports.AnswerType{
		Text:    []string{msg},
		IsReply: false,
	}
}

func (s *Stats) GetTopStats(count int) *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.startStreamTime.IsZero() || len(s.countMessages) == 0 {
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
	}, 0, len(s.countMessages))

	for k, v := range s.countMessages {
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

	diff := s.startTime.Sub(s.startStreamTime)
	if diff >= 5*time.Minute {
		sb.WriteString(fmt.Sprintf(" (статистика велась с %s)", s.startTime.Format("15:04:05")))
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
