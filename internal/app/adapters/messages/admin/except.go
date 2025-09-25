package admin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddExcept struct {
	template   ports.TemplatePort
	typeExcept string
}

func (e *AddExcept) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return e.handleExceptAdd(cfg, text)
}

func (e *AddExcept) handleExceptAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := e.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

	idx := 2 // id параметра, с которого начинаются аргументы команды
	if words[2] == "add" {
		idx = 3
	}

	// !am ex <кол-во сообщений> <наказания через запятую> <слова/фразы через запятую или regex>
	// или !am ex add <кол-во сообщений> <наказания через запятую> <слова/фразы через запятую или regex>
	if len(words) < idx+3 {
		return NonParametr
	}

	messageLimit, err := strconv.Atoi(words[idx])
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{"не указан лимит сообщений!"},
			IsReply: true,
		}
	}

	var punishments []config.Punishment
	for _, pa := range strings.Split(words[idx+1], ",") {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

		p, err := e.template.ParsePunishment(pa, true)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
				IsReply: true,
			}
		}

		if p.Action == "inherit" {
			if e.typeExcept == "emote" {
				punishments = cfg.Spam.SettingsEmotes.Punishments
			} else {
				punishments = cfg.Spam.SettingsDefault.Punishments
			}
			break
		}
		punishments = append(punishments, p)
	}

	exSettings := cfg.Spam.Exceptions
	if e.typeExcept == "emote" {
		exSettings = cfg.Spam.SettingsEmotes.Exceptions
	}

	if _, ok := opts["-regex"]; ok {
		re, err := regexp.Compile(strings.Join(words[idx+2:], " "))
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		optsMerged := mergeSpamOptions(config.SpamOptions{}, opts)
		if except, ok := exSettings[strings.Join(words[idx+2:], " ")]; ok {
			optsMerged = mergeSpamOptions(except.Options, opts)
		}

		exSettings[strings.Join(words[idx+2:], " ")] = &config.ExceptionsSettings{
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Regexp:       re,
			Options:      optsMerged,
		}

		return nil
	}

	for _, word := range strings.Split(strings.Join(words[idx+2:], " "), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		optsMerged := mergeSpamOptions(config.SpamOptions{}, opts)
		if except, ok := exSettings[word]; ok {
			optsMerged = mergeSpamOptions(except.Options, opts)
		}

		exSettings[word] = &config.ExceptionsSettings{
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Options:      optsMerged,
		}
	}

	return nil
}

type SetExcept struct {
	template   ports.TemplatePort
	typeExcept string
}

func (e *SetExcept) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return e.handleExceptSet(cfg, text)
}

func (e *SetExcept) handleExceptSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()                                         // !am ex set ml/p <параметр 1> <слова или фразы>
	opts := e.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

	cmds := map[string]func(exWord *config.ExceptionsSettings, param string) *ports.AnswerType{
		"ml": func(exWord *config.ExceptionsSettings, param string) *ports.AnswerType {
			value, err := strconv.Atoi(param)
			if err != nil {
				return NonParametr
			}

			exWord.MessageLimit = value
			return nil
		},
		"p": func(exWord *config.ExceptionsSettings, param string) *ports.AnswerType {
			var punishments []config.Punishment
			for _, pa := range strings.Split(param, ",") {
				pa = strings.TrimSpace(pa)
				if pa == "" {
					continue
				}

				p, err := e.template.ParsePunishment(pa, true)
				if err != nil {
					return &ports.AnswerType{
						Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
						IsReply: true,
					}
				}

				if p.Action == "inherit" {
					punishments = cfg.Spam.SettingsDefault.Punishments
					break
				}

				punishments = append(punishments, p)
			}

			exWord.Punishments = punishments
			return nil
		},
	}

	exSettings := cfg.Spam.Exceptions
	if e.typeExcept == "emote" {
		exSettings = cfg.Spam.SettingsEmotes.Exceptions
	}

	var updated, notFound []string
	processWord := func(exSettings map[string]*config.ExceptionsSettings, word string) *ports.AnswerType {
		exWord, ok := exSettings[word]
		if !ok {
			notFound = append(notFound, word)
			return nil
		}
		exWord.Options = mergeSpamOptions(exWord.Options, opts)

		if cmd, ok := cmds[words[3]]; ok {
			if out := cmd(exWord, words[4]); out != nil {
				return out
			}
			updated = append(updated, word)
		}
		return nil
	}

	if regex, ok := exSettings[text.Tail(5)]; ok {
		if out := processWord(exSettings, text.Tail(5)); out != nil {
			return out
		}
		_ = regex
	} else {
		for _, word := range strings.Split(words[5], ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			if out := processWord(exSettings, word); out != nil {
				return out
			}
		}
	}

	return buildResponse(updated, "изменены", notFound, "не найдены", "исключения не указаны")
}

