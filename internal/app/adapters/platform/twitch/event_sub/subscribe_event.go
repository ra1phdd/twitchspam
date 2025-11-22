package event_sub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

const maxRetries = 5
const initialBackoff = 3 * time.Second

func (es *EventSub) subscribeEvents(ctx context.Context, sessionID, channelID string) {
	events := []struct {
		name, version string
		condition     map[string]string
	}{
		{
			name:    "channel.chat.message",
			version: "1",
			condition: map[string]string{
				"broadcaster_user_id": channelID,
				"user_id":             es.cfg.App.UserID,
			},
		},
		{
			name:    "automod.message.hold",
			version: "1",
			condition: map[string]string{
				"broadcaster_user_id": channelID,
				"moderator_user_id":   es.cfg.App.UserID,
			},
		},
		{
			name:    "stream.online",
			version: "1",
			condition: map[string]string{
				"broadcaster_user_id": channelID,
			},
		},
		{
			name:    "stream.offline",
			version: "1",
			condition: map[string]string{
				"broadcaster_user_id": channelID,
			},
		},
		{
			name:    "channel.update",
			version: "2",
			condition: map[string]string{
				"broadcaster_user_id": channelID,
			},
		},
		{
			name:    "channel.moderate",
			version: "2",
			condition: map[string]string{
				"broadcaster_user_id": channelID,
				"moderator_user_id":   es.cfg.App.UserID,
			},
		},
	}

	for _, e := range events {
		var attempt int
		backoff := initialBackoff

		for {
			err := es.subscribeEvent(ctx, e.name, e.version, e.condition, sessionID)
			if err == nil {
				break
			}

			es.log.Error("Failed to subscribe to event", err,
				slog.String("event", e.name),
				slog.Int("attempt", attempt+1),
			)

			attempt++
			if attempt >= maxRetries {
				es.log.Error("Giving up on event after max retries", err, slog.String("event", e.name))
				break
			}

			es.log.Warn("Retrying event subscription", slog.String("event", e.name), slog.Duration("backoff", backoff))
			time.Sleep(backoff)
			backoff *= 2
		}
	}

	es.log.Info("Successfully subscribed to events")
}

func (es *EventSub) subscribeEvent(ctx context.Context, eventType, version string, condition map[string]string, sessionID string) error {
	body := map[string]interface{}{
		"type":      eventType,
		"version":   version,
		"condition": condition,
		"transport": map[string]string{
			"method":     "websocket",
			"session_id": sessionID,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal subscription body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.twitch.tv/helix/eventsub/subscriptions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create subscription request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+es.cfg.App.OAuth)
	req.Header.Set("Client-Id", es.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.client.Do(req)
	if err != nil {
		return fmt.Errorf("send subscription request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusForbidden {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	return nil
}
