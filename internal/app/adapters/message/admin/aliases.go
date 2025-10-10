package admin

import (
	"fmt"
	"regexp"
	"strings"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

// Одиночные алиасы

type ListAlias struct {
	fs ports.FileServerPort
}

func (a *ListAlias) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return a.handleAliasesList(cfg)
}

func (a *ListAlias) handleAliasesList(cfg *config.Config) *ports.AnswerType {
	aliases := make(map[string]string) // !am al list
	for k, v := range cfg.Aliases {
		aliases[k] = v
	}

	for _, als := range cfg.AliasGroups {
		if len(als.Aliases) == 0 {
			continue
		}

		var alias []string
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

func (a *AddAlias) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleAliasesAdd(cfg, text)
}

func (a *AddAlias) handleAliasesAdd(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
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

	if cfg.Aliases[original] != "" {
		original = cfg.Aliases[original]
	}

	for _, alias := range strings.Split(aliases, ",") {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}

		if strings.Contains(alias, "!am al ") || strings.Contains(alias, "!am alg ") || alias == original {
			return aliasDenied
		}

		cfg.Aliases[alias] = original
	}

	a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups, cfg.GlobalAliases)
	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("алиасы `%s` добавлены для команды `%s`!", aliases, original)},
		IsReply: true,
	}
}

type DelAlias struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *DelAlias) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleAliasesDel(cfg, text)
}

func (a *DelAlias) handleAliasesDel(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am al del <алиасы через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	var removed, notFound []string
	for _, alias := range strings.Split(strings.TrimSpace(matches[1]), ",") {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}

		if _, ok := cfg.Aliases[alias]; ok {
			delete(cfg.Aliases, alias)
			removed = append(removed, alias)
		} else {
			notFound = append(notFound, alias)
		}
	}

	a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups, cfg.GlobalAliases)
	return buildResponse("алиасы не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

// Группы алиасов

type CreateAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *CreateAliasGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleAlgCreate(cfg, text)
}

func (a *CreateAliasGroup) handleAlgCreate(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg create <название_группы> <оригинальная команда>
	if len(matches) != 3 {
		return incorrectSyntax
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.AliasGroups[groupName]; exists {
		return existsAliasGroup
	}

	original := strings.TrimSpace(matches[2])
	if !strings.HasPrefix(original, "!") {
		original = "!" + original
	}

	cfg.AliasGroups[groupName] = &config.AliasGroups{
		Enabled:  true,
		Aliases:  make(map[string]struct{}),
		Original: original,
	}
	a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups, cfg.GlobalAliases)

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("группа алиасов `%s` создана!", groupName)},
		IsReply: true,
	}
}

type AddAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *AddAliasGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleAlgAdd(cfg, text)
}

func (a *AddAliasGroup) handleAlgAdd(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg add <название_группы> <алиасы через запятую>
	if len(matches) != 3 {
		return incorrectSyntax
	}

	groupName := strings.TrimSpace(matches[1])
	group, exists := cfg.AliasGroups[groupName]
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

		cfg.AliasGroups[groupName].Aliases[alias] = struct{}{}
	}

	a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups, cfg.GlobalAliases)
	return success
}

type DelAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *DelAliasGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleAlgDel(cfg, text)
}

func (a *DelAliasGroup) handleAlgDel(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg del <название_группы> <алиасы через запятую или ничего для удаления группы>
	if len(matches) < 2 {
		return incorrectSyntax
	}

	groupName := strings.TrimSpace(matches[1])
	group, exists := cfg.AliasGroups[groupName]
	if !exists {
		return notFoundAliasGroup
	}
	defer a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups, cfg.GlobalAliases)

	if len(matches) < 3 || strings.TrimSpace(matches[2]) == "" {
		delete(cfg.AliasGroups, groupName)
		return success
	}

	var removed, notFound []string
	for _, w := range strings.Split(strings.TrimSpace(matches[2]), ",") {
		if _, ok := group.Aliases[w]; ok {
			delete(cfg.AliasGroups[groupName].Aliases, w)
			removed = append(removed, w)
		} else {
			notFound = append(notFound, w)
		}
	}

	return buildResponse("алиасы не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type OnOffAliasGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *OnOffAliasGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return a.handleAlgOnOff(cfg, text)
}

func (a *OnOffAliasGroup) handleAlgOnOff(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am alg on/off <название_группы>
	if len(matches) != 3 {
		return incorrectSyntax
	}

	state := strings.ToLower(strings.TrimSpace(matches[1]))
	groupName := strings.TrimSpace(matches[2])

	if _, exists := cfg.AliasGroups[groupName]; !exists {
		return notFoundAliasGroup
	}

	cfg.AliasGroups[groupName].Enabled = state == "on"
	a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups, cfg.GlobalAliases)
	return success
}
