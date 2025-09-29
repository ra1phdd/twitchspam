package admin

import (
	"fmt"
	"regexp"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

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

type AddCommand struct {
	re *regexp.Regexp
}

func (c *AddCommand) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandAdd(cfg, text)
}

func (c *AddCommand) handleCommandAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Original) // !am cmd add <команда> <текст>
	if len(matches) != 3 {
		return NonParametr
	}

	cmd := strings.TrimSpace(matches[1])
	if !strings.HasPrefix(cmd, "!") {
		cmd = "!" + cmd
	}

	cfg.Commands[cmd] = &config.Commands{
		Text: strings.TrimSpace(matches[2]),
	}

	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
}

type DelCommand struct {
	re *regexp.Regexp
}

func (c *DelCommand) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandDel(cfg, text)
}

func (c *DelCommand) handleCommandDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Original) // !am cmd del <команды через запятую>
	if len(matches) != 2 {
		return NonParametr
	}

	var removed, notFound []string
	for _, cmd := range strings.Split(strings.TrimSpace(matches[1]), ",") {
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

	return buildResponse("команды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type AliasesCommand struct {
	re *regexp.Regexp
}

func (c *AliasesCommand) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandAliases(cfg, text)
}

func (c *AliasesCommand) handleCommandAliases(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Original) // !am cmd aliases <команда>
	if len(matches) != 2 {
		return NonParametr
	}

	cmd := strings.TrimSpace(matches[1])
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
