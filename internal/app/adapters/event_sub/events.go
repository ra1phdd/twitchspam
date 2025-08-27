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

type Automod struct {
	log        logger.Logger
	cfg        *config.Config
	moderation ports.ModerationPort
	bwords     ports.BanwordsPort
	stream     ports.StreamPort

	client *http.Client
}

var ErrForbidden = errors.New("forbidden: subscription missing proper authorization")

func NewEventSub(log logger.Logger, cfg *config.Config, stream ports.StreamPort, client *http.Client) *Automod {
	return &Automod{
		log:        log,
		cfg:        cfg,
		moderation: moderation.New(log, cfg, stream, client),
		bwords:     banwords.New(cfg.Banwords),
		stream:     stream,
		client:     client,
	}
}

func (a *Automod) RunEventLoop() {
	for {
		err := a.connectAndHandleEvents()
		if err != nil {
			if errors.Is(err, ErrForbidden) {
				a.log.Error("Stopping EventLoop due to 403 Forbidden", err)
				return
			}

			a.log.Warn("Websocket connection lost, retrying...", slog.String("error", err.Error()))
			time.Sleep(5 * time.Second)
		}
	}
}

func (a *Automod) connectAndHandleEvents() error {
	ws, _, err := websocket.DefaultDialer.Dial("wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		a.log.Error("Failed to connect to Twitch EventSub", err)
		return err
	}
	defer ws.Close()

	a.log.Info("Connected to Twitch EventSub WebSocket")
	for {
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			a.log.Error("Error while reading websocket message", err)
			return err
		}

		var event EventSubMessage
		if err := json.Unmarshal(msgBytes, &event); err != nil {
			a.log.Error("Failed to decode EventSub message", err, slog.String("event", string(msgBytes)))
			continue
		}

		switch event.Metadata.MessageType {
		case "session_welcome":
			a.log.Debug("Received session_welcome on EventSub")

			var payload SessionWelcomePayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				a.log.Error("Failed to decode session_welcome payload", err, slog.String("event", string(msgBytes)))
				break
			}

			if err := a.subscribeEvent("automod.message.hold", "1", map[string]string{
				"broadcaster_user_id": a.stream.ChannelID(),
				"moderator_user_id":   a.cfg.App.UserID,
			}, payload.Session.ID); err != nil {
				a.log.Error("Failed to subscribe to event", err, slog.String("event", "automod.message.hold"))
				return err
			}

			if err := a.subscribeEvent("stream.online", "1", map[string]string{
				"broadcaster_user_id": a.stream.ChannelID(),
			}, payload.Session.ID); err != nil {
				a.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.online"))
				return err
			}

			if err := a.subscribeEvent("stream.offline", "1", map[string]string{
				"broadcaster_user_id": a.stream.ChannelID(),
			}, payload.Session.ID); err != nil {
				a.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.offline"))
				return err
			}
			break
		case "session_keepalive":
			a.log.Trace("Received session_keepalive on EventSub")
			break
		case "notification":
			a.log.Debug("Received notification on EventSub")

			var amEvent Message
			if err := json.Unmarshal(event.Payload, &amEvent); err != nil {
				a.log.Error("Failed to decode notification payload", err, slog.String("event", string(msgBytes)))
				break
			}

			switch amEvent.Subscription.Type {
			case "automod.message.hold":
				a.log.Info("AutoMod held message", slog.String("user_id", amEvent.Event.UserID), slog.String("message_id", amEvent.Event.MessageID), slog.String("text", amEvent.Event.Message.Text))
				text := strings.ToLower(domain.NormalizeText(amEvent.Event.Message.Text))
				words := strings.Fields(text)

				if a.bwords.CheckMessage(words) {
					time.Sleep(time.Duration(a.cfg.Spam.DelayAutomod) * time.Second)
					a.moderation.Ban(amEvent.Event.UserID, "банворд")
				}
			case "stream.online":
				a.log.Info("Stream started")
				a.stream.SetIslive(true)
			case "stream.offline":
				a.log.Info("Stream ended")
				a.stream.SetIslive(false)
			}
		case "session_reconnect":
			a.log.Debug("Received session_reconnect on EventSub")
			return nil
		}
	}
}

func (a *Automod) subscribeEvent(eventType, version string, condition map[string]string, sessionID string) error {
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

	req.Header.Set("Authorization", "Bearer "+a.cfg.App.OAuth)
	req.Header.Set("Client-Id", a.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
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
