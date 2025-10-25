package message

import (
	"github.com/prometheus/client_golang/prometheus"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/file_server"
	"twitchspam/internal/app/adapters/message/admin"
	"twitchspam/internal/app/adapters/message/checker"
	"twitchspam/internal/app/adapters/message/user"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/infrastructure/timers"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Message struct {
	log         logger.Logger
	cfg         *config.Config
	api         ports.APIPort
	stream      ports.StreamPort
	template    ports.TemplatePort
	admin, user ports.CommandPort
	checker     ports.CheckerPort

	messages ports.StorePort[storage.Message]
	timeouts ports.StorePort[int]
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, api ports.APIPort, client *http.Client) *Message {
	cfg := manager.Get()
	fs := file_server.New(log, client)
	timer := timers.NewTimingWheel(100*time.Millisecond, 600)

	m := &Message{
		log:    log,
		cfg:    cfg,
		api:    api,
		stream: stream,
		template: template.New(
			template.WithAliases(cfg.Aliases, cfg.AliasGroups, cfg.GlobalAliases),
			template.WithPlaceholders(stream),
			template.WithBanwords(log, cfg.Banwords.Words, cfg.Banwords.Regexp),
			template.WithMword(cfg.Mword, cfg.MwordGroup),
		),
		messages: storage.New[storage.Message](50, time.Duration(cfg.WindowSecs)*time.Second),
		timeouts: storage.New[int](15, 0),
	}
	m.admin = admin.New(log, manager, stream, api, m.template, fs, timer, m.messages)
	m.user = user.New(log, manager, stream, m.template, fs, api)
	m.checker = checker.NewCheck(log, cfg, stream, m.template, m.messages, m.timeouts, client)

	for cmd, data := range cfg.Commands {
		if data.Timer == nil {
			continue
		}

		(&admin.AddTimer{Cfg: cfg, Timers: timer, Stream: stream, Api: api}).AddTimer(cmd, data)
	}

	return m
}

func (m *Message) Check(msg *domain.ChatMessage) {
	m.log.Trace("Processing new message", slog.String("username", msg.Chatter.Username), slog.String("message", msg.Message.Text.Text()))
	if m.stream.IsLive() {
		m.stream.Stats().AddMessage(msg.Chatter.Username)
		m.log.Trace("Added message to stream stats", slog.String("channel", m.stream.ChannelName()), slog.String("username", msg.Chatter.Username))
	}

	m.messages.Push(msg.Chatter.Username, msg.Message.ID, storage.Message{
		Data:           msg,
		Time:           time.Now(),
		IgnoreAntispam: !m.cfg.Enabled || !m.template.SpamPause().CanProcess() || !m.cfg.Spam.SettingsDefault.Enabled,
	})
	m.log.Trace("Message pushed to storage", slog.String("username", msg.Chatter.Username), slog.String("message_id", msg.Message.ID))

	if !strings.HasPrefix(msg.Message.Text.Text(), "!am al ") && !strings.HasPrefix(msg.Message.Text.Text(), "!am alg ") {
		startProcessing := time.Now()

		text, ok := m.template.Aliases().Replace(msg.Message.Text.Words(domain.RemovePunctuationOption))
		if ok {
			m.log.Debug("Message text replaced via alias",
				slog.String("username", msg.Chatter.Username),
				slog.String("original_text", msg.Message.Text.Text()),
				slog.String("new_text", text),
			)
			msg.Message.Text.ReplaceOriginal(text)
		} else {
			m.log.Trace("No alias replacement applied", slog.String("username", msg.Chatter.Username))
		}

		endProcessing := time.Since(startProcessing).Seconds()
		metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "aliases"}).Observe(endProcessing)
	}

	if adminAction := m.admin.FindMessages(msg); adminAction != nil {
		adminAction.ReplyUsername = msg.Chatter.Username
		m.api.SendChatMessages(m.stream.ChannelID(), adminAction)
		return
	}

	if userAction := m.user.FindMessages(msg); userAction != nil {
		if userAction.IsReply && userAction.ReplyUsername == "" {
			userAction.ReplyUsername = msg.Chatter.Username
		}
		m.api.SendChatMessages(m.stream.ChannelID(), userAction)
		return
	}

	action := m.checker.Check(msg, true)
	m.getAction(action, msg)
}

func (m *Message) CheckAutomod(msg *domain.ChatMessage) {
	if m.stream.IsLive() {
		m.stream.Stats().AddMessage(msg.Chatter.Username)
	}

	if m.cfg.Enabled && m.cfg.Automod.Enabled {
		return
	}

	if m.cfg.Automod.Delay > 0 {
		time.Sleep(time.Duration(m.cfg.Automod.Delay) * time.Second)
	}

	if msg.Message.Text.Text(domain.RemoveDuplicateLettersOption) == "(" {
		err := m.api.ManageHeldAutoModMessage(m.cfg.App.UserID, msg.Message.ID, "ALLOW")
		if err != nil {
			m.log.Error("Failed to manage held automod", err)
		}
	}

	action := m.checker.Check(msg, false)
	m.getAction(action, msg)
}

func (m *Message) getAction(action *ports.CheckerAction, msg *domain.ChatMessage) {
	switch action.Type {
	case checker.None:
		return
	case checker.Ban:
		m.log.Warn("Ban user", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Text()))
		m.api.BanUser(m.stream.ChannelName(), m.stream.ChannelID(), msg.Chatter.UserID, action.Reason)
	case checker.Timeout:
		m.log.Warn("Timeout user", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Text()), slog.Int("duration", int(action.Duration.Seconds())))
		m.api.TimeoutUser(m.stream.ChannelName(), m.stream.ChannelID(), msg.Chatter.UserID, int(action.Duration.Seconds()), action.Reason)
	case checker.Delete:
		m.log.Warn("Delete message", slog.String("username", msg.Chatter.Username), slog.String("text", msg.Message.Text.Text()))
		if err := m.api.DeleteChatMessage(m.stream.ChannelName(), m.stream.ChannelID(), msg.Message.ID); err != nil {
			m.log.Error("Failed to delete message on chat", err)
		}
	}
}
