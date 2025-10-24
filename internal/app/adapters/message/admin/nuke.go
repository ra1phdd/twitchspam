package admin

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/message/checker"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Nuke struct {
	re       *regexp.Regexp
	reWords  *regexp.Regexp
	log      logger.Logger
	api      ports.APIPort
	template ports.TemplatePort
	stream   ports.StreamPort
	messages ports.StorePort[storage.Message]
}

func (n *Nuke) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return n.handleNuke(text)
}

func (n *Nuke) handleNuke(text *domain.MessageText) *ports.AnswerType {
	// !am nuke <*наказание> <*длительность> <*scrollback> <слова/фразы через запятую или regex>
	matches := n.re.FindStringSubmatch(text.Text())
	if len(matches) != 5 {
		return nonParametr
	}

	var globalErrs []string
	punishment := config.Punishment{
		Action:   "timeout",
		Duration: 60,
	}
	duration := 5 * time.Minute

	scrollback := n.messages.GetTTL()
	if scrollback > 60*time.Second {
		scrollback = 60 * time.Second
	}

	if strings.TrimSpace(matches[1]) != "" {
		p, err := n.template.Punishment().Parse(strings.TrimSpace(matches[1]), false)
		if err != nil {
			globalErrs = append(globalErrs, "не удалось распарсить наказание, применено дефолтное (60))")
		} else {
			punishment = p
		}
	}

	if strings.TrimSpace(matches[2]) != "" {
		if val, ok := n.template.Parser().ParseIntArg(strings.TrimSpace(matches[2]), 1, 3600); ok {
			duration = time.Duration(val) * time.Second
		}
	}

	if strings.TrimSpace(matches[3]) != "" {
		if val, ok := n.template.Parser().ParseIntArg(strings.TrimSpace(matches[3]), 1, 180); ok {
			scrollback = time.Duration(val) * time.Second
		}
	}

	if strings.TrimSpace(matches[4]) == "" {
		return &ports.AnswerType{
			Text:    []string{"не указаны слова для массбана!"},
			IsReply: true,
		}
	}
	wordsMatches := n.reWords.FindAllStringSubmatch(strings.TrimSpace(matches[4]), -1)

	var containsWords, words []string
	var re *regexp.Regexp
	for _, m := range wordsMatches {
		switch {
		case strings.TrimSpace(m[1]) != "": // r'...'
			var err error
			re, err = regexp.Compile(strings.TrimSpace(m[1]))
			if err != nil {
				return invalidRegex
			}
		case strings.TrimSpace(m[2]) != "": // r"..."
			var err error
			re, err = regexp.Compile(strings.TrimSpace(m[2]))
			if err != nil {
				return invalidRegex
			}
		case strings.TrimSpace(m[3]) != "": // '...'
			words = append(words, strings.TrimSpace(m[3]))
		case strings.TrimSpace(m[4]) != "": // "..."
			words = append(words, strings.TrimSpace(m[4]))
		case strings.TrimSpace(m[5]) != "": // bareword
			containsWords = append(containsWords, strings.TrimSpace(m[5]))
		}
	}

	n.template.Nuke().Start(punishment, duration, containsWords, words, re, func(ctx context.Context) {
		checkCtx := func() bool {
			select {
			case <-ctx.Done():
				n.log.Warn("Mass action canceled", slog.String("reason", ctx.Err().Error()))
				return false
			default:
				return true
			}
		}

		executeAction := func(username string, messageID string, msg storage.Message) {
			err := n.api.Pool().Submit(func() {
				if !checkCtx() {
					return
				}

				switch punishment.Action {
				case checker.Ban:
					n.log.Warn("Ban user", slog.String("username", username), slog.String("text", msg.Data.Message.Text.Text()))
					n.api.BanUser(n.stream.ChannelName(), n.stream.ChannelID(), msg.Data.Chatter.UserID, "массбан")
				case checker.Timeout:
					n.log.Warn("Timeout user", slog.String("username", username),
						slog.String("text", msg.Data.Message.Text.Text()),
						slog.Int("duration", int((time.Duration(punishment.Duration)*time.Second).Seconds())),
					)
					n.api.TimeoutUser(n.stream.ChannelName(), n.stream.ChannelID(), msg.Data.Chatter.UserID, punishment.Duration, "массбан")
				case checker.Delete:
					n.log.Warn("Delete message", slog.String("username", username), slog.String("text", msg.Data.Message.Text.Text()))
					if err := n.api.DeleteChatMessage(n.stream.ChannelName(), n.stream.ChannelID(), messageID); err != nil {
						n.log.Error("Failed to delete message on chat", err)
					}
				}
			})
			if err != nil {
				n.log.Error("Failed to submit request", err)
			}
		}

		now := time.Now()
		for username, msgs := range n.messages.GetAllData() {
			if !checkCtx() {
				return
			}

			for messageID, msg := range msgs {
				if !checkCtx() {
					return
				}

				if now.Sub(msg.Time) >= scrollback {
					continue
				}

				n.messages.Update(username, messageID, func(cur storage.Message, exists bool) storage.Message {
					if !exists {
						return cur
					}

					cur.IgnoreNuke = true
					return cur
				})

				action := n.template.Nuke().Check(&msg.Data.Message.Text, msg.IgnoreNuke)
				if action == nil || msg.Data.Chatter.IsBroadcaster || msg.Data.Chatter.IsMod {
					continue
				}

				executeAction(username, messageID, msg)
				if punishment.Action != "delete" {
					break
				}
			}
		}
	})

	if len(globalErrs) != 0 {
		return &ports.AnswerType{
			Text:    []string{strings.Join(globalErrs, " • ")},
			IsReply: true,
		}
	}
	return success
}

type NukeStop struct {
	template ports.TemplatePort
}

func (n *NukeStop) Execute(_ *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return n.handleNukeStop()
}

func (n *NukeStop) handleNukeStop() *ports.AnswerType {
	// !am nuke stop
	n.template.Nuke().Cancel()

	return success
}

type ReNuke struct {
	template ports.TemplatePort
}

func (n *ReNuke) Execute(_ *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return n.handleReNuke()
}

func (n *ReNuke) handleReNuke() *ports.AnswerType {
	// !am nuke re
	if err := n.template.Nuke().Restart(); err != nil {
		return &ports.AnswerType{
			Text:    []string{"повтор предыдущего массбана не возможен"},
			IsReply: true,
		}
	}

	return success
}
