package admin

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleMw(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 3 { // !am mw add/del/list
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"del":  a.handleMwDel,
		"list": a.handleMwList,
	}

	mwCmd := text.Words()[2]
	if handler, ok := handlers[mwCmd]; ok {
		return handler(cfg, text)
	}

	return a.handleMwAdd(cfg, text)
}

func (a *Admin) handleMwAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	isRegex, opts := a.template.ParseOptions(&words) // ParseOptions удаляет опции из слайса words

	idx := 2 // id параметра, с которого начинаются аргументы команды
	if words[2] == "add" {
		idx = 3
	}

	// !am mw <наказания через запятую> <слова/фразы через запятую>
	// или !am mw add <наказания через запятую> <слова/фразы через запятую>
	if len(words) <= idx+2 {
		return NonParametr
	}

	var punishments []config.Punishment
	for _, pa := range strings.Split(words[idx], ",") {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

		p, err := parsePunishment(pa, false)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
				IsReply: true,
			}
		}
		punishments = append(punishments, p)
	}

	if isRegex {
		re, err := regexp2.Compile(strings.Join(words[idx+1:], " "), regexp2.None)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		cfg.Mword[strings.Join(words[idx+1:], " ")] = &config.Mword{
			Punishments: punishments,
			Regexp:      re,
			Options:     opts,
		}
		return nil
	}

	for _, word := range strings.Split(strings.Join(words[idx+1:], " "), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		cfg.Mword[word] = &config.Mword{
			Punishments: punishments,
			Options:     opts,
		}
	}

	return nil
}

func (a *Admin) handleMwDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 4 { // !am mw del <слова/фразы через запятую или regex>
		return NonParametr
	}

	var removed, notFound []string
	if _, ok := cfg.Mword[text.Tail(3)]; ok {
		delete(cfg.Mword, text.Tail(3))
		removed = append(removed, text.Tail(3))
	} else {
		for _, word := range strings.Split(text.Tail(3), ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			if _, ok := cfg.Mword[word]; ok {
				delete(cfg.Mword, word)
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
			Text:    []string{"мворды не найдены!"},
			IsReply: true,
		}
	}

	if len(removed) > 0 && len(notFound) == 0 {
		return &ports.AnswerType{
			Text:    []string{"успешно!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleMwList(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	if len(cfg.Mword) == 0 {
		return &ports.AnswerType{
			Text:    []string{"мворды не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for word, mw := range cfg.Mword {
		parts = append(parts, fmt.Sprintf("- %s (наказания: %s)", word, formatPunishments(mw.Punishments)))
	}
	msg := "мворды: \n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}
