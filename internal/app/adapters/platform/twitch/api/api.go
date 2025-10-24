package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
	"twitchspam/internal/app/infrastructure/config"
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

const (
	maxRetries  = 5
	baseBackoff = time.Second
	maxBackoff  = 30 * time.Second
)

type twitchRequest struct {
	Method string
	URL    string
	Token  *config.UserTokens
	Body   io.Reader
}

type TwitchAPIError struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func (t *Twitch) doTwitchRequest(ctx context.Context, reqData twitchRequest, target interface{}) (int, error) {
	t.log.Trace("Preparing Twitch request",
		slog.String("method", reqData.Method),
		slog.String("url", reqData.URL),
		slog.Bool("hasToken", reqData.Token != nil),
	)

	req, err := http.NewRequestWithContext(ctx, reqData.Method, reqData.URL, reqData.Body)
	if err != nil {
		t.log.Error("Failed to create HTTP request", err, slog.String("method", reqData.Method), slog.String("url", reqData.URL))
		return 0, err
	}

	auth, clientID := t.cfg.App.OAuth, t.cfg.App.ClientID
	if reqData.Token != nil {
		auth, clientID = reqData.Token.AccessToken, t.cfg.UserAccess.ClientID
	}

	req.Header.Set("Authorization", "Bearer "+auth)
	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Content-Type", "application/json")

	var resp *http.Response
	for attempt := 1; attempt <= maxRetries; attempt++ {
		t.log.Debug("Sending Twitch request", slog.Int("attempt", attempt), slog.String("method", reqData.Method), slog.String("url", reqData.URL))

		resp, err = t.client.Do(req)
		if err != nil {
			t.log.Error("HTTP request failed", err, slog.Int("attempt", attempt), slog.String("url", reqData.URL))
			return 0, err
		}

		raw, err := io.ReadAll(resp.Body)
		if cerr := resp.Body.Close(); cerr != nil {
			t.log.Error("Failed to close response body", cerr)
		}
		if err != nil {
			t.log.Error("Failed to read response body", err, slog.Int("status", resp.StatusCode), slog.String("url", reqData.URL))
			return resp.StatusCode, err
		}

		t.log.Trace("Response received", slog.Int("status", resp.StatusCode), slog.String("body", string(raw)))
		switch resp.StatusCode {
		case http.StatusOK, http.StatusNoContent:
			if target == nil {
				t.log.Debug("No target provided, discarding body", slog.Int("status", resp.StatusCode))
				return resp.StatusCode, nil
			}

			if err := json.Unmarshal(raw, target); err != nil {
				t.log.Error("Failed to decode response JSON", err, slog.Int("status", resp.StatusCode), slog.String("body", string(raw)))
				return resp.StatusCode, err
			}

			t.log.Debug("Request succeeded", slog.Int("status", resp.StatusCode))
			return resp.StatusCode, nil

		case http.StatusTooManyRequests:
			wait := calcWaitDuration(resp.Header.Get("Ratelimit-Reset"))

			if wait <= 0 {
				wait = time.Duration(attempt) * baseBackoff
			}
			if wait > maxBackoff {
				wait = maxBackoff
			}

			t.log.Warn("Rate limit hit, backing off", slog.Int("attempt", attempt), slog.String("wait", wait.String()))
			time.Sleep(wait)
			continue

		default:
			var apiErr TwitchAPIError
			if err := json.Unmarshal(raw, &apiErr); err != nil {
				t.log.Error("Failed to decode Twitch API error", err, slog.Int("status", resp.StatusCode), slog.String("body", string(raw)))
				return resp.StatusCode, fmt.Errorf("twitch API returned: %s", string(raw))
			}

			t.log.Error("Twitch API returned an error", errors.New(apiErr.Message), slog.Int("status", resp.StatusCode), slog.String("url", reqData.URL))
			return resp.StatusCode, errors.New(apiErr.Message)
		}
	}

	t.log.Error("Twitch request failed after max retries", nil,
		slog.Int("maxRetries", maxRetries),
		slog.String("url", reqData.URL),
	)
	return 0, fmt.Errorf("twitch request failed after %d retries", maxRetries)
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
