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
	"strings"
	"sync"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
	"unicode/utf8"
)

type EventSub struct {
	log logger.Logger
	cfg *config.Config
	api ports.APIPort
	irc ports.IRCPort

	mu        sync.Mutex
	channels  map[string]channel
	sessionID string

	client *http.Client
}

type channel struct {
	username string
	stream   ports.StreamPort
	message  ports.MessagePort
}

func NewTwitch(log logger.Logger, cfg *config.Config, api ports.APIPort, irc ports.IRCPort, client *http.Client) *EventSub {
	es := &EventSub{
		log:      log,
		cfg:      cfg,
		api:      api,
		irc:      irc,
		client:   client,
		channels: make(map[string]channel),
	}
	go es.runEventLoop()

	for es.sessionID == "" {
		time.Sleep(time.Millisecond)
	}

	return es
}

func (es *EventSub) AddChannel(channelID, channelName string, stream ports.StreamPort, message ports.MessagePort) {
	es.mu.Lock()
	if _, ok := es.channels[channelID]; ok {
		es.mu.Unlock()
		return
	}

	es.channels[channelID] = channel{
		username: channelName,
		stream:   stream,
		message:  message,
	}
	es.mu.Unlock()

	es.subscribeEvents(context.Background(), es.sessionID, channelID)
}

func (es *EventSub) runEventLoop() {
	for {
		err := es.connectAndHandleEvents()
		if err != nil {
			es.log.Warn("Websocket connection lost, retrying...", slog.String("error", err.Error()))
			time.Sleep(5 * time.Second)
		}
	}
}

func (es *EventSub) connectAndHandleEvents() error {
	dialer := websocket.Dialer{
		NetDialContext:   es.client.Transport.(*http.Transport).DialContext,
		HandshakeTimeout: 10 * time.Second,
	}

	ws, resp, err := dialer.Dial("wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		if resp != nil {
			err := resp.Body.Close()
			if err != nil {
				es.log.Error("Failed to close response body", err)
			}
		}
		return fmt.Errorf("websocket dial: %w", err)
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	msgChan := make(chan []byte, 10000)
	es.startWorkers(ctx, &wg, msgChan)

	es.log.Info("Connected to EventSub WebSocket")
	return es.readMessages(ctx, ws, msgChan, &wg)
}

func (es *EventSub) startWorkers(ctx context.Context, wg *sync.WaitGroup, msgChan chan []byte) {
	for range runtime.NumCPU() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			es.workerLoop(ctx, msgChan)
		}()
	}
}

func (es *EventSub) workerLoop(ctx context.Context, msgChan chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgBytes, ok := <-msgChan:
			if !ok {
				return
			}
			es.handleMessage(ctx, msgBytes)
		}
	}
}

