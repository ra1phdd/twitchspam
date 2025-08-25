package automod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log/slog"
	"net/http"
	"time"
	"twitchspam/pkg/logger"
)

type Automod struct {
	log logger.Logger

	OAuth       string
	ClientID    string
	ModeratorID string
	ChannelID   string

	client *http.Client
}

var ErrForbidden = errors.New("forbidden: subscription missing proper authorization")

func New(log logger.Logger, oauth, clientID, moderatorID, channelID string, client *http.Client) *Automod {
	return &Automod{
		log:         log,
		OAuth:       oauth,
		ClientID:    clientID,
		ModeratorID: moderatorID,
		ChannelID:   channelID,
		client:      client,
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

			if err := a.subscribeToAutoModEvents(payload.Session.ID); err != nil {
				a.log.Error("Failed to subscribe to automod", err, slog.String("session_id", payload.Session.ID))
				if errors.Is(err, ErrForbidden) {
					return ErrForbidden
				}
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

			a.log.Info("AutoMod held message", slog.String("user_id", amEvent.Event.UserID), slog.String("message_id", amEvent.Event.MessageID), slog.String("text", amEvent.Event.Message.Text))
		case "session_reconnect":
			a.log.Debug("Received session_reconnect on EventSub")
			return nil
		}
	}
}

func (a *Automod) subscribeToAutoModEvents(sessionID string) error {
	body := map[string]interface{}{
		"type":    "automod.message.hold",
		"version": "1",
		"condition": map[string]string{
			"broadcaster_user_id": a.ChannelID,
			"moderator_user_id":   a.ModeratorID,
		},
		"transport": map[string]string{
			"method":     "websocket",
			"session_id": sessionID,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		a.log.Error("Failed to marshal subscription body", err)
		return err
	}

	req, err := http.NewRequest("POST", "https://api.twitch.tv/helix/eventsub/subscriptions", bytes.NewBuffer(jsonBody))
	if err != nil {
		a.log.Error("Failed to create subscription request", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+a.OAuth)
	req.Header.Set("Client-Id", a.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		a.log.Error("Failed to send subscription request", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(resp.Body)
		a.log.Error("Twitch returned Non-OK status for EventSub subscription", nil, slog.Int("status_code", resp.StatusCode), slog.String("body", string(raw)))

		if resp.StatusCode == http.StatusForbidden {
			return ErrForbidden
		}
		return fmt.Errorf("failed to subscribe automod, status: %s", resp.Status)
	}

	a.log.Info("Successfully subscribed to automod")
	return nil
}
