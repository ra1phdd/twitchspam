package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type OnOffAntispam struct {
	enabled  bool
	typeSpam string
}

func (a *OnOffAntispam) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return a.handleAntiSpamOnOff(cfg)
}

func (a *OnOffAntispam) handleAntiSpamOnOff(cfg *config.Config) *ports.AnswerType {
	targetMap := map[string]*bool{
		"vip":     &cfg.Spam.SettingsVIP.Enabled,
		"emote":   &cfg.Spam.SettingsEmotes.Enabled,
		"default": &cfg.Spam.SettingsDefault.Enabled,
	}
	if target, ok := targetMap[a.typeSpam]; ok {
		*target = a.enabled
	}

	return nil
}

type InfoAntispam struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (a *InfoAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAntiSpamInfo(cfg, text)
}

func (a *InfoAntispam) handleAntiSpamInfo(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	whitelistUsers := "не найдены"
	if len(cfg.Spam.WhitelistUsers) > 0 {
		var sb strings.Builder
		first := true
		for user := range cfg.Spam.WhitelistUsers {
			if !first {
				sb.WriteString(", ")
			}
			sb.WriteString(user)
			first = false
		}
		whitelistUsers = sb.String()
	}

	exceptions := "не найдены"
	if len(cfg.Spam.Exceptions) > 0 {
		var sb strings.Builder
		for word, ex := range cfg.Spam.Exceptions {
			sb.WriteString(fmt.Sprintf("  - %s (лимит сообщений: %d, наказания: %s)\n", word, ex.MessageLimit, strings.Join(a.template.FormatPunishments(ex.Punishments), ", ")))
		}
		exceptions = sb.String()
	}

	parts := []string{
		"- режим: " + cfg.Spam.Mode,
		"- окно проверки сообщений: " + fmt.Sprint(cfg.Spam.CheckWindowSeconds),
		"- разрешенные пользователи: " + whitelistUsers,
		"\nобщие:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsDefault.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsDefault.SimilarityThreshold),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsDefault.MessageLimit),
		"- наказания: " + strings.Join(a.template.FormatPunishments(cfg.Spam.SettingsDefault.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsDefault.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + fmt.Sprint(cfg.Spam.SettingsDefault.MaxWordLength),
		"- наказание за превышение длины слова: " + a.template.FormatPunishment(cfg.Spam.SettingsDefault.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + fmt.Sprint(cfg.Spam.SettingsDefault.MinGapMessages),
		"\nвиперы:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsVIP.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.SimilarityThreshold),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.MessageLimit),
		"- наказания: " + strings.Join(a.template.FormatPunishments(cfg.Spam.SettingsVIP.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsVIP.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + fmt.Sprint(cfg.Spam.SettingsVIP.MaxWordLength),
		"- наказание за превышение длины слова: " + a.template.FormatPunishment(cfg.Spam.SettingsVIP.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + fmt.Sprint(cfg.Spam.SettingsVIP.MinGapMessages),
		"\nэмоуты:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsEmotes.Enabled),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsEmotes.MessageLimit),
		"- наказания: " + strings.Join(a.template.FormatPunishments(cfg.Spam.SettingsEmotes.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsEmotes.DurationResetPunishments),
		"- ограничение количества эмоутов в сообщении: " + fmt.Sprint(cfg.Spam.SettingsEmotes.MaxEmotesLength),
		"- наказание за превышение количества эмоутов в сообщении: " + a.template.FormatPunishment(cfg.Spam.SettingsEmotes.MaxEmotesPunishment),
		"\nисключения:",
		exceptions,
	}
	msg := "настройки:\n" + strings.Join(parts, "\n")

	key, err := a.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}

	return &ports.AnswerType{
		Text:    []string{a.fs.GetURL(key)},
		IsReply: true,
	}
}

type ModeAntispam struct {
	mode string
}

func (a *ModeAntispam) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return a.handleAntispamMode(cfg, a.mode)
}

func (a *ModeAntispam) handleAntispamMode(cfg *config.Config, mode string) *ports.AnswerType {
	cfg.Spam.Mode = mode
	return nil
}

type SimAntispam struct {
	template ports.TemplatePort
	typeSpam string
}

func (a *SimAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleSim(cfg, text, a.typeSpam)
}

func (a *SimAntispam) handleSim(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	words := text.Words()
	target, idx := &cfg.Spam.SettingsDefault.SimilarityThreshold, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.SimilarityThreshold, 3
	}

	if len(words) < idx+1 { // !am sim <значение> или !am vip sim <значение>
		return NonParametr
	}

	if val, ok := a.template.ParseFloatArg(words[idx], 0.1, 1); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение порога схожести сообщений должно быть от 0.1 до 1.0!"},
		IsReply: true,
	}
}

type MsgAntispam struct {
	template ports.TemplatePort
	typeSpam string
}

func (a *MsgAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleMsg(cfg, text, a.typeSpam)
}

func (a *MsgAntispam) handleMsg(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	words := text.Words()
	target, idx := &cfg.Spam.SettingsDefault.MessageLimit, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.MessageLimit, 3
	} else if typeSpam == "emote" {
		target, idx = &cfg.Spam.SettingsEmotes.MessageLimit, 3
	}

	if len(words) < idx+1 { // !am msg <значение> или !am vip/emote msg <значение>
		return NonParametr
	}

	if val, ok := a.template.ParseIntArg(words[idx], 2, 15); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение лимита сообщений должно быть от 2 до 15!"},
		IsReply: true,
	}
}

type PunishmentsAntispam struct {
	template ports.TemplatePort
	typeSpam string
}

func (a *PunishmentsAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handlePunishments(cfg, text, a.typeSpam)
}

func (a *PunishmentsAntispam) handlePunishments(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	words := text.Words()
	target, idx := &cfg.Spam.SettingsDefault.Punishments, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.Punishments, 3
	} else if typeSpam == "emote" {
		target, idx = &cfg.Spam.SettingsEmotes.Punishments, 3
	}

	if len(words) < idx+1 { // !am p <наказания через запятую> или !am vip/emote p <наказания через запятую>
		return NonParametr
	}

	var punishments []config.Punishment
	for i, str := range strings.Split(text.Tail(idx), ",") {
		if i >= 15 {
			break
		}

		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}

		allowInherit := typeSpam != "default"
		p, err := a.template.ParsePunishment(str, allowInherit)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказание (%s)!", str)},
				IsReply: true,
			}
		}

		if p.Action == "inherit" {
			if typeSpam != "default" {
				punishments = cfg.Spam.SettingsDefault.Punishments
				break
			}

			return &ports.AnswerType{
				Text:    []string{"невозможно скопировать наказания!"},
				IsReply: true,
			}
		}
		punishments = append(punishments, p)
	}

	if len(punishments) == 0 {
		return NonParametr
	}

	*target = punishments
	return nil
}

