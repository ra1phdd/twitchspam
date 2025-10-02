package template

import (
	"sync/atomic"
	"time"
)

type SpamPauseTemplate struct {
	paused int32
}

func NewSpamPause() *SpamPauseTemplate {
	return &SpamPauseTemplate{}
}

func (p *SpamPauseTemplate) Pause(duration time.Duration) {
	if duration == 0 {
		atomic.StoreInt32(&p.paused, 0)
	}

	atomic.StoreInt32(&p.paused, 1)
	time.AfterFunc(duration, func() {
		atomic.StoreInt32(&p.paused, 0)
	})
}

func (p *SpamPauseTemplate) CanProcess() bool {
	return atomic.LoadInt32(&p.paused) == 0
}
