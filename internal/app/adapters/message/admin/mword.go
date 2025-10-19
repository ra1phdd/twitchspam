package admin

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"twitchspam/internal/app/domain"
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

func (m *AddMword) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwAdd(cfg, text)
}

func (m *AddMword) handleMwAdd(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Text(), template.MwordOptions)

	// !am mw (add) <наказания через запятую> <слова/фразы через запятую>
	// или !am mw (add) <наказания через запятую> re <name> <regex>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 6 {
		return nonParametr
	}

	parts := strings.Split(strings.TrimSpace(matches[1]), ",")
	punishments := make([]config.Punishment, 0, len(parts))

	for _, pa := range parts {
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

		cfg.Mword = append(cfg.Mword, config.Mword{
			Punishments: punishments,
			Options:     m.template.Options().MergeMword(config.MwordOptions{}, opts),
			NameRegexp:  name,
			Regexp:      re,
		})

		m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
		return success
	}

	words := strings.Split(strings.TrimSpace(matches[5]), ",")
	added := make([]string, 0, len(words))
	exists := make([]string, 0, len(words))

	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if slices.ContainsFunc(cfg.Mword, func(w config.Mword) bool { return w.Word == word }) {
			exists = append(exists, word)
			continue
		}

		cfg.Mword = append(cfg.Mword, config.Mword{
			Punishments: punishments,
			Options:     m.template.Options().MergeMword(config.MwordOptions{}, opts),
			Word:        word,
		})
		added = append(added, word)
	}

	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: added, Name: "добавлены"}, RespArg{Items: exists, Name: "уже существуют"})
}

type SetMword struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *SetMword) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwSet(cfg, text)
}

func (m *SetMword) handleMwSet(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Text(), template.MwordOptions)

	// или !am mw set <наказания через запятую> <слова или фразы через запятую>
	// или !am mw set <слова или фразы через запятую>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 3 {
		return nonParametr
	}

	var punishments []config.Punishment
	if strings.TrimSpace(matches[1]) != "" {
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
	}

	var edited, notFound []string
	for _, word := range strings.Split(strings.TrimSpace(matches[2]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		i := slices.IndexFunc(cfg.Mword, func(w config.Mword) bool { return w.Word == word })
		if i == -1 {
			notFound = append(notFound, word)
			continue
		}

		if len(punishments) != 0 {
			cfg.Mword[i].Punishments = punishments
		}
		cfg.Mword[i].Options = m.template.Options().MergeMword(cfg.Mword[i].Options, opts)

		edited = append(edited, word)
	}

	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type DelMword struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *DelMword) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwDel(cfg, text)
}

func (m *DelMword) handleMwDel(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := m.re.FindStringSubmatch(text.Text()) // !am mw del <слова/фразы через запятую или regex>
	if len(matches) != 2 {
		return nonParametr
	}

	var removed, notFound []string
	for _, word := range strings.Split(strings.TrimSpace(matches[1]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		index := slices.IndexFunc(cfg.Mword, func(w config.Mword) bool { return w.Word == word })
		if index == -1 {
			notFound = append(notFound, word)
			continue
		}

		removed = append(removed, word)
		cfg.Mword = slices.Delete(cfg.Mword, index, index+1)
	}

	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type ListMword struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (m *ListMword) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return m.handleMwList(cfg)
}

func (m *ListMword) handleMwList(cfg *config.Config) *ports.AnswerType {
	mwords := make(map[string]config.Mword)
	for i, mw := range cfg.Mword {
		mwords[strconv.Itoa(i)] = mw
	}

	return buildList(mwords, "мворды", "мворды не найдены!",
		func(_ string, mw config.Mword) string {
			if mw.Regexp != nil {
				return fmt.Sprintf("- `%s` (название мворда: %s, наказания: %s)",
					mw.Regexp.String(), mw.NameRegexp, strings.Join(m.template.Punishment().FormatAll(mw.Punishments), ", "))
			}

			return fmt.Sprintf("- %s (наказания: %s)",
				mw.Word, strings.Join(m.template.Punishment().FormatAll(mw.Punishments), ", "))
		}, m.fs)
}

// Группы мутвордов

type CreateMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *CreateMwordGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgCreate(cfg, text)
}

func (m *CreateMwordGroup) handleMwgCreate(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Text(), template.MwordOptions)

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
	}

	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return success
}

type AddMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *AddMwordGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgAdd(cfg, text)
}

func (m *AddMwordGroup) handleMwgAdd(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	// !am mwg add <название_группы> <слова/фразы через запятую>
	// или !am mwg add <название_группы> re <name> <regex>
	matches := m.re.FindStringSubmatch(text.Text())
	if len(matches) != 6 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, ok := cfg.MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	if strings.ToLower(strings.TrimSpace(matches[3])) == "re" {
		name, reStr := strings.TrimSpace(matches[4]), strings.TrimSpace(matches[5])

		re, err := regexp.Compile(reStr)
		if err != nil {
			return invalidRegex
		}

		cfg.MwordGroup[groupName].Words = append(cfg.MwordGroup[groupName].Words, config.Mword{
			NameRegexp: name,
			Regexp:     re,
		})
		m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
		return success
	}

	var added, exists []string
	for _, word := range strings.Split(strings.TrimSpace(matches[6]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if slices.ContainsFunc(cfg.Mword, func(w config.Mword) bool { return w.Word == word }) {
			exists = append(exists, word)
			continue
		}

		cfg.MwordGroup[groupName].Words = append(cfg.MwordGroup[groupName].Words, config.Mword{
			Word: word,
		})
		added = append(added, word)
	}

	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: added, Name: "добавлены"}, RespArg{Items: exists, Name: "уже существуют"})
}

type GlobalSetMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *GlobalSetMwordGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgGlobalSet(cfg, text)
}

func (m *GlobalSetMwordGroup) handleMwgGlobalSet(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Text(), template.MwordOptions)

	// !am mwg set <название_группы> <*наказания через запятую> <*опции>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return notFoundMwordGroup
	}

	var punishments []config.Punishment
	if strings.TrimSpace(matches[2]) != "" {
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
	}

	cfg.MwordGroup[groupName].Options = m.template.Options().MergeMword(cfg.MwordGroup[groupName].Options, opts)
	if len(punishments) != 0 {
		cfg.MwordGroup[groupName].Punishments = punishments
	}

	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return success
}

type SetMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *SetMwordGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgSet(cfg, text)
}

func (m *SetMwordGroup) handleMwgSet(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Text(), template.MwordOptions)

	// !am mwg set <название_группы> <*наказания через запятую> <слова/фразы или regex> <*опции>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 4 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return notFoundMwordGroup
	}

	var punishments []config.Punishment
	if strings.TrimSpace(matches[2]) != "" {
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
	}

	var edited, notFound []string
	for _, word := range strings.Split(strings.TrimSpace(matches[3]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		i := slices.IndexFunc(cfg.MwordGroup[groupName].Words, func(w config.Mword) bool { return w.Word == word })
		if i == -1 {
			notFound = append(notFound, word)
			continue
		}

		if len(punishments) != 0 {
			cfg.MwordGroup[groupName].Words[i].Punishments = punishments
		}
		cfg.MwordGroup[groupName].Words[i].Options = m.template.Options().MergeMword(cfg.MwordGroup[groupName].Words[i].Options, opts)

		edited = append(edited, word)
	}

	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type DelMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *DelMwordGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgDel(cfg, text)
}

func (m *DelMwordGroup) handleMwgDel(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	// !am mwg del <название_группы> <слова/фразы через запятую или ничего для all>
	matches := m.re.FindStringSubmatch(text.Text())
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, ok := cfg.MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	if strings.TrimSpace(matches[2]) == "" {
		delete(cfg.MwordGroup, groupName)

		m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
		return success
	}

	var removed, notFound []string
	for _, word := range strings.Split(strings.TrimSpace(matches[2]), ",") {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		index := slices.IndexFunc(cfg.MwordGroup[groupName].Words, func(w config.Mword) bool { return w.Word == word || w.NameRegexp == word })
		if index == -1 {
			notFound = append(notFound, word)
			continue
		}

		removed = append(removed, word)
		cfg.MwordGroup[groupName].Words = slices.Delete(cfg.MwordGroup[groupName].Words, index, index+1)
	}

	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type OnOffMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *OnOffMwordGroup) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgOnOff(cfg, text)
}

func (m *OnOffMwordGroup) handleMwgOnOff(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := m.re.FindStringSubmatch(text.Text()) // !am mwg on/off <название_группы>
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[2])
	state := strings.ToLower(strings.TrimSpace(matches[1]))

	if _, ok := cfg.MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	cfg.MwordGroup[groupName].Enabled = state == "on"
	m.template.Mword().Update(cfg.Mword, cfg.MwordGroup)
	return success
}

type ListMwordGroup struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (m *ListMwordGroup) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return m.handleMwgList(cfg)
}

func (m *ListMwordGroup) handleMwgList(cfg *config.Config) *ports.AnswerType {
	return buildList(cfg.MwordGroup, "мворд группы", "мворд группы не найдены!",
		func(name string, mwg *config.MwordGroup) string {
			var sb strings.Builder
			sb.WriteString(name + ":\n")
			sb.WriteString(fmt.Sprintf("- глобальные наказания: %v\n", strings.Join(m.template.Punishment().FormatAll(mwg.Punishments), ", ")))
			sb.WriteString(fmt.Sprintf("- глобальные опции: %+v\n", m.template.Options().MwordToString(mwg.Options)))

			sb.WriteString("- мворды:\n")
			for _, mw := range mwg.Words {
				punishments := "стандартные"
				if len(mw.Punishments) > 0 {
					punishments = fmt.Sprintf("%v", mw.Punishments)
				}

				options := "стандартные"
				if mw.Options != (config.MwordOptions{}) {
					options = m.template.Options().MwordToString(mw.Options)
				}

				if mw.Regexp != nil {
					sb.WriteString(fmt.Sprintf("  - %s (название: %s, наказания: %s, опции: %s)\n",
						mw.Regexp.String(), mw.NameRegexp, punishments, options))
					continue
				}

				sb.WriteString(fmt.Sprintf("  - %s (наказания: %s, опции: %s)\n", mw.Word, punishments, options))
			}

			sb.WriteString("\n")
			return sb.String()
		}, m.fs)
}
