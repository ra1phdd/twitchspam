package timers

import (
	"sync"
	"time"
)

type Timer struct {
	ID       string
	Rounds   int
	Interval time.Duration
	Task     func(map[string]any)
	Args     map[string]any
	Repeat   bool
	stop     chan struct{}
}

type slot struct {
	timers map[string]*Timer
}

type TimingWheel struct {
	tickDuration time.Duration
	slots        []*slot
	currentPos   int
	slotsCount   int
	mutex        sync.Mutex
	ticker       *time.Ticker
}

func NewTimingWheel(tickDuration time.Duration, slotsCount int) *TimingWheel {
	tw := &TimingWheel{
		tickDuration: tickDuration,
		slotsCount:   slotsCount,
		slots:        make([]*slot, slotsCount),
		currentPos:   0,
	}

	for i := range tw.slots {
		tw.slots[i] = &slot{timers: make(map[string]*Timer)}
	}

	tw.ticker = time.NewTicker(tickDuration)
	go tw.start()
	return tw
}

func (tw *TimingWheel) AddTimer(id string, interval time.Duration, repeat bool, args map[string]any, task func(map[string]any)) {
	t := &Timer{
		ID:       id,
		Interval: interval,
		Task:     task,
		Args:     args,
		Repeat:   repeat,
		stop:     make(chan struct{}),
	}

	pos := (tw.currentPos + int(interval/tw.tickDuration)) % tw.slotsCount
	t.Rounds = int(interval/tw.tickDuration) / tw.slotsCount

	tw.mutex.Lock()
	tw.slots[pos].timers[id] = t
	tw.mutex.Unlock()
}

func (tw *TimingWheel) UpdateTimerTTL(id string, newInterval time.Duration) bool {
	tw.mutex.Lock()
	defer tw.mutex.Unlock()

	for _, s := range tw.slots {
		if t, ok := s.timers[id]; ok {
			delete(s.timers, id)
			if newInterval <= 0 {
				go t.Task(t.Args)
				return true
			}

			t.Rounds = int(newInterval / tw.tickDuration / time.Duration(tw.slotsCount))
			pos := (tw.currentPos + int(newInterval/tw.tickDuration)) % tw.slotsCount

			tw.slots[pos].timers[id] = t
			return true
		}
	}
	return false
}

func (tw *TimingWheel) ActiveTimers() map[string]time.Duration {
	result := make(map[string]time.Duration)

	tw.mutex.Lock()
	for i, s := range tw.slots {
		for id, timer := range s.timers {
			remaining := time.Duration(timer.Rounds)*tw.tickDuration +
				tw.tickDuration*time.Duration((i-tw.currentPos+tw.slotsCount)%tw.slotsCount)
			result[id] = remaining
		}
	}
	tw.mutex.Unlock()

	return result
}

func (tw *TimingWheel) RemoveTimer(id string) {
	tw.mutex.Lock()
	for _, s := range tw.slots {
		delete(s.timers, id)
	}
	tw.mutex.Unlock()
}

func (tw *TimingWheel) start() {
	for range tw.ticker.C {
		tw.mutex.Lock()
		currentSlot := tw.slots[tw.currentPos]
		tw.mutex.Unlock()

		for id, timer := range currentSlot.timers {
			if timer.Rounds > 0 {
				timer.Rounds--
				continue
			}
			go timer.Task(timer.Args)

			if timer.Repeat {
				nextPos := (tw.currentPos + int(timer.Interval/tw.tickDuration)) % tw.slotsCount

				tw.mutex.Lock()
				tw.slots[nextPos].timers[id] = timer
				tw.mutex.Unlock()
			}
			delete(currentSlot.timers, id)
		}

		tw.currentPos = (tw.currentPos + 1) % tw.slotsCount
	}
}
