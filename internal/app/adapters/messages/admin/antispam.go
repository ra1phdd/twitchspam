package admin

import (
	"fmt"
	"slices"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleAntiSpam(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	spamCmd, spamArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) *ports.AnswerType{
		"on": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, cmd, args, "default")
		},
		"off": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, cmd, args, "default")
		},
		"info": a.handleAntiSpamInfo,
	}

	if handler, ok := handlers[spamCmd]; ok {
		return handler(cfg, spamCmd, spamArgs)
	}
	return nil
}

func (a *Admin) handleAntiSpamInfo(cfg *config.Config, _ string, _ []string) *ports.AnswerType {
	exceptions := []string{"не найдены"}
	if len(cfg.Spam.Exceptions) > 0 {
		exceptions = []string{}
		for word, ex := range cfg.Spam.Exceptions {
			exceptions = append(exceptions, fmt.Sprintf("    - %s (message_limit: %d, punishments: %s)", word, ex.MessageLimit, formatPunishments(ex.Punishments)))
		}
	}

	parts := []string{
		"- режим: " + cfg.Spam.Mode,
		"- окно проверки сообщений: " + fmt.Sprint(cfg.Spam.CheckWindowSeconds),
		"- разрешенные пользователи: " + strings.Join(cfg.Spam.WhitelistUsers, ", "),
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
		strings.Join(exceptions, "\n"),
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

func (a *Admin) handleAntiSpamOnOff(cfg *config.Config, cmd string, _ []string, typeSpam string) *ports.AnswerType {
	switch typeSpam {
	case "vip":
		cfg.Spam.SettingsVIP.Enabled = cmd == "on"
	case "emote":
		cfg.Spam.SettingsEmotes.Enabled = cmd == "on"
	default:
		cfg.Spam.SettingsDefault.Enabled = cmd == "on"
	}
	return nil
}

func (a *Admin) handleMode(cfg *config.Config, cmd string, _ []string) *ports.AnswerType {
	cfg.Spam.Mode = cmd
	return nil
}

func (a *Admin) handleSim(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *float64

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.SimilarityThreshold
	default:
		target = &cfg.Spam.SettingsDefault.SimilarityThreshold
	}

	if val, ok := parseFloatArg(args, 0.1, 1); ok {
		*target = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение порога схожести сообщений должно быть от 0.1 до 1.0!"},
		IsReply: true,
	}
}

func (a *Admin) handleMsg(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MessageLimit
	case "emote":
		target = &cfg.Spam.SettingsEmotes.MessageLimit
	default:
		target = &cfg.Spam.SettingsDefault.MessageLimit
	}

	if val, ok := parseIntArg(args, 2, 15); ok {
		*target = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение лимита сообщений должно быть от 2 до 15!"},
		IsReply: true,
	}
}

func (a *Admin) handlePunishments(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	if len(args) == 0 {
		return NonParametr
	}

	parts := strings.Split(strings.Join(args, " "), ",")
	var punishments []config.Punishment

	for i, str := range parts {
		if i >= 15 {
			break
		}

		allowInherit := typeSpam == "default"
		p, err := parsePunishment(str, allowInherit)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{fmt.Sprintf("не удалось распарсить наказания (%s)!", str)},
				IsReply: true,
			}
		}

		if p.Action == "inherit" {
			if typeSpam == "default" {
				return &ports.AnswerType{
					Text:    []string{"невозможно скопировать наказания!"},
					IsReply: true,
				}
			}

			punishments = cfg.Spam.SettingsDefault.Punishments
			break
		}
		punishments = append(punishments, p)
	}

	if len(punishments) == 0 {
		return NonParametr
	}

	switch typeSpam {
	case "vip":
		cfg.Spam.SettingsVIP.Punishments = punishments
	case "emote":
		cfg.Spam.SettingsEmotes.Punishments = punishments
	default:
		cfg.Spam.SettingsDefault.Punishments = punishments
	}
	return nil
}

func (a *Admin) handleDurationResetPunishments(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.DurationResetPunishments
	case "emote":
		target = &cfg.Spam.SettingsEmotes.DurationResetPunishments
	default:
		target = &cfg.Spam.SettingsDefault.DurationResetPunishments
	}

	if val, ok := parseIntArg(args, 1, 86400); ok {
		*target = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение времени сброса наказаний должно быть от 1 до 86400!"},
		IsReply: true,
	}
}

