package admin

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleSpam(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	spamCmd, spamArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) ports.ActionType{
		"on": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleSpamOnOff(cfg, cmd, args, "default")
		},
		"off": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleSpamOnOff(cfg, cmd, args, "default")
		},
	}

	if handler, ok := handlers[spamCmd]; ok {
		return handler(cfg, spamCmd, spamArgs)
	}
	return NotFound
}

func (a *Admin) handleSpamOnOff(cfg *config.Config, cmd string, _ []string, typeSpam string) ports.ActionType {
	switch typeSpam {
	case "vip":
		cfg.Spam.SettingsVIP.Enabled = cmd == "on"
	default:
		cfg.Spam.SettingsDefault.Enabled = cmd == "on"
	}
	return None
}

func (a *Admin) handleMode(cfg *config.Config, cmd string, _ []string) ports.ActionType {
	cfg.Spam.Mode = cmd
	return None
}

func (a *Admin) handleSim(cfg *config.Config, _ string, args []string, typeSpam string) ports.ActionType {
	var target *float64

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.SimilarityThreshold
	default:
		target = &cfg.Spam.SettingsDefault.SimilarityThreshold
	}

	if val, ok := parseFloatArg(args, 0, 1); ok {
		*target = val
		return None
	}
	return ErrSimilarityThreshold
}

func (a *Admin) handleMsg(cfg *config.Config, _ string, args []string, typeSpam string) ports.ActionType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MessageLimit
	default:
		target = &cfg.Spam.SettingsDefault.MessageLimit
	}

	if val, ok := parseIntArg(args, 2, 15); ok {
		*target = val
		return None
	}
	return ErrMessageLimit
}

func (a *Admin) handleTo(cfg *config.Config, _ string, args []string, typeSpam string) ports.ActionType {
	if len(args) == 0 {
		return NonValue
	}

	parts := strings.Split(args[0], ",")
	var timeouts []int

	for i, str := range parts {
		if i >= 15 {
			break
		}

		if t, err := strconv.Atoi(str); err == nil {
			timeouts = append(timeouts, t)
		} else {
			return NonValue
		}
	}

	if len(timeouts) == 0 {
		return NonValue
	}

	switch typeSpam {
	case "vip":
		cfg.Spam.SettingsVIP.Timeouts = timeouts
	default:
		cfg.Spam.SettingsDefault.Timeouts = timeouts
	}
	return None
}

func (a *Admin) handleRto(cfg *config.Config, _ string, args []string, typeSpam string) ports.ActionType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.ResetTimeoutSeconds
	default:
		target = &cfg.Spam.SettingsDefault.ResetTimeoutSeconds
	}

	if val, ok := parseIntArg(args, 1, 86400); ok {
		*target = val
		return None
	}
	return ErrResetTimeoutSeconds
}

func (a *Admin) handleMwLen(cfg *config.Config, _ string, args []string, typeSpam string) ports.ActionType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MaxWordLength
	default:
		target = &cfg.Spam.SettingsDefault.MaxWordLength
	}

	if val, ok := parseIntArg(args, 0, 500); ok {
		*target = val
		return None
	}
	return ErrMaxWordLength
}

func (a *Admin) handleMwt(cfg *config.Config, _ string, args []string, typeSpam string) ports.ActionType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MaxWordTimeoutTime
	default:
		target = &cfg.Spam.SettingsDefault.MaxWordTimeoutTime
	}

	if val, ok := parseIntArg(args, 0, 1209600); ok {
		*target = val
		return None
	}
	return ErrMaxWordTimeoutTime
}

func (a *Admin) handleMinGap(cfg *config.Config, _ string, args []string, typeSpam string) ports.ActionType {
	var target *int

	switch typeSpam {
	case "vip":
		target = &cfg.Spam.SettingsVIP.MinGapMessages
	default:
		target = &cfg.Spam.SettingsDefault.MinGapMessages
	}

	if val, ok := parseIntArg(args, 0, 15); ok {
		*target = val
		return None
	}
	return ErrMinGapMessages
}

func (a *Admin) handleTime(cfg *config.Config, _ string, args []string) ports.ActionType {
	if val, ok := parseIntArg(args, 1, 300); ok {
		cfg.Spam.CheckWindowSeconds = val
		return None
	}
	return ErrCheckWindowSeconds
}

func (a *Admin) handleAdd(cfg *config.Config, _ string, args []string) ports.ActionType {
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
		return None
	}

	return ports.ActionType(strings.Join(msgParts, " • "))
}

func (a *Admin) handleDel(cfg *config.Config, _ string, args []string) ports.ActionType {
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
		return None
	}
	return ports.ActionType(strings.Join(msgParts, " • "))
}

func (a *Admin) handleVip(cfg *config.Config, _ string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	vipCmd, vipArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) ports.ActionType{
		"on": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleSpamOnOff(cfg, cmd, args, "vip")
		},
		"off": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleSpamOnOff(cfg, cmd, args, "vip")
		},
		"sim": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleSim(cfg, cmd, args, "vip")
		},
		"msg": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMsg(cfg, cmd, args, "vip")
		},
		"to": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleTo(cfg, cmd, args, "vip")
		},
		"rto": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleRto(cfg, cmd, args, "vip")
		},
		"mwlen": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMwLen(cfg, cmd, args, "vip")
		},
		"mwt": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMwt(cfg, cmd, args, "vip")
		},
		"min_gap": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMinGap(cfg, cmd, args, "vip")
		},
	}

	if handler, ok := handlers[vipCmd]; ok {
		return handler(cfg, vipCmd, vipArgs)
	}
	return NotFound
}
