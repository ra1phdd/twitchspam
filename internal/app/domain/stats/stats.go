package stats

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"twitchspam/internal/app/ports"
)

type Stats struct {
	mu sync.Mutex

	startTime time.Time
	endTime   time.Time
	online    struct {
		maxViewers int
		sumViewers int64
		count      int
	}
	countMessages map[string]int
	countDeletes  map[string]int
	countTimeouts map[string]int
	countBans     map[string]int
}

func New() *Stats {
	return &Stats{}
}

func (s *Stats) SetStartTime(t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	*s = Stats{
		startTime: t,
		endTime:   t,
		online: struct {
			maxViewers int
			sumViewers int64
			count      int
		}{maxViewers: 0, sumViewers: 0, count: 0},
		countMessages: make(map[string]int),
		countDeletes:  make(map[string]int),
		countTimeouts: make(map[string]int),
		countBans:     make(map[string]int),
	}
}

func (s *Stats) GetStartTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.startTime
}

func (s *Stats) SetEndTime(t time.Time) {
	s.mu.Lock()
	s.endTime = t
	s.mu.Unlock()
}

func (s *Stats) GetEndTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.endTime
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

func (s *Stats) GetStats() *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.startTime.IsZero() {
		return &ports.AnswerType{
			Text:    []string{"нет данных за последний стрим!"},
			IsReply: false,
		}
	}

	var countBans, countTimeouts, countDeletes, countMessages int
	combined := make(map[string]int)
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

	var list []kv
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
		s.endTime.Sub(s.startTime).Round(time.Second).String(), math.Round(avgViewers), s.online.maxViewers,
		countMessages, len(s.countMessages), float64(countMessages)/s.endTime.Sub(s.startTime).Seconds(), countBans, countTimeouts, countDeletes)

	top := 3
	if len(list) < 3 {
		top = len(list)
	}

	for i := 0; i < top; i++ {
		if i > 0 {
			msg += ", "
		}
		msg += fmt.Sprintf("%s (%d)", list[i].key, list[i].value)
	}
	msg += "• посмотреть свою стату - !stats"

	return &ports.AnswerType{
		Text:    []string{msg},
		IsReply: false,
	}
}

func (s *Stats) GetUserStats(username string) *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.startTime.IsZero() || len(s.countMessages) == 0 {
		return &ports.AnswerType{
			Text:    []string{"нет данных за последний стрим!"},
			IsReply: false,
		}
	}

	type kv struct {
		Key   string
		Value int
	}

	var pairs []kv
	for k, v := range s.countMessages {
		pairs = append(pairs, kv{k, v})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Value > pairs[j].Value
	})
	username = strings.ToLower(strings.TrimPrefix(username, "@"))

	position := -1
	for i, p := range pairs {
		if p.Key == username {
			position = i + 1
			break
		}
	}

	msg := fmt.Sprintf("кол-во сообщений за стрим: %d", s.countMessages[username])
	if position != -1 {
		msg += fmt.Sprintf(" (топ-%d чаттер)", position)
	}

	if s.countBans[username] > 0 || s.countTimeouts[username] > 0 || s.countDeletes[username] > 0 {
		msg += fmt.Sprintf(" • кол-во банов: %d • кол-во мутов: %d • кол-во удаленных сообщений: %d",
			s.countBans[username], s.countTimeouts[username], s.countDeletes[username])
	}

	return &ports.AnswerType{
		Text:    []string{msg},
		IsReply: false,
	}
}

func (s *Stats) GetTopStats(count int) *ports.AnswerType {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.startTime.IsZero() || len(s.countMessages) == 0 {
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
	sb.WriteString(fmt.Sprintf("топ-%d чаттеров по кол-ву сообщений за стрим:", count))

	sep := ", "
	if count > 5 {
		sep = "\n"
	}

	for i := 0; i < count && i < len(pairs); i++ {
		if i > 0 {
			sb.WriteString(sep)
		}
		sb.WriteString(fmt.Sprintf("%s (%d)", pairs[i].Key, pairs[i].Val))
	}

	return &ports.AnswerType{
		Text:    []string{sb.String()},
		IsReply: false,
	}
}
