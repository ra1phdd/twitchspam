package twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/messages/admin"
	"twitchspam/internal/app/adapters/messages/user"
	"twitchspam/internal/app/adapters/stats"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/antispam"
	"twitchspam/internal/app/domain/banwords"
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
	moderation ports.ModerationPort
	checker    ports.CheckerPort
	admin      ports.AdminPort
	user       ports.UserPort
	bwords     ports.BanwordsPort
	stats      ports.StatsPort

	client *http.Client
}

func New(log logger.Logger, manager *config.Manager, client *http.Client, modChannel string) (*Twitch, error) {
	c := &Twitch{
		log:    log,
		cfg:    manager.Get(),
		client: client,
	}

	channelID, err := c.GetChannelID(modChannel)
	if err != nil {
		return nil, err
	}

	viewerCount, isLive, err := c.GetOnline(modChannel)
	if err != nil {
		return nil, err
	}

	c.stream = stream.NewStream(channelID, modChannel)
	c.stream.SetIslive(isLive)

	c.stats = stats.New()
	if isLive {
		c.log.Info("Stream started")
		c.stream.SetIslive(true)
		c.stats.SetStartTime(time.Now())
		c.stats.SetOnline(viewerCount)
	}

	c.moderation = twitch.New(log, c.cfg, c.stream, client)
	c.checker = antispam.NewCheck(log, c.cfg, c.stream, c.stats)
	c.admin = admin.New(log, manager, c.stream)
	c.user = user.New(log, manager, c.stream, c.stats)
	c.bwords = banwords.New(c.cfg.Banwords)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			viewerCount, isLive, err := c.GetOnline(modChannel)
			if err != nil {
				log.Error("Error getting viewer count", err)
				return
			}

			if isLive {
				c.stats.SetOnline(viewerCount)
			}
		}
	}()
	go c.runEventLoop()

	return c, nil
}

