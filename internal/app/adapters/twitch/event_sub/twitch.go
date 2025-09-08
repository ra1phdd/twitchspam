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
	"twitchspam/internal/app/adapters/messages/admin"
	"twitchspam/internal/app/adapters/messages/checker"
	"twitchspam/internal/app/adapters/messages/user"
	twitch2 "twitchspam/internal/app/adapters/twitch"
	"twitchspam/internal/app/adapters/twitch/api"
	"twitchspam/internal/app/domain/aliases"
	"twitchspam/internal/app/domain/banwords"
	"twitchspam/internal/app/domain/regex"
	"twitchspam/internal/app/domain/stats"
	"twitchspam/internal/app/domain/stream"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/twitch"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Twitch struct {
	log        logger.Logger
	cfg        *config.Config
	stream     ports.StreamPort
	api        ports.APIPort
	moderation ports.ModerationPort
	checker    ports.CheckerPort
	admin      ports.AdminPort
	user       ports.UserPort
	aliases    ports.AliasesPort
	bwords     ports.BanwordsPort
	stats      ports.StatsPort

	client *http.Client
}

func New(log logger.Logger, manager *config.Manager, client *http.Client, modChannel string) (*Twitch, error) {
	t := &Twitch{
		log:    log,
		cfg:    manager.Get(),
		client: client,
	}
	t.api = api.NewTwitch(t.cfg, t.client)

	channelID, err := t.api.GetChannelID(modChannel)
	if err != nil {
		return nil, err
	}

	live, err := t.api.GetLiveStream(channelID)
	if err != nil {
		return nil, err
	}

	t.stream = stream.NewStream(channelID, modChannel)
	t.stream.SetIslive(live.IsOnline)

	t.stats = stats.New()
	if live.IsOnline {
		t.log.Info("Stream started")
		t.stream.SetIslive(true)
		t.stream.SetStreamID(live.ID)

		t.stats.SetStartTime(live.StartedAt)
		t.stats.SetOnline(live.ViewerCount)
	}

	r := regex.New()
	t.aliases = aliases.New(t.cfg.Aliases)
	t.bwords = banwords.New(t.cfg.Banwords.Words, t.cfg.Banwords.Regexp)
	t.moderation = twitch.New(log, t.cfg, t.stream, client)
	t.checker = checker.NewCheck(log, t.cfg, t.stream, t.stats, t.bwords, r)
	t.admin = admin.New(log, manager, t.stream, r, t.api, t.aliases)
	t.user = user.New(log, t.cfg, t.stream, t.stats)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			live, err := t.api.GetLiveStream(channelID)
			if err != nil {
				log.Error("Error getting viewer count", err)
				return
			}

			if live.IsOnline {
				t.stream.SetIslive(true)
				t.stream.SetStreamID(live.ID)
				t.stats.SetOnline(live.ViewerCount)
			}
		}
	}()
	go t.runEventLoop()

	return t, nil
}

func (t *Twitch) runEventLoop() {
	for {
		err := t.connectAndHandleEvents()
		if err != nil {
			t.log.Warn("Websocket connection lost, retrying...", slog.String("error", err.Error()))
			time.Sleep(5 * time.Second)
		}
	}
}

func (t *Twitch) connectAndHandleEvents() error {
	ws, _, err := websocket.DefaultDialer.Dial("wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		t.log.Error("Failed to connect to Twitch Twitch", err)
		return err
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	msgChan := make(chan []byte, 10000)
	t.startWorkers(ctx, cancel, &wg, msgChan)

	t.log.Info("Connected to Twitch Twitch WebSocket")
	return t.readMessages(ctx, ws, msgChan, &wg)
}

func (t *Twitch) startWorkers(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, msgChan chan []byte) {
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.workerLoop(ctx, cancel, msgChan)
		}()
	}
}

