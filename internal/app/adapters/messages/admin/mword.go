package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleMw(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	mwCmd, mwArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) ports.ActionType{
		"del":  a.handleMwDel,
		"list": a.handleMwList,
	}

	if handler, ok := handlers[mwCmd]; ok {
		return handler(cfg, mwCmd, mwArgs)
	}

	return a.handleMwAdd(cfg, mwCmd, mwArgs)
}

func (a *Admin) handleMwAdd(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	words := a.regexp.SplitWords(strings.Join(args[1:], " "))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		re, err := a.regexp.Parse(word)
		if err != nil {
			return ports.ActionType(err.Error())
		}

		action, duration, err := parsePunishment(args[0])
		if err != nil {
			return ErrFound
		}

		cfg.Mword[word] = &config.Mword{
			Action:   action,
			Duration: duration,
			Regexp:   re,
		}
	}

	return Success
}

func (a *Admin) handleMwDel(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}

	wordsToDelete := a.regexp.SplitWords(strings.Join(args, " "))
	for _, w := range wordsToDelete {
		if _, ok := cfg.Mword[w]; ok {
			delete(cfg.Mword, w)
		}
	}

	return Success
}

func (a *Admin) handleMwList(cfg *config.Config, _ string, _ []string) ports.ActionType {
	if len(cfg.Mword) == 0 {
		return "мворды отсутствуют"
	}

	var parts []string
	for word, mw := range cfg.Mword {
		parts = append(parts, fmt.Sprintf("%s(%d)", word, mw.Duration))
	}

	msg := "мворды: " + strings.Join(parts, ", ")
	return ports.ActionType(msg)
}
