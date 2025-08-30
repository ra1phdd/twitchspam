package user

import (
	"golang.org/x/time/rate"
	"strings"
	"time"
	"twitchspam/config"
	"twitchspam/internal/app/domain"
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

func (u *User) FindMessages(irc *ports.IRCMessage) ports.ActionType {
	text := strings.ToLower(domain.NormalizeText(irc.Text))
	if _, exists := u.usersLimiter[irc.Username]; !exists {
		u.usersLimiter[irc.Username] = rate.NewLimiter(rate.Every(time.Minute), 3)
	}

	if strings.HasPrefix(irc.Text, "!stats") && u.usersLimiter[irc.Username].Allow() {
		parts := strings.Fields(irc.Text)
		if len(parts) < 2 {
			return NonParametr
		}

		args := parts[1:]
		if len(args) > 0 && args[0] == "all" {
			return ports.ActionType(u.stats.GetStats())
		}

		if irc.IsMod {
			return ports.ActionType(u.stats.GetModeratorStats(irc.Username))
		}
		return ports.ActionType(u.stats.GetUserStats(irc.Username))
	}

	if u.stream.IsLive() && u.stream.Category() != "Just Chatting" && u.limiterGame.Allow() && u.usersLimiter[irc.Username].Allow() {
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

	for _, q := range []string{
		"афсигга плохой",
		"афсига плохой",
		"афсугга плохой",
		"афсуга плохой",
	} {
		if strings.Contains(text, q) && u.usersLimiter[irc.Username].Allow() {
			return "(("
		}
	}

	for _, q := range []string{
		"афсигга хороший",
		"афсига хороший",
		"афсугга хороший",
		"афсуга хороший",
	} {
		if strings.Contains(text, q) && u.usersLimiter[irc.Username].Allow() {
			return "nya"
		}
	}

	return None
}
