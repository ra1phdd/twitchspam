package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleAntiSpam(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) != 3 { // !am as on/off/info
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"on": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, true, "default")
		},
		"off": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, false, "default")
		},
		"info": a.handleAntiSpamInfo,
	}

	antispamCmd := text.Words()[2]
	if handler, ok := handlers[antispamCmd]; ok {
		return handler(cfg, text)
	}
	return nil
}

func (a *Admin) handleAntiSpamOnOff(cfg *config.Config, enabled bool, typeSpam string) *ports.AnswerType {
	targetMap := map[string]*bool{
		"vip":     &cfg.Spam.SettingsVIP.Enabled,
		"emote":   &cfg.Spam.SettingsEmotes.Enabled,
		"default": &cfg.Spam.SettingsDefault.Enabled,
	}
	if target, ok := targetMap[typeSpam]; ok {
		*target = enabled
	}

	return nil
}

func (a *Admin) handleAntiSpamInfo(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
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
			sb.WriteString(fmt.Sprintf("    - %s (message_limit: %d, punishments: %s)\n", word, ex.MessageLimit, strings.Join(formatPunishments(ex.Punishments), ", ")))
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
		"- наказания: " + strings.Join(formatPunishments(cfg.Spam.SettingsDefault.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsDefault.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + fmt.Sprint(cfg.Spam.SettingsDefault.MaxWordLength),
		"- наказание за превышение длины слова: " + formatPunishment(cfg.Spam.SettingsDefault.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + fmt.Sprint(cfg.Spam.SettingsDefault.MinGapMessages),
		"\nвиперы:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsVIP.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.SimilarityThreshold),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.MessageLimit),
		"- наказания: " + strings.Join(formatPunishments(cfg.Spam.SettingsVIP.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsVIP.DurationResetPunishments),
		"- ограничение максимальной длины слова: " + fmt.Sprint(cfg.Spam.SettingsVIP.MaxWordLength),
		"- наказание за превышение длины слова: " + formatPunishment(cfg.Spam.SettingsVIP.MaxWordPunishment),
		"- минимальное количество разных сообщений между спамом: " + fmt.Sprint(cfg.Spam.SettingsVIP.MinGapMessages),
		"\nэмоуты:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsEmotes.Enabled),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsEmotes.MessageLimit),
		"- наказания: " + strings.Join(formatPunishments(cfg.Spam.SettingsEmotes.Punishments), ", "),
		"- время сброса счётчика наказаний: " + fmt.Sprint(cfg.Spam.SettingsEmotes.DurationResetPunishments),
		"- ограничение количества эмоутов в сообщении: " + fmt.Sprint(cfg.Spam.SettingsEmotes.MaxEmotesLength),
		"- наказание за превышение количества эмоутов в сообщении: " + formatPunishment(cfg.Spam.SettingsEmotes.MaxEmotesPunishment),
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

func (a *Admin) handleMode(cfg *config.Config, mode string) *ports.AnswerType {
	cfg.Spam.Mode = mode
	return nil
}

func (a *Admin) handleSim(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	target, idx := &cfg.Spam.SettingsDefault.SimilarityThreshold, 2 // !am sim <значение> или !am vip sim <значение>
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.SimilarityThreshold, 3
	}

	if len(text.Words()) != idx+1 {
		return NonParametr
	}

	if val, ok := parseFloatArg(text.Words()[idx], 0.1, 1); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение порога схожести сообщений должно быть от 0.1 до 1.0!"},
		IsReply: true,
	}
}

func (a *Admin) handleMsg(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	target, idx := &cfg.Spam.SettingsDefault.MessageLimit, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.MessageLimit, 3
	} else if typeSpam == "emote" {
		target, idx = &cfg.Spam.SettingsEmotes.MessageLimit, 3
	}

	if len(text.Words()) != idx+1 { // !am msg <значение> или !am vip/emote msg <значение>
		return NonParametr
	}

	if val, ok := parseIntArg(text.Words()[idx], 2, 15); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение лимита сообщений должно быть от 2 до 15!"},
		IsReply: true,
	}
}

func (a *Admin) handlePunishments(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	target, idx := &cfg.Spam.SettingsDefault.Punishments, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.Punishments, 3
	} else if typeSpam == "emote" {
		target, idx = &cfg.Spam.SettingsEmotes.Punishments, 3
	}

	if len(text.Words()) != idx+1 { // !am p <наказания через запятую> или !am vip/emote p <наказания через запятую>
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
		p, err := parsePunishment(str, allowInherit)
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

func (a *Admin) handleDurationResetPunishments(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	target, idx := &cfg.Spam.SettingsDefault.DurationResetPunishments, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.DurationResetPunishments, 3
	} else if typeSpam == "emote" {
		target, idx = &cfg.Spam.SettingsEmotes.DurationResetPunishments, 3
	}

	if len(text.Words()) != idx+1 { // !am rp <значение> или !am vip/emote rp <значение>
		return NonParametr
	}

	if val, ok := parseIntArg(text.Words()[idx], 1, 86400); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение времени сброса наказаний должно быть от 1 до 86400!"},
		IsReply: true,
	}
}