type ResetPunishmentsAntispam struct {
	template ports.TemplatePort
	typeSpam string
}

func (a *ResetPunishmentsAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleDurationResetPunishments(cfg, text, a.typeSpam)
}

func (a *ResetPunishmentsAntispam) handleDurationResetPunishments(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	words := text.Words()
	target, idx := &cfg.Spam.SettingsDefault.DurationResetPunishments, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.DurationResetPunishments, 3
	} else if typeSpam == "emote" {
		target, idx = &cfg.Spam.SettingsEmotes.DurationResetPunishments, 3
	}

	if len(words) < idx+1 { // !am rp <значение> или !am vip/emote rp <значение>
		return NonParametr
	}

	if val, ok := a.template.ParseIntArg(words[idx], 1, 86400); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение времени сброса наказаний должно быть от 1 до 86400!"},
		IsReply: true,
	}
}

type MaxLenAntispam struct {
	template ports.TemplatePort
	typeSpam string
}

func (a *MaxLenAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleMaxLen(cfg, text, a.typeSpam)
}

func (a *MaxLenAntispam) handleMaxLen(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	words := text.Words()
	params := map[string]struct {
		target *int
		idx    int
		max    int
		errMsg string
	}{
		"vip":     {&cfg.Spam.SettingsVIP.MaxWordLength, 3, 500, "значение максимальной длины слова должно быть от 0 до 500!"},
		"emote":   {&cfg.Spam.SettingsEmotes.MaxEmotesLength, 3, 30, "значение максимального количества эмоутов должно быть от 0 до 30!"},
		"default": {&cfg.Spam.SettingsDefault.MaxWordLength, 2, 500, "значение максимальной длины слова должно быть от 0 до 500!"},
	}

	if param, ok := params[typeSpam]; ok {
		if len(words) < param.idx+1 { // !am mwlen <значение> или !am vip/emote mwlen/melen <значение>
			return NonParametr
		}

		if val, ok := a.template.ParseIntArg(words[param.idx], 0, param.max); ok {
			*param.target = val
			return nil
		}

		return &ports.AnswerType{
			Text:    []string{param.errMsg},
			IsReply: true,
		}
	}

	return nil
}

