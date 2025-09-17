package admin

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"slices"
	"strings"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

var NotFoundMwordGroup = &ports.AnswerType{
	Text:    []string{"мворд группа не найдена!"},
	IsReply: true,
}

type ListMwordGroup struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (m *ListMwordGroup) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return m.handleMwgList(cfg)
}

func (m *ListMwordGroup) handleMwgList(cfg *config.Config) *ports.AnswerType {
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

		parts = append(parts, fmt.Sprintf("- %s (enabled: %v, punishments: (%s), words: %s, regexp: %s)",
			name, mwg.Enabled, m.template.FormatPunishments(mwg.Punishments), strings.Join(mwg.Words, ", "), strings.Join(re, ", ")))
	}
	msg := "мворд группы: \n" + strings.Join(parts, "\n")

	key, err := m.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{m.fs.GetURL(key)},
		IsReply: true,
	}
}

type CreateMwordGroup struct {
	template ports.TemplatePort
}

func (m *CreateMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgCreate(cfg, text)
}

func (m *CreateMwordGroup) handleMwgCreate(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := m.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

	if len(words) < 5 { // !am mwg create <название_группы> <наказания через запятую>
		return NonParametr
	}

	groupName := words[3]
	if _, exists := cfg.MwordGroup[groupName]; exists {
		return &ports.AnswerType{
			Text:    []string{"мворд группа уже существует!"},
			IsReply: true,
		}
	}

	var punishments []config.Punishment
	for _, pa := range strings.Split(text.Tail(4), ",") {
		p, err := m.template.ParsePunishment(pa, false)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
				IsReply: true,
			}
		}
		punishments = append(punishments, p)
	}

	mwg := cfg.MwordGroup[groupName]
	mwg = &config.MwordGroup{
		Enabled:     true,
		Punishments: punishments,
		Options:     mergeSpamOptions(mwg.Options, opts),
	}

	m.template.UpdateMwords(cfg.MwordGroup, cfg.Mword)
	return nil
}

type SetMwordGroup struct {
	template ports.TemplatePort
}

func (m *SetMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgSet(cfg, text)
}

func (m *SetMwordGroup) handleMwgSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := m.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

	if len(words) < 3 { // !am mwg set <название_группы> <наказания через запятую ИЛИ опции>
		return NonParametr
	}

	mwg, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}
	mwg.Options = mergeSpamOptions(mwg.Options, opts)

	if len(words) >= 4 {
		var punishments []config.Punishment
		punishmentsArgs := strings.Split(strings.Join(words[4:], " "), ",")
		for _, pa := range punishmentsArgs {
			p, err := m.template.ParsePunishment(pa, false)
			if err != nil {
				return &ports.AnswerType{
					Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
					IsReply: true,
				}
			}
			punishments = append(punishments, p)
		}
		mwg.Punishments = punishments
	}

	m.template.UpdateMwords(cfg.MwordGroup, cfg.Mword)
	return nil
}

type AddMwordGroup struct {
	template ports.TemplatePort
}

func (m *AddMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgAdd(cfg, text)
}

func (m *AddMwordGroup) handleMwgAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := m.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

	if len(words) < 5 { // !am mwg add <название_группы> <слова/фразы через запятую>
		return NonParametr
	}

	group, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}

	joined := strings.Join(words[4:], " ")
	if _, ok := opts["-regex"]; ok {
		re, err := regexp2.Compile(joined, regexp2.None)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		group.Regexp = append(group.Regexp, re)
		return nil
	}

	for _, word := range strings.Split(joined, ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if !slices.Contains(group.Words, word) {
			group.Words = append(group.Words, word)
		}
	}

	m.template.UpdateMwords(cfg.MwordGroup, cfg.Mword)
	return nil
}

type DelMwordGroup struct {
	template ports.TemplatePort
}

func (m *DelMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgDel(cfg, text)
}

func (m *DelMwordGroup) handleMwgDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 5 { // !am mwg del <название_группы> <слова/фразы через запятую или all>
		return NonParametr
	}

	group, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}

	if words[4] == "all" {
		delete(cfg.MwordGroup, words[3])
		return nil
	}

	var removed, notFound []string
	newSlice := group.Regexp[:0]
	for _, r := range group.Regexp {
		if r.String() != text.Tail(4) {
			newSlice = append(newSlice, r)
		} else {
			removed = append(removed, text.Tail(4))
		}
	}
	group.Regexp = newSlice

	args := words[4:]
	argsSet := make(map[string]struct{}, len(args))
	for _, a := range args {
		argsSet[a] = struct{}{}
	}

	newWords := group.Words[:0]
	for _, w := range group.Words {
		if _, ok := argsSet[w]; ok {
			removed = append(removed, w)
		} else {
			newWords = append(newWords, w)
		}
	}
	group.Words = newWords

	m.template.UpdateMwords(cfg.MwordGroup, cfg.Mword)
	return buildResponse(removed, "удалены", notFound, "не найдены", "слова в мворд группе не указаны")
}

type OnOffMwordGroup struct {
	template ports.TemplatePort
}

func (m *OnOffMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgOnOff(cfg, text)
}

func (m *OnOffMwordGroup) handleMwgOnOff(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am mwg on/off <название_группы>
		return NonParametr
	}

	mwg, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}
	mwg.Enabled = words[2] == "on"

	m.template.UpdateMwords(cfg.MwordGroup, cfg.Mword)
	return nil
}