func (c *Twitch) runEventLoop() {
	for {
		err := c.connectAndHandleEvents()
		if err != nil {
			c.log.Warn("Websocket connection lost, retrying...", slog.String("error", err.Error()))
			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Twitch) connectAndHandleEvents() error {
	ws, _, err := websocket.DefaultDialer.Dial("wss://eventsub.wss.twitch.tv/ws", nil)
	if err != nil {
		c.log.Error("Failed to connect to Twitch Twitch", err)
		return err
	}
	defer ws.Close()

	c.log.Info("Connected to Twitch Twitch WebSocket")
	for {
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			c.log.Error("Error while reading websocket message", err)
			return err
		}

		var event EventSubMessage
		if err := json.Unmarshal(msgBytes, &event); err != nil {
			c.log.Error("Failed to decode Twitch message", err, slog.String("event", string(msgBytes)))
			continue
		}

		switch event.Metadata.MessageType {
		case "session_welcome":
			c.log.Debug("Received session_welcome on Twitch")

			var payload SessionWelcomePayload
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				c.log.Error("Failed to decode session_welcome payload", err, slog.String("event", string(msgBytes)))
				break
			}

			if err := c.subscribeEvent("channel.chat.message", "1", map[string]string{
				"broadcaster_user_id": c.stream.ChannelID(),
				"user_id":             c.cfg.App.UserID,
			}, payload.Session.ID); err != nil {
				c.log.Error("Failed to subscribe to event channel.chat.message", err, slog.String("event", "channel.chat.message"))
			}

			if err := c.subscribeEvent("automod.message.hold", "1", map[string]string{
				"broadcaster_user_id": c.stream.ChannelID(),
				"moderator_user_id":   c.cfg.App.UserID,
			}, payload.Session.ID); err != nil {
				c.log.Error("Failed to subscribe to event automod", err, slog.String("event", "automod.message.hold"))
			}

			if err := c.subscribeEvent("stream.online", "1", map[string]string{
				"broadcaster_user_id": c.stream.ChannelID(),
			}, payload.Session.ID); err != nil {
				c.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.online"))
			}

			if err := c.subscribeEvent("stream.offline", "1", map[string]string{
				"broadcaster_user_id": c.stream.ChannelID(),
			}, payload.Session.ID); err != nil {
				c.log.Error("Failed to subscribe to event", err, slog.String("event", "stream.offline"))
			}

			if err := c.subscribeEvent("channel.update", "2", map[string]string{
				"broadcaster_user_id": c.stream.ChannelID(),
			}, payload.Session.ID); err != nil {
				c.log.Error("Failed to subscribe to event", err, slog.String("event", "channel.update"))
			}

			if err := c.subscribeEvent("channel.moderate", "2", map[string]string{
				"broadcaster_user_id": c.stream.ChannelID(),
				"moderator_user_id":   c.cfg.App.UserID,
			}, payload.Session.ID); err != nil {
				c.log.Error("Failed to subscribe to event", err, slog.String("event", "channel.moderate"))
			}

			break
		case "session_keepalive":
			c.log.Trace("Received session_keepalive on Twitch")
			break
		case "notification":
			c.log.Debug("Received notification on Twitch")

			var envelope EventSubEnvelope
			if err := json.Unmarshal(event.Payload, &envelope); err != nil {
				c.log.Error("Failed to decode Twitch envelope", err)
				break
			}

			switch envelope.Subscription.Type {
			case "channel.chat.message":
				var msgEvent ChatMessageEvent
				if err := json.Unmarshal(envelope.Event, &msgEvent); err != nil {
					c.log.Error("Failed to decode channel.chat.message event", err)
					break
				}
				c.log.Debug("New message", slog.String("username", msgEvent.ChatterUserName), slog.String("text", msgEvent.Message.Text))

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

				msg := &ports.ChatMessage{
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

				if adminAction := c.admin.FindMessages(msg); adminAction != admin.None {
					if err := c.SendChatMessage(msg.Broadcaster.UserID, fmt.Sprintf("@%s, %s!", msg.Chatter.Username, adminAction)); err != nil {
						c.log.Error("Failed to send message on chat", err)
					}
					continue
				}

				if userAction := c.user.FindMessages(msg); userAction != user.None {
					if err := c.SendChatMessage(msg.Broadcaster.UserID, fmt.Sprintf("@%s, %s!", msg.Chatter.Username, userAction)); err != nil {
						c.log.Error("Failed to send message on chat", err)
					}
					continue
				}

				action := c.checker.Check(msg)
				switch action.Type {
				case antispam.Ban:
					c.log.Warn("Banword in phrase", slog.String("username", action.Username), slog.String("text", action.Text))
					c.moderation.Ban(action.UserID, action.Reason)
				case antispam.Timeout:
					c.log.Warn("Spam is found", slog.String("username", action.Username), slog.String("text", action.Text), slog.Int("duration", int(action.Duration.Seconds())))
					if c.cfg.Spam.SettingsDefault.Enabled {
						c.moderation.Timeout(action.UserID, int(action.Duration.Seconds()), action.Reason)
					}
				case antispam.Delete:
					c.log.Warn("Muteword in phrase", slog.String("username", action.Username), slog.String("text", action.Text))
					if err := c.DeleteChatMessage(msg.Broadcaster.UserID, msg.Message.ID); err != nil {
						c.log.Error("Failed to delete message on chat", err)
					}
				}
			case "automod.message.hold":
				var am AutomodHoldEvent
				if err := json.Unmarshal(envelope.Event, &am); err != nil {
					c.log.Error("Failed to decode automod event", err)
					break
				}
				c.log.Info("AutoMod held message", slog.String("user_id", am.UserID), slog.String("message_id", am.MessageID), slog.String("text", am.Message.Text))

				text := strings.ToLower(domain.NormalizeText(am.Message.Text))
				words := strings.Fields(text)

				if c.bwords.CheckMessage(words) {
					time.Sleep(time.Duration(c.cfg.Spam.DelayAutomod) * time.Second)
					c.moderation.Ban(am.UserID, "банворд")
				}

				if c.cfg.PunishmentOnline && c.bwords.CheckOnline(text) {
					c.moderation.Ban(am.UserID, "тупое")
				}
			case "stream.online":
				c.log.Info("Stream started")
				c.stream.SetIslive(true)
				c.stats.SetStartTime(time.Now())
			case "stream.offline":
				c.log.Info("Stream ended")
				c.stream.SetIslive(false)
				c.stats.SetEndTime(time.Now())
			case "channel.update":
				var upd ChannelUpdateEvent
				if err := json.Unmarshal(envelope.Event, &upd); err != nil {
					c.log.Error("Failed to decode channel.update event", err)
					break
				}
				c.log.Info("Channel updated", slog.String("title", upd.Title), slog.String("category", upd.CategoryName), slog.String("lang", upd.Language))

				if upd.CategoryName != "" { // TODO
					c.stream.SetCategory(upd.CategoryName)
				}
			case "channel.moderate":
				var modEvent ChannelModerateEvent
				if err := json.Unmarshal(envelope.Event, &modEvent); err != nil {
					c.log.Error("Failed to decode channel.moderate event", err)
					break
				}

				switch modEvent.Action {
				case "delete":
					c.log.Info("The moderator deleted the user's message", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Timeout.Username))
					c.stats.AddDeleted(modEvent.ModeratorUserName)
				case "timeout":
					c.log.Info("The moderator muted the user", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Timeout.Username), slog.Time("expires_at", modEvent.Timeout.ExpiresAt), slog.String("reason", modEvent.Timeout.Reason))
					c.stats.AddTimeout(modEvent.ModeratorUserName)
				case "ban":
					c.log.Info("The moderator banned the user", slog.String("mod_username", modEvent.ModeratorUserName), slog.String("username", modEvent.Ban.Username), slog.String("reason", modEvent.Ban.Reason))
					c.stats.AddBan(modEvent.ModeratorUserName)
				}
			case "session_reconnect":
				c.log.Debug("Received session_reconnect on Twitch")
				return nil
			}
		}
	}
}

func (c *Twitch) subscribeEvent(eventType, version string, condition map[string]string, sessionID string) error {
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

	req.Header.Set("Authorization", "Bearer "+c.cfg.App.OAuth)
	req.Header.Set("Client-Id", c.cfg.App.ClientID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
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