func (t *Twitch) workerLoop(ctx context.Context, cancel context.CancelFunc, msgChan chan []byte) {
	for {
		select {
		case <-ctx.Done():
			return
		case msgBytes, ok := <-msgChan:
			if !ok {
				return
			}
			t.handleMessage(cancel, msgBytes)
		}
	}
}

func (t *Twitch) handleMessage(cancel context.CancelFunc, msgBytes []byte) {
	var event twitch2.EventSubMessage
	if err := json.Unmarshal(msgBytes, &event); err != nil {
		t.log.Error("Failed to decode Twitch message", err, slog.String("event", string(msgBytes)))
		return
	}

	switch event.Metadata.MessageType {
	case "session_reconnect":
		t.log.Debug("Received session_reconnect on Twitch")
		cancel()
		return
	case "session_welcome":
		t.log.Debug("Received session_welcome on Twitch")

		var payload twitch2.SessionWelcomePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.log.Error("Failed to decode session_welcome payload", err, slog.String("event", string(msgBytes)))
			return
		}

		t.subscribeEvents(payload)
	case "session_keepalive":
		t.log.Trace("Received session_keepalive on Twitch")
	case "notification":
		t.log.Debug("Received notification on Twitch")

		var envelope twitch2.EventSubEnvelope
		if err := json.Unmarshal(event.Payload, &envelope); err != nil {
			t.log.Error("Failed to decode Twitch envelope", err)
			return
		}

		switch envelope.Subscription.Type {
		case "channel.chat.message":
			var msgEvent twitch2.ChatMessageEvent
			if err := json.Unmarshal(envelope.Event, &msgEvent); err != nil {
				t.log.Error("Failed to decode channel.chat.message event", err)
				return
			}
			t.log.Debug("New message", slog.String("username", msgEvent.ChatterUserName), slog.String("text", msgEvent.Message.Text))

			t.checkMessage(msgEvent)
		case "automod.message.hold":
			var am twitch2.AutomodHoldEvent
			if err := json.Unmarshal(envelope.Event, &am); err != nil {
				t.log.Error("Failed to decode automod event", err)
				return
			}
			t.log.Info("AutoMod held message", slog.String("user_id", am.UserID), slog.String("message_id", am.MessageID), slog.String("text", am.Message.Text))

			t.checkAutomod(am)
		case "stream.online":
			t.log.Info("Stream started")
			t.stream.SetIslive(true)
			t.stats.SetStartTime(time.Now())
		case "stream.offline":
			t.log.Info("Stream ended")
			t.stream.SetIslive(false)
			t.stats.SetEndTime(time.Now())

			if err := t.api.SendChatMessage(t.stream.ChannelID(), t.stats.GetStats()); err != nil {
				t.log.Error("Failed to send message on chat", err)
			}
		case "channel.update":
			var upd twitch2.ChannelUpdateEvent
			if err := json.Unmarshal(envelope.Event, &upd); err != nil {
				t.log.Error("Failed to decode channel.update event", err)
				return
			}
			t.log.Info("Channel updated", slog.String("title", upd.Title), slog.String("category", upd.CategoryName), slog.String("lang", upd.Language))

			if upd.CategoryName != "" { // TODO
				t.stream.SetCategory(upd.CategoryName)
			}
		case "channel.moderate":
			var modEvent twitch2.ChannelModerateEvent
			if err := json.Unmarshal(envelope.Event, &modEvent); err != nil {
				t.log.Error("Failed to decode channel.moderate event", err)
				return
			}

			t.checkModerate(modEvent)
		}
	}
}

func (t *Twitch) readMessages(ctx context.Context, ws *websocket.Conn, msgChan chan []byte, wg *sync.WaitGroup) error {
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

func (t *Twitch) subscribeEvent(eventType, version string, condition map[string]string, sessionID string) error {
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

	req.Header.Set("Authorization", "Bearer "+t.cfg.App.OAuth)
	req.Header.Set("Client-Id", t.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
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

func (t *Twitch) convertMap(msgEvent twitch2.ChatMessageEvent) *ports.ChatMessage {
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