func (es *EventSub) handleMessage(ctx context.Context, msgBytes []byte) {
	var event EventSubMessage
	if err := json.Unmarshal(msgBytes, &event); err != nil {
		es.log.Error("Failed to decode EventSub message", err, slog.String("event", string(msgBytes)))
		return
	}

	switch event.Metadata.MessageType {
	case "session_reconnect":
		es.log.Debug("Received session_reconnect on EventSub")

		if cancelFunc, ok := ctx.Value("cancelFunc").(context.CancelFunc); ok {
			cancelFunc()
		}
		return
	case "session_welcome":
		es.log.Debug("Received session_welcome on EventSub")

		var payload SessionWelcomePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			es.log.Error("Failed to decode session_welcome payload", err, slog.String("event", string(msgBytes)))
			return
		}
		es.sessionID = payload.Session.ID

		es.mu.Lock()
		defer es.mu.Unlock()

		for id := range es.channels {
			es.subscribeEvents(ctx, es.sessionID, id)
		}
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
			es.log.Debug("NewTwitch message", slog.String("username", msgEvent.ChatterUserName), slog.String("text", msgEvent.Message.Text))

			if c, ok := es.channels[msgEvent.BroadcasterUserID]; ok {
				c.message.Check(es.convertMap(msgEvent))
			}
		case "automod.message.hold":
			var am AutomodHoldEvent
			if err := json.Unmarshal(envelope.Event, &am); err != nil {
				es.log.Error("Failed to decode automod event", err)
				return
			}
			es.log.Info("AutoMod held message", slog.String("user_id", am.UserID), slog.String("message_id", am.MessageID), slog.String("text", am.Message.Text))

			msg := &domain.ChatMessage{
				Broadcaster: domain.Broadcaster{
					UserID: am.UserID,
				},
				Chatter: domain.Chatter{
					UserID:   am.UserID,
					Username: am.UserName,
				},
				Message: domain.Message{
					ID: am.MessageID,
					Text: domain.MessageText{
						Original: am.Message.Text,
					},
				},
			}

			if c, ok := es.channels[am.BroadcasterUserID]; ok {
				go c.message.CheckAutomod(msg)
			}
		case "stream.online":
			var sm StreamMessageEvent
			if err := json.Unmarshal(envelope.Event, &sm); err != nil {
				es.log.Error("Failed to decode stream message event", err)
				return
			}
			es.log.Info("Stream started")

			if c, ok := es.channels[sm.BroadcasterUserID]; ok {
				c.stream.SetIslive(true)
				c.stream.Stats().SetStartTime(time.Now())
			}
		case "stream.offline":
			var sm StreamMessageEvent
			if err := json.Unmarshal(envelope.Event, &sm); err != nil {
				es.log.Error("Failed to decode stream message event", err)
				return
			}
			es.log.Info("Stream ended")

			if c, ok := es.channels[sm.BroadcasterUserID]; ok {
				c.stream.SetIslive(false)
				c.stream.Stats().SetEndTime(time.Now())

				if es.cfg.Channels[sm.BroadcasterUserLogin].Enabled {
					es.api.SendChatMessages(c.stream.ChannelID(), c.stream.Stats().GetStats())
				}
			}
		case "channel.update":
			var upd ChannelUpdateEvent
			if err := json.Unmarshal(envelope.Event, &upd); err != nil {
				es.log.Error("Failed to decode channel.update event", err)
				return
			}
			es.log.Info("channel updated", slog.String("title", upd.Title), slog.String("category", upd.CategoryName), slog.String("lang", upd.Language))

			if c, ok := es.channels[upd.BroadcasterUserID]; ok {
				c.stream.SetCategory(upd.CategoryName)
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

	if resp.StatusCode != http.StatusAccepted {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("twitch returned %s: %s", resp.Status, string(raw))
	}

	return nil
}

func (es *EventSub) convertMap(msgEvent ChatMessageEvent) *domain.ChatMessage {
	var isBroadcaster, isMod, isVip, isSubscriber bool
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

	emoteChars := 0
	textChars := 0

	var emotes []string
	for _, fragment := range msgEvent.Message.Fragments {
		text := strings.TrimSpace(fragment.Text)
		if text == "" {
			continue
		}

		if fragment.Type == "text" {
			textChars += utf8.RuneCountInString(text)
		}
		if fragment.Type == "emote" {
			emotes = append(emotes, text)
			emoteChars += utf8.RuneCountInString(text)
		}
	}

	total := emoteChars + textChars
	emoteOnly := total > 0 && float64(emoteChars)/float64(total) >= es.cfg.Channels[msgEvent.BroadcasterUserLogin].Spam.SettingsEmotes.EmoteThreshold

	msg := &domain.ChatMessage{
		Broadcaster: domain.Broadcaster{
			UserID:   msgEvent.BroadcasterUserID,
			Login:    msgEvent.BroadcasterUserLogin,
			Username: msgEvent.BroadcasterUserName,
		},
		Chatter: domain.Chatter{
			UserID:        msgEvent.ChatterUserID,
			Login:         msgEvent.ChatterUserLogin,
			Username:      msgEvent.ChatterUserName,
			IsBroadcaster: isBroadcaster,
			IsMod:         isMod,
			IsVip:         isVip,
			IsSubscriber:  isSubscriber,
		},
		Message: domain.Message{
			ID: msgEvent.MessageID,
			Text: domain.MessageText{
				Original: msgEvent.Message.Text,
			},
			EmoteOnly: emoteOnly,
			Emotes:    emotes,
			IsFirst: func() bool {
				isFirst, _ := es.irc.WaitForIRC(msgEvent.MessageID, 250*time.Millisecond)
				return isFirst
			},
		},
	}

	if msgEvent.Reply != nil {
		msg.Reply = &domain.Reply{
			ParentChatter: domain.Chatter{
				UserID:   msgEvent.Reply.ParentUserID,
				Login:    msgEvent.Reply.ParentUserLogin,
				Username: msgEvent.Reply.ParentUserName,
			},
			ParentMessage: domain.Message{
				ID: msgEvent.Reply.ParentMessageID,
				Text: domain.MessageText{
					Original: msgEvent.Reply.ParentMessageBody,
				},
			},
		}
	}

	return msg
}
