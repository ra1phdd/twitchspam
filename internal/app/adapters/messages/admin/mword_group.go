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

func (a *Admin) handleMwg(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am mwg list/create/set/...
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"list":   a.handleMwgList,
		"create": a.handleMwgCreate,
		"set":    a.handleMwgSet,
		"add":    a.handleMwgAdd,
		"del":    a.handleMwgDel,
		"on":     a.handleMwgOnOff,
		"off":    a.handleMwgOnOff,
	}

	mwgCmd := words[2]
	if handler, ok := handlers[mwgCmd]; ok {
		return handler(cfg, text)
	}
	return NotFoundCmd
}

func (a *Admin) handleMwgList(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
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
			name, mwg.Enabled, formatPunishments(mwg.Punishments), strings.Join(mwg.Words, ", "), strings.Join(re, ", ")))
	}
	msg := "мворд группы: \n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}

func (a *Admin) handleMwgCreate(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := a.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

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
		p, err := parsePunishment(pa, false)
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
		Options:     a.mergeSpamOptions(mwg.Options, opts),
	}

	return nil
}

func (a *Admin) handleMwgSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := a.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

	if len(words) < 3 { // !am mwg set <название_группы> <наказания через запятую ИЛИ опции>
		return NonParametr
	}

	mwg, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}
	mwg.Options = a.mergeSpamOptions(mwg.Options, opts)

	if len(words) >= 4 {
		var punishments []config.Punishment
		punishmentsArgs := strings.Split(strings.Join(words[4:], " "), ",")
		for _, pa := range punishmentsArgs {
			p, err := parsePunishment(pa, false)
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

	return nil
}

func (a *Admin) handleMwgAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := a.template.ParseOptions(&words, template.SpamOptions) // ParseOptions удаляет опции из слайса words

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

	return nil
}

func (a *Admin) handleMwgDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
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

	return a.buildResponse(removed, "удалены", notFound, "не найдены", "слова в мворд группе не указаны")
}

func (a *Admin) handleMwgOnOff(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am mwg on/off <название_группы>
		return NonParametr
	}

	mwg, exists := cfg.MwordGroup[words[3]]
	if !exists {
		return NotFoundMwordGroup
	}
	mwg.Enabled = words[2] == "on"

	return nil
}
