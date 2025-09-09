package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleLink(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	linkCmd, linkArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, args []string) *ports.AnswerType{
		"add":  a.handleLinkAdd,
		"del":  a.handleLinkDel,
		"list": a.handleLinkList,
	}

	if handler, ok := handlers[linkCmd]; ok {
		return handler(cfg, linkArgs)
	}
	return NotFoundCmd
}

func (a *Admin) handleLinkAdd(cfg *config.Config, args []string) *ports.AnswerType {
	if len(args) < 2 {
		return NonParametr
	}

	key := args[0]
	text := strings.Join(args[1:], " ")

	cfg.Links[key] = &config.Links{
		Text: text,
	}

	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
}

func (a *Admin) handleLinkDel(cfg *config.Config, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}

	key := args[0]
	if _, ok := cfg.Links[key]; !ok {
		return &ports.AnswerType{
			Text:    []string{"ссылка не найдена!"},
			IsReply: true,
		}
	}

	delete(cfg.Links, key)
	for alias, original := range cfg.Aliases {
		if original == key {
			delete(cfg.Aliases, alias)
		}
	}

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("ссылка %s удалена!", key)},
		IsReply: true,
	}
}

func (a *Admin) handleLinkList(cfg *config.Config, _ []string) *ports.AnswerType {
	if len(cfg.Links) == 0 {
		return &ports.AnswerType{
			Text:    []string{"ссылки не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for key, link := range cfg.Links {
		parts = append(parts, fmt.Sprintf("- %s -> %s", key, link.Text))
	}
	msg := "ссылки: \n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}
