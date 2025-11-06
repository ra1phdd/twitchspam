package admin

import (
	"fmt"
	"regexp"
	"strings"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type ListCommand struct {
	fs ports.FileServerPort
}

func (c *ListCommand) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
	return buildList(cfg.Channels[channel].Commands, "команды", "команды не найдены!",
		func(key string, cmd *config.Commands) string {
			return fmt.Sprintf("- %s -> %s", key, cmd.Text)
		}, c.fs)
}

type AddCommand struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (c *AddCommand) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	textWithoutOpts, opts := c.template.Options().ParseAll(msg.Message.Text.Text(), template.CommandOptions)

	matches := c.re.FindStringSubmatch(textWithoutOpts) // !am cmd add <команда> <текст>
	if len(matches) != 3 {
		return nonParametr
	}

	cmd := strings.ToLower(strings.TrimSpace(matches[1]))
	if !strings.HasPrefix(cmd, "!") {
		cmd = "!" + cmd
	}

	cfg.Channels[channel].Commands[cmd] = &config.Commands{
		Text:    strings.TrimSpace(matches[2]),
		Options: c.template.Options().MergeCommand(nil, opts),
	}

	return success
}

type SetCommand struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (c *SetCommand) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	textWithoutOpts, opts := c.template.Options().ParseAll(msg.Message.Text.Text(), template.CommandOptions)

	matches := c.re.FindStringSubmatch(textWithoutOpts) // !am cmd set <команда> <*текст>
	if len(matches) != 3 {
		return nonParametr
	}

	cmd := strings.ToLower(strings.TrimSpace(matches[1]))
	if !strings.HasPrefix(cmd, "!") {
		cmd = "!" + cmd
	}

	if _, ok := cfg.Channels[channel].Commands[cmd]; !ok {
		return &ports.AnswerType{
			Text:    []string{"команда не найдена!"},
			IsReply: true,
		}
	}

	cmdText := strings.TrimSpace(matches[2])
	if cmdText != "" {
		cfg.Channels[channel].Commands[cmd].Text = cmdText
	}
	cfg.Channels[channel].Commands[cmd].Options = c.template.Options().MergeCommand(cfg.Channels[channel].Commands[cmd].Options, opts)

	return success
}

type DelCommand struct {
	re *regexp.Regexp
}

func (c *DelCommand) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(msg.Message.Text.Text()) // !am word del <команды через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	removed, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if word == "" {
			continue
		}

		if !strings.HasPrefix(word, "!") {
			word = "!" + word
		}

		if _, ok := cfg.Channels[channel].Commands[word]; !ok {
			notFound = append(notFound, word)
			continue
		}

		delete(cfg.Channels[channel].Commands, word)
		for alias, original := range cfg.Channels[channel].Aliases {
			if original == word {
				delete(cfg.Channels[channel].Aliases, alias)
			}
		}
		removed = append(removed, word)
	}

	return buildResponse("команды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type AliasesCommand struct {
	re *regexp.Regexp
}

func (c *AliasesCommand) Execute(cfg *config.Config, channel string, msg *message.ChatMessage) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(msg.Message.Text.Text()) // !am cmd aliases <команда>
	if len(matches) != 2 {
		return nonParametr
	}

	cmd := strings.ToLower(strings.TrimSpace(matches[1]))
	if !strings.HasPrefix(cmd, "!") {
		cmd = "!" + cmd
	}

	if orig, ok := cfg.Channels[channel].Aliases[cmd]; ok {
		cmd = orig
	}

	for _, alg := range cfg.Channels[channel].AliasGroups {
		if _, ok := alg.Aliases[cmd]; ok {
			cmd = alg.Original
			break
		}
	}

	aliases := make([]string, 0, len(cfg.Channels[channel].Aliases))
	for alias, orig := range cfg.Channels[channel].Aliases {
		if strings.Contains(cmd, orig) {
			aliases = append(aliases, alias)
		}
	}

	for _, alg := range cfg.Channels[channel].AliasGroups {
		if !strings.Contains(cmd, alg.Original) {
			continue
		}

		for als := range alg.Aliases {
			aliases = append(aliases, als)
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
