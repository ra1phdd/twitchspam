package template

import (
	"sync/atomic"
	"time"
)

type SpamPauseTemplate struct {
	paused  int32
	endTime atomic.Value // хранит time.Time
}

func NewSpamPause() *SpamPauseTemplate {
	return &SpamPauseTemplate{}
}

func (p *SpamPauseTemplate) Pause(duration time.Duration) {
	if duration == 0 {
		atomic.StoreInt32(&p.paused, 0)
		p.endTime.Store(time.Time{})
		return
	}

	atomic.StoreInt32(&p.paused, 1)
	end := time.Now().Add(duration)
	p.endTime.Store(end)

	time.AfterFunc(duration, func() {
		atomic.StoreInt32(&p.paused, 0)
		p.endTime.Store(time.Time{})
	})
}

func (p *SpamPauseTemplate) CanProcess() bool {
	return atomic.LoadInt32(&p.paused) == 0
}

func (p *SpamPauseTemplate) Remaining() time.Duration {
	val := p.endTime.Load()
	if val == nil {
		return 0
	}
	end, ok := val.(time.Time)
	if !ok || end.IsZero() {
		return 0
	}
	remaining := time.Until(end)
	if remaining < 0 {
		return 0
	}
	return remaining
}
