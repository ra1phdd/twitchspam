package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Moderation struct {
	log    logger.Logger
	cfg    *config.Config
	stream ports.StreamPort

	client *http.Client
}

func New(log logger.Logger, cfg *config.Config, stream ports.StreamPort, client *http.Client) *Moderation {
	return &Moderation{
		log:    log,
		cfg:    cfg,
		stream: stream,
		client: client,
	}
}

func (m *Moderation) Timeout(userID string, duration int, reason string) {
	reqBody := TimeoutRequest{
		Data: TimeoutData{
			UserID:   userID,
			Duration: duration,
			Reason:   reason,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		m.log.Error("Failed to marshal timeout request body", err)
		return
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/moderation/bans?broadcaster_id=%s&moderator_id=%s", m.stream.ChannelID(), m.cfg.App.UserID)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		m.log.Error("Failed to create timeout request", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+m.cfg.App.OAuth)
	req.Header.Set("Client-Id", m.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		m.log.Error("Failed to send timeout request", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.log.Error("Twitch returned Non-OK status for moderation ban", nil, slog.Int("status_code", resp.StatusCode), slog.String("body", string(body)))
		return
	}

	m.log.Info("Timeout applied successfully", slog.String("user_id", userID), slog.Int("duration", duration), slog.String("reason", reason))
}

func (m *Moderation) Ban(userID string, reason string) {
	m.Timeout(userID, 0, reason)
}
