package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddAlias struct {
	template ports.TemplatePort
}

func (a *AddAlias) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAliasesAdd(cfg, text)
}

func (a *AddAlias) handleAliasesAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 6 { // !am alias add <алиас> from <оригинальная команда>
		return NonParametr
	}

	var fromIndex = -1
	for i, arg := range words {
		if arg == "from" {
			fromIndex = i
			break
		}
	}

	if fromIndex == -1 || fromIndex == 0 || fromIndex == len(words)-1 {
		return &ports.AnswerType{
			Text:    []string{"некорректный синтаксис!"},
			IsReply: true,
		}
	}

	alias := strings.Join(words[3:fromIndex], " ")
	original := strings.Join(words[fromIndex+1:], " ")

	if !strings.HasPrefix(original, "!") {
		original = "!" + original
	}

	if strings.Contains(alias, "!am alias") || alias == original {
		return &ports.AnswerType{
			Text:    []string{"нельзя добавить алиас на эту команду!"},
			IsReply: true,
		}
	}

	if cfg.Aliases[original] != "" {
		original = cfg.Aliases[original]
	}

	cfg.Aliases[alias] = original
	a.template.UpdateAliases(cfg.Aliases)
	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("алиас `%s` добавлен для команды `%s`!", alias, original)},
		IsReply: true,
	}
}

type DelAlias struct {
	template ports.TemplatePort
}

func (a *DelAlias) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAliasesDel(cfg, text)
}

func (a *DelAlias) handleAliasesDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am alias del <алиас>
		return NonParametr
	}

	alias := text.Tail(3)
	if _, ok := cfg.Aliases[alias]; !ok {
		return &ports.AnswerType{
			Text:    []string{"алиас не найден!"},
			IsReply: true,
		}
	}

	delete(cfg.Aliases, alias)
	a.template.UpdateAliases(cfg.Aliases)
	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("алиас `%s` удален!", alias)},
		IsReply: true,
	}
}

type ListAlias struct {
	fs ports.FileServerPort
}

func (a *ListAlias) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAliasesList(cfg, text)
}

func (a *ListAlias) handleAliasesList(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return buildList(cfg.Aliases, "алиасы", "алиасы не найдены!",
		func(alias, original string) string {
			return fmt.Sprintf("- %s → %s", alias, original)
		}, a.fs)
}
