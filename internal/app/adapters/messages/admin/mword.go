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

var (
	notFoundMwordGroup = &ports.AnswerType{
		Text:    []string{"мворд группа не найдена!"},
		IsReply: true,
	}
	existsMwordGroup = &ports.AnswerType{
		Text:    []string{"мворд группа уже существует!"},
		IsReply: true,
	}
)

// Одиночные мутворды

type AddMword struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *AddMword) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwAdd(cfg, text)
}

func (m *AddMword) handleMwAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Original, template.MwordOptions)

	// !am mw (add) <наказания через запятую> <слова/фразы через запятую>
	// или !am mw (add) <наказания через запятую> re <name> <regex>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 6 {
		return nonParametr
	}

	var punishments []config.Punishment
	for _, pa := range strings.Split(strings.TrimSpace(matches[1]), ",") {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

		p, err := m.template.Punishment().Parse(pa, false)
		if err != nil {
			return errorPunishmentParse
		}
		punishments = append(punishments, p)
	}

	if len(punishments) == 0 {
		return invalidPunishmentFormat
	}

	if strings.ToLower(strings.TrimSpace(matches[2])) == "re" {
		name, reStr := strings.TrimSpace(matches[3]), strings.TrimSpace(matches[4])

		re, err := regexp.Compile(reStr)
		if err != nil {
			return invalidRegex
		}

		cfg.Mword[name] = &config.Mword{
			Punishments: punishments,
			Regexp:      re,
			Options:     m.template.Options().MergeMword(config.MwordOptions{}, opts),
		}

		return success
	}

	var added, exists []string
	for _, word := range strings.Split(strings.TrimSpace(matches[5]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if _, ok := cfg.Mword[word]; ok {
			exists = append(exists, word)
			continue
		}

		cfg.Mword[word] = &config.Mword{
			Punishments: punishments,
			Options:     m.template.Options().MergeMword(config.MwordOptions{}, opts),
		}
		added = append(added, word)
	}

	return buildResponse("мворды не указаны", RespArg{Items: added, Name: "добавлены"}, RespArg{Items: exists, Name: "уже существуют"})
}

type SetMword struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *SetMword) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwSet(cfg, text)
}

func (m *SetMword) handleMwSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Original, template.MwordOptions)

	// или !am mw set <наказания через запятую> <слова или фразы через запятую>
	// или !am mw set <слова или фразы через запятую>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 3 {
		return nonParametr
	}

	var punishments []config.Punishment
	for _, pa := range strings.Split(strings.TrimSpace(matches[1]), ",") {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

		p, err := m.template.Punishment().Parse(pa, false)
		if err != nil {
			return errorPunishmentParse
		}
		punishments = append(punishments, p)
	}

	var edited, notFound []string
	for _, word := range strings.Split(strings.TrimSpace(matches[2]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if _, ok := cfg.Mword[word]; !ok {
			notFound = append(notFound, word)
			continue
		}

		if len(punishments) != 0 {
			cfg.Mword[word].Punishments = punishments
		}
		cfg.Mword[word].Options = m.template.Options().MergeMword(cfg.Mword[word].Options, opts)

		edited = append(edited, word)
	}

	return buildResponse("мворды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type DelMword struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *DelMword) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwDel(cfg, text)
}

func (m *DelMword) handleMwDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := m.re.FindStringSubmatch(text.Original) // !am mw del <слова/фразы через запятую или regex>
	if len(matches) != 2 {
		return nonParametr
	}

	var removed, notFound []string
	for _, word := range strings.Split(strings.TrimSpace(matches[1]), ",") {
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

	return buildResponse("мворды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type ListMword struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (m *ListMword) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return m.handleMwList(cfg)
}

func (m *ListMword) handleMwList(cfg *config.Config) *ports.AnswerType {
	return buildList(cfg.Mword, "мворды", "мворды не найдены!",
		func(word string, mw *config.Mword) string {
			if mw.Regexp != nil {
				return fmt.Sprintf("- `%s` (название мворда: %s, наказания: %s)",
					mw.Regexp.String(), word, strings.Join(m.template.Punishment().FormatAll(mw.Punishments), ", "))
			}

			return fmt.Sprintf("- %s (наказания: %s)",
				word, strings.Join(m.template.Punishment().FormatAll(mw.Punishments), ", "))
		}, m.fs)
}

// Группы мутвордов

type CreateMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *CreateMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgCreate(cfg, text)
}

func (m *CreateMwordGroup) handleMwgCreate(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Original, template.MwordOptions)

	matches := m.re.FindStringSubmatch(textWithoutOpts) // !am mwg create <название_группы> <наказания через запятую>
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.MwordGroup[groupName]; exists {
		return existsMwordGroup
	}

	var punishments []config.Punishment
	for _, pa := range strings.Split(strings.TrimSpace(matches[2]), ",") {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

		p, err := m.template.Punishment().Parse(pa, false)
		if err != nil {
			return errorPunishmentParse
		}
		punishments = append(punishments, p)
	}

	if len(punishments) == 0 {
		return invalidPunishmentFormat
	}

	cfg.MwordGroup[groupName] = &config.MwordGroup{
		Enabled:     true,
		Punishments: punishments,
		Options:     m.template.Options().MergeMword(config.MwordOptions{}, opts),
		Regexp:      make(map[string]*regexp.Regexp),
	}
	return success
}

type AddMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *AddMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgAdd(cfg, text)
}

func (m *AddMwordGroup) handleMwgAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	// !am mwg add <название_группы> <слова/фразы через запятую>
	// или !am mwg add <название_группы> re <name> <regex>
	matches := m.re.FindStringSubmatch(text.Original)
	if len(matches) != 6 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, ok := cfg.MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	if strings.ToLower(strings.TrimSpace(matches[2])) == "re" {
		if len(matches) < 5 {
			return nonParametr
		}
		name, reStr := strings.TrimSpace(matches[3]), strings.TrimSpace(matches[4])

		re, err := regexp.Compile(reStr)
		if err != nil {
			return invalidRegex
		}

		cfg.MwordGroup[groupName].Regexp[name] = re
		return success
	}

	var added, exists []string
	for _, word := range strings.Split(strings.TrimSpace(matches[5]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if slices.Contains(cfg.MwordGroup[groupName].Words, word) {
			exists = append(exists, word)
			continue
		}

		cfg.MwordGroup[groupName].Words = append(cfg.MwordGroup[groupName].Words, word)
		added = append(added, word)
	}

	return buildResponse("мворды не указаны", RespArg{Items: added, Name: "добавлены"}, RespArg{Items: exists, Name: "уже существуют"})
}

type SetMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *SetMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgSet(cfg, text)
}

func (m *SetMwordGroup) handleMwgSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Original, template.MwordOptions)

	// !am mwg set <название_группы> <наказания через запятую>
	// или !am mwg set <название_группы>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return notFoundMwordGroup
	}
	cfg.MwordGroup[groupName].Options = m.template.Options().MergeMword(cfg.MwordGroup[groupName].Options, opts)

	var punishments []config.Punishment
	for _, pa := range strings.Split(strings.TrimSpace(matches[2]), ",") {
		pa = strings.TrimSpace(pa)
		if pa == "" {
			continue
		}

		p, err := m.template.Punishment().Parse(pa, false)
		if err != nil {
			return errorPunishmentParse
		}
		punishments = append(punishments, p)
	}

	if len(punishments) != 0 {
		cfg.MwordGroup[groupName].Punishments = punishments
	}
	return success
}

type DelMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *DelMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgDel(cfg, text)
}

func (m *DelMwordGroup) handleMwgDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	// !am mwg del <название_группы> <слова/фразы через запятую или ничего для all>
	matches := m.re.FindStringSubmatch(text.Original)
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, ok := cfg.MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	if strings.TrimSpace(matches[2]) == "" {
		delete(cfg.MwordGroup, groupName)
		return success
	}

	var removed, notFound []string
	for _, word := range strings.Split(strings.TrimSpace(matches[2]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		isFound := false
		if _, ok := cfg.MwordGroup[groupName].Regexp[word]; ok {
			delete(cfg.MwordGroup[groupName].Regexp, word)
			removed = append(removed, word)
			isFound = true
		}

		for i, w := range cfg.MwordGroup[groupName].Words {
			if w == word {
				cfg.MwordGroup[groupName].Words = append(cfg.MwordGroup[groupName].Words[:i], cfg.MwordGroup[groupName].Words[i+1:]...)
				removed = append(removed, word)
				isFound = true
				break
			}
		}

		if !isFound {
			notFound = append(notFound, word)
		}
	}

	return buildResponse("мворды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type OnOffMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *OnOffMwordGroup) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return m.handleMwgOnOff(cfg, text)
}

func (m *OnOffMwordGroup) handleMwgOnOff(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := m.re.FindStringSubmatch(text.Original) // !am mwg on/off <название_группы>
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[2])
	state := strings.ToLower(strings.TrimSpace(matches[1]))

	if _, ok := cfg.MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	cfg.MwordGroup[groupName].Enabled = state == "on"
	return success
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

			if len(re) == 0 {
				re = append(re, "отсутствуют")
			}

			words := mwg.Words
			if len(words) == 0 {
				words = append(words, "отсутствуют")
			}

			return fmt.Sprintf("%s\n- включено: %v\n- наказания: %s\n- слова: %s\n- регулярные выражения: %s\n\n",
				name,
				mwg.Enabled,
				strings.Join(m.template.Punishment().FormatAll(mwg.Punishments), ", "),
				strings.Join(words, ", "),
				strings.Join(re, ", "),
			)
		}, m.fs)
}
