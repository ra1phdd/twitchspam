package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleCommand(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am cmd add/del/list
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"add":   a.handleCommandAdd,
		"del":   a.handleCommandDel,
		"list":  a.handleCommandList,
		"timer": a.handleCommandTimers,
	}

	linkCmd := words[2]
	if handler, ok := handlers[linkCmd]; ok {
		return handler(cfg, text)
	}
	return NotFoundCmd
}

func (a *Admin) handleCommandAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 5 { // !am cmd add <команда> <текст>
		return NonParametr
	}

	cmd := words[3]
	if !strings.HasPrefix(cmd, "!") {
		cmd = "!" + cmd
	}

	cfg.Commands[cmd] = &config.Commands{
		Text: text.Tail(4),
	}

	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
}

func (a *Admin) handleCommandDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am cmd del <команды через запятую>
		return NonParametr
	}

	var removed, notFound []string
	for _, cmd := range strings.Split(words[3], ",") {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		if !strings.HasPrefix(cmd, "!") {
			cmd = "!" + cmd
		}

		if _, ok := cfg.Commands[cmd]; !ok {
			notFound = append(notFound, cmd)
			continue
		}

		delete(cfg.Commands, cmd)
		for alias, original := range cfg.Aliases {
			if original == cmd {
				delete(cfg.Aliases, alias)
			}
		}
		removed = append(removed, cmd)
	}

	return a.buildResponse(removed, "удалены", notFound, "не найдены", "команды не указаны")
}

func (a *Admin) handleCommandList(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	if len(cfg.Commands) == 0 {
		return &ports.AnswerType{
			Text:    []string{"команды не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for key, link := range cfg.Commands {
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
