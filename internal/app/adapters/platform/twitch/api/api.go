package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Twitch struct {
	log    logger.Logger
	cfg    *config.Config
	client *http.Client
	pool   *TwitchPool
}

type TwitchPool struct {
	wg       sync.WaitGroup
	tasks    chan func()
	shutdown chan struct{}
}

func NewTwitch(log logger.Logger, cfg *config.Config, client *http.Client, workerCount int) *Twitch {
	t := &Twitch{
		log:    log,
		cfg:    cfg,
		client: client,
		pool: &TwitchPool{
			tasks:    make(chan func()),
			shutdown: make(chan struct{}),
		},
	}

	for i := 0; i < workerCount; i++ {
		t.pool.wg.Add(1)
		go t.pool.worker()
	}

	return t
}

func (t *Twitch) Pool() ports.APIPollPort {
	return t.pool
}

func (p *TwitchPool) Submit(task func()) error {
	select {
	case p.tasks <- task:
		return nil
	default:
		return fmt.Errorf("worker pool queue is full")
	}
}

func (p *TwitchPool) Stop() {
	close(p.shutdown)
	p.wg.Wait()
	close(p.tasks)
}

func (p *TwitchPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case task := <-p.tasks:
			task()
		case <-p.shutdown:
			return
		}
	}
}

const (
	maxRetries  = 5
	baseBackoff = time.Second
	maxBackoff  = 30 * time.Second
)

func (t *Twitch) doTwitchRequest(method, url string, body io.Reader, target interface{}) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+t.cfg.App.OAuth)
	req.Header.Set("Client-Id", t.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err = t.client.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
			if target == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				return nil
			}
			return json.NewDecoder(resp.Body).Decode(target)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resetHeader := resp.Header.Get("Ratelimit-Reset")
			wait := calcWaitDuration(resetHeader)

			if wait <= 0 {
				wait = time.Duration(attempt) * baseBackoff
			}

			if wait > maxBackoff {
				wait = maxBackoff
			}

			t.log.Warn("Rate limit hit, backing off",
				slog.Int("attempt", attempt),
				slog.String("wait", wait.String()),
			)

			time.Sleep(wait)
			continue
		}

		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	return fmt.Errorf("twitch request failed after %d retries", maxRetries)
}

func calcWaitDuration(resetHeader string) time.Duration {
	if resetHeader == "" {
		return 0
	}

	ts, err := strconv.ParseInt(resetHeader, 10, 64)
	if err != nil {
		return 0
	}

	resetTime := time.Unix(ts, 0)
	now := time.Now()

	if resetTime.Before(now) {
		return 0
	}
	return resetTime.Sub(now)
}
