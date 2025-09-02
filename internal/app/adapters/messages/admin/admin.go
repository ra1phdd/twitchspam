package admin

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"log/slog"
	"math"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

const (
	None                   ports.ActionType = "none"
	Success                ports.ActionType = "успешно"
	NotFound               ports.ActionType = "неизвестная команда"
	ErrFound               ports.ActionType = "неизвестная ошибка"
	NonParametr            ports.ActionType = "не указан параметр"
	NonValue               ports.ActionType = "неверное значение"
	ErrSimilarityThreshold ports.ActionType = "значение должно быть от 0.0 до 1.0"
	ErrMessageLimit        ports.ActionType = "значение должно быть от 2 до 15"
	ErrCheckWindowSeconds  ports.ActionType = "значение должно быть от 0 до 300"
	ErrMaxWordLength       ports.ActionType = "значение должно быть от 0 до 500"
	ErrMaxWordTimeoutTime  ports.ActionType = "значение должно быть от 0 до 1209600"
	ErrMinGapMessages      ports.ActionType = "значение должно быть от 0 до 15"
	ErrResetTimeoutSeconds ports.ActionType = "значение должно быть от 1 до 86400"
	ErrDelayAutomod        ports.ActionType = "значение должно быть от 1 до 10"
	NoStream               ports.ActionType = "стрим выключен"
	ErrFoundMwordGroup     ports.ActionType = "группа уже существует"
	ErrNotFoundMwordGroup  ports.ActionType = "группа не найдена"
	ErrNotFoundMwordGroups ports.ActionType = "группы не найдены"
)

type Admin struct {
	log     logger.Logger
	manager *config.Manager
	stream  ports.StreamPort
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort) *Admin {
	return &Admin{
		log:     log,
		manager: manager,
		stream:  stream,
	}
}

type cmdHandler func(cfg *config.Config, cmd string, args []string) ports.ActionType

var startApp = time.Now()

func (a *Admin) FindMessages(msg *ports.ChatMessage) ports.ActionType {
	if (!msg.Chatter.IsBroadcaster || !msg.Chatter.IsMod) && !strings.HasPrefix(msg.Message.Text, "!am") {
		return None
	}

	parts := strings.Fields(msg.Message.Text)
	if len(parts) < 2 {
		return NonParametr
	}

	cmd, args := parts[1], parts[2:]
	if cmd == "ping" {
		return a.handlePing()
	}

	handlers := map[string]cmdHandler{
		"on": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleOnOff(cfg, cmd, args, "default")
		},
		"off": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleOnOff(cfg, cmd, args, "default")
		},
		"online": a.handleMode,
		"always": a.handleMode,
		"sim": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleSim(cfg, cmd, args, "default")
		},
		"msg": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMsg(cfg, cmd, args, "default")
		},
		"time": a.handleTime,
		"to": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleTo(cfg, cmd, args, "default")
		},
		"rto": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleRto(cfg, cmd, args, "default")
		},
		"mw": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMw(cfg, cmd, args, "default")
		},
		"mwt": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMwt(cfg, cmd, args, "default")
		},
		"min_gap": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMinGap(cfg, cmd, args, "default")
		},
		"da":    a.handleDelayAutomod,
		"reset": a.handleReset,
		"add":   a.handleAdd,
		"del":   a.handleDel,
		"vip":   a.handleVip,
		"game":  a.handleCategory,
		"mwg":   a.handleMwg,
	}

	handler, ok := handlers[cmd]
	if !ok {
		a.log.Info("cmd", slog.String("cmd", cmd))
		return NotFound
	}

	var result ports.ActionType
	if err := a.manager.Update(func(cfg *config.Config) {
		result = handler(cfg, cmd, args)
	}); err != nil {
		a.log.Error("Failed update config", err, slog.String("msg", msg.Message.Text))
		return ErrFound
	}

	if result != None {
		return result
	}
	return Success
}