type MaxPunishmentAntispam struct {
	template ports.TemplatePort
	typeSpam string
}

func (a *MaxPunishmentAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleMaxPunishment(cfg, text, a.typeSpam)
}

func (a *MaxPunishmentAntispam) handleMaxPunishment(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	words := text.Words()
	target, idx := &cfg.Spam.SettingsDefault.MaxWordPunishment, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.MaxWordPunishment, 3
	} else if typeSpam == "emote" {
		target, idx = &cfg.Spam.SettingsEmotes.MaxEmotesPunishment, 3
	}

	if len(words) < idx+1 { // !am mwp <наказание> или !am vip/emote mwp/mep <наказание>
		return NonParametr
	}

	p, err := a.template.ParsePunishment(words[idx], true)
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{fmt.Sprintf("не удалось распарсить наказание (%s)!", words[2])},
			IsReply: true,
		}
	}

	if p.Action == "inherit" {
		defaults := map[string]config.Punishment{
			"default": cfg.Spam.SettingsDefault.Punishments[0],
			"vip":     cfg.Spam.SettingsVIP.Punishments[0],
			"emote":   cfg.Spam.SettingsEmotes.Punishments[0],
		}
		if val, ok := defaults[typeSpam]; ok {
			p = val
		}
	}

	*target = p
	return nil
}

type MinGapAntispam struct {
	template ports.TemplatePort
	typeSpam string
}

func (a *MinGapAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleMinGap(cfg, text, a.typeSpam)
}

func (a *MinGapAntispam) handleMinGap(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	words := text.Words()
	target, idx := &cfg.Spam.SettingsDefault.MinGapMessages, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.MinGapMessages, 3
	}

	if len(words) < idx+1 { // !am min_gap <значение> или !am vip min_gap <значение>
		return NonParametr
	}

	if val, ok := a.template.ParseIntArg(words[idx], 0, 15); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение должно быть от 0 до 15!"},
		IsReply: true,
	}
}

type TimeAntispam struct {
	template ports.TemplatePort
}

func (a *TimeAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleTime(cfg, text)
}

func (a *TimeAntispam) handleTime(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am time <значение>
		return NonParametr
	}

	if val, ok := a.template.ParseIntArg(words[2], 1, 300); ok {
		cfg.Spam.CheckWindowSeconds = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение окна проверки сообщений должно быть от 1 до 300!"},
		IsReply: true,
	}
}

type AddAntispam struct{}

func (a *AddAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAdd(cfg, text)
}

func (a *AddAntispam) handleAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am add <пользователи через запятую>
		return NonParametr
	}

	var added, alreadyExists []string
	for _, user := range strings.Split(words[2], ",") {
		user = strings.TrimSpace(user)
		if user == "" {
			continue
		}

		if _, ok := cfg.Spam.WhitelistUsers[user]; ok {
			alreadyExists = append(alreadyExists, user)
		} else {
			cfg.Spam.WhitelistUsers[user] = struct{}{}
			added = append(added, user)
		}
	}

	return buildResponse(added, "добавлены в список", alreadyExists, "уже есть в списке", "пользователи не указаны")
}

type DelAntispam struct{}

func (a *DelAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleDel(cfg, text)
}

func (a *DelAntispam) handleDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am del <пользователи через запятую>
		return NonParametr
	}

	var removed, notFound []string
	for _, user := range strings.Split(words[2], ",") {
		user = strings.TrimSpace(user)
		if user == "" {
			continue
		}

		if _, ok := cfg.Spam.WhitelistUsers[user]; ok {
			delete(cfg.Spam.WhitelistUsers, user)
			removed = append(removed, user)
		} else {
			notFound = append(notFound, user)
		}
	}

	return buildResponse(removed, "удалены из списока", notFound, "не найдены", "пользователи не указаны")
}