func (a *Admin) handleMaxLen(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
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
		if len(text.Words()) != param.idx+1 { // !am mwlen <значение> или !am vip/emote mwlen/melen <значение>
			return NonParametr
		}

		if val, ok := parseIntArg(text.Words()[param.idx], 0, param.max); ok {
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

func (a *Admin) handleMaxPunishment(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	target, idx := &cfg.Spam.SettingsDefault.MaxWordPunishment, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.MaxWordPunishment, 3
	} else if typeSpam == "emote" {
		target, idx = &cfg.Spam.SettingsEmotes.MaxEmotesPunishment, 3
	}

	if len(text.Words()) != idx+1 { // !am mwp <наказание> или !am vip/emote mwp/mep <наказание>
		return NonParametr
	}

	p, err := parsePunishment(text.Words()[idx], true)
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{fmt.Sprintf("не удалось распарсить наказание (%s)!", text.Words()[2])},
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

func (a *Admin) handleMinGap(cfg *config.Config, text *ports.MessageText, typeSpam string) *ports.AnswerType {
	target, idx := &cfg.Spam.SettingsDefault.MinGapMessages, 2
	if typeSpam == "vip" {
		target, idx = &cfg.Spam.SettingsVIP.MinGapMessages, 3
	}

	if len(text.Words()) != idx+1 { // !am min_gap <значение> или !am vip min_gap <значение>
		return NonParametr
	}

	if val, ok := parseIntArg(text.Words()[idx], 0, 15); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение должно быть от 0 до 15!"},
		IsReply: true,
	}
}

func (a *Admin) handleTime(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) != 3 { // !am time <значение>
		return NonParametr
	}

	if val, ok := parseIntArg(text.Words()[2], 1, 300); ok {
		cfg.Spam.CheckWindowSeconds = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{"значение окна проверки сообщений должно быть от 1 до 300!"},
		IsReply: true,
	}
}

func (a *Admin) handleAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) != 3 { // !am add <пользователи через запятую>
		return NonParametr
	}

	var added, alreadyExists []string
	for _, user := range strings.Split(text.Tail(2), ",") {
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

	var msgParts []string
	if len(added) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("добавлены в список: %s", strings.Join(added, ", ")))
	}
	if len(alreadyExists) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("уже в списке: %s", strings.Join(alreadyExists, ", ")))
	}

	if len(msgParts) == 0 {
		return &ports.AnswerType{
			Text:    []string{"пользователи не указаны!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) != 3 { // !am del <пользователи через запятую>
		return NonParametr
	}

	var removed, notFound []string
	for _, user := range strings.Split(text.Tail(2), ",") {
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

	var msgParts []string
	if len(removed) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("удалены из списка: %s", strings.Join(removed, ", ")))
	}
	if len(notFound) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("не найдены: %s", strings.Join(notFound, ", ")))
	}

	if len(msgParts) == 0 {
		return &ports.AnswerType{
			Text:    []string{"пользователи не указаны!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleVip(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 3 { // !am vip on/off/sim/...
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"on": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, true, "vip")
		},
		"off": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, false, "vip")
		},
		"sim": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleSim(cfg, text, "vip")
		},
		"msg": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMsg(cfg, text, "vip")
		},
		"p": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handlePunishments(cfg, text, "vip")
		},
		"rp": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleDurationResetPunishments(cfg, text, "vip")
		},
		"mwlen": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMaxLen(cfg, text, "vip")
		},
		"mwp": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMaxPunishment(cfg, text, "vip")
		},
		"min_gap": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMinGap(cfg, text, "vip")
		},
	}

	vipCmd := text.Words()[2]
	if handler, ok := handlers[vipCmd]; ok {
		return handler(cfg, text)
	}
	return NotFoundCmd
}

func (a *Admin) handleEmote(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	if len(text.Words()) < 3 { // !am emote on/off/sim/...
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"on": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, true, "emote")
		},
		"off": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, false, "emote")
		},
		"msg": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMsg(cfg, text, "emote")
		},
		"p": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handlePunishments(cfg, text, "emote")
		},
		"rp": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleDurationResetPunishments(cfg, text, "emote")
		},
		"melen": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMaxLen(cfg, text, "emote")
		},
		"mep": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMaxPunishment(cfg, text, "emote")
		},
	}

	emoteCmd := text.Words()[2]
	if handler, ok := handlers[emoteCmd]; ok {
		return handler(cfg, text)
	}
	return NotFoundCmd
}
