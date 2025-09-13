package admin

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"strconv"
	"strings"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleExcept(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am ex list/add/set/del
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"list": a.handleExceptList,
		"set":  a.handleExceptSet,
		"del":  a.handleExDel,
	}

	exceptCmd := words[2]
	if handler, ok := handlers[exceptCmd]; ok {
		return handler(cfg, text)
	}
	return a.handleExceptAdd(cfg, text)
}

func (a *Admin) handleExceptList(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	if len(cfg.Spam.Exceptions) == 0 {
		return &ports.AnswerType{
			Text:    []string{"исключений не найдено!"},
			IsReply: true,
		}
	}

	var parts []string
	for word, ex := range cfg.Spam.Exceptions {
		parts = append(parts, fmt.Sprintf("- %s (лимит сообщений: %d, наказания: %s)", word, ex.MessageLimit, strings.Join(formatPunishments(ex.Punishments), ", ")))
	}
	msg := "исключения: \n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}

	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}

func (a *Admin) handleExceptAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := a.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

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

		p, err := parsePunishment(pa, true)
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

	if _, ok := opts["-regex"]; ok {
		re, err := regexp2.Compile(strings.Join(words[idx+2:], " "), regexp2.None)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		except := cfg.Spam.Exceptions[strings.Join(words[idx+2:], " ")]
		except = &config.SpamExceptionsSettings{
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Regexp:       re,
			Options:      a.mergeSpamOptions(except.Options, opts),
		}

		return nil
	}

	for _, word := range strings.Split(strings.Join(words[idx+2:], " "), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		except := cfg.Spam.Exceptions[word]
		except = &config.SpamExceptionsSettings{
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Options:      a.mergeSpamOptions(except.Options, opts),
		}
	}

	return nil
}

func (a *Admin) handleExceptSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()                                         // !am ex set ml/p <параметр 1> <слова или фразы>
	opts := a.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

	cmds := map[string]func(exWord *config.SpamExceptionsSettings, param string) *ports.AnswerType{
		"ml": func(exWord *config.SpamExceptionsSettings, param string) *ports.AnswerType {
			value, err := strconv.Atoi(param)
			if err != nil {
				return NonParametr
			}

			exWord.MessageLimit = value
			return nil
		},
		"p": func(exWord *config.SpamExceptionsSettings, param string) *ports.AnswerType {
			var punishments []config.Punishment
			for _, pa := range strings.Split(param, ",") {
				pa = strings.TrimSpace(pa)
				if pa == "" {
					continue
				}

				p, err := parsePunishment(pa, true)
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

	var updated, notFound []string
	processWord := func(word string) *ports.AnswerType {
		exWord, ok := cfg.Spam.Exceptions[word]
		if !ok {
			notFound = append(notFound, word)
			return nil
		}
		exWord.Options = a.mergeSpamOptions(exWord.Options, opts)

		if cmd, ok := cmds[words[3]]; ok {
			if out := cmd(exWord, words[4]); out != nil {
				return out
			}
			updated = append(updated, word)
		}
		return nil
	}

	if regex, ok := cfg.Spam.Exceptions[text.Tail(5)]; ok {
		if out := processWord(text.Tail(5)); out != nil {
			return out
		}
		_ = regex
	} else {
		for _, word := range strings.Split(words[5], ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			if out := processWord(word); out != nil {
				return out
			}
		}
	}

	return a.buildResponse(updated, "изменены", notFound, "не найдены", "исключения не указаны")
}

func (a *Admin) handleExDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am ex del <слова/фразы через запятую или regex>
		return NonParametr
	}

	var removed, notFound []string
	if _, ok := cfg.Spam.Exceptions[text.Tail(3)]; ok {
		delete(cfg.Spam.Exceptions, text.Tail(3))
		removed = append(removed, text.Tail(3))
	} else {
		for _, word := range strings.Split(words[3], ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			if _, ok := cfg.Spam.Exceptions[word]; ok {
				delete(cfg.Spam.Exceptions, word)
				removed = append(removed, word)
			} else {
				notFound = append(notFound, word)
			}
		}
	}

	return a.buildResponse(removed, "удалены", notFound, "не найдены", "исключения не указаны")
}
