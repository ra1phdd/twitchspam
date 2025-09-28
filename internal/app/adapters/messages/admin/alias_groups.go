package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

var NotFoundAliasGroup = &ports.AnswerType{
	Text:    []string{"группа алиасов не найдена!"},
	IsReply: true,
}

type CreateAliasGroup struct {
	template ports.TemplatePort
}

func (a *CreateAliasGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAlgCreate(cfg, text)
}

func (a *CreateAliasGroup) handleAlgCreate(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 5 { // !am alg create <название_группы> <оригинальная команда>
		return NonParametr
	}

	groupName := words[3]
	if _, exists := cfg.AliasGroups[groupName]; exists {
		return &ports.AnswerType{
			Text:    []string{"группа алиасов уже существует!"},
			IsReply: true,
		}
	}

	original := text.Tail(4)
	if !strings.HasPrefix(original, "!") {
		original = "!" + original
	}

	cfg.AliasGroups[groupName] = &config.AliasGroups{
		Aliases:  make(map[string]struct{}),
		Original: original,
	}
	a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups)

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("группа алиасов `%s` создана!", groupName)},
		IsReply: true,
	}
}

type AddAliasGroup struct {
	template ports.TemplatePort
}

func (a *AddAliasGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAlgAdd(cfg, text)
}

func (a *AddAliasGroup) handleAlgAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 5 { // !am alg add <название_группы> <алиасы>
		return NonParametr
	}

	groupName := words[3]
	aliasList := strings.Split(text.Tail(4), ",")

	_, exists := cfg.AliasGroups[groupName]
	if !exists {
		return NotFoundAliasGroup
	}

	for _, alias := range aliasList {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			continue
		}

		cfg.AliasGroups[groupName].Aliases[alias] = struct{}{}
	}

	a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups)
	return nil
}

type DelAliasGroup struct {
	template ports.TemplatePort
}

func (a *DelAliasGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAlgDel(cfg, text)
}

func (a *DelAliasGroup) handleAlgDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am alg del <название_группы> <алиасы через запятую или all>
		return NonParametr
	}

	groupName := words[3]
	group, exists := cfg.AliasGroups[groupName]
	if !exists {
		return NotFoundAliasGroup
	}

	if words[4] == "all" {
		delete(cfg.AliasGroups, groupName)
		a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups)
		return nil
	}

	args := strings.Split(words[4], ",")
	argsSet := make(map[string]struct{}, len(args))
	for _, w := range args {
		argsSet[w] = struct{}{}
	}

	var removed, notFound []string
	for w := range argsSet {
		if _, ok := group.Aliases[w]; ok {
			delete(cfg.AliasGroups[groupName].Aliases, w)
			removed = append(removed, w)
		} else {
			notFound = append(notFound, w)
		}
	}

	cfg.AliasGroups[groupName] = group
	a.template.Aliases().Update(cfg.Aliases, cfg.AliasGroups)

	return buildResponse(removed, "удалены", notFound, "не найдены", "слова в мворд группе не указаны")
}

type OnOffAliasGroup struct {
	template ports.TemplatePort
}

func (a *OnOffAliasGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAlgOnOff(cfg, text)
}

func (a *OnOffAliasGroup) handleAlgOnOff(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am alg on/off <название_группы>
		return NonParametr
	}

	_, exists := cfg.AliasGroups[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}
	cfg.MwordGroup[words[3]].Enabled = words[2] == "on"

	return nil
}
