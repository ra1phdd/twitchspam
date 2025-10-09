package event_sub

import (
	"log/slog"
	"time"
)

const maxRetries = 5
const initialBackoff = 3 * time.Second

func (es *EventSub) subscribeEvents(sessionID, channelID string) {
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
			err := es.subscribeEvent(e.name, e.version, e.condition, sessionID)
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
