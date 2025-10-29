package admin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddExcept struct {
	re         *regexp.Regexp
	template   ports.TemplatePort
	typeExcept string
}

func (e *AddExcept) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return e.handleExceptAdd(cfg, channel, text)
}

func (e *AddExcept) handleExceptAdd(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	textWithoutOpts, opts := e.template.Options().ParseAll(text.Text(), template.ExceptOptions)

	// !am ex (add) <кол-во сообщений> <наказания через запятую> <слова/фразы через запятую>
	// или !am ex (add) <кол-во сообщений> <наказания через запятую> re <name> <regex>
	matches := e.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) < 7 {
		return nonParametr
	}

	messageLimit, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil {
		return invalidMessageLimitFormat
	}

	if messageLimit < 2 || messageLimit > 15 {
		return invalidMessageLimitValue
	}

	parts := strings.Split(strings.TrimSpace(matches[2]), ",")
	punishments := make([]config.Punishment, 0, len(parts))

	for _, pa := range parts {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

		p, err := e.template.Punishment().Parse(pa, true)
		if err != nil {
			return errorPunishmentParse
		}

		if p.Action == "inherit" {
			if e.typeExcept == "emote" {
				punishments = cfg.Channels[channel].Spam.SettingsEmotes.Punishments
			} else {
				punishments = cfg.Channels[channel].Spam.SettingsDefault.Punishments
			}
			break
		}
		punishments = append(punishments, p)
	}

	if len(punishments) == 0 {
		return invalidPunishmentFormat
	}

	exSettings := cfg.Channels[channel].Spam.Exceptions
	fn := e.template.Options().MergeExcept
	if e.typeExcept == "emote" {
		exSettings = cfg.Channels[channel].Spam.SettingsEmotes.Exceptions
		fn = e.template.Options().MergeEmoteExcept
	}

	if strings.ToLower(strings.TrimSpace(matches[3])) == "re" {
		name, reStr := strings.TrimSpace(matches[4]), strings.TrimSpace(matches[5])

		re, err := regexp.Compile(reStr)
		if err != nil {
			return invalidRegex
		}

		exSettings[name] = &config.ExceptionsSettings{
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Regexp:       re,
			Options:      fn(nil, opts),
		}

		return success
	}

	words := strings.Split(strings.TrimSpace(matches[6]), ",")
	added, exists := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if _, ok := exSettings[word]; ok {
			exists = append(exists, word)
			continue
		}

		var options *config.ExceptOptions
		if message.HasDoubleLetters(word) || message.HasSpecialSymbols(word) {
			options = &config.ExceptOptions{}

			trueVal := true
			if message.HasDoubleLetters(word) {
				options.NoRepeat = &trueVal
			}
			if message.HasSpecialSymbols(word) {
				options.SavePunctuation = &trueVal
			}
		}

		exSettings[word] = &config.ExceptionsSettings{
			Enabled:      true,
			MessageLimit: messageLimit,
			Punishments:  punishments,
			Options:      e.template.Options().MergeExcept(options, opts),
		}
		added = append(added, word)
	}

	return buildResponse("исключения не указаны", RespArg{Items: added, Name: "добавлены"}, RespArg{Items: exists, Name: "уже существуют"})
}

type SetExcept struct {
	re         *regexp.Regexp
	template   ports.TemplatePort
	typeExcept string
}

func (e *SetExcept) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return e.handleExceptSet(cfg, channel, text)
}

