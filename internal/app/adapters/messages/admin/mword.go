package admin

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"strings"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddMword struct {
	template ports.TemplatePort
}

func (m *AddMword) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwAdd(cfg, text)
}

func (m *AddMword) handleMwAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := m.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

	idx := 2 // id параметра, с которого начинаются аргументы команды
	if words[2] == "add" {
		idx = 3
	}

	// !am mw <наказания через запятую> <слова/фразы через запятую>
	// или !am mw add <наказания через запятую> <слова/фразы через запятую>
	if len(words) < idx+3 {
		return NonParametr
	}

	var punishments []config.Punishment
	for _, pa := range strings.Split(words[idx], ",") {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

		p, err := m.template.ParsePunishment(pa, false)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
				IsReply: true,
			}
		}
		punishments = append(punishments, p)
	}

	if _, ok := opts["-regex"]; ok {
		re, err := regexp2.Compile(strings.Join(words[idx+1:], " "), regexp2.None)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		mword := cfg.Mword[strings.Join(words[idx+1:], " ")]
		mword = &config.Mword{
			Punishments: punishments,
			Regexp:      re,
			Options:     mergeSpamOptions(mword.Options, opts),
		}
		m.template.UpdateMwords(cfg.MwordGroup, cfg.Mword)
		return nil
	}

	for _, word := range strings.Split(strings.Join(words[idx+1:], " "), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		mword := cfg.Mword[word]
		mword = &config.Mword{
			Punishments: punishments,
			Options:     mergeSpamOptions(mword.Options, opts),
		}
	}

	m.template.UpdateMwords(cfg.MwordGroup, cfg.Mword)
	return nil
}

type DelMword struct {
	template ports.TemplatePort
}

func (m *DelMword) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwDel(cfg, text)
}

func (m *DelMword) handleMwDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am mw del <слова/фразы через запятую или regex>
		return NonParametr
	}

	var removed, notFound []string
	if _, ok := cfg.Mword[text.Tail(3)]; ok {
		delete(cfg.Mword, text.Tail(3))
		removed = append(removed, text.Tail(3))
	} else {
		for _, word := range strings.Split(text.Tail(3), ",") {
			word = strings.TrimSpace(word)
			if word == "" {
				continue
			}

			if _, ok := cfg.Mword[word]; ok {
				delete(cfg.Mword, word)
				removed = append(removed, word)
			} else {
				notFound = append(notFound, word)
			}
		}
	}

	m.template.UpdateMwords(cfg.MwordGroup, cfg.Mword)
	return buildResponse(removed, "удалены", notFound, "не найдены", "мворды не указаны")
}

type ListMword struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (m *ListMword) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return m.handleMwList(cfg)
}

func (m *ListMword) handleMwList(cfg *config.Config) *ports.AnswerType {
	if len(cfg.Mword) == 0 {
		return &ports.AnswerType{
			Text:    []string{"мворды не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for word, mw := range cfg.Mword {
		parts = append(parts, fmt.Sprintf("- %s (наказания: %s)", word, m.template.FormatPunishments(mw.Punishments)))
	}
	msg := "мворды: \n" + strings.Join(parts, "\n")

	key, err := m.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{m.fs.GetURL(key)},
		IsReply: true,
	}
}
