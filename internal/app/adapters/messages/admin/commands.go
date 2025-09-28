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
	return buildList(cfg.Commands, "команды", "команды не найдены!",
		func(key string, cmd *config.Commands) string {
			return fmt.Sprintf("- %s -> %s", key, cmd.Text)
		}, c.fs)
}

type AliasesCommand struct{}

func (c *AliasesCommand) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandAliases(cfg, text)
}

func (c *AliasesCommand) handleCommandAliases(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 5 { // !am cmd aliases <команда>
		return NonParametr
	}

	cmd := words[3]
	if !strings.HasPrefix(cmd, "!") {
		cmd = "!" + cmd
	}

	if orig, ok := cfg.Aliases[cmd]; ok {
		cmd = orig
	}

	var aliases []string
	for alias, orig := range cfg.Aliases {
		if strings.Contains(cmd, orig) {
			aliases = append(aliases, alias)
		}
	}

	if len(aliases) == 0 {
		return &ports.AnswerType{
			Text:    []string{fmt.Sprintf("команда: %s, алиасы: не найдены", cmd)},
			IsReply: true,
		}
	}

	msg := fmt.Sprintf("команда: %s, алиасы: %s", cmd, strings.Join(aliases, ","))
	return &ports.AnswerType{
		Text:    []string{msg},
		IsReply: true,
	}
}
