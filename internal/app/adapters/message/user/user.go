package user

import (
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type User struct {
	log          logger.Logger
	manager      *config.Manager
	stream       ports.StreamPort
	template     ports.TemplatePort
	fs           ports.FileServerPort
	api          ports.APIPort
	muLimiter    sync.Mutex
	usersLimiter map[string]*rate.Limiter
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, template ports.TemplatePort, fs ports.FileServerPort, api ports.APIPort) *User {
	return &User{
		log:          log,
		manager:      manager,
		stream:       stream,
		template:     template,
		fs:           fs,
		api:          api,
		usersLimiter: make(map[string]*rate.Limiter),
	}
}

func (u *User) FindMessages(msg *message.ChatMessage) *ports.AnswerType {
	cfg := u.manager.Get()
	u.ensureUserLimiter(msg.Chatter.Username, cfg.Limiter)

	if !cfg.Channels[u.stream.ChannelName()].Enabled {
		u.log.Debug("Bot disabled, skipping message",
			slog.String("username", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
		)
		return nil
	}

	startProcessing := time.Now()
	if action := u.handleStats(msg); action != nil {
		u.log.Debug("Handled stats command",
			slog.String("username", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
		)
		return action
	}
	endProcessing := time.Since(startProcessing).Seconds()
	metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "stats"}).Observe(endProcessing)

	startProcessing = time.Now()
	if action := u.handleCommands(msg); action != nil {
		u.log.Debug("Handled user command",
			slog.String("username", msg.Chatter.Username),
			slog.String("message", msg.Message.Text.Text()),
		)
		return action
	}
	endProcessing = time.Since(startProcessing).Seconds()
	metrics.ModulesProcessingTime.With(prometheus.Labels{"module": "user_commands"}).Observe(endProcessing)

	return nil
}

func (u *User) handleStats(msg *message.ChatMessage) *ports.AnswerType {
	if !strings.HasPrefix(msg.Message.Text.Text(message.LowerOption), "!stats") {
		u.log.Trace("Message does not start with !stats, skipping", slog.String("username", msg.Chatter.Username))
		return nil
	}

	if u.stream.IsLive() {
		u.log.Debug("Stream is live, stats command not allowed", slog.String("username", msg.Chatter.Username))
		return nil
	}

	if !u.allowUser(msg.Chatter.Username) {
		u.log.Debug("User not allowed to access stats due to limiter", slog.String("username", msg.Chatter.Username))
		return nil
	}

	target := msg.Chatter.Username
	words := msg.Message.Text.Words(message.LowerOption)

	u.log.Info("User command executed",
		slog.String("user", msg.Chatter.Username),
		slog.String("message", msg.Message.Text.Text()),
	)

	if len(words) > 1 {
		switch words[1] {
		case "all":
			return u.stream.Stats().GetStats()
		case "top":
			count := 0
			if len(words) > 2 {
				count, _ = strconv.Atoi(words[2])
			}
			return u.stream.Stats().GetTopStats(count)
		default:
			target = words[1]
		}
	}

	return u.stream.Stats().GetUserStats(target)
}

func (u *User) handleCommands(msg *message.ChatMessage) *ports.AnswerType {
	var replyUsername string
	if msg.Reply != nil {
		replyUsername = msg.Reply.ParentChatter.Username
		u.log.Debug("Message is a reply", slog.String("reply_username", replyUsername))
	}

	text, count, isAnnounce := "", 1, false
	cfg := u.manager.Get()
	words := msg.Message.Text.Words()

	for i, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if strings.HasPrefix(word, "@") && replyUsername == "" {
			replyUsername = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(word, "@"), ","))
			u.log.Debug("Detected reply username from @ mention", slog.String("reply_username", replyUsername))
		}

		cmd, ok := cfg.Channels[u.stream.ChannelName()].Commands[strings.ToLower(word)]
		if !ok {
			continue
		}

		if cmd.Options != nil && cmd.Options.IsPrivate != nil && *cmd.Options.IsPrivate && !msg.Chatter.IsBroadcaster && !msg.Chatter.IsMod {
			u.log.Warn("Private command attempted by unauthorized user",
				slog.String("username", msg.Chatter.Username),
				slog.String("command", word))
			return nil
		}

		if (msg.Chatter.IsBroadcaster || msg.Chatter.IsMod) && len(words) > i+1 {
			next := strings.TrimSpace(words[i+1])

			if strings.HasSuffix(next, "a") {
				isAnnounce = true
				next = strings.TrimSuffix(next, "a")
				u.log.Debug("Announcement mode detected", slog.Bool("is_announce", isAnnounce))
			}

			if strings.HasSuffix(next, "а") {
				isAnnounce = true
				next = strings.TrimSuffix(next, "а")
			}

			c, err := strconv.Atoi(next)
			if err == nil && c > 0 {
				if c > 100 {
					c = 100
				}
				count = c
			}
		}

		mode := config.AlwaysMode
		if cmd.Options != nil && cmd.Options.Mode != nil {
			mode = *cmd.Options.Mode
		}

		if (mode == config.OnlineMode && !u.stream.IsLive()) || (mode == config.OfflineMode && u.stream.IsLive()) {
			u.log.Trace("Command mode does not match stream status, skipping",
				slog.String("command", word),
				slog.Int("option_mode", mode),
				slog.Bool("stream_live", u.stream.IsLive()),
			)
			return nil
		}

		if !u.allowUser(msg.Chatter.Username) {
			u.log.Warn("User blocked by limiter during command execution", slog.String("username", msg.Chatter.Username), slog.String("command", word))
			return nil
		}

		if !u.allowCommand(word, cmd.Limiter) {
			u.log.Warn("Command blocked by limiter", slog.String("username", msg.Chatter.Username), slog.String("command", word))
			return nil
		}

		text = u.template.Placeholders().ReplaceAll(cmd.Text, words)
		if text == "" {
			return nil
		}

		metrics.UserCommands.With(prometheus.Labels{"channel": msg.Broadcaster.Login, "command": strings.TrimSpace(word)}).Inc()
		if replyUsername != "" {
			break
		}
	}

	if text == "" {
		u.log.Trace("No command matched or text empty, returning nil")
		return nil
	}

	u.log.Info("User command executed",
		slog.String("user", msg.Chatter.Username),
		slog.String("message", msg.Message.Text.Text()),
	)

	msgs := make([]string, count)
	for i := range count {
		msgs[i] = text
	}

	if _, ok := u.manager.Get().UsersTokens[u.stream.ChannelID()]; ok && isAnnounce {
		u.log.Info("Sending chat announcement", slog.String("channel_id", u.stream.ChannelID()), slog.Int("count", count))
		u.api.SendChatAnnouncements(u.stream.ChannelID(), &ports.AnswerType{
			Text:    msgs,
			IsReply: false,
		}, "primary")
		return nil
	}

	noReply := ((msg.Chatter.IsBroadcaster || msg.Chatter.IsMod) && replyUsername == "") || count > 1
	return &ports.AnswerType{
		Text:          msgs,
		IsReply:       !noReply,
		ReplyUsername: replyUsername,
	}
}

func (u *User) ensureUserLimiter(username string, limiter config.Limiter) {
	if limiter.Requests == 0 || limiter.Per == 0 {
		return
	}

	u.muLimiter.Lock()
	if _, exists := u.usersLimiter[username]; !exists {
		u.usersLimiter[username] = rate.NewLimiter(rate.Every(limiter.Per), limiter.Requests)
	}
	u.muLimiter.Unlock()
}

func (u *User) allowUser(username string) bool {
	u.muLimiter.Lock()
	defer u.muLimiter.Unlock()

	limiter, ok := u.usersLimiter[username]
	if !ok {
		return false
	}

	return limiter.Allow()
}

func (u *User) allowCommand(command string, limiter *config.Limiter) bool {
	if limiter == nil {
		return true
	}

	if limiter.Rate == nil && limiter.Per > 0 && limiter.Requests > 0 {
		if err := u.manager.Update(func(cfg *config.Config) {
			cfg.Channels[u.stream.ChannelName()].Commands[command].Limiter.Rate = rate.NewLimiter(rate.Every(limiter.Per), limiter.Requests)
		}); err != nil {
			u.log.Error("Failed to update command rate limiter", err)
		}
	}

	return limiter.Rate == nil || limiter.Rate.Allow()
}
