package admin

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"strconv"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleExcept(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 3 { // !am ex list/add/set/del
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"list": a.handleExceptList,
		"set":  a.handleExSet,
		"del":  a.handleExDel,
	}

	exceptCmd := text.Words()[2]
	if handler, ok := handlers[exceptCmd]; ok {
		return handler(cfg, text)
	}
	return a.handleExceptAdd(cfg, text)
}

func (a *Admin) handleExceptList(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	if len(cfg.Spam.Exceptions) == 0 {
		return &ports.AnswerType{
			Text:    []string{"исключений не найдено!"},
			IsReply: true,
		}
	}

	var parts []string
	for word, ex := range cfg.Spam.Exceptions {
		parts = append(parts, fmt.Sprintf("- %s (лимит сообщений: %d, наказания: %s)", word, ex.MessageLimit, strings.Join(formatPunishments(ex.Punishments), ", ")))
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

func (a *Admin) handleExceptAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	isRegex, opts := a.template.ParseOptions(&words) // ParseOptions удаляет опции из слайса words

	idx := 2 // id параметра, с которого начинаются аргументы команды
	if words[2] == "add" {
		idx = 3
	}

	// !am ex <кол-во сообщений> <наказания через запятую> <слова/фразы через запятую или regex>
	// или !am ex add <кол-во сообщений> <наказания через запятую> <слова/фразы через запятую или regex>
	if len(words) <= idx+2 {
		return NonParametr
	}

	messageLimit, err := strconv.Atoi(words[idx])
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{"не указан лимит сообщений!"},
			IsReply: true,
		}
	}

	var punishments []config.Punishment
	for _, pa := range strings.Split(words[idx+1], ",") {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

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

	if isRegex {
		re, err := regexp2.Compile(strings.Join(words[idx+2:], " "), regexp2.None)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		cfg.Spam.Exceptions[strings.Join(words[idx+2:], " ")] = &config.SpamExceptionsSettings{
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Regexp:       re,
			Options:      opts,
		}
		return nil
	}

	for _, word := range strings.Split(strings.Join(words[idx+2:], " "), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		cfg.Spam.Exceptions[word] = &config.SpamExceptionsSettings{
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Options:      opts,
		}
	}

	return nil
}

func (a *Admin) handleExSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 6 { // !am ex set ml/p <параметр 1> <слова/фразы через запятую или regex>
		return NonParametr
	}

	cmds := map[string]func(exWord *config.SpamExceptionsSettings, param string) *ports.AnswerType{
		"ml": func(exWord *config.SpamExceptionsSettings, param string) *ports.AnswerType {
			value, err := strconv.Atoi(param)
			if err != nil {
				return NonParametr
			}

			exWord.MessageLimit = value
			return nil
		},
		"p": func(exWord *config.SpamExceptionsSettings, param string) *ports.AnswerType {
			var punishments []config.Punishment
			punishmentsArgs := strings.Split(param, ",")
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
			return nil
		},
	}

	var updated, notFound []string
	if regex, ok := cfg.Spam.Exceptions[text.Tail(5)]; ok {
		if cmd, cmdOk := cmds[text.Words()[3]]; cmdOk {
			if out := cmd(regex, text.Words()[4]); out != nil {
				return out
			}
			updated = append(updated, text.Tail(5))
		}
	} else {
		for _, word := range strings.Split(text.Tail(5), ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			exWord, exOk := cfg.Spam.Exceptions[word]
			if !exOk {
				notFound = append(notFound, word)
			}

			if cmd, cmdOk := cmds[text.Words()[3]]; cmdOk {
				if out := cmd(exWord, text.Words()[4]); out != nil {
					return out
				}
				updated = append(updated, word)
			}

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
		return &ports.AnswerType{
			Text:    []string{"исключения не найдены!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleExDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 4 { // !am ex del <слова/фразы через запятую или regex>
		return NonParametr
	}

	var removed, notFound []string
	if _, ok := cfg.Spam.Exceptions[text.Tail(3)]; ok {
		delete(cfg.Spam.Exceptions, text.Tail(3))
		removed = append(removed, text.Tail(3))
	} else {
		for _, word := range strings.Split(text.Tail(3), ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			if _, ok := cfg.Spam.Exceptions[word]; ok {
				delete(cfg.Spam.Exceptions, word)
				removed = append(removed, word)
			} else {
				notFound = append(notFound, word)
			}
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
		return &ports.AnswerType{
			Text:    []string{"исключения не найдены!"},
			IsReply: true,
		}
	}

	if len(removed) > 0 && len(notFound) == 0 {
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}
