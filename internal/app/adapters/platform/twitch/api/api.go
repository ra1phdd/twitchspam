package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
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

var UserAuthNotCompleted = errors.New("user auth failed")

func NewTwitch(log logger.Logger, manager *config.Manager, client *http.Client, workerCount int) *Twitch {
	t := &Twitch{
		log:    log,
		cfg:    manager.Get(),
		client: client,
		pool: &TwitchPool{
			tasks:    make(chan func(), 300),
			shutdown: make(chan struct{}),
		},
	}

	for range workerCount {
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
		return errors.New("worker pool queue is full")
	}
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

const (
	maxRetries  = 5
	baseBackoff = time.Second
	maxBackoff  = 30 * time.Second
)

func (t *Twitch) doTwitchRequest(method, url string, token *config.UserTokens, body io.Reader, target interface{}) error {
	req, err := http.NewRequestWithContext(context.Background(), method, url, body)
	if err != nil {
		return err
	}

	auth := t.cfg.App.OAuth
	clientID := t.cfg.App.ClientID
	if token != nil {
		auth = token.AccessToken
		clientID = t.cfg.UserAccess.ClientID
	}

	req.Header.Set("Authorization", "Bearer "+auth)
	req.Header.Set("Client-Id", clientID)
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

		switch resp.StatusCode {
		case http.StatusOK, http.StatusNoContent:
			if target == nil {
				_, _ = io.Copy(io.Discard, resp.Body)
				return nil
			}
			return json.NewDecoder(resp.Body).Decode(target)

		case http.StatusBadRequest:
			raw, _ := io.ReadAll(resp.Body)
			t.log.Error("Twitch returned 400", nil, slog.String("raw", string(raw)))
			return errors.New("400")

		case http.StatusUnauthorized:
			raw, _ := io.ReadAll(resp.Body)
			t.log.Warn("Access token expired", slog.String("raw", string(raw)))

			return errors.New("access token expired")

		case http.StatusTooManyRequests:
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

		default:
			raw, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
		}
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

func (t *Twitch) ensureUserToken(broadcasterID string) (*config.UserTokens, error) {
	token, ok := t.cfg.UsersTokens[broadcasterID]
	if !ok {
		return nil, UserAuthNotCompleted
	}

	if time.Now().After(token.ObtainedAt.Add(time.Duration(token.ExpiresIn-300) * time.Second)) {
		resp, err := t.refreshUserToken(token)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh user token: %w", err)
		}

		newToken := &config.UserTokens{
			AccessToken:  resp.AccessToken,
			RefreshToken: resp.RefreshToken,
			ExpiresIn:    resp.ExpiresIn,
			ObtainedAt:   time.Now(),
		}
		t.cfg.UsersTokens[broadcasterID] = newToken

		return newToken, nil
	}

	return token, nil
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (t *Twitch) refreshUserToken(token *config.UserTokens) (*TokenResponse, error) {
	if token == nil {
		return nil, errors.New("token is nil")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token.RefreshToken)
	data.Set("client_id", t.cfg.UserAccess.ClientID)
	data.Set("client_secret", t.cfg.UserAccess.ClientSecret)

	resp, err := t.client.PostForm("https://id.twitch.tv/oauth2/token", data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to refresh token: %s", string(raw))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}
