package admin

import (
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
	// !am nuke <*наказание> <*scrollback> <слова/фразы через запятую или regex>
	matches := n.re.FindStringSubmatch(text.Text())
	if len(matches) != 4 {
		return nonParametr
	}

	var globalErrs []string
	punishment := config.Punishment{
		Action:   "timeout",
		Duration: 60,
	}
	scrollback := n.messages.GetTTL()

	if strings.TrimSpace(matches[1]) != "" {
		p, err := n.template.Punishment().Parse(strings.TrimSpace(matches[1]), false)
		if err != nil {
			globalErrs = append(globalErrs, "не удалось распарсить наказание, применено дефолтное (60))")
		} else {
			punishment = p
		}
	}

	if strings.TrimSpace(matches[2]) != "" {
		if val, ok := n.template.Parser().ParseIntArg(strings.TrimSpace(matches[2]), 1, 180); ok {
			scrollback = time.Duration(val) * time.Second
		}
	}

	if strings.TrimSpace(matches[3]) == "" {
		return &ports.AnswerType{
			Text:    []string{"не указаны слова для массбана!"},
			IsReply: true,
		}
	}
	wordsMatches := n.reWords.FindAllStringSubmatch(strings.TrimSpace(matches[3]), -1)

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

	dur := time.Duration(punishment.Duration) * time.Second
	apply := func(msgID, userID, username, text string) {
		switch punishment.Action {
		case checker.Ban:
			n.log.Warn("Ban user", slog.String("username", username), slog.String("text", text))
			n.api.BanUser(n.stream.ChannelID(), userID, "массбан")
		case checker.Timeout:
			n.log.Warn("Timeout user", slog.String("username", username), slog.String("text", text), slog.Int("duration", int(dur.Seconds())))
			n.api.TimeoutUser(n.stream.ChannelID(), userID, int(dur.Seconds()), "массбан")
		case checker.Delete:
			n.log.Warn("Delete message", slog.String("username", username), slog.String("text", text))
			if err := n.api.DeleteChatMessage(n.stream.ChannelID(), msgID); err != nil {
				n.log.Error("Failed to delete message on chat", err)
			}
		}
	}

	now := time.Now()
	n.template.Nuke().Start(punishment, containsWords, words, re)
	for username, msgs := range n.messages.GetAllData() {
		for messageID, msg := range msgs {
			if now.Sub(msg.Time) >= scrollback {
				continue
			}

			action := n.template.Nuke().Check(&msg.Data.Message.Text)
			if action != nil && !msg.Data.Chatter.IsBroadcaster && !msg.Data.Chatter.IsMod {
				apply(messageID, msg.Data.Chatter.UserID, username, msg.Data.Message.Text.Text())
				if punishment.Action != "delete" {
					break
				}
			}
		}
	}

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
