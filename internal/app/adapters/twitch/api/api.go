package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Twitch struct {
	log    logger.Logger
	cfg    *config.Config
	stream ports.StreamPort
	client *http.Client
}

func NewTwitch(log logger.Logger, cfg *config.Config, stream ports.StreamPort, client *http.Client) *Twitch {
	return &Twitch{
		log:    log,
		cfg:    cfg,
		stream: stream,
		client: client,
	}
}

func (t *Twitch) doTwitchRequest(method, url string, body io.Reader, target interface{}) error {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+t.cfg.App.OAuth)
	req.Header.Set("Client-Id", t.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	if target == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
