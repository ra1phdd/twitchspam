package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleMw(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	mwCmd, mwArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) *ports.AnswerType{
		"del":  a.handleMwDel,
		"list": a.handleMwList,
	}

	if handler, ok := handlers[mwCmd]; ok {
		return handler(cfg, mwCmd, mwArgs)
	}

	return a.handleMwAdd(cfg, mwCmd, mwArgs)
}

func (a *Admin) handleMwAdd(cfg *config.Config, mwCmd string, args []string) *ports.AnswerType {
	if mwCmd == "add" && len(args) < 2 {
		return NonParametr
	} else if len(args) < 1 {
		return NonParametr
	}

	wordsArgs := args
	punishArg := mwCmd
	if mwCmd == "add" {
		punishArg = args[0]
		wordsArgs = args[1:]
	}

	words := a.regexp.SplitWords(strings.Join(wordsArgs, " "))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		re, err := a.regexp.Parse(word)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		var punishments []config.Punishment
		punishmentsArgs := strings.Split(punishArg, ",")
		for _, pa := range punishmentsArgs {
			p, err := parsePunishment(pa, false)
			if err != nil {
				return &ports.AnswerType{
					Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
					IsReply: true,
				}
			}
			punishments = append(punishments, p)
		}

		cfg.Mword[word] = config.Mword{
			Punishments: punishments,
			Regexp:      re,
		}
	}

	return nil
}

func (a *Admin) handleMwDel(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}

	wordsToDelete := a.regexp.SplitWords(strings.Join(args, " "))
	for _, w := range wordsToDelete {
		if _, ok := cfg.Mword[w]; ok {
			delete(cfg.Mword, w)
		}
	}

	return nil
}

func (a *Admin) handleMwList(cfg *config.Config, _ string, _ []string) *ports.AnswerType {
	if len(cfg.Mword) == 0 {
		return &ports.AnswerType{
			Text:    []string{"мворды не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for word, mw := range cfg.Mword {
		parts = append(parts, fmt.Sprintf("- %s (punishments: %s)", word, formatPunishments(mw.Punishments)))
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
