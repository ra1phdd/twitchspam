package event_sub

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type EventSub struct {
	log     logger.Logger
	cfg     *config.Config
	stream  ports.StreamPort
	api     ports.APIPort
	checker ports.CheckerPort
	admin   ports.AdminPort
	user    ports.UserPort
	aliases ports.AliasesPort
	bwords  ports.BanwordsPort
	stats   ports.StatsPort

	client *http.Client
}

func New(log logger.Logger, cfg *config.Config, stream ports.StreamPort, api ports.APIPort, checker ports.CheckerPort, admin ports.AdminPort, user ports.UserPort, aliases ports.AliasesPort, bwords ports.BanwordsPort, stats ports.StatsPort, client *http.Client) *EventSub {
	es := &EventSub{
		log:     log,
		cfg:     cfg,
		stream:  stream,
		api:     api,
		checker: checker,
		admin:   admin,
		user:    user,
		aliases: aliases,
		bwords:  bwords,
		stats:   stats,
		client:  client,
	}

	return es
}

func (es *EventSub) RunEventLoop() {
	for {
		err := es.connectAndHandleEvents()
		if err != nil {
			es.log.Warn("Websocket connection lost, retrying...", slog.String("error", err.Error()))
			time.Sleep(5 * time.Second)
		}
	}
}

func (es *EventSub) connectAndHandleEvents() error {
	ws, _, err := websocket.DefaultDialer.Dial("wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		es.log.Error("Failed to connect to EventSub EventSub", err)
		return err
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	msgChan := make(chan []byte, 10000)
	es.startWorkers(ctx, cancel, &wg, msgChan)

	es.log.Info("Connected to EventSub WebSocket")
	return es.readMessages(ctx, ws, msgChan, &wg)
}

func (es *EventSub) startWorkers(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, msgChan chan []byte) {
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.workerLoop(ctx, cancel, msgChan)
		}()
	}
}

func (es *EventSub) workerLoop(ctx context.Context, cancel context.CancelFunc, msgChan chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgBytes, ok := <-msgChan:
			if !ok {
				return
			}
			es.handleMessage(cancel, msgBytes)
		}
	}
}

func (es *EventSub) handleMessage(cancel context.CancelFunc, msgBytes []byte) {
	var event EventSubMessage
	if err := json.Unmarshal(msgBytes, &event); err != nil {
		es.log.Error("Failed to decode EventSub message", err, slog.String("event", string(msgBytes)))
		return
	}

	switch event.Metadata.MessageType {
	case "session_reconnect":
		es.log.Debug("Received session_reconnect on EventSub")
		cancel()
		return
	case "session_welcome":
		es.log.Debug("Received session_welcome on EventSub")

		var payload SessionWelcomePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			es.log.Error("Failed to decode session_welcome payload", err, slog.String("event", string(msgBytes)))
			return
		}

		es.subscribeEvents(payload)
	case "session_keepalive":
		es.log.Trace("Received session_keepalive on EventSub")
	case "notification":
		es.log.Debug("Received notification on EventSub")

		var envelope EventSubEnvelope
		if err := json.Unmarshal(event.Payload, &envelope); err != nil {
			es.log.Error("Failed to decode EventSub envelope", err)
			return
		}

		switch envelope.Subscription.Type {
		case "channel.chat.message":
			var msgEvent ChatMessageEvent
			if err := json.Unmarshal(envelope.Event, &msgEvent); err != nil {
				es.log.Error("Failed to decode channel.chat.message event", err)
				return
			}
			es.log.Debug("New message", slog.String("username", msgEvent.ChatterUserName), slog.String("text", msgEvent.Message.Text))

			es.checkMessage(msgEvent)
		case "automod.message.hold":
			var am AutomodHoldEvent
			if err := json.Unmarshal(envelope.Event, &am); err != nil {
				es.log.Error("Failed to decode automod event", err)
				return
			}
			es.log.Info("AutoMod held message", slog.String("user_id", am.UserID), slog.String("message_id", am.MessageID), slog.String("text", am.Message.Text))

			es.checkAutomod(am)
		case "stream.online":
			es.log.Info("Stream started")
			es.stream.SetIslive(true)
			es.stats.SetStartTime(time.Now())
		case "stream.offline":
			es.log.Info("Stream ended")
			es.stream.SetIslive(false)
			es.stats.SetEndTime(time.Now())

			if err := es.api.SendChatMessage(es.stats.GetStats()); err != nil {
				es.log.Error("Failed to send message on chat", err)
			}
		case "channel.update":
			var upd ChannelUpdateEvent
			if err := json.Unmarshal(envelope.Event, &upd); err != nil {
				es.log.Error("Failed to decode channel.update event", err)
				return
			}
			es.log.Info("Channel updated", slog.String("title", upd.Title), slog.String("category", upd.CategoryName), slog.String("lang", upd.Language))

			if upd.CategoryName != "" { // TODO
				es.stream.SetCategory(upd.CategoryName)
			}
		case "channel.moderate":
			var modEvent ChannelModerateEvent
			if err := json.Unmarshal(envelope.Event, &modEvent); err != nil {
				es.log.Error("Failed to decode channel.moderate event", err)
				return
			}

			es.checkModerate(modEvent)
		}
	}
}

func (es *EventSub) readMessages(ctx context.Context, ws *websocket.Conn, msgChan chan []byte, wg *sync.WaitGroup) error {
	for {
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			wg.Wait()
			close(msgChan)
			return nil
		case msgChan <- msgBytes:
		}
	}
}

func (es *EventSub) subscribeEvent(eventType, version string, condition map[string]string, sessionID string) error {
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

	req.Header.Set("Authorization", "Bearer "+es.cfg.App.OAuth)
	req.Header.Set("Client-Id", es.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := es.client.Do(req)
	if err != nil {
		return fmt.Errorf("send subscription request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	return nil
}

func (es *EventSub) convertMap(msgEvent ChatMessageEvent) *ports.ChatMessage {
	var isBroadcaster, isMod, isVip, isSubscriber, emoteOnly bool
	var emotes []string

	for _, badge := range msgEvent.Badges {
		switch badge.SetID {
		case "broadcaster":
			isBroadcaster = true
		case "moderator":
			isMod = true
		case "vip":
			isVip = true
		case "subscriber":
			isSubscriber = true
		}
	}

	emoteOnly = true
	for _, fragment := range msgEvent.Message.Fragments {
		if fragment.Type == "text" {
			emoteOnly = false
		}
		if fragment.Type == "emote" {
			emotes = append(emotes, fragment.Text)
		}
	}

	return &ports.ChatMessage{
		Broadcaster: ports.Broadcaster{
			UserID:   msgEvent.BroadcasterUserID,
			Login:    msgEvent.BroadcasterUserLogin,
			Username: msgEvent.BroadcasterUserName,
		},
		Chatter: ports.Chatter{
			UserID:        msgEvent.ChatterUserID,
			Login:         msgEvent.ChatterUserLogin,
			Username:      msgEvent.ChatterUserName,
			IsBroadcaster: isBroadcaster,
			IsMod:         isMod,
			IsVip:         isVip,
			IsSubscriber:  isSubscriber,
		},
		Message: ports.Message{
			ID:        msgEvent.MessageID,
			Text:      msgEvent.Message.Text,
			EmoteOnly: emoteOnly,
			Emotes:    emotes,
		},
	}
}
