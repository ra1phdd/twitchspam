package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleAliases(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	aliasCmd, aliasArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) *ports.AnswerType{
		"add":  a.handleAliasesAdd,
		"del":  a.handleAliasesDel,
		"list": a.handleAliasesList,
	}

	if handler, ok := handlers[aliasCmd]; ok {
		return handler(cfg, aliasCmd, aliasArgs)
	}
	return NotFoundCmd
}

func (a *Admin) handleAliasesAdd(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 3 {
		return NonParametr
	}

	var fromIndex = -1
	for i, arg := range args {
		if arg == "from" {
			fromIndex = i
			break
		}
	}

	if fromIndex == -1 || fromIndex == 0 || fromIndex == len(args)-1 {
		return &ports.AnswerType{
			Text:    []string{"некорректный синтаксис!"},
			IsReply: true,
		}
	}

	alias := strings.Join(args[:fromIndex], " ")
	original := strings.Join(args[fromIndex+1:], " ")

	if !strings.HasPrefix(alias, "!") {
		alias = "!" + alias
	}

	if !strings.HasPrefix(original, "!") {
		original = "!" + original
	}

	cfg.Aliases[alias] = original
	a.aliases.Update(cfg.Aliases)
	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("алиас `%s` добавлен для команды `%s`!", alias, original)},
		IsReply: true,
	}
}

func (a *Admin) handleAliasesDel(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}

	alias := strings.Join(args, " ")
	if !strings.HasPrefix(alias, "!") {
		alias = "!" + alias
	}

	if _, ok := cfg.Aliases[alias]; ok {
		delete(cfg.Aliases, alias)
		a.aliases.Update(cfg.Aliases)
		return &ports.AnswerType{
			Text:    []string{fmt.Sprintf("алиас `%s` удален!", alias)},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text:    []string{"алиас не найден!"},
		IsReply: true,
	}
}

func (a *Admin) handleAliasesList(cfg *config.Config, _ string, _ []string) *ports.AnswerType {
	if len(cfg.Aliases) == 0 {
		return &ports.AnswerType{
			Text:    []string{"алиасы не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for alias, original := range cfg.Aliases {
		parts = append(parts, fmt.Sprintf("- %s → %s", alias, original))
	}
	msg := "алиасы: \n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}
