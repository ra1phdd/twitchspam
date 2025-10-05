package event_sub

import (
	"log/slog"
	"time"
)

const maxRetries = 10

func (es *EventSub) subscribeEvents(payload SessionWelcomePayload) {
	events := []struct {
		name, version string
		condition     map[string]string
	}{
		{
			name:    "channel.chat.message",
			version: "1",
			condition: map[string]string{
				"broadcaster_user_id": es.stream.ChannelID(),
				"user_id":             es.cfg.App.UserID,
			},
		},
		{
			name:    "automod.message.hold",
			version: "1",
			condition: map[string]string{
				"broadcaster_user_id": es.stream.ChannelID(),
				"moderator_user_id":   es.cfg.App.UserID,
			},
		},
		{
			name:    "stream.online",
			version: "1",
			condition: map[string]string{
				"broadcaster_user_id": es.stream.ChannelID(),
			},
		},
		{
			name:    "stream.offline",
			version: "1",
			condition: map[string]string{
				"broadcaster_user_id": es.stream.ChannelID(),
			},
		},
		{
			name:    "channel.update",
			version: "2",
			condition: map[string]string{
				"broadcaster_user_id": es.stream.ChannelID(),
			},
		},
		{
			name:    "channel.moderate",
			version: "2",
			condition: map[string]string{
				"broadcaster_user_id": es.stream.ChannelID(),
				"moderator_user_id":   es.cfg.App.UserID,
			},
		},
	}

	for _, e := range events {
		var attempt int
		var backoff = 1 * time.Second

		for {
			err := es.subscribeEvent(e.name, e.version, e.condition, payload.Session.ID)
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
}
