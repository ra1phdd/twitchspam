package admin

import (
	"fmt"
	"regexp"
	"strings"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type ListCommand struct {
	fs ports.FileServerPort
}

func (c *ListCommand) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return c.handleCommandList(cfg)
}

func (c *ListCommand) handleCommandList(cfg *config.Config) *ports.AnswerType {
	return buildList(cfg.Commands, "команды", "команды не найдены!",
		func(key string, cmd *config.Commands) string {
			return fmt.Sprintf("- %s -> %s", key, cmd.Text)
		}, c.fs)
}

type AddCommand struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (c *AddCommand) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleCommandAdd(cfg, text)
}

func (c *AddCommand) handleCommandAdd(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := c.template.Options().ParseAll(text.Text(), template.CommandOptions)

	matches := c.re.FindStringSubmatch(textWithoutOpts) // !am cmd add <команда> <текст>
	if len(matches) != 3 {
		return nonParametr
	}

	cmd := strings.ToLower(strings.TrimSpace(matches[1]))
	if !strings.HasPrefix(cmd, "!") {
		cmd = "!" + cmd
	}

	cfg.Commands[cmd] = &config.Commands{
		Text:    strings.TrimSpace(matches[2]),
		Options: c.template.Options().MergeCommand(config.CommandOptions{}, opts),
	}

	return success
}

type SetCommand struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (c *SetCommand) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleCommandSet(cfg, text)
}

func (c *SetCommand) handleCommandSet(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := c.template.Options().ParseAll(text.Text(), template.CommandOptions)

	matches := c.re.FindStringSubmatch(textWithoutOpts) // !am cmd set <команды через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	edited := make([]string, 0, len(words))
	notFound := make([]string, 0, len(words))

	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			continue
		}

		if !strings.HasPrefix(word, "!") {
			word = "!" + word
		}

		if _, ok := cfg.Commands[word]; !ok {
			notFound = append(notFound, word)
			continue
		}

		cfg.Commands[word].Options = c.template.Options().MergeCommand(cfg.Commands[word].Options, opts)
		edited = append(edited, word)
	}

	return buildResponse("команды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type DelCommand struct {
	re *regexp.Regexp
}

func (c *DelCommand) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleCommandDel(cfg, text)
}

func (c *DelCommand) handleCommandDel(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Text()) // !am cmd del <команды через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	removed := make([]string, 0, len(words))
	notFound := make([]string, 0, len(words))

	for _, cmd := range words {
		cmd = strings.ToLower(strings.TrimSpace(cmd))
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

func (c *AliasesCommand) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleCommandAliases(cfg, text)
}

func (c *AliasesCommand) handleCommandAliases(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Text()) // !am cmd aliases <команда>
	if len(matches) != 2 {
		return nonParametr
	}

	cmd := strings.ToLower(strings.TrimSpace(matches[1]))
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

	aliasesStr := strings.Join(aliases, ",")
	if len(aliases) == 0 {
		aliasesStr = "не найдены"
	}

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("команда: %s, алиасы: %s", cmd, aliasesStr)},
		IsReply: true,
	}
}
