package admin

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"twitchspam/internal/app/domain/message"
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

func (n *Nuke) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	// !am nuke <*наказание> <*длительность> <*scrollback> <слова/фразы через запятую или regex>
	matches := n.re.FindStringSubmatch(msg.Message.Text.Text(message.NormalizeCommaSpacesOption))
	if len(matches) != 5 {
		return nonParametr
	}

	var globalErrs []string
	punishment := config.Punishment{
		Action:   "timeout",
		Duration: 60,
	}
	duration := 5 * time.Minute
	scrollback := 1 * time.Minute

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

			if scrollback > 180*time.Second {
				scrollback = 180 * time.Second
			}
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

	n.template.Nuke().Start(punishment, duration, containsWords, words, re, msg.Chatter.Username, func(ctx context.Context) {
		checkCtx := func() bool {
			select {
			case <-ctx.Done():
				n.log.Warn("Mass action canceled", slog.String("reason", ctx.Err().Error()))
				return false
			default:
				return true
			}
		}

		now := time.Now()
		for username, messages := range n.messages.GetAllData() {
			if !checkCtx() {
				return
			}

			for messageID, item := range messages {
				if !checkCtx() {
					return
				}

				if now.Sub(item.Time) >= scrollback {
					continue
				}

				n.messages.Update(username, messageID, func(cur *storage.Message, exists bool) *storage.Message {
					if !exists {
						return cur
					}

					cur.IgnoreNuke = true
					return cur
				})

				action := n.template.Nuke().Check(&item.Data.Message.Text, item.IgnoreNuke)
				if action == nil || item.Data.Chatter.IsBroadcaster || item.Data.Chatter.IsMod {
					continue
				}

				n.api.Pool().Submit(func() {
					if !checkCtx() {
						return
					}

					ExecuteModAction(n.log, n.api, n.stream, action, item.Data.Chatter.UserID, item.Data.Chatter.Username, item.Data.Message.ID, item.Data.Message.Text.Text())
				})

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

func (n *NukeStop) Execute(_ *config.Config, _ string, _ *message.ChatMessage) *ports.AnswerType {
	// !am nuke stop
	n.template.Nuke().Cancel()

	return success
}

type ReNuke struct {
	template ports.TemplatePort
}

func (n *ReNuke) Execute(_ *config.Config, _ string, _ *message.ChatMessage) *ports.AnswerType {
	// !am nuke re
	if err := n.template.Nuke().Restart(); err != nil {
		return &ports.AnswerType{
			Text:    []string{"повтор предыдущего массбана не возможен!"},
			IsReply: true,
		}
	}

	return success
}
