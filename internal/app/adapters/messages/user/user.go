package user

import (
	"golang.org/x/time/rate"
	"slices"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type User struct {
	log          logger.Logger
	cfg          *config.Config
	stream       ports.StreamPort
	stats        ports.StatsPort
	template     ports.TemplatePort
	fs           ports.FileServerPort
	limiterGame  *rate.Limiter
	usersLimiter map[string]*rate.Limiter
}

func New(log logger.Logger, cfg *config.Config, stream ports.StreamPort, stats ports.StatsPort, template ports.TemplatePort, fs ports.FileServerPort) *User {
	return &User{
		log:          log,
		cfg:          cfg,
		stream:       stream,
		stats:        stats,
		template:     template,
		fs:           fs,
		limiterGame:  rate.NewLimiter(rate.Every(30*time.Second), 1),
		usersLimiter: make(map[string]*rate.Limiter),
	}
}

func (u *User) FindMessages(msg *ports.ChatMessage) *ports.AnswerType {
	if _, exists := u.usersLimiter[msg.Chatter.Username]; !exists {
		u.usersLimiter[msg.Chatter.Username] = rate.NewLimiter(rate.Every(time.Minute), 3)
	}

	if action := u.handleStats(msg); action != nil {
		return action
	}

	if !u.cfg.Enabled {
		return nil
	}

	if action := u.handleCommands(msg); action != nil {
		return action
	}

	if action := u.handleAnswers(msg); action != nil {
		return action
	}

	if action := u.handleGameQuery(msg, msg.Message.Text.Lower()); action != nil {
		return action
	}

	return nil
}

func (u *User) handleStats(msg *ports.ChatMessage) *ports.AnswerType {
	if !strings.HasPrefix(msg.Message.Text.Original, "!stats") || !u.usersLimiter[msg.Chatter.Username].Allow() {
		return nil
	}

	target := msg.Chatter.Username
	words := msg.Message.Text.WordsLower()

	if len(words) > 1 {
		switch words[1] {
		case "all":
			return u.stats.GetStats()
		case "top":
			count := 0
			if len(words) > 2 {
				count, _ = strconv.Atoi(words[2])
			}
			return u.stats.GetTopStats(count)
		default:
			target = words[1]
		}
	}

	return u.stats.GetUserStats(target)
}

func (u *User) handleCommands(msg *ports.ChatMessage) *ports.AnswerType {
	words := msg.Message.Text.WordsLower()
	var text, replyUsername string
	for _, word := range words {
		if len(word) == 0 {
			continue
		}

		if word[0] == '@' && replyUsername == "" {
			replyUsername = strings.TrimSuffix(word[1:], ",")
			continue
		}

		if link, ok := u.cfg.Links[word]; ok {
			text = u.template.ReplacePlaceholders(link.Text, words)
			break
		}
	}

	if replyUsername == "" && msg.Reply != nil {
		replyUsername = msg.Reply.ParentChatter.Username
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

func (u *User) handleAnswers(msg *ports.ChatMessage) *ports.AnswerType {
	words := msg.Message.Text.WordsLower()
	for _, answer := range u.cfg.Answers {
		if !answer.Enabled {
			continue
		}

		for _, phrase := range answer.Words {
			if phrase == "" {
				continue
			}

			if u.template.MatchPhrase(words, phrase) {
				text := u.template.ReplacePlaceholders(answer.Text, words)
				if text == "" {
					return nil
				}

				return &ports.AnswerType{
					Text:    []string{text},
					IsReply: true,
				}
			}
		}

		for _, re := range answer.Regexp {
			if re == nil {
				continue
			}

			if isMatch, _ := re.MatchString(msg.Message.Text.Lower()); isMatch {
				text := u.template.ReplacePlaceholders(answer.Text, words)
				if text == "" {
					return nil
				}

				return &ports.AnswerType{
					Text:    []string{text},
					IsReply: true,
				}
			}
		}
	}

	return nil
}

func (u *User) handleGameQuery(msg *ports.ChatMessage, text string) *ports.AnswerType {
	if !u.stream.IsLive() || u.stream.Category() == "Just Chatting" ||
		!u.limiterGame.Allow() || !u.usersLimiter[msg.Chatter.Username].Allow() {
		return nil
	}

	queries := []string{
		"че за игра",
		"чё за игра",
		"что за игра",
		"как игра называется",
		"как игра называеться",
	}

	if slices.Contains(queries, text) {
		return &ports.AnswerType{
			Text:    []string{u.stream.Category()},
			IsReply: true,
		}
	}
	return nil
}
