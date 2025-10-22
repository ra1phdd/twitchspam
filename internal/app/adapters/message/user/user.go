package user

import (
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/time/rate"
	"strconv"
	"strings"
	"sync"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/domain"
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

func (u *User) FindMessages(msg *domain.ChatMessage) *ports.AnswerType {
	cfg := u.manager.Get()
	u.ensureUserLimiter(msg.Chatter.Username, cfg.Limiter)

	if !cfg.Enabled {
		return nil
	}

	if action := u.handleStats(msg); action != nil {
		return action
	}

	if action := u.handleCommands(msg); action != nil {
		return action
	}

	return nil
}

func (u *User) handleStats(msg *domain.ChatMessage) *ports.AnswerType {
	if !strings.HasPrefix(msg.Message.Text.Text(domain.LowerOption), "!stats") {
		return nil
	}

	if u.stream.IsLive() || !u.allowUser(msg.Chatter.Username) {
		return nil
	}

	target := msg.Chatter.Username
	words := msg.Message.Text.Words(domain.LowerOption)

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

func (u *User) handleCommands(msg *domain.ChatMessage) *ports.AnswerType {
	var replyUsername string
	if msg.Reply != nil {
		replyUsername = msg.Reply.ParentChatter.Username
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
		}

		cmd, ok := cfg.Commands[strings.ToLower(word)]
		if !ok {
			continue
		}

		if cmd.Options.IsPrivate && !msg.Chatter.IsBroadcaster && !msg.Chatter.IsMod {
			return nil
		}

		if (msg.Chatter.IsBroadcaster || msg.Chatter.IsMod) && len(words) > i+1 {
			next := strings.TrimSpace(words[i+1])

			if strings.HasSuffix(next, "a") {
				isAnnounce = true
				next = strings.TrimSuffix(next, "a")
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

		if (cmd.Options.Mode == config.OnlineMode && !u.stream.IsLive()) ||
			(cmd.Options.Mode == config.OfflineMode && u.stream.IsLive()) {
			return nil
		}

		if !u.allowUser(msg.Chatter.Username) || !u.allowCommand(word, cmd.Limiter) {
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
		return nil
	}
	var msgs []string
	for range count {
		msgs = append(msgs, text)
	}

	if _, ok := u.manager.Get().UsersTokens[u.stream.ChannelID()]; ok && isAnnounce {
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
			cfg.Commands[command].Limiter.Rate = rate.NewLimiter(rate.Every(limiter.Per), limiter.Requests)
		}); err != nil {
			u.log.Error("Failed to update command rate limiter", err)
		}
	}

	return limiter.Rate == nil || limiter.Rate.Allow()
}
