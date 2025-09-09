package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleCommand(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	linkCmd, linkArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, args []string) *ports.AnswerType{
		"add":  a.handleCommandAdd,
		"del":  a.handleCommandDel,
		"list": a.handleCommandList,
	}

	if handler, ok := handlers[linkCmd]; ok {
		return handler(cfg, linkArgs)
	}
	return NotFoundCmd
}

func (a *Admin) handleCommandAdd(cfg *config.Config, args []string) *ports.AnswerType {
	if len(args) < 2 {
		return NonParametr
	}

	key := args[0]
	text := strings.Join(args[1:], " ")

	cfg.Links[key] = config.Links{
		Text: text,
	}

	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
}

func (a *Admin) handleCommandDel(cfg *config.Config, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}

	var removed, notFound []string
	keys := strings.Split(args[0], ",")
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if _, ok := cfg.Links[key]; !ok {
			notFound = append(notFound, key)
			continue
		}

		delete(cfg.Links, key)
		for alias, original := range cfg.Aliases {
			if original == key {
				delete(cfg.Aliases, alias)
			}
		}
		removed = append(removed, key)
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

func (a *Admin) handleCommandList(cfg *config.Config, _ []string) *ports.AnswerType {
	if len(cfg.Links) == 0 {
		return &ports.AnswerType{
			Text:    []string{"команды не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for key, link := range cfg.Links {
		parts = append(parts, fmt.Sprintf("- %s -> %s", key, link.Text))
	}
	msg := "команды: \n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}
