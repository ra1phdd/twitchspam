package admin

import (
	"fmt"
	"slices"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

var NotFoundMwordGroup = &ports.AnswerType{
	Text:    []string{"мворд группа не найдена!"},
	IsReply: true,
}

func (a *Admin) handleMwg(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	mwgCmd, mwgArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) *ports.AnswerType{
		"list":   a.handleMwgList,
		"create": a.handleMwgCreate,
		"set":    a.handleMwgSet,
		"add":    a.handleMwgAdd,
		"del":    a.handleMwgDel,
		"on":     a.handleMwgOnOff,
		"off":    a.handleMwgOnOff,
	}

	if handler, ok := handlers[mwgCmd]; ok {
		return handler(cfg, mwgCmd, mwgArgs)
	}
	return NotFoundCmd
}

func (a *Admin) handleMwgList(cfg *config.Config, _ string, _ []string) *ports.AnswerType {
	if len(cfg.MwordGroup) == 0 {
		return &ports.AnswerType{
			Text:    []string{"мворд группы не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for name, mwg := range cfg.MwordGroup {
		var re []string
		for _, pattern := range mwg.Regexp {
			re = append(re, pattern.String())
		}

		parts = append(parts, fmt.Sprintf("- %s (enabled: %v, action: %s, duration: %d, words: %s, regexp: %s)",
			name, mwg.Enabled, mwg.Action, mwg.Duration, strings.Join(mwg.Words, ", "), strings.Join(re, ", ")))
	}
	msg := "мворд группы: \n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}

func (a *Admin) handleMwgCreate(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	punishment := args[1]

	if _, exists := cfg.MwordGroup[groupName]; exists {
		return &ports.AnswerType{
			Text:    []string{"мворд группа уже существует!"},
			IsReply: true,
		}
	}

	action, duration, err := parsePunishment(punishment)
	if err != nil {
		return UnknownPunishment
	}

	cfg.MwordGroup[groupName] = &config.MwordGroup{
		Action:   action,
		Duration: duration,
		Enabled:  true,
	}

	return nil
}

func (a *Admin) handleMwgSet(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	punishment := args[1]

	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return NotFoundMwordGroup
	}

	action, duration, err := parsePunishment(punishment)
	if err != nil {
		return UnknownPunishment
	}

	cfg.MwordGroup[groupName].Action = action
	cfg.MwordGroup[groupName].Duration = duration

	return nil
}

func (a *Admin) handleMwgAdd(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return NotFoundMwordGroup
	}

	words := a.regexp.SplitWords(strings.Join(args[1:], " "))
	for _, word := range words {
		trimmed := strings.TrimSpace(word)
		if trimmed == "" {
			continue
		}

		if re, err := a.regexp.Parse(trimmed); err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		} else if re != nil {
			if !regexExists(group.Regexp, re) {
				group.Regexp = append(group.Regexp, re)
			}
			continue
		}

		if !slices.Contains(group.Words, trimmed) {
			group.Words = append(group.Words, trimmed)
		}
	}

	return nil
}

func (a *Admin) handleMwgDel(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	target := strings.Join(args[1:], " ")

	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return NotFoundMwordGroup
	}

	if target == "all" {
		delete(cfg.MwordGroup, groupName)
		return nil
	}

	wordsToRemove := a.regexp.SplitWords(target)
	var newWords []string

	for _, existingWord := range group.Words {
		if !slices.Contains(wordsToRemove, existingWord) {
			newWords = append(newWords, existingWord)
		}
	}
	group.Words = newWords

	return nil
}

func (a *Admin) handleMwgOnOff(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	groupName := args[0]

	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return NotFoundMwordGroup
	}
	cfg.MwordGroup[groupName].Enabled = cmd == "on"

	return nil
}
