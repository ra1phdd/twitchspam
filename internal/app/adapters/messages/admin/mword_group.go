package admin

import (
	"fmt"
	"slices"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleMwg(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	mwgCmd, mwgArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) ports.ActionType{
		"list":   a.handleMwgList,
		"create": a.handleMwgCreate,
		"set":    a.handleMwgSet,
		"add":    a.handleMwgAdd,
		"del":    a.handleMwgDel,
		"on":     a.handleMwgOnOff,
		"off":    a.handleMwgOnOff,
		"words":  a.handleMwgWords,
	}

	if handler, ok := handlers[mwgCmd]; ok {
		return handler(cfg, mwgCmd, mwgArgs)
	}
	return NotFound
}

func (a *Admin) handleMwgList(cfg *config.Config, _ string, _ []string) ports.ActionType {
	if len(cfg.MwordGroup) == 0 {
		return ErrNotFoundMwordGroups
	}

	msg := "группы:"
	for name := range cfg.MwordGroup {
		msg += fmt.Sprintf(" %s", name)
	}

	return ports.ActionType(msg)
}

func (a *Admin) handleMwgCreate(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	punishment := args[1]

	if _, exists := cfg.MwordGroup[groupName]; exists {
		return ErrFoundMwordGroup
	}

	action, duration, err := parsePunishment(punishment)
	if err != nil {
		return ErrFound
	}

	cfg.MwordGroup[groupName] = &config.MwordGroup{
		Action:   action,
		Duration: duration,
		Enabled:  true,
		Words:    []string{},
	}

	return Success
}

func (a *Admin) handleMwgSet(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	punishment := args[1]

	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return ErrNotFoundMwordGroup
	}

	action, duration, err := parsePunishment(punishment)
	if err != nil {
		return ErrFound
	}

	cfg.MwordGroup[groupName].Action = action
	cfg.MwordGroup[groupName].Duration = duration

	return Success
}

func (a *Admin) handleMwgAdd(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return NotFound
	}

	words := a.regexp.SplitWords(strings.Join(args[1:], " "))
	for _, word := range words {
		trimmed := strings.TrimSpace(word)
		if trimmed == "" {
			continue
		}

		if re, err := a.regexp.Parse(trimmed); err != nil {
			return ports.ActionType(err.Error())
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

	return Success
}

func (a *Admin) handleMwgDel(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	target := strings.Join(args[1:], " ")

	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return NotFound
	}

	if target == "all" {
		delete(cfg.MwordGroup, groupName)
		return Success
	}

	wordsToRemove := a.regexp.SplitWords(target)
	var newWords []string

	for _, existingWord := range group.Words {
		if !slices.Contains(wordsToRemove, existingWord) {
			newWords = append(newWords, existingWord)
		}
	}
	group.Words = newWords

	return Success
}

func (a *Admin) handleMwgOnOff(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	groupName := args[0]

	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return ErrNotFoundMwordGroup
	}
	cfg.MwordGroup[groupName].Enabled = cmd == "on"

	return Success
}

func (a *Admin) handleMwgWords(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	groupName := args[0]

	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return ErrNotFoundMwordGroup
	}

	if len(group.Words) == 0 {
		return "cлова в группе отсутствуют"
	}

	msg := "cлова в группе: " + strings.Join(group.Words, ", ")
	return ports.ActionType(msg)
}
