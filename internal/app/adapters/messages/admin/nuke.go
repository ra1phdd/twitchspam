package admin

import (
	"regexp"
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Nuke struct {
	re       *regexp.Regexp
	reWords  *regexp.Regexp
	log      logger.Logger
	manager  *config.Manager
	api      ports.APIPort
	template ports.TemplatePort
}

func (n *Nuke) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return n.handleNuke(cfg, text)
}

func (n *Nuke) handleNuke(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	// !am nuke <*наказание> <*scrollback> <слова/фразы через запятую или regex>
	matches := n.re.FindStringSubmatch(text.Text())
	if len(matches) != 4 {
		return nonParametr
	}

	var globalErrs []string
	punishment := config.Punishment{
		Action:   "timeout",
		Duration: 600,
	}
	scrollback := n.template.Store().Messages().GetTTL()

	if strings.TrimSpace(matches[1]) != "" {
		p, err := n.template.Punishment().Parse(strings.TrimSpace(matches[1]), false)
		if err != nil {
			globalErrs = append(globalErrs, "не удалось распарсить наказание, применено дефолтное (600))")
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

	var words map[bool]string // key - nocontains
	var regexps []*regexp.Regexp
	for _, m := range wordsMatches {
		switch {
		case strings.TrimSpace(m[1]) != "": // r'...'
			re, err := regexp.Compile(strings.TrimSpace(m[1]))
			if err != nil {
				return invalidRegex
			}

			regexps = append(regexps, re)
		case strings.TrimSpace(m[2]) != "": // r"..."
			re, err := regexp.Compile(strings.TrimSpace(m[2]))
			if err != nil {
				return invalidRegex
			}

			regexps = append(regexps, re)
		case strings.TrimSpace(m[3]) != "": // '...'
			words[true] = strings.TrimSpace(m[3])
		case strings.TrimSpace(m[4]) != "": // "..."
			words[true] = strings.TrimSpace(m[4])
		case strings.TrimSpace(m[5]) != "": // bareword
			words[false] = strings.TrimSpace(m[5])
		}
	}

	cfg.Nuke = config.Nuke{
		Enabled:    true,
		ExpiresAt:  time.Now().Add(10 * time.Minute),
		Punishment: punishment,
		Scrollback: scrollback,
		Words:      words,
		Regexp:     regexps,
	}

	time.AfterFunc(10*time.Minute, func() {
		if err := n.manager.Update(func(cfg *config.Config) {
			cfg.Nuke.Enabled = false
		}); err != nil {
			n.log.Error("Failed update nuke in config", err)
		}
	})

	//dur := time.Duration(punishment.Duration) * time.Second
	//apply := func(msgID, userID, username, text string) {
	//	switch punishment.Action {
	//	case checker.Ban:
	//		n.log.Warn("Ban user", slog.String("username", username), slog.String("text", text))
	//		n.api.BanUser(userID, "массбан")
	//	case checker.Timeout:
	//		n.log.Warn("Timeout user", slog.String("username", username), slog.String("text", text), slog.Int("duration", int(dur.Seconds())))
	//		n.api.TimeoutUser(userID, int(dur.Seconds()), "массбан")
	//	case checker.Delete:
	//		n.log.Warn("Delete message", slog.String("username", username), slog.String("text", text))
	//		if err := n.api.DeleteChatMessage(msgID); err != nil {
	//			n.log.Error("Failed to delete message on chat", err)
	//		}
	//	}
	//}

	//for username, msgs := range n.template.Store().Messages().GetAll() {
	//	skipUser := false
	//	for _, msg := range msgs {
	//		for noContains, word := range words {
	//			if noContains && !slices.Contains(msg.Text.Words(), word) {
	//				continue
	//			}
	//
	//			if !noContains && !strings.Contains(msg.Text.Text(domain.RemoveDuplicateLetters), word) {
	//				continue
	//			}
	//
	//			apply(msg.MessageID, msg.UserID, username, msg.Text.Text())
	//			if punishment.Action != "delete" {
	//				skipUser = true
	//			}
	//			break
	//		}
	//
	//		if skipUser {
	//			break
	//		}
	//
	//		for _, re := range regexps {
	//			if re.MatchString(msg.Text.Text(domain.RemoveDuplicateLetters)) {
	//				apply(msg.MessageID, msg.UserID, username, msg.Text.Text())
	//				break
	//			}
	//		}
	//	}
	//}

	if len(globalErrs) != 0 {
		return &ports.AnswerType{
			Text:    []string{strings.Join(globalErrs, " • ")},
			IsReply: true,
		}
	}
	return success
}
