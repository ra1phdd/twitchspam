package irc

import (
	"time"
)

func (i *IRC) WaitForIRC(msgID string, timeout time.Duration) (bool, bool) {
	ch := make(chan bool, 1)

	i.mu.Lock()
	i.chans[msgID] = ch
	i.mu.Unlock()

	defer func() {
		i.mu.Lock()
		delete(i.chans, msgID)
		i.mu.Unlock()
	}()

	select {
	case isFirst := <-ch:
		return isFirst, true // второй параметр - дождался ответа или нет
	case <-time.After(timeout):
		return false, false
	}
}

func (i *IRC) NotifyIRC(msgID string, isFirst bool) {
	i.mu.Lock()
	ch, ok := i.chans[msgID]
	i.mu.Unlock()

	if ok {
		ch <- isFirst
	}
}

func (i *IRC) cleanupLoop() {
	ticker := time.NewTicker(i.ttl)
	defer ticker.Stop()

	for range ticker.C {
		i.mu.Lock()
		for id, ch := range i.chans {
			select {
			case <-ch:
			default:
				close(ch)
			}
			delete(i.chans, id)
		}
		i.mu.Unlock()
	}
}
