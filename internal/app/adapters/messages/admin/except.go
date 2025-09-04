package admin

import (
	"fmt"
	"strconv"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleEx(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	mwgCmd, mwgArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) ports.ActionType{
		"list": a.handleExList,
		"add":  a.handleExAdd,
		"set":  a.handleExSet,
		"del":  a.handleExDel,
	}

	if handler, ok := handlers[mwgCmd]; ok {
		return handler(cfg, mwgCmd, mwgArgs)
	}
	return NotFound
}

func (a *Admin) handleExList(cfg *config.Config, _ string, _ []string) ports.ActionType {
	if len(cfg.Spam.Exceptions) == 0 {
		return "исключения отсутствуют"
	}

	var parts []string
	for word, ex := range cfg.Spam.Exceptions {
		parts = append(parts, fmt.Sprintf("%s(ML: %d, TO: %d)", word, ex.MessageLimit, ex.Timeout))
	}

	msg := "исключения: " + strings.Join(parts, ", ")
	return ports.ActionType(msg)
}

func (a *Admin) handleExAdd(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 3 {
		return NonParametr
	}

	messageLimit, err := strconv.Atoi(args[0])
	if err != nil {
		return ErrFound
	}

	timeout, err := strconv.Atoi(args[1])
	if err != nil {
		return ErrFound
	}

	words := a.regexp.SplitWords(strings.Join(args[2:], " "))
	for _, w := range words {
		trimmed := strings.TrimSpace(w)
		if trimmed == "" {
			continue
		}

		re, err := a.regexp.Parse(trimmed)
		if err != nil {
			return ports.ActionType(err.Error())
		}

		cfg.Spam.Exceptions[w] = &config.SpamExceptionsSettings{
			MessageLimit: messageLimit,
			Timeout:      timeout,
			Regexp:       re,
		}
	}

	return Success
}

func (a *Admin) handleExSet(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 3 {
		return NonParametr
	}

	field := args[0]
	value, err := strconv.Atoi(args[1])
	if err != nil {
		return ErrFound
	}

	var updated, notFound []string
	words := a.regexp.SplitWords(strings.Join(args[2:], " "))

	for _, w := range words {
		if exWord, ok := cfg.Spam.Exceptions[w]; ok {
			switch field {
			case "ml":
				exWord.MessageLimit = value
			case "to":
				exWord.Timeout = value
			default:
				return NotFound
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
		return None
	}
	return ports.ActionType(strings.Join(msgParts, " • "))
}

func (a *Admin) handleExDel(cfg *config.Config, _ string, args []string) ports.ActionType {
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
		return None
	}
	return ports.ActionType(strings.Join(msgParts, " • "))
}