func (a *Admin) handleMwLen(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *int
	var maxInt int
	var errText string

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MaxWordLength
		maxInt = 500
		errText = "значение максимальной длины слова должно быть от 0 до 500!"
	case "emote":
		target = &cfg.Spam.SettingsEmotes.MaxEmotesLength
		maxInt = 30
		errText = "значение максимального количества эмоутов должно быть от 0 до 30!"
	default:
		target = &cfg.Spam.SettingsDefault.MaxWordLength
		maxInt = 500
		errText = "значение максимальной длины слова должно быть от 0 до 500!"
	}

	if val, ok := parseIntArg(args, 0, maxInt); ok {
		*target = val
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{errText},
		IsReply: true,
	}
}

func (a *Admin) handleMwPunishment(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	if len(args) == 0 {
		return NonParametr
	}

	allowInherit := typeSpam == "default"
	p, err := parsePunishment(args[0], allowInherit)
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{fmt.Sprintf("не удалось распарсить наказание (%s)!", args[0])},
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

	targets := map[string]*config.Punishment{
		"default": &cfg.Spam.SettingsDefault.MaxWordPunishment,
		"vip":     &cfg.Spam.SettingsVIP.MaxWordPunishment,
		"emote":   &cfg.Spam.SettingsEmotes.MaxEmotesPunishment,
	}
	if field, ok := targets[typeSpam]; ok {
		*field = p
	}

	return nil
}

func (a *Admin) handleMinGap(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MinGapMessages
	default:
		target = &cfg.Spam.SettingsDefault.MinGapMessages
	}

	if val, ok := parseIntArg(args, 0, 15); ok {
		*target = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение должно быть от 0 до 15!"},
		IsReply: true,
	}
}

func (a *Admin) handleTime(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if val, ok := parseIntArg(args, 1, 300); ok {
		cfg.Spam.CheckWindowSeconds = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение окна проверки сообщений должно быть от 1 до 300!"},
		IsReply: true,
	}
}

func (a *Admin) handleAdd(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) == 0 {
		return NonParametr
	}

	var added []string
	var alreadyExists []string

	for _, u := range args {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if slices.Contains(cfg.Spam.WhitelistUsers, u) {
			alreadyExists = append(alreadyExists, u)
		} else {
			cfg.Spam.WhitelistUsers = append(cfg.Spam.WhitelistUsers, u)
			added = append(added, u)
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
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleDel(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) == 0 {
		return NonParametr
	}

	var removed []string
	var notFound []string

	cfg.Spam.WhitelistUsers = slices.DeleteFunc(cfg.Spam.WhitelistUsers, func(w string) bool {
		for _, u := range args {
			if strings.TrimSpace(u) == w {
				removed = append(removed, w)
				return true
			}
		}
		return false
	})

	for _, u := range args {
		u = strings.TrimSpace(u)
		if u == "" || slices.Contains(removed, u) {
			continue
		}
		notFound = append(notFound, u)
	}

	var msgParts []string
	if len(removed) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("удалены из списка: %s", strings.Join(removed, ", ")))
	}
	if len(notFound) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("нет в списке: %s", strings.Join(notFound, ", ")))
	}

	if len(msgParts) == 0 {
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleVip(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	vipCmd, vipArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) *ports.AnswerType{
		"on": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, cmd, args, "vip")
		},
		"off": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, cmd, args, "vip")
		},
		"sim": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleSim(cfg, cmd, args, "vip")
		},
		"msg": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMsg(cfg, cmd, args, "vip")
		},
		"p": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handlePunishments(cfg, cmd, args, "vip")
		},
		"rp": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleDurationResetPunishments(cfg, cmd, args, "vip")
		},
		"mwlen": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMwLen(cfg, cmd, args, "vip")
		},
		"mwp": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMwPunishment(cfg, cmd, args, "vip")
		},
		"min_gap": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMinGap(cfg, cmd, args, "vip")
		},
	}

	if handler, ok := handlers[vipCmd]; ok {
		return handler(cfg, vipCmd, vipArgs)
	}
	return NotFoundCmd
}

func (a *Admin) handleEmote(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 1 {
		return NonParametr
	}
	emoteCmd, emoteArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) *ports.AnswerType{
		"on": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, cmd, args, "emote")
		},
		"off": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleAntiSpamOnOff(cfg, cmd, args, "emote")
		},
		"msg": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMsg(cfg, cmd, args, "emote")
		},
		"p": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handlePunishments(cfg, cmd, args, "emote")
		},
		"rp": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleDurationResetPunishments(cfg, cmd, args, "emote")
		},
		"melen": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMwLen(cfg, cmd, args, "emote")
		},
		"mep": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMwPunishment(cfg, cmd, args, "emote")
		},
	}

	if handler, ok := handlers[emoteCmd]; ok {
		return handler(cfg, emoteCmd, emoteArgs)
	}
	return NotFoundCmd
}
