package user

import (
	"golang.org/x/time/rate"
	"strconv"
	"strings"
	"sync"
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
	muLimiter    sync.Mutex
	usersLimiter map[string]*rate.Limiter
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, template ports.TemplatePort, fs ports.FileServerPort) *User {
	return &User{
		log:          log,
		manager:      manager,
		stream:       stream,
		template:     template,
		fs:           fs,
		usersLimiter: make(map[string]*rate.Limiter),
	}
}

func (u *User) FindMessages(msg *ports.ChatMessage) *ports.AnswerType {
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

func (u *User) handleStats(msg *ports.ChatMessage) *ports.AnswerType {
	if !strings.HasPrefix(msg.Message.Text.Original, "!stats") {
		return nil
	}

	if !u.allowUser(msg.Chatter.Username) {
		return nil
	}

	target := msg.Chatter.Username
	words := msg.Message.Text.WordsLower()

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

func (u *User) handleCommands(msg *ports.ChatMessage) *ports.AnswerType {
	var text, replyUsername string
	if msg.Reply != nil {
		replyUsername = msg.Reply.ParentChatter.Username
	}

	words := msg.Message.Text.WordsLower()
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if strings.HasPrefix(word, "@") && replyUsername == "" {
			replyUsername = strings.TrimPrefix(word, "@")
		}

		cfg := u.manager.Get()
		if link, ok := cfg.Commands[word]; ok {
			if !u.allowUser(msg.Chatter.Username) || !u.allowCommand(word, link.Limiter) {
				return nil
			}

			text = u.template.Placeholders().ReplaceAll(link.Text, words)
			if text == "" {
				return nil
			}

			if replyUsername != "" {
				break
			}
		}
	}

	if text == "" {
		return nil
	}

	return &ports.AnswerType{
		Text:          []string{text},
		IsReply:       true,
		ReplyUsername: replyUsername,
	}
}

func (u *User) ensureUserLimiter(username string, limiter *config.Limiter) {
	if limiter == nil {
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
	return ok && limiter.Allow()
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
