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
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/domain/trusts"
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
	trusts      ports.TrustsPort
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
		trusts: trusts.New(cfg.Channels[stream.ChannelName()].Roles, cfg.GlobalRoles, cfg.Channels[stream.ChannelName()].Trusts),
		stream: stream,
		template: template.New(
			template.WithAliases(cfg.Channels[stream.ChannelName()].Aliases, cfg.Channels[stream.ChannelName()].AliasGroups, cfg.GlobalAliases),
			template.WithPlaceholders(stream),
			template.WithBanwords(cfg.Banwords),
			template.WithMword(cfg.Channels[stream.ChannelName()].Mword, cfg.Channels[stream.ChannelName()].MwordGroup),
		),
		messages: storage.NewV2[storage.Message](
			storage.WithSubCapacity[storage.Message](50),
			storage.WithMode[storage.Message](storage.ExpireAfterAccess),
			storage.WithSubTTL[storage.Message](5*time.Minute),
		),
		timeouts: storage.NewV2[int](
			storage.WithSubCapacity[int](15),
		),
	}
	m.admin = admin.New(log, manager, stream, m.trusts, api, m.template, fs, timer, m.messages)
	m.user = user.New(log, manager, stream, m.trusts, m.template, fs, api)
	m.checker = checker.NewCheck(log, cfg, stream, m.trusts, m.template, m.messages, m.timeouts, client)

	return m
}

func (m *Message) Check(msg *message.ChatMessage) {
	startProcessing := time.Now()
	m.log.Trace("Processing new message", slog.String("username", msg.Chatter.Username), slog.String("message", msg.Message.Text.Text()))
	if m.stream.IsLive() {
		m.stream.Stats().AddMessage(msg.Chatter.Username)
		m.log.Trace("Added message to stream stats", slog.String("channel", m.stream.ChannelName()), slog.String("username", msg.Chatter.Username))
	}

	startModuleProcessing := time.Now()
	m.messages.Push(msg.Chatter.Username, msg.Message.ID, storage.Message{
		Data:           msg,
		Time:           time.Now(),
		IgnoreAntispam: !m.cfg.Channels[m.stream.ChannelName()].Enabled || !m.template.SpamPause().CanProcess() || !m.cfg.Channels[m.stream.ChannelName()].Spam.SettingsDefault.Enabled,
	})
	endModuleProcessing := time.Since(startModuleProcessing).Seconds()
	metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "push_message"}).Observe(endModuleProcessing)
	m.log.Trace("Message pushed to storage", slog.String("username", msg.Chatter.Username), slog.String("message_id", msg.Message.ID))

	skip := false
	for _, prefix := range []string{
		"!am title ", "!am cat ", "!am mw ", "!am mwg ",
		"!am cmd ", "!am ex ", "!am emote ex ",
		"!am pred ", "!am poll ", "!am nuke ",
		"!am mark ", "!stats ",
	} {
		if strings.HasPrefix(msg.Message.Text.Text(), prefix) {
			skip = true
			break
		}
	}

	if !skip {
		startModuleProcessing = time.Now()

		text, ok := m.template.Aliases().Replace(msg.Message.Text.Words())
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

		endModuleProcessing = time.Since(startModuleProcessing).Seconds()
		metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "aliases"}).Observe(endModuleProcessing)
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
	endProcessing := time.Since(startProcessing).Seconds()
	metrics.MessageProcessingTime.Observe(endProcessing)

	admin.ExecuteModAction(m.log, m.api, m.stream, action, msg.Chatter.UserID, msg.Chatter.Username, msg.Message.ID, msg.Message.Text.Text())
}

func (m *Message) CheckAutomod(msg *message.ChatMessage) {
	startProcessing := time.Now()
	if m.stream.IsLive() {
		m.stream.Stats().AddMessage(msg.Chatter.Username)
	}

	if !m.cfg.Channels[m.stream.ChannelName()].Enabled || !m.cfg.Channels[m.stream.ChannelName()].Automod.Enabled {
		return
	}

	if m.cfg.Channels[m.stream.ChannelName()].Automod.Delay > 0 {
		time.Sleep(time.Duration(m.cfg.Channels[m.stream.ChannelName()].Automod.Delay) * time.Second)
	}

	if msg.Message.Text.Text(message.RemoveDuplicateLettersOption) == "(" {
		err := m.api.ManageHeldAutoModMessage(m.cfg.App.UserID, msg.Message.ID, "ALLOW")
		if err != nil {
			m.log.Error("Failed to manage held automod", err)
		}
	}

	action := m.checker.Check(msg, false)
	endProcessing := time.Since(startProcessing).Seconds()
	metrics.MessageProcessingTime.Observe(endProcessing)

	admin.ExecuteModAction(m.log, m.api, m.stream, action, msg.Chatter.UserID, msg.Chatter.Username, msg.Message.ID, msg.Message.Text.Text())
}
