package admin

import (
	"fmt"
	"regexp"
	"strings"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

// Одиночные алиасы

type ListAlias struct {
	fs ports.FileServerPort
}

func (a *ListAlias) Execute(cfg *config.Config, channel string, _ *message.Text) *ports.AnswerType {
	return a.handleAliasesList(cfg, channel)
}

func (a *ListAlias) handleAliasesList(cfg *config.Config, channel string) *ports.AnswerType {
	aliases := make(map[string]string) // !am al list
	for k, v := range cfg.Channels[channel].Aliases {
		aliases[k] = v
	}

	for _, als := range cfg.Channels[channel].AliasGroups {
		if len(als.Aliases) == 0 {
			continue
		}

		alias := make([]string, 0, len(als.Aliases))
		for key := range als.Aliases {
			alias = append(alias, key)
		}

		data := strings.Join(alias, ", ") + func() string {
			if !als.Enabled {
				return " (выключено)"
			}
			return ""
		}()
		aliases[data] = als.Original
	}

	return buildList(aliases, "алиасы", "алиасы не найдены!",
		func(alias, original string) string {
			return fmt.Sprintf("- %s → %s", alias, original)
		}, a.fs)
}

type AddAlias struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *AddAlias) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return a.handleAliasesAdd(cfg, channel, text)
}

func (a *AddAlias) handleAliasesAdd(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am al add <алиасы через запятую> from <оригинальная команда>
	if len(matches) != 3 {
		return incorrectSyntax
	}

	aliases := strings.TrimSpace(matches[1])
	original := strings.TrimSpace(matches[2])

	if aliases == "" || original == "" {
		return incorrectSyntax
	}

	// алиас может ссылаться только на команду, а не на текст
	// при этом сам алиас может быть текстом, пример - "что за игра"
	if !strings.HasPrefix(original, "!") {
		original = "!" + original
	}

	if cfg.Channels[channel].Aliases[original] != "" {
		original = cfg.Channels[channel].Aliases[original]
	}

	for _, alias := range strings.Split(aliases, ",") {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}

		if strings.Contains(alias, "!am al ") || strings.Contains(alias, "!am alg ") || alias == original {
			return aliasDenied
		}

		cfg.Channels[channel].Aliases[alias] = original
	}

	a.template.Aliases().Update(cfg.Channels[channel].Aliases, cfg.Channels[channel].AliasGroups, cfg.GlobalAliases)
	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("алиасы `%s` добавлены для команды `%s`!", aliases, original)},
		IsReply: true,
	}
}

type DelAlias struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *DelAlias) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return a.handleAliasesDel(cfg, channel, text)
}

func (a *DelAlias) handleAliasesDel(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am al del <алиасы через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	removed, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if _, ok := cfg.Channels[channel].Aliases[word]; ok {
			delete(cfg.Channels[channel].Aliases, word)
			removed = append(removed, word)
		} else {
			notFound = append(notFound, word)
		}
	}

	a.template.Aliases().Update(cfg.Channels[channel].Aliases, cfg.Channels[channel].AliasGroups, cfg.GlobalAliases)
	return buildResponse("алиасы не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

// Группы алиасов

type CreateAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *CreateAliasGroup) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return a.handleAlgCreate(cfg, channel, text)
}

func (a *CreateAliasGroup) handleAlgCreate(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg create <название_группы> <оригинальная команда>
	if len(matches) != 3 {
		return incorrectSyntax
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.Channels[channel].AliasGroups[groupName]; exists {
		return existsAliasGroup
	}

	original := strings.TrimSpace(matches[2])
	if !strings.HasPrefix(original, "!") {
		original = "!" + original
	}

	cfg.Channels[channel].AliasGroups[groupName] = &config.AliasGroups{
		Enabled:  true,
		Aliases:  make(map[string]struct{}),
		Original: original,
	}
	a.template.Aliases().Update(cfg.Channels[channel].Aliases, cfg.Channels[channel].AliasGroups, cfg.GlobalAliases)

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("группа алиасов `%s` создана!", groupName)},
		IsReply: true,
	}
}

type AddAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *AddAliasGroup) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return a.handleAlgAdd(cfg, channel, text)
}

func (a *AddAliasGroup) handleAlgAdd(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg add <название_группы> <алиасы через запятую>
	if len(matches) != 3 {
		return incorrectSyntax
	}

	groupName := strings.TrimSpace(matches[1])
	group, exists := cfg.Channels[channel].AliasGroups[groupName]
	if !exists {
		return notFoundAliasGroup
	}

	aliases := strings.Split(strings.TrimSpace(matches[2]), ",")
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}

		if strings.Contains(alias, "!am al ") || strings.Contains(alias, "!am alg ") || alias == group.Original {
			return aliasDenied
		}

		cfg.Channels[channel].AliasGroups[groupName].Aliases[alias] = struct{}{}
	}

	a.template.Aliases().Update(cfg.Channels[channel].Aliases, cfg.Channels[channel].AliasGroups, cfg.GlobalAliases)
	return success
}

type SetAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *SetAliasGroup) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return a.handleAlgSet(cfg, channel, text)
}

func (a *SetAliasGroup) handleAlgSet(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg set <название_группы> <оригинальная команда>
	if len(matches) != 3 {
		return incorrectSyntax
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.Channels[channel].AliasGroups[groupName]; !exists {
		return notFoundAliasGroup
	}

	original := strings.TrimSpace(matches[2])
	if !strings.HasPrefix(original, "!") {
		original = "!" + original
	}

	cfg.Channels[channel].AliasGroups[groupName].Original = original
	a.template.Aliases().Update(cfg.Channels[channel].Aliases, cfg.Channels[channel].AliasGroups, cfg.GlobalAliases)
	return success
}

type DelAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *DelAliasGroup) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return a.handleAlgDel(cfg, channel, text)
}

func (a *DelAliasGroup) handleAlgDel(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg del <название_группы> <алиасы через запятую или ничего для удаления группы>
	if len(matches) < 2 {
		return incorrectSyntax
	}

	groupName := strings.TrimSpace(matches[1])
	group, exists := cfg.Channels[channel].AliasGroups[groupName]
	if !exists {
		return notFoundAliasGroup
	}
	defer a.template.Aliases().Update(cfg.Channels[channel].Aliases, cfg.Channels[channel].AliasGroups, cfg.GlobalAliases)

	if len(matches) < 3 || strings.TrimSpace(matches[2]) == "" {
		delete(cfg.Channels[channel].AliasGroups, groupName)
		return success
	}

	words := strings.Split(strings.TrimSpace(matches[2]), ",")
	removed, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		if _, ok := group.Aliases[word]; ok {
			delete(cfg.Channels[channel].AliasGroups[groupName].Aliases, word)
			removed = append(removed, word)
		} else {
			notFound = append(notFound, word)
		}
	}

	return buildResponse("алиасы не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type OnOffAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *OnOffAliasGroup) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return a.handleAlgOnOff(cfg, channel, text)
}

func (a *OnOffAliasGroup) handleAlgOnOff(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg on/off <название_группы>
	if len(matches) != 3 {
		return incorrectSyntax
	}

	state := strings.ToLower(strings.TrimSpace(matches[1]))
	groupName := strings.TrimSpace(matches[2])

	if _, exists := cfg.Channels[channel].AliasGroups[groupName]; !exists {
		return notFoundAliasGroup
	}

	cfg.Channels[channel].AliasGroups[groupName].Enabled = state == "on"
	a.template.Aliases().Update(cfg.Channels[channel].Aliases, cfg.Channels[channel].AliasGroups, cfg.GlobalAliases)
	return success
}
