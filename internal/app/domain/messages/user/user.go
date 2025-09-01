package user

import (
	"golang.org/x/time/rate"
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

const (
	None        ports.ActionType = "none"
	NonParametr ports.ActionType = "не указан параметр"
)

type User struct {
	log          logger.Logger
	manager      *config.Manager
	stream       ports.StreamPort
	stats        ports.StatsPort
	limiterGame  *rate.Limiter
	usersLimiter map[string]*rate.Limiter
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, stats ports.StatsPort) *User {
	return &User{
		log:          log,
		manager:      manager,
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

	if strings.HasPrefix(msg.Message.Text, "!stats") && u.usersLimiter[msg.Chatter.Username].Allow() {
		parts := strings.Fields(msg.Message.Text)
		target := msg.Chatter.Username
		if len(parts) > 1 {
			if parts[1] == "all" {
				return ports.ActionType(u.stats.GetStats())
			}
			target = parts[1]
		}
		return ports.ActionType(u.stats.GetUserStats(target))
	}

	if u.stream.IsLive() && u.stream.Category() != "Just Chatting" && u.limiterGame.Allow() && u.usersLimiter[msg.Chatter.Username].Allow() {
		for _, q := range []string{
			"че за игра",
			"чё за игра",
			"что за игра",
			"как игра называется",
			"как игра называеться",
		} {
			if strings.Contains(text, q) {
				return ports.ActionType(u.stream.Category())
			}
		}
	}

	return None
}