type DelExcept struct {
	typeExcept string
}

func (e *DelExcept) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return e.handleExceptDel(cfg, text)
}

func (e *DelExcept) handleExceptDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am ex del <слова/фразы через запятую или regex>
		return NonParametr
	}

	exSettings := cfg.Spam.Exceptions
	if e.typeExcept == "emote" {
		exSettings = cfg.Spam.SettingsEmotes.Exceptions
	}

	var removed, notFound []string
	if _, ok := exSettings[text.Tail(3)]; ok {
		delete(exSettings, text.Tail(3))
		removed = append(removed, text.Tail(3))
	} else {
		for _, word := range strings.Split(words[3], ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			if _, ok := exSettings[word]; ok {
				delete(exSettings, word)
				removed = append(removed, word)
			} else {
				notFound = append(notFound, word)
			}
		}
	}

	return buildResponse(removed, "удалены", notFound, "не найдены", "исключения не указаны")
}

type ListExcept struct {
	template   ports.TemplatePort
	fs         ports.FileServerPort
	typeExcept string
}

func (e *ListExcept) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return e.handleExceptList(cfg)
}

func (e *ListExcept) handleExceptList(cfg *config.Config) *ports.AnswerType {
	exSettings := cfg.Spam.Exceptions
	if e.typeExcept == "emote" {
		exSettings = cfg.Spam.SettingsEmotes.Exceptions
	}

	return buildList(exSettings, "исключения", "исключений не найдено!",
		func(word string, ex *config.ExceptionsSettings) string {
			return fmt.Sprintf("- %s (лимит сообщений: %d, наказания: %s)",
				word, ex.MessageLimit, strings.Join(e.template.FormatPunishments(ex.Punishments), ", "))
		}, e.fs)
}

type OnOffExcept struct {
	template   ports.TemplatePort
	typeExcept string
}

func (e *OnOffExcept) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return e.handleExceptOnOff(cfg, text)
}

func (e *OnOffExcept) handleExceptOnOff(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := e.template.ParseOptions(&words, template.SpamOptions)

	if len(words) < 4 { // !am ex on/off <команды через запятую>
		return NonParametr
	}

	exSettings := cfg.Spam.Exceptions
	if e.typeExcept == "emote" {
		exSettings = cfg.Spam.SettingsEmotes.Exceptions
	}

	if _, ok := opts["-regex"]; ok {
		except, ok := exSettings[strings.Join(words[3:], " ")]
		if !ok {
			return &ports.AnswerType{
				Text:    []string{"исключение не найдено"},
				IsReply: true,
			}
		}

		except.Enabled = words[2] == "on"
		return nil
	}

	var edited, notFound []string
	for _, key := range strings.Split(words[3], ",") {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		except, ok := exSettings[key]
		if !ok {
			notFound = append(notFound, key)
			continue
		}

		except.Enabled = words[2] == "on"
		edited = append(edited, key)
	}

	return buildResponse(edited, "изменены", notFound, "не найдены", "исключения не указаны")
}
