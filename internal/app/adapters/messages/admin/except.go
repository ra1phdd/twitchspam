package admin

import (
	"fmt"
	"strconv"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleEx(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	mwgCmd, mwgArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) *ports.AnswerType{
		"list": a.handleExList,
		"add":  a.handleExAdd,
		"set":  a.handleExSet,
		"del":  a.handleExDel,
	}

	if handler, ok := handlers[mwgCmd]; ok {
		return handler(cfg, mwgCmd, mwgArgs)
	}
	return NotFoundCmd
}

func (a *Admin) handleExList(cfg *config.Config, _ string, _ []string) *ports.AnswerType {
	if len(cfg.Spam.Exceptions) == 0 {
		return &ports.AnswerType{
			Text:    []string{"исключений не найдено!"},
			IsReply: true,
		}
	}

	var parts []string
	for word, ex := range cfg.Spam.Exceptions {
		parts = append(parts, fmt.Sprintf("- %s (message_limit: %d, punishments: %s)", word, ex.MessageLimit, formatPunishments(ex.Punishments)))
	}
	msg := "исключения: \n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}

	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}

func (a *Admin) handleExAdd(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 3 {
		return NonParametr
	}

	messageLimit, err := strconv.Atoi(args[0])
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{"не указан лимит сообщений!"},
			IsReply: true,
		}
	}

	var punishments []config.Punishment
	punishmentsArgs := strings.Split(args[1], ",")
	for _, pa := range punishmentsArgs {
		p, err := parsePunishment(pa, true)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
				IsReply: true,
			}
		}

		if p.Action == "inherit" {
			punishments = cfg.Spam.SettingsDefault.Punishments
			break
		}

		punishments = append(punishments, p)
	}

	words := a.regexp.SplitWords(strings.Join(args[2:], " "))
	for _, w := range words {
		trimmed := strings.TrimSpace(w)
		if trimmed == "" {
			continue
		}

		re, err := a.regexp.Parse(trimmed)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		cfg.Spam.Exceptions[w] = config.SpamExceptionsSettings{
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Regexp:       re,
		}
	}

	return nil
}

func (a *Admin) handleExSet(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 3 {
		return NonParametr
	}
	field := args[0]

	var updated, notFound []string
	words := a.regexp.SplitWords(strings.Join(args[2:], " "))

	for _, w := range words {
		if exWord, ok := cfg.Spam.Exceptions[w]; ok {
			switch field {
			case "ml":
				value, err := strconv.Atoi(args[1])
				if err != nil {
					return NonParametr
				}
				exWord.MessageLimit = value
			case "p":
				var punishments []config.Punishment
				punishmentsArgs := strings.Split(args[1], ",")
				for _, pa := range punishmentsArgs {
					p, err := parsePunishment(pa, true)
					if err != nil {
						return &ports.AnswerType{
							Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
							IsReply: true,
						}
					}

					if p.Action == "inherit" {
						punishments = cfg.Spam.SettingsDefault.Punishments
						break
					}

					punishments = append(punishments, p)
				}
				exWord.Punishments = punishments
			default:
				return NonParametr
			}
			updated = append(updated, w)
		} else {
			notFound = append(notFound, w)
		}
	}

	var msgParts []string
	if len(updated) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("изменены: %s", strings.Join(updated, ", ")))
	}
	if len(notFound) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("не найдены: %s", strings.Join(notFound, ", ")))
	}

	if len(msgParts) == 0 {
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleExDel(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}

	var removed, notFound []string
	wordsToRemove := a.regexp.SplitWords(strings.Join(args, " "))

	for _, w := range wordsToRemove {
		if _, ok := cfg.Spam.Exceptions[w]; ok {
			delete(cfg.Spam.Exceptions, w)
			removed = append(removed, w)
		} else {
			notFound = append(notFound, w)
		}
	}

	var msgParts []string
	if len(removed) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("удалены: %s", strings.Join(removed, ", ")))
	}
	if len(notFound) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("не найдены: %s", strings.Join(notFound, ", ")))
	}

	if len(msgParts) == 0 {
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}
