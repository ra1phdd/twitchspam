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

// Одиночные мутворды

type AddMword struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *AddMword) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwAdd(cfg, channel, text)
}

func (m *AddMword) handleMwAdd(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
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

		cfg.Channels[channel].Mword = append(cfg.Channels[channel].Mword, config.Mword{
			Punishments: punishments,
			Options:     m.template.Options().MergeMword(nil, opts),
			NameRegexp:  name,
			Regexp:      re,
		})

		m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
		return success
	}

	words := strings.Split(strings.TrimSpace(matches[5]), ",")
	added, exists := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if slices.ContainsFunc(cfg.Channels[channel].Mword, func(w config.Mword) bool { return w.Word == word }) {
			exists = append(exists, word)
			continue
		}

		cfg.Channels[channel].Mword = append(cfg.Channels[channel].Mword, config.Mword{
			Punishments: punishments,
			Options:     m.template.Options().MergeMword(nil, opts),
			Word:        word,
		})
		added = append(added, word)
	}

	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: added, Name: "добавлены"}, RespArg{Items: exists, Name: "уже существуют"})
}

type SetMword struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *SetMword) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwSet(cfg, channel, text)
}

func (m *SetMword) handleMwSet(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
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

	words := strings.Split(strings.TrimSpace(matches[2]), ",")
	edited, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		index := slices.IndexFunc(cfg.Channels[channel].Mword, func(w config.Mword) bool { return w.Word == word })
		if index == -1 {
			notFound = append(notFound, word)
			continue
		}

		if len(punishments) != 0 {
			cfg.Channels[channel].Mword[index].Punishments = punishments
		}
		cfg.Channels[channel].Mword[index].Options = m.template.Options().MergeMword(cfg.Channels[channel].Mword[index].Options, opts)

		edited = append(edited, word)
	}

	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type DelMword struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *DelMword) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwDel(cfg, channel, text)
}

func (m *DelMword) handleMwDel(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	matches := m.re.FindStringSubmatch(text.Text()) // !am mw del <слова/фразы через запятую или regex>
	if len(matches) != 2 {
		return nonParametr
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	removed, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		index := slices.IndexFunc(cfg.Channels[channel].Mword, func(w config.Mword) bool { return w.Word == word })
		if index == -1 {
			notFound = append(notFound, word)
			continue
		}

		cfg.Channels[channel].Mword = slices.Delete(cfg.Channels[channel].Mword, index, index+1)
		removed = append(removed, word)
	}

	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type ListMword struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (m *ListMword) Execute(cfg *config.Config, channel string, _ *domain.MessageText) *ports.AnswerType {
	return m.handleMwList(cfg, channel)
}

func (m *ListMword) handleMwList(cfg *config.Config, channel string) *ports.AnswerType {
	mwords := make(map[string]config.Mword, len(cfg.Channels[channel].Mword))
	for i, mw := range cfg.Channels[channel].Mword {
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

func (m *CreateMwordGroup) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgCreate(cfg, channel, text)
}

func (m *CreateMwordGroup) handleMwgCreate(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Text(), template.MwordOptions)

	matches := m.re.FindStringSubmatch(textWithoutOpts) // !am mwg create <название_группы> <наказания через запятую>
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.Channels[channel].MwordGroup[groupName]; exists {
		return existsMwordGroup
	}

	parts := strings.Split(strings.TrimSpace(matches[2]), ",")
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

	cfg.Channels[channel].MwordGroup[groupName] = &config.MwordGroup{
		Enabled:     true,
		Punishments: punishments,
		Options:     m.template.Options().MergeMword(nil, opts),
	}

	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return success
}

type AddMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *AddMwordGroup) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgAdd(cfg, channel, text)
}

func (m *AddMwordGroup) handleMwgAdd(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	// !am mwg add <название_группы> <слова/фразы через запятую>
	// или !am mwg add <название_группы> re <name> <regex>
	matches := m.re.FindStringSubmatch(text.Text())
	if len(matches) != 6 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, ok := cfg.Channels[channel].MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	if strings.ToLower(strings.TrimSpace(matches[3])) == "re" {
		name, reStr := strings.TrimSpace(matches[4]), strings.TrimSpace(matches[5])

		re, err := regexp.Compile(reStr)
		if err != nil {
			return invalidRegex
		}

		cfg.Channels[channel].MwordGroup[groupName].Words = append(cfg.Channels[channel].MwordGroup[groupName].Words, config.Mword{
			NameRegexp: name,
			Regexp:     re,
		})
		m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
		return success
	}

	words := strings.Split(strings.TrimSpace(matches[6]), ",")
	added, exists := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if slices.ContainsFunc(cfg.Channels[channel].Mword, func(w config.Mword) bool { return w.Word == word }) {
			exists = append(exists, word)
			continue
		}

		cfg.Channels[channel].MwordGroup[groupName].Words = append(cfg.Channels[channel].MwordGroup[groupName].Words, config.Mword{
			Word: word,
		})
		added = append(added, word)
	}

	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: added, Name: "добавлены"}, RespArg{Items: exists, Name: "уже существуют"})
}

type GlobalSetMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *GlobalSetMwordGroup) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgGlobalSet(cfg, channel, text)
}

func (m *GlobalSetMwordGroup) handleMwgGlobalSet(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Text(), template.MwordOptions)

	// !am mwg glset <название_группы> <*наказания через запятую> <*опции>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.Channels[channel].MwordGroup[groupName]; !exists {
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

	cfg.Channels[channel].MwordGroup[groupName].Options = m.template.Options().MergeMword(cfg.Channels[channel].MwordGroup[groupName].Options, opts)
	if len(punishments) != 0 {
		cfg.Channels[channel].MwordGroup[groupName].Punishments = punishments
	}

	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return success
}

type SetMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *SetMwordGroup) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgSet(cfg, channel, text)
}

func (m *SetMwordGroup) handleMwgSet(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := m.template.Options().ParseAll(text.Text(), template.MwordOptions)

	// !am mwg set <название_группы> <*наказания через запятую> <слова/фразы или regex> <*опции>
	matches := m.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 4 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, exists := cfg.Channels[channel].MwordGroup[groupName]; !exists {
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

	words := strings.Split(strings.TrimSpace(matches[3]), ",")
	edited, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		index := slices.IndexFunc(cfg.Channels[channel].MwordGroup[groupName].Words, func(w config.Mword) bool { return w.Word == word })
		if index == -1 {
			notFound = append(notFound, word)
			continue
		}

		if len(punishments) != 0 {
			cfg.Channels[channel].MwordGroup[groupName].Words[index].Punishments = punishments
		}
		cfg.Channels[channel].MwordGroup[groupName].Words[index].Options = m.template.Options().MergeMword(cfg.Channels[channel].MwordGroup[groupName].Words[index].Options, opts)

		edited = append(edited, word)
	}

	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type DelMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *DelMwordGroup) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgDel(cfg, channel, text)
}

func (m *DelMwordGroup) handleMwgDel(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	// !am mwg del <название_группы> <слова/фразы через запятую или ничего для all>
	matches := m.re.FindStringSubmatch(text.Text())
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[1])
	if _, ok := cfg.Channels[channel].MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	if strings.TrimSpace(matches[2]) == "" {
		delete(cfg.Channels[channel].MwordGroup, groupName)

		m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
		return success
	}

	words := strings.Split(strings.TrimSpace(matches[2]), ",")
	removed, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		index := slices.IndexFunc(cfg.Channels[channel].MwordGroup[groupName].Words, func(w config.Mword) bool { return w.Word == word || w.NameRegexp == word })
		if index == -1 {
			notFound = append(notFound, word)
			continue
		}

		removed = append(removed, word)
		cfg.Channels[channel].MwordGroup[groupName].Words = slices.Delete(cfg.Channels[channel].MwordGroup[groupName].Words, index, index+1)
	}

	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return buildResponse("мворды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type OnOffMwordGroup struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (m *OnOffMwordGroup) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return m.handleMwgOnOff(cfg, channel, text)
}

func (m *OnOffMwordGroup) handleMwgOnOff(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	matches := m.re.FindStringSubmatch(text.Text()) // !am mwg on/off <название_группы>
	if len(matches) != 3 {
		return nonParametr
	}

	groupName := strings.TrimSpace(matches[2])
	state := strings.ToLower(strings.TrimSpace(matches[1]))

	if _, ok := cfg.Channels[channel].MwordGroup[groupName]; !ok {
		return notFoundMwordGroup
	}

	cfg.Channels[channel].MwordGroup[groupName].Enabled = state == "on"
	m.template.Mword().Update(cfg.Channels[channel].Mword, cfg.Channels[channel].MwordGroup)
	return success
}

type ListMwordGroup struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (m *ListMwordGroup) Execute(cfg *config.Config, channel string, _ *domain.MessageText) *ports.AnswerType {
	return m.handleMwgList(cfg, channel)
}

func (m *ListMwordGroup) handleMwgList(cfg *config.Config, channel string) *ports.AnswerType {
	return buildList(cfg.Channels[channel].MwordGroup, "мворд группы", "мворд группы не найдены!",
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
				if mw.Options != (nil) {
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
