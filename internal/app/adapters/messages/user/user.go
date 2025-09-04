package user

import (
	"golang.org/x/time/rate"
	"slices"
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

const (
	None ports.ActionType = "none"
)

type User struct {
	log          logger.Logger
	cfg          *config.Config
	stream       ports.StreamPort
	stats        ports.StatsPort
	limiterGame  *rate.Limiter
	usersLimiter map[string]*rate.Limiter
}

func New(log logger.Logger, cfg *config.Config, stream ports.StreamPort, stats ports.StatsPort) *User {
	return &User{
		log:          log,
		cfg:          cfg,
		stream:       stream,
		stats:        stats,
		limiterGame:  rate.NewLimiter(rate.Every(30*time.Second), 1),
		usersLimiter: make(map[string]*rate.Limiter),
	}
}

func (u *User) FindMessages(msg *ports.ChatMessage) ports.ActionType {
	text := strings.ToLower(domain.NormalizeText(msg.Message.Text))
	if _, exists := u.usersLimiter[msg.Chatter.Username]; !exists {
		u.usersLimiter[msg.Chatter.Username] = rate.NewLimiter(rate.Every(time.Minute), 3)
	}

	if action := u.handleStats(msg); action != None {
		return action
	}

	if !u.cfg.Enabled {
		return None
	}

	if action := u.handleGameQuery(msg, text); action != None {
		return action
	}

	return None
}

func (u *User) handleStats(msg *ports.ChatMessage) ports.ActionType {
	if !strings.HasPrefix(msg.Message.Text, "!stats") || !u.usersLimiter[msg.Chatter.Username].Allow() {
		return None
	}

	parts := strings.Fields(msg.Message.Text)
	target := msg.Chatter.Username
	if len(parts) > 1 && parts[1] != "all" {
		target = parts[1]
	}

	if len(parts) > 1 && parts[1] == "all" {
		return ports.ActionType(u.stats.GetStats())
	}
	return ports.ActionType(u.stats.GetUserStats(target))
}

func (u *User) handleGameQuery(msg *ports.ChatMessage, text string) ports.ActionType {
	if !u.stream.IsLive() || u.stream.Category() == "Just Chatting" ||
		!u.limiterGame.Allow() || !u.usersLimiter[msg.Chatter.Username].Allow() {
		return None
	}

	queries := []string{
		"че за игра",
		"чё за игра",
		"что за игра",
		"как игра называется",
		"как игра называеться",
	}

	if slices.Contains(queries, text) {
		return ports.ActionType(u.stream.Category())
	}
	return None
}
