package admin

import (
	"fmt"
	"regexp"
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
		return Success
	}

	return NotFoundCmd
}

type InfoAntispam struct {
	template ports.TemplatePort
	fs       ports.FileServerPort
}

func (a *InfoAntispam) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return a.handleAntiSpamInfo(cfg)
}

func (a *InfoAntispam) handleAntiSpamInfo(cfg *config.Config) *ports.AnswerType {
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

	formatExceptions := func(exceptions map[string]*config.ExceptionsSettings) string {
		if len(exceptions) == 0 {
			return "не найдены"
		}

		var sb strings.Builder
		for word, ex := range exceptions {
			if ex.Regexp != nil {
				sb.WriteString(fmt.Sprintf("- %s (название исключения: %s, включено: %v, лимит сообщений: %d, наказания: %s, опции: %s)\n",
					ex.Regexp.String(), word, ex.Enabled, ex.MessageLimit,
					strings.Join(a.template.Punishment().FormatAll(ex.Punishments), ", "),
					a.template.Options().ExceptToString(ex.Options),
				))
				continue
			}

			sb.WriteString(fmt.Sprintf("- %s (включено: %v, лимит сообщений: %d, наказания: %s, опции: %s)\n",
				word, ex.Enabled, ex.MessageLimit,
				strings.Join(a.template.Punishment().FormatAll(ex.Punishments), ", "),
				a.template.Options().ExceptToString(ex.Options),
			))
		}
		return sb.String()
	}

	parts := []string{
		"- режим: " + cfg.Spam.Mode,
		"- окно проверки сообщений: " + fmt.Sprint(cfg.Spam.CheckWindowSeconds),
		"- разрешенные пользователи: " + whitelistUsers,
		"\nобщие:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsDefault.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsDefault.SimilarityThreshold),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsDefault.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Spam.SettingsDefault.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsDefault.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + fmt.Sprint(cfg.Spam.SettingsDefault.MaxWordLength),
		"- наказание за превышение длины слова: " + a.template.Punishment().Format(cfg.Spam.SettingsDefault.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + fmt.Sprint(cfg.Spam.SettingsDefault.MinGapMessages),
		"\nвиперы:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsVIP.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.SimilarityThreshold),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Spam.SettingsVIP.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsVIP.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + fmt.Sprint(cfg.Spam.SettingsVIP.MaxWordLength),
		"- наказание за превышение длины слова: " + a.template.Punishment().Format(cfg.Spam.SettingsVIP.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + fmt.Sprint(cfg.Spam.SettingsVIP.MinGapMessages),
		"\nэмоуты:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsEmotes.Enabled),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsEmotes.MessageLimit),
		"- наказания: " + strings.Join(a.template.Punishment().FormatAll(cfg.Spam.SettingsEmotes.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsEmotes.DurationResetPunishments),
		"- ограничение количества эмоутов в сообщении: " + fmt.Sprint(cfg.Spam.SettingsEmotes.MaxEmotesLength),
		"- наказание за превышение количества эмоутов в сообщении: " + a.template.Punishment().Format(cfg.Spam.SettingsEmotes.MaxEmotesPunishment),
		"- исключения: " + formatExceptions(cfg.Spam.SettingsEmotes.Exceptions),
		"\nисключения:",
		formatExceptions(cfg.Spam.Exceptions),
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
	return Success
}

type SimAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *SimAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleSim(cfg, text, a.typeSpam)
}

func (a *SimAntispam) handleSim(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.SimilarityThreshold
	if typeSpam == "vip" {
		target = &cfg.Spam.SettingsVIP.SimilarityThreshold
	}

	matches := a.re.FindStringSubmatch(text.Original) // !am sim <значение> или !am vip sim <значение>
	if len(matches) != 2 {
		return NonParametr
	}

	if val, ok := a.template.Parser().ParseFloatArg(strings.TrimSpace(matches[1]), 0.1, 1); ok {
		*target = val
		return Success
	}

	return &ports.AnswerType{
		Text:    []string{"значение порога схожести сообщений должно быть от 0.1 до 1.0!"},
		IsReply: true,
	}
}

type MsgAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *MsgAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleMsg(cfg, text)
}

func (a *MsgAntispam) handleMsg(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.MessageLimit
	switch a.typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MessageLimit
	case "emote":
		target = &cfg.Spam.SettingsEmotes.MessageLimit
	}

	matches := a.re.FindStringSubmatch(text.Original) // !am msg <значение> или !am vip/emote msg <значение>
	if len(matches) != 2 {
		return NonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 2, 15); ok {
		*target = val
		return Success
	}

	return &ports.AnswerType{
		Text:    []string{"значение лимита сообщений должно быть от 2 до 15!"},
		IsReply: true,
	}
}

type PunishmentsAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *PunishmentsAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handlePunishments(cfg, text)
}

func (a *PunishmentsAntispam) handlePunishments(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.Punishments
	switch a.typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.Punishments
	case "emote":
		target = &cfg.Spam.SettingsEmotes.Punishments
	}

	matches := a.re.FindStringSubmatch(text.Original) // !am p <наказания через запятую> или !am vip/emote p <наказания через запятую>
	if len(matches) != 2 {
		return NonParametr
	}

	var punishments []config.Punishment
	for i, str := range strings.Split(strings.TrimSpace(matches[1]), ",") {
		if i >= 15 {
			break
		}

		str = strings.TrimSpace(str)
		if str == "" {
			continue
		}

		allowInherit := a.typeSpam != "default"
		p, err := a.template.Punishment().Parse(str, allowInherit)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказание (%s)!", str)},
				IsReply: true,
			}
		}

		if p.Action == "inherit" {
			if a.typeSpam != "default" {
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
	return Success
}

type ResetPunishmentsAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *ResetPunishmentsAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleDurationResetPunishments(cfg, text)
}