func (a *Admin) handlePing() ports.ActionType {
	uptime := time.Since(startApp)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	percent, _ := cpu.Percent(0, false)
	if len(percent) == 0 {
		percent = append(percent, 0)
	}

	return ports.ActionType(fmt.Sprintf("бот работает %v • загрузка CPU %.2f%% • потребление ОЗУ %v MB", uptime.Truncate(time.Second), percent[0], m.Sys/1024/1024))
}

func (a *Admin) handleOnOff(cfg *config.Config, cmd string, args []string, typeSpam string) ports.ActionType {
	switch typeSpam {
	case "vip":
		cfg.Spam.SettingsVIP.Enabled = cmd == "on"
	default:
		cfg.Spam.SettingsDefault.Enabled = cmd == "on"
	}
	return None
}

func (a *Admin) handleMode(cfg *config.Config, cmd string, args []string) ports.ActionType {
	cfg.Spam.Mode = cmd
	return None
}

func (a *Admin) handleSim(cfg *config.Config, cmd string, args []string, typeSpam string) ports.ActionType {
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

func (a *Admin) handleMsg(cfg *config.Config, cmd string, args []string, typeSpam string) ports.ActionType {
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

func (a *Admin) handleTo(cfg *config.Config, cmd string, args []string, typeSpam string) ports.ActionType {
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

func (a *Admin) handleRto(cfg *config.Config, cmd string, args []string, typeSpam string) ports.ActionType {
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

func (a *Admin) handleMw(cfg *config.Config, cmd string, args []string, typeSpam string) ports.ActionType {
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

func (a *Admin) handleMwt(cfg *config.Config, cmd string, args []string, typeSpam string) ports.ActionType {
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

func (a *Admin) handleMinGap(cfg *config.Config, cmd string, args []string, typeSpam string) ports.ActionType {
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

func (a *Admin) handleDelayAutomod(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if val, ok := parseIntArg(args, 0, 10); ok {
		cfg.Spam.DelayAutomod = val
		return None
	}
	return ErrDelayAutomod
}

func (a *Admin) handleReset(cfg *config.Config, cmd string, args []string) ports.ActionType {
	cfg.Spam = a.manager.GetDefault().Spam
	return None
}

func (a *Admin) handleTime(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if val, ok := parseIntArg(args, 1, 300); ok {
		cfg.Spam.CheckWindowSeconds = val
		return None
	}
	return ErrCheckWindowSeconds
}

func (a *Admin) handleAdd(cfg *config.Config, cmd string, args []string) ports.ActionType {
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

func (a *Admin) handleDel(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) == 0 {
		return NonParametr
	}

	var removed []string
	var notFound []string

	cfg.Spam.WhitelistUsers = slices.DeleteFunc(cfg.Spam.WhitelistUsers, func(w string) bool {
		for _, u := range args {
			u = strings.TrimSpace(u)
			if u == w {
				removed = append(removed, w)
				return true
			}
		}
		return false
	})

	for _, u := range args {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		found := false
		for _, r := range removed {
			if u == r {
				found = true
				break
			}
		}
		if !found {
			notFound = append(notFound, u)
		}
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

func (a *Admin) handleVip(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	vipCmd, vipArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) ports.ActionType{
		"on": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleOnOff(cfg, cmd, args, "vip")
		},
		"off": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleOnOff(cfg, cmd, args, "vip")
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
		"mw": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMw(cfg, cmd, args, "vip")
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

func (a *Admin) handleCategory(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if !a.stream.IsLive() {
		return NoStream
	}

	a.stream.SetCategory(strings.Join(args, " "))
	return Success
}

func (a *Admin) handleMwg(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	mwgCmd, mwgArgs := args[0], args[1:]

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) ports.ActionType{
		"list":   a.handleMwgList,
		"create": a.handleMwgCreate,
		"set":    a.handleMwgSet,
		"add":    a.handleMwgAdd,
		"del":    a.handleMwgDel,
		"on":     a.handleMwgOnOff,
		"off":    a.handleMwgOnOff,
		"words":  a.handleMwgWords,
	}

	if handler, ok := handlers[mwgCmd]; ok {
		return handler(cfg, mwgCmd, mwgArgs)
	}
	return NotFound
}

func (a *Admin) handleMwgList(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(cfg.MwordGroup) == 0 {
		return ErrNotFoundMwordGroups
	}

	msg := "группы:"
	for name := range cfg.MwordGroup {
		msg += fmt.Sprintf(" %s", name)
	}

	return ports.ActionType(msg)
}

func (a *Admin) handleMwgCreate(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	punishment := args[1]

	if _, exists := cfg.MwordGroup[groupName]; exists {
		return ErrFoundMwordGroup
	}

	action, duration, err := parsePunishment(punishment)
	if err != nil {
		return ErrFound
	}

	cfg.MwordGroup[groupName] = &config.MwordGroup{
		Action:   action,
		Duration: duration,
		Enabled:  true,
		Words:    []string{},
	}

	return Success
}

func (a *Admin) handleMwgSet(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	punishment := args[1]

	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return ErrNotFoundMwordGroup
	}

	action, duration, err := parsePunishment(punishment)
	if err != nil {
		return ErrFound
	}

	cfg.MwordGroup[groupName].Action = action
	cfg.MwordGroup[groupName].Duration = duration

	return Success
}

func (a *Admin) handleMwgAdd(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	words := strings.Split(strings.Join(args[1:], " "), ",")

	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return NotFound
	}

	for _, word := range words {
		trimmedWord := strings.TrimSpace(word)
		if trimmedWord == "" {
			continue
		}

		found := false
		for _, existingWord := range group.Words {
			if existingWord == trimmedWord {
				found = true
				break
			}
		}

		if !found {
			group.Words = append(cfg.MwordGroup[groupName].Words, trimmedWord)
		}
	}

	return Success
}

func (a *Admin) handleMwgDel(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	target := strings.Join(args[1:], " ")

	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return NotFound
	}

	if target == "all" {
		delete(cfg.MwordGroup, groupName)

		return Success
	}

	wordsToRemove := strings.Split(target, ",")
	var newWords []string

	for _, existingWord := range group.Words {
		keep := true
		for _, wordToRemove := range wordsToRemove {
			if existingWord == strings.TrimSpace(wordToRemove) {
				keep = false
				break
			}
		}
		if keep {
			newWords = append(newWords, existingWord)
		}
	}
	group.Words = newWords

	return Success
}

func (a *Admin) handleMwgOnOff(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	groupName := args[0]

	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return ErrNotFoundMwordGroup
	}
	cfg.MwordGroup[groupName].Enabled = cmd == "on"

	return Success
}

func (a *Admin) handleMwgWords(cfg *config.Config, cmd string, args []string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	groupName := args[0]

	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return ErrNotFoundMwordGroup
	}

	if len(group.Words) == 0 {
		return "cлова в группе отсутствуют"
	}

	msg := "cлова в группе: " + strings.Join(group.Words, ", ")
	return ports.ActionType(msg)
}

func parsePunishment(punishment string) (string, int, error) {
	punishment = strings.TrimSpace(punishment)
	if punishment == "-" {
		return "delete", 0, nil
	}

	if punishment == "0" {
		return "ban", 0, nil
	}

	duration, err := strconv.Atoi(punishment)
	if err != nil || duration < 1 || duration > 1209600 {
		return "", 0, fmt.Errorf("invalid timeout value")
	}

	return "timeout", duration, nil
}

func parseIntArg(args []string, min, max int) (int, bool) {
	if len(args) == 0 {
		return 0, false
	}
	val, err := strconv.Atoi(args[0])
	if err != nil {
		return 0, false
	}
	if (min != -1 && val < min) || (max != -1 && val > max) {
		return 0, false
	}
	return val, true
}

func parseFloatArg(args []string, min, max float64) (float64, bool) {
	if len(args) == 0 {
		return 0, false
	}
	val, err := strconv.ParseFloat(args[0], 64)
	if err != nil || val < min || val > max {
		return 0, false
	}

	return math.Round(val*100) / 100, true
}
