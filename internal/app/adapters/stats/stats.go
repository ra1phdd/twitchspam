package stats

import (
	"fmt"
	"math"
	"sort"
	"time"
	"twitchspam/pkg/logger"
)

type Stats struct {
	log logger.Logger

	startTime time.Time
	endTime   time.Time
	online    struct {
		maxViewers int
		sumViewers int64
		count      int
	}
	countFirstMessages int
	countMessages      map[string]int
	countBans          map[string]int
	countTimeouts      map[string]int
}

func New(log logger.Logger) *Stats {
	return &Stats{
		log: log,
	}
}

func (s *Stats) SetStartTime(t time.Time) {
	*s = Stats{
		startTime: t,
		endTime:   t,
		online: struct {
			maxViewers int
			sumViewers int64
			count      int
		}{maxViewers: 0, sumViewers: 0, count: 0},
		countMessages: make(map[string]int),
		countBans:     make(map[string]int),
		countTimeouts: make(map[string]int),
	}
}

func (s *Stats) SetEndTime(t time.Time) {
	s.endTime = t
}

func (s *Stats) SetOnline(viewers int) {
	if viewers > s.online.maxViewers {
		s.online.maxViewers = viewers
	}

	s.online.sumViewers = int64(viewers)
	s.online.count++
}

func (s *Stats) CountFirstMessages() {
	s.countFirstMessages++
}

func (s *Stats) AddMessage(username string) {
	if s.countMessages == nil {
		return
	}

	s.countMessages[username]++
}

func (s *Stats) AddBan(username string) {
	if s.countBans == nil {
		return
	}

	s.countBans[username]++
}

func (s *Stats) AddTimeout(username string) {
	if s.countTimeouts == nil {
		return
	}

	s.countTimeouts[username]++
}

func (s *Stats) GetStats() string {
	if s.startTime.IsZero() {
		return "нет данных"
	}

	msg := fmt.Sprintf("длительность стрима: %s • средний онлайн: %.0f • максимальный онлайн: %d • всего сообщений: %d • первых сообщений: %d • скорость сообщений: %.1f/сек • кол-во банов: %d • кол-во мутов: %d • топ 3 модератора за стрим: ",
		s.endTime.Sub(s.startTime), math.Round(float64(s.online.sumViewers/int64(s.online.count))), s.online.maxViewers, len(s.countMessages),
		s.countFirstMessages, float64(len(s.countMessages))/s.endTime.Sub(s.startTime).Seconds(), len(s.countBans), len(s.countTimeouts))

	combined := make(map[string]int)
	for k, v := range s.countBans {
		combined[k] += v
	}
	for k, v := range s.countTimeouts {
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
	msg += "• посмотреть свою стату - !am stats"

	return msg
}

func (s *Stats) GetModeratorStats(username string) string {
	return fmt.Sprintf("кол-во сообщений за стрим: %d • кол-во банов: %d • кол-во мутов: %d", s.countMessages[username], s.countBans[username], s.countTimeouts[username])
}

func (s *Stats) GetUserStats(username string) string {
	return fmt.Sprintf("кол-во сообщений за стрим: %d", s.countMessages[username])
}