func (a *ResetPunishmentsAntispam) handleDurationResetPunishments(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.DurationResetPunishments
	switch a.typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.DurationResetPunishments
	case "emote":
		target = &cfg.Spam.SettingsEmotes.DurationResetPunishments
	}

	matches := a.re.FindStringSubmatch(text.Original) // !am rp <значение> или !am vip/emote rp <значение>
	if len(matches) != 2 {
		return NonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 1, 86400); ok {
		*target = val
		return Success
	}

	return &ports.AnswerType{
		Text:    []string{"значение времени сброса наказаний должно быть от 1 до 86400!"},
		IsReply: true,
	}
}

type MaxLenAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *MaxLenAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleMaxLen(cfg, text)
}

func (a *MaxLenAntispam) handleMaxLen(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	params := map[string]struct {
		target *int
		max    int
		errMsg string
	}{
		"vip":     {&cfg.Spam.SettingsVIP.MaxWordLength, 500, "значение максимальной длины слова должно быть от 0 до 500!"},
		"emote":   {&cfg.Spam.SettingsEmotes.MaxEmotesLength, 30, "значение максимального количества эмоутов должно быть от 0 до 30!"},
		"default": {&cfg.Spam.SettingsDefault.MaxWordLength, 500, "значение максимальной длины слова должно быть от 0 до 500!"},
	}

	if param, ok := params[a.typeSpam]; ok {
		matches := a.re.FindStringSubmatch(text.Original) // !am mlen <значение> или !am vip/emote mlen <значение>
		if len(matches) != 2 {
			return NonParametr
		}

		if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 0, param.max); ok {
			*param.target = val
			return Success
		}

		return &ports.AnswerType{
			Text:    []string{param.errMsg},
			IsReply: true,
		}
	}

	return NotFoundCmd
}

type MaxPunishmentAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *MaxPunishmentAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleMaxPunishment(cfg, text)
}

func (a *MaxPunishmentAntispam) handleMaxPunishment(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.MaxWordPunishment
	switch a.typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MaxWordPunishment
	case "emote":
		target = &cfg.Spam.SettingsEmotes.MaxEmotesPunishment
	}

	matches := a.re.FindStringSubmatch(text.Original) // !am mp <наказание> или !am vip/emote mp <наказание>
	if len(matches) != 2 {
		return NonParametr
	}

	p, err := a.template.Punishment().Parse(strings.TrimSpace(matches[1]), true)
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{fmt.Sprintf("не удалось распарсить наказание (%s)!", matches[1])},
			IsReply: true,
		}
	}

	if p.Action == "inherit" {
		defaults := map[string]config.Punishment{
			"default": cfg.Spam.SettingsDefault.Punishments[0],
			"vip":     cfg.Spam.SettingsVIP.Punishments[0],
			"emote":   cfg.Spam.SettingsEmotes.Punishments[0],
		}
		if val, ok := defaults[a.typeSpam]; ok {
			p = val
		}
	}

	*target = p
	return Success
}

type MinGapAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	typeSpam string
}

func (a *MinGapAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleMinGap(cfg, text)
}

func (a *MinGapAntispam) handleMinGap(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	target := &cfg.Spam.SettingsDefault.MinGapMessages
	if a.typeSpam == "vip" {
		target = &cfg.Spam.SettingsVIP.MinGapMessages
	}

	matches := a.re.FindStringSubmatch(text.Original) // !am mg <значение> или !am vip mg <значение>
	if len(matches) != 2 {
		return NonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 0, 15); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение должно быть от 0 до 15!"},
		IsReply: true,
	}
}

type TimeAntispam struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *TimeAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleTime(cfg, text)
}

func (a *TimeAntispam) handleTime(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Original) // !am time <значение>
	if len(matches) != 2 {
		return NonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 1, 300); ok {
		cfg.Spam.CheckWindowSeconds = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение окна проверки сообщений должно быть от 1 до 300!"},
		IsReply: true,
	}
}

type AddAntispam struct {
	re *regexp.Regexp
}

func (a *AddAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleAdd(cfg, text)
}

func (a *AddAntispam) handleAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Original) // !am add <пользователи через запятую>
	if len(matches) != 2 {
		return NonParametr
	}

	var added, alreadyExists []string
	for _, user := range strings.Split(strings.TrimSpace(matches[1]), ",") {
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

	return buildResponse("пользователи не указаны", RespArg{Items: added, Name: "добавлены в список"}, RespArg{Items: alreadyExists, Name: "уже есть в списке"})
}

type DelAntispam struct {
	re *regexp.Regexp
}

func (a *DelAntispam) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleDel(cfg, text)
}

func (a *DelAntispam) handleDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Original) // !am del <пользователи через запятую>
	if len(matches) != 2 {
		return NonParametr
	}

	var removed, notFound []string
	for _, user := range strings.Split(strings.TrimSpace(matches[1]), ",") {
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

	return buildResponse("пользователи не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}
