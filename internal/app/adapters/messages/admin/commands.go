package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddCommand struct{}

func (c *AddCommand) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandAdd(cfg, text)
}

func (c *AddCommand) handleCommandAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
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

type DelCommand struct{}

func (c *DelCommand) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandDel(cfg, text)
}

func (c *DelCommand) handleCommandDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
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

	return buildResponse(removed, "удалены", notFound, "не найдены", "команды не указаны")
}

type ListCommand struct {
	fs ports.FileServerPort
}

func (c *ListCommand) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return c.handleCommandList(cfg)
}

func (c *ListCommand) handleCommandList(cfg *config.Config) *ports.AnswerType {
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

	key, err := c.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{c.fs.GetURL(key)},
		IsReply: true,
	}
}
