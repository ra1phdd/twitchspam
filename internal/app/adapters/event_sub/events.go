package event_sub

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"twitchspam/config"
	"twitchspam/internal/app/adapters/banwords"
	"twitchspam/internal/app/adapters/moderation"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type EventSub struct {
	log        logger.Logger
	cfg        *config.Config
	stream     ports.StreamPort
	moderation ports.ModerationPort
	bwords     ports.BanwordsPort
	stats      ports.StatsPort

	client *http.Client
}

var ErrForbidden = errors.New("forbidden: subscription missing proper authorization")

func NewEventSub(log logger.Logger, cfg *config.Config, stream ports.StreamPort, stats ports.StatsPort, client *http.Client) *EventSub {
	return &EventSub{
		log:        log,
		cfg:        cfg,
		moderation: moderation.New(log, cfg, stream, client),
		bwords:     banwords.New(cfg.Banwords),
		stats:      stats,
		stream:     stream,
		client:     client,
	}
}

func (e *EventSub) RunEventLoop() {
	for {
		err := e.connectAndHandleEvents()
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				e.log.Error("Stopping EventLoop due to 403 Forbidden", err)
				return
			}

			e.log.Warn("Websocket connection lost, retrying...", slog.String("error", err.Error()))
			time.Sleep(5 * time.Second)
		}
	}
}

func (e *EventSub) connectAndHandleEvents() error {
	ws, _, err := websocket.DefaultDialer.Dial("wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		e.log.Error("Failed to connect to Twitch EventSub", err)
		return err
	}
	defer ws.Close()

	e.log.Info("Connected to Twitch EventSub WebSocket")
	for {
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			e.log.Error("Error while reading websocket message", err)
			return err
		}

		var event EventSubMessage
		if err := json.Unmarshal(msgBytes, &event); err != nil {
			e.log.Error("Failed to decode EventSub message", err, slog.String("event", string(msgBytes)))
			continue
		}

		switch event.Metadata.MessageType {
		case "session_welcome":
			e.log.Debug("Received session_welcome on EventSub")

			var payload SessionWelcomePayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				e.log.Error("Failed to decode session_welcome payload", err, slog.String("event", string(msgBytes)))
				break
			}

			if err := e.subscribeEvent("automod.message.hold", "1", map[string]string{
				"broadcaster_user_id": e.stream.ChannelID(),
				"moderator_user_id":   e.cfg.App.UserID,
			}, payload.Session.ID); err != nil {
				e.log.Error("Failed to subscribe to event", err, slog.String("event", "automod.message.hold"))
				return err
			}

			if err := e.subscribeEvent("stream.online", "1", map[string]string{
				"broadcaster_user_id": e.stream.ChannelID(),
			}, payload.Session.ID); err != nil {
				e.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.online"))
				return err
			}

			if err := e.subscribeEvent("stream.offline", "1", map[string]string{
				"broadcaster_user_id": e.stream.ChannelID(),
			}, payload.Session.ID); err != nil {
				e.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.offline"))
				return err
			}

			if err := e.subscribeEvent("channel.update", "2", map[string]string{
				"broadcaster_user_id": e.stream.ChannelID(),
			}, payload.Session.ID); err != nil {
				e.log.Error("Failed to subscribe to event", err, slog.String("event", "channel.update"))
				return err
			}

			if err := e.subscribeEvent("channel.moderate", "2", map[string]string{
				"broadcaster_user_id": e.stream.ChannelID(),
				"moderator_user_id":   e.cfg.App.UserID,
			}, payload.Session.ID); err != nil {
				e.log.Error("Failed to subscribe to event", err, slog.String("event", "channel.moderate"))
				return err
			}

			break
		case "session_keepalive":
			e.log.Trace("Received session_keepalive on EventSub")
			break
		case "notification":
			e.log.Debug("Received notification on EventSub")

			var envelope EventSubEnvelope
			if err := json.Unmarshal(event.Payload, &envelope); err != nil {
				e.log.Error("Failed to decode EventSub envelope", err)
				break
			}

			switch envelope.Subscription.Type {
			case "automod.message.hold":
				var am AutomodHoldEvent
				if err := json.Unmarshal(envelope.Event, &am); err != nil {
					e.log.Error("Failed to decode automod event", err)
					break
				}
				e.log.Info("AutoMod held message", slog.String("user_id", am.UserID), slog.String("message_id", am.MessageID), slog.String("text", am.Message.Text))

				text := strings.ToLower(domain.NormalizeText(am.Message.Text))
				words := strings.Fields(text)

				if e.bwords.CheckMessage(words) {
					time.Sleep(time.Duration(e.cfg.Spam.DelayAutomod) * time.Second)
					e.moderation.Ban(am.UserID, "банворд")
				}

				if e.cfg.PunishmentOnline && e.bwords.CheckOnline(text) {
					e.moderation.Ban(am.UserID, "тупое")
				}
			case "stream.online":
				e.log.Info("Stream started")
				e.stream.SetIslive(true)
				e.stats.SetStartTime(time.Now())
			case "stream.offline":
				e.log.Info("Stream ended")
				e.stream.SetIslive(false)
				e.stats.SetEndTime(time.Now())
			case "channel.update":
				var upd ChannelUpdateEvent
				if err := json.Unmarshal(envelope.Event, &upd); err != nil {
					e.log.Error("Failed to decode channel.update event", err)
					break
				}
				e.log.Info("Channel updated", slog.String("title", upd.Title), slog.String("category", upd.CategoryName), slog.String("lang", upd.Language))

				if upd.CategoryName != "" { // TODO
					e.stream.SetCategory(upd.CategoryName)
				}
			case "channel.moderate":
				var modEvent ChannelModerateEvent
				if err := json.Unmarshal(envelope.Event, &modEvent); err != nil {
					e.log.Error("Failed to decode channel.moderate event", err)
					break
				}

				switch modEvent.Action {
				case "timeout":
					e.log.Info("User muted", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Timeout.Username), slog.Time("expires_at", modEvent.Timeout.ExpiresAt), slog.String("reason", modEvent.Timeout.Reason))
					e.stats.AddTimeout(modEvent.ModeratorUserName)
				case "ban":
					e.log.Info("User banned", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Ban.Username), slog.String("reason", modEvent.Ban.Reason))
					e.stats.AddBan(modEvent.ModeratorUserName)
				}
			case "session_reconnect":
				e.log.Debug("Received session_reconnect on EventSub")
				return nil
			}
		}
	}
}

func (e *EventSub) subscribeEvent(eventType, version string, condition map[string]string, sessionID string) error {
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

	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/eventsub/subscriptions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("create subscription request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+e.cfg.App.OAuth)
	req.Header.Set("Client-Id", e.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return fmt.Errorf("send subscription request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusForbidden {
			return ErrForbidden
		}
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	return nil
}
