package admin

import (
	"fmt"
	"regexp"
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
	return buildList(cfg.MwordGroup, "мворд группы", "мворд группы не найдены!",
		func(name string, mwg *config.MwordGroup) string {
			var re []string
			for _, pattern := range mwg.Regexp {
				re = append(re, pattern.String())
			}
			return fmt.Sprintf("- %s (enabled: %v, punishments: (%s), words: %s, regexp: %s)",
				name,
				mwg.Enabled,
				m.template.Punishment().FormatAll(mwg.Punishments),
				strings.Join(mwg.Words, ", "),
				strings.Join(re, ", "),
			)
		}, m.fs)
}

type CreateMwordGroup struct {
	template ports.TemplatePort
}

func (m *CreateMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgCreate(cfg, text)
}

func (m *CreateMwordGroup) handleMwgCreate(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := m.template.Options().ParseAll(&words, template.MwordOptions) // ParseOptions удаляет опции из слайса words

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
		p, err := m.template.Punishment().Parse(pa, false)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
				IsReply: true,
			}
		}
		punishments = append(punishments, p)
	}

	optsMerged := m.template.Options().MergeMword(config.MwordOptions{}, opts)
	if mwg, ok := cfg.MwordGroup[groupName]; ok {
		optsMerged = m.template.Options().MergeMword(mwg.Options, opts)
	}

	cfg.MwordGroup[groupName] = &config.MwordGroup{
		Enabled:     true,
		Punishments: punishments,
		Options:     optsMerged,
	}
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
	opts := m.template.Options().ParseAll(&words, template.MwordOptions) // ParseOptions удаляет опции из слайса words

	if len(words) < 3 { // !am mwg set <название_группы> <наказания через запятую ИЛИ опции>
		return NonParametr
	}

	mwg, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}
	cfg.MwordGroup[words[3]].Options = m.template.Options().MergeMword(mwg.Options, opts)

	if len(words) >= 4 {
		var punishments []config.Punishment
		punishmentsArgs := strings.Split(strings.Join(words[4:], " "), ",")
		for _, pa := range punishmentsArgs {
			p, err := m.template.Punishment().Parse(pa, false)
			if err != nil {
				return &ports.AnswerType{
					Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", pa)},
					IsReply: true,
				}
			}
			punishments = append(punishments, p)
		}
		cfg.MwordGroup[words[3]].Punishments = punishments
	}
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
	opts := m.template.Options().ParseAll(&words, template.MwordOptions) // ParseOptions удаляет опции из слайса words

	if len(words) < 5 { // !am mwg add <название_группы> <слова/фразы через запятую>
		return NonParametr
	}

	group, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}

	joined := strings.Join(words[4:], " ")
	if _, ok := opts["-regex"]; ok {
		re, err := regexp.Compile(joined)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"неверное регулярное выражение!"},
				IsReply: true,
			}
		}

		cfg.MwordGroup[words[3]].Regexp = append(group.Regexp, re)
		return nil
	}

	for _, word := range strings.Split(joined, ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if !slices.Contains(group.Words, word) {
			cfg.MwordGroup[words[3]].Words = append(group.Words, word)
		}
	}
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
	cfg.MwordGroup[words[3]].Regexp = newSlice

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
	cfg.MwordGroup[words[3]].Words = newWords

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

	_, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}
	cfg.MwordGroup[words[3]].Enabled = words[2] == "on"

	return nil
}
