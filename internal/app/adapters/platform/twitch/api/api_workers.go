package api

import (
	"twitchspam/internal/app/ports"
)

func (t *Twitch) Pool() ports.APIPollPort {
	return t.pool
}

func (p *TwitchPool) Submit(task func()) {
	p.tasks <- task
}

func (p *TwitchPool) Stop() {
	close(p.tasks)
	p.wg.Wait()
	close(p.shutdown)
}

func (p *TwitchPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			task()
		case <-p.shutdown:
			return
		}
	}
}
