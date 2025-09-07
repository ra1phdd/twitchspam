package admin

import (
	"fmt"
	"slices"
	"strconv"
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
	timeoutsDefault := make([]string, len(cfg.Spam.SettingsDefault.Timeouts))
	for i, v := range cfg.Spam.SettingsDefault.Timeouts {
		timeoutsDefault[i] = fmt.Sprint(v)
	}
	timeoutsVip := make([]string, len(cfg.Spam.SettingsVIP.Timeouts))
	for i, v := range cfg.Spam.SettingsVIP.Timeouts {
		timeoutsVip[i] = fmt.Sprint(v)
	}

	parts := []string{
		"- режим: " + cfg.Spam.Mode,
		"- окно проверки сообщений: " + fmt.Sprint(cfg.Spam.CheckWindowSeconds),
		"- разрешенные пользователи: " + strings.Join(cfg.Spam.WhitelistUsers, ", "),
		"\nобщие:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsDefault.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsDefault.SimilarityThreshold),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsDefault.MessageLimit),
		"- таймауты: " + strings.Join(timeoutsDefault, ", "),
		"- сброс счётчика таймаутов: " + fmt.Sprint(cfg.Spam.SettingsDefault.ResetTimeoutSeconds),
		"- ограничение максимальной длины слова: " + fmt.Sprint(cfg.Spam.SettingsDefault.MaxWordLength),
		"- таймаут за превышение длины слова: " + fmt.Sprint(cfg.Spam.SettingsDefault.MaxWordTimeoutTime),
		"- минимальное количество разных сообщений между спамом: " + fmt.Sprint(cfg.Spam.SettingsDefault.MinGapMessages),
		"\nвиперы:",
		"- включен: " + fmt.Sprint(cfg.Spam.SettingsVIP.Enabled),
		"- порог схожести сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.SimilarityThreshold),
		"- кол-во похожих сообщений: " + fmt.Sprint(cfg.Spam.SettingsVIP.MessageLimit),
		"- таймауты: " + strings.Join(timeoutsVip, ", "),
		"- сброс счётчика таймаутов: " + fmt.Sprint(cfg.Spam.SettingsVIP.ResetTimeoutSeconds),
		"- ограничение максимальной длины слова: " + fmt.Sprint(cfg.Spam.SettingsVIP.MaxWordLength),
		"- таймаут за превышение длины слова: " + fmt.Sprint(cfg.Spam.SettingsVIP.MaxWordTimeoutTime),
		"- минимальное количество разных сообщений между спамом: " + fmt.Sprint(cfg.Spam.SettingsVIP.MinGapMessages),
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

	if val, ok := parseFloatArg(args, 0, 1); ok {
		*target = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение порога схожести сообщений должно быть от 0.0 до 1.0!"},
		IsReply: true,
	}
}

func (a *Admin) handleMsg(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MessageLimit
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

func (a *Admin) handleTo(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	if len(args) == 0 {
		return NonParametr
	}

	parts := strings.Split(strings.Join(args, " "), ",")
	var timeouts []int

	for i, str := range parts {
		if i >= 15 {
			break
		}

		if t, err := strconv.Atoi(str); err == nil {
			timeouts = append(timeouts, t)
		} else {
			return &ports.AnswerType{
				Text:    []string{"одно из значений не является числом!"},
				IsReply: true,
			}
		}
	}

	if len(timeouts) == 0 {
		return NonParametr
	}

	switch typeSpam {
	case "vip":
		cfg.Spam.SettingsVIP.Timeouts = timeouts
	default:
		cfg.Spam.SettingsDefault.Timeouts = timeouts
	}
	return nil
}

func (a *Admin) handleRto(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.ResetTimeoutSeconds
	default:
		target = &cfg.Spam.SettingsDefault.ResetTimeoutSeconds
	}

	if val, ok := parseIntArg(args, 1, 86400); ok {
		*target = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение сброса таймаутов должно быть от 1 до 86400!"},
		IsReply: true,
	}
}

func (a *Admin) handleMwLen(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MaxWordLength
	default:
		target = &cfg.Spam.SettingsDefault.MaxWordLength
	}

	if val, ok := parseIntArg(args, 0, 500); ok {
		*target = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение максимальной длины слова должно быть от 0 до 500!"},
		IsReply: true,
	}
}

func (a *Admin) handleMwt(cfg *config.Config, _ string, args []string, typeSpam string) *ports.AnswerType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MaxWordTimeoutTime
	default:
		target = &cfg.Spam.SettingsDefault.MaxWordTimeoutTime
	}

	if val, ok := parseIntArg(args, 0, 1209600); ok {
		*target = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение таймаута должно быть от 0 до 1209600!"},
		IsReply: true,
	}
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
		"to": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleTo(cfg, cmd, args, "vip")
		},
		"rto": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleRto(cfg, cmd, args, "vip")
		},
		"mwlen": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMwLen(cfg, cmd, args, "vip")
		},
		"mwt": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMwt(cfg, cmd, args, "vip")
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
