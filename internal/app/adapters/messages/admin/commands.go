package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleCommand(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 3 { // !am cmd add/del/list
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"add":  a.handleCommandAdd,
		"del":  a.handleCommandDel,
		"list": a.handleCommandList,
	}

	linkCmd := text.Words()[2]
	if handler, ok := handlers[linkCmd]; ok {
		return handler(cfg, text)
	}
	return NotFoundCmd
}

func (a *Admin) handleCommandAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 5 { // !am cmd add <команда> <текст>
		return NonParametr
	}

	cfg.Links[text.Words()[3]] = &config.Links{
		Text: text.Tail(4),
	}

	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
}

func (a *Admin) handleCommandDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) != 4 { // !am cmd del <команды через запятую>
		return NonParametr
	}

	var removed, notFound []string
	for _, key := range strings.Split(text.Tail(3), ",") {
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
		return &ports.AnswerType{
			Text:    []string{"пользователи не указаны!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleCommandList(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
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