func (e *SetExcept) handleExceptSet(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	textWithoutOpts, opts := e.template.Options().ParseAll(text.Text(), template.ExceptOptions)

	// !am ex set ml <значение> <слова или фразы через запятую>
	// или !am ex set p <наказания через запятую> <слова или фразы через запятую>
	// или !am ex set <слова или фразы через запятую>
	matches := e.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 4 {
		return nonParametr
	}

	cmds := map[string]func(exWord *config.ExceptionsSettings, param string) *ports.AnswerType{
		"ml": func(exWord *config.ExceptionsSettings, param string) *ports.AnswerType {
			messageLimit, err := strconv.Atoi(param)
			if err != nil {
				return nonParametr
			}

			if messageLimit == 0 {
				exWord.Enabled = false
				return success
			}

			if messageLimit < 2 || messageLimit > 15 {
				return invalidMessageLimitValue
			}

			exWord.MessageLimit = messageLimit
			return success
		},
		"p": func(exWord *config.ExceptionsSettings, param string) *ports.AnswerType {
			var punishments []config.Punishment
			for _, pa := range strings.Split(param, ",") {
				pa = strings.TrimSpace(pa)
				if pa == "" {
					continue
				}

				p, err := e.template.Punishment().Parse(pa, true)
				if err != nil {
					return &ports.AnswerType{
						Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
						IsReply: true,
					}
				}

				if p.Action == "inherit" {
					punishments = cfg.Channels[channel].Spam.SettingsDefault.Punishments
					break
				}

				punishments = append(punishments, p)
			}

			if len(punishments) == 0 {
				return invalidPunishmentFormat
			}

			exWord.Punishments = punishments
			return success
		},
	}

	exSettings := cfg.Channels[channel].Spam.Exceptions
	fn := e.template.Options().MergeExcept
	if e.typeExcept == "emote" {
		exSettings = cfg.Channels[channel].Spam.SettingsEmotes.Exceptions
		fn = e.template.Options().MergeEmoteExcept
	}

	words := strings.Split(strings.TrimSpace(matches[3]), ",")
	edited, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if _, ok := exSettings[word]; !ok {
			notFound = append(notFound, word)
			continue
		}

		exSettings[word].Options = fn(exSettings[word].Options, opts)

		if strings.TrimSpace(matches[1]) != "" {
			answer := cmds[strings.TrimSpace(matches[1])](exSettings[word], strings.TrimSpace(matches[2]))
			if answer != nil && answer != success {
				return answer
			}
		}
		edited = append(edited, word)
	}

	return buildResponse("исключения не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type DelExcept struct {
	re         *regexp.Regexp
	typeExcept string
}

func (e *DelExcept) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return e.handleExceptDel(cfg, channel, text)
}

func (e *DelExcept) handleExceptDel(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := e.re.FindStringSubmatch(text.Text()) // !am ex del <слова/фразы через запятую или regex>
	if len(matches) != 2 {
		return nonParametr
	}

	exSettings := cfg.Channels[channel].Spam.Exceptions
	if e.typeExcept == "emote" {
		exSettings = cfg.Channels[channel].Spam.SettingsEmotes.Exceptions
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	removed, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
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

	return buildResponse("исключения не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type ListExcept struct {
	template   ports.TemplatePort
	fs         ports.FileServerPort
	typeExcept string
}

func (e *ListExcept) Execute(cfg *config.Config, channel string, _ *message.Text) *ports.AnswerType {
	return e.handleExceptList(cfg, channel)
}

func (e *ListExcept) handleExceptList(cfg *config.Config, channel string) *ports.AnswerType {
	exSettings := cfg.Channels[channel].Spam.Exceptions
	if e.typeExcept == "emote" {
		exSettings = cfg.Channels[channel].Spam.SettingsEmotes.Exceptions
	}

	return buildList(exSettings, "исключения", "исключений не найдено!",
		func(word string, ex *config.ExceptionsSettings) string {
			return fmt.Sprintf("- %s (включено: %v, лимит сообщений: %d, наказания: %s)",
				word, ex.Enabled, ex.MessageLimit, strings.Join(e.template.Punishment().FormatAll(ex.Punishments), ", "))
		}, e.fs)
}

type OnOffExcept struct {
	re         *regexp.Regexp
	template   ports.TemplatePort
	typeExcept string
}

func (e *OnOffExcept) Execute(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	return e.handleExceptOnOff(cfg, channel, text)
}

func (e *OnOffExcept) handleExceptOnOff(cfg *config.Config, channel string, text *message.Text) *ports.AnswerType {
	matches := e.re.FindStringSubmatch(text.Text()) // !am ex on/off <слова/фразы через запятую>
	if len(matches) != 3 {
		return nonParametr
	}

	exSettings := cfg.Channels[channel].Spam.Exceptions
	if e.typeExcept == "emote" {
		exSettings = cfg.Channels[channel].Spam.SettingsEmotes.Exceptions
	}

	state := strings.ToLower(strings.TrimSpace(matches[1]))

	words := strings.Split(strings.TrimSpace(matches[2]), ",")
	edited, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		except, ok := exSettings[word]
		if !ok {
			notFound = append(notFound, word)
			continue
		}

		except.Enabled = state == "on"
		edited = append(edited, word)
	}

	return buildResponse("исключения не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}
