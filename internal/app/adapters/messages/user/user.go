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

type User struct {
	log          logger.Logger
	cfg          *config.Config
	stream       ports.StreamPort
	stats        ports.StatsPort
	regexp       ports.RegexPort
	aliases      ports.AliasesPort
	limiterGame  *rate.Limiter
	usersLimiter map[string]*rate.Limiter
}

func New(log logger.Logger, cfg *config.Config, stream ports.StreamPort, stats ports.StatsPort, regexp ports.RegexPort, aliases ports.AliasesPort) *User {
	return &User{
		log:          log,
		cfg:          cfg,
		stream:       stream,
		stats:        stats,
		regexp:       regexp,
		aliases:      aliases,
		limiterGame:  rate.NewLimiter(rate.Every(30*time.Second), 1),
		usersLimiter: make(map[string]*rate.Limiter),
	}
}

func (u *User) FindMessages(msg *ports.ChatMessage) *ports.AnswerType {
	text := strings.ToLower(domain.NormalizeText(msg.Message.Text))
	if _, exists := u.usersLimiter[msg.Chatter.Username]; !exists {
		u.usersLimiter[msg.Chatter.Username] = rate.NewLimiter(rate.Every(time.Minute), 3)
	}
	words := strings.Fields(msg.Message.Text)

	if action := u.handleStats(msg, words); action != nil {
		return action
	}

	if !u.cfg.Enabled {
		return nil
	}

	if action := u.handleLinks(msg, words); action != nil {
		return action
	}

	if action := u.handleAnswers(text); action != nil {
		return action
	}

	if action := u.handleGameQuery(msg, text); action != nil {
		return action
	}

	return nil
}

func (u *User) handleStats(msg *ports.ChatMessage, words []string) *ports.AnswerType {
	if !strings.HasPrefix(msg.Message.Text, "!stats") || !u.usersLimiter[msg.Chatter.Username].Allow() {
		return nil
	}

	target := msg.Chatter.Username
	if len(words) > 1 && words[1] != "all" {
		target = words[1]
	}

	if len(words) > 1 && words[1] == "all" {
		return &ports.AnswerType{
			Text:    []string{u.stats.GetStats()},
			IsReply: false,
		}
	}
	return &ports.AnswerType{
		Text:    []string{u.stats.GetUserStats(target)},
		IsReply: true,
	}
}

func (u *User) handleLinks(msg *ports.ChatMessage, words []string) *ports.AnswerType {
	var text, replyUsername string
	for _, word := range words {
		if len(word) == 0 {
			continue
		}

		if word[0] == '@' && replyUsername == "" {
			replyUsername = strings.TrimSuffix(word[1:], ",")
			continue
		}

		if link, ok := u.cfg.Links[word]; ok && text == "" {
			text = u.aliases.ReplacePlaceholders(link.Text, words)
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

func (u *User) handleAnswers(text string) *ports.AnswerType {
	words := u.regexp.SplitWordsBySpace(text)
	for _, answer := range u.cfg.Answers {
		if !answer.Enabled {
			continue
		}

		for _, phrase := range answer.Words {
			if phrase == "" {
				continue
			}

			if u.regexp.MatchPhrase(words, phrase) {
				return &ports.AnswerType{
					Text:    []string{u.aliases.ReplacePlaceholders(answer.Text, words)},
					IsReply: true,
				}
			}
		}

		for _, re := range answer.Regexp {
			if re == nil {
				continue
			}

			if isMatch, _ := re.MatchString(text); isMatch {
				return &ports.AnswerType{
					Text:    []string{u.aliases.ReplacePlaceholders(answer.Text, words)},
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
