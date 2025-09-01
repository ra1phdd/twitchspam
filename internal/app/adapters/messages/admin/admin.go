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

type cmdHandler func(cfg *config.Config, args []string, cmd string) ports.ActionType

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
		"on": func(cfg *config.Config, _ []string, cmd string) ports.ActionType {
			return handleOnOffSettings(cmd, &cfg.Spam.SettingsDefault)
		},
		"off": func(cfg *config.Config, _ []string, cmd string) ports.ActionType {
			return handleOnOffSettings(cmd, &cfg.Spam.SettingsDefault)
		},
		"online":  a.handleMode,
		"always":  a.handleMode,
		"sim":     a.handleSim,
		"msg":     a.handleMsg,
		"time":    a.handleTime,
		"to":      a.handleTo,
		"rto":     a.handleRto,
		"mw":      a.handleMw,
		"mwt":     a.handleMwt,
		"min_gap": a.handleMinGap,
		"da":      a.handleDelayAutomod,
		"reset":   a.handleReset,
		"add":     a.handleAdd,
		"del":     a.handleDel,
		"vip":     a.handleVip,
		"game":    a.handleCategory,
		"mwg":     a.handleMwg,
	}

	handler, ok := handlers[cmd]
	if !ok {
		a.log.Info("cmd", slog.String("cmd", cmd))
		return NotFound
	}

	var result ports.ActionType
	if err := a.manager.Update(func(cfg *config.Config) {
		result = handler(cfg, args, cmd)
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
	return ports.ActionType(fmt.Sprintf("бот работает %v • загрузка CPU %.2f%% • потребление ОЗУ %v MB", uptime.Truncate(time.Second), percent[0], m.Sys/1024/1024))
}

func (a *Admin) handleMode(cfg *config.Config, _ []string, cmd string) ports.ActionType {
	cfg.Spam.Mode = cmd
	return None
}

func (a *Admin) handleSim(cfg *config.Config, args []string, _ string) ports.ActionType {
	return handleSimSettings(args, &cfg.Spam.SettingsDefault)
}

func (a *Admin) handleMsg(cfg *config.Config, args []string, _ string) ports.ActionType {
	return handleMsgSettings(args, &cfg.Spam.SettingsDefault)
}

func (a *Admin) handleTo(cfg *config.Config, args []string, _ string) ports.ActionType {
	return handleToSettings(args, &cfg.Spam.SettingsDefault)
}

func (a *Admin) handleRto(cfg *config.Config, args []string, _ string) ports.ActionType {
	return handleRtoSettings(args, &cfg.Spam.SettingsDefault)
}

func (a *Admin) handleMw(cfg *config.Config, args []string, _ string) ports.ActionType {
	return handleMwSettings(args, &cfg.Spam.SettingsDefault)
}

func (a *Admin) handleMwt(cfg *config.Config, args []string, _ string) ports.ActionType {
	return handleMwtSettings(args, &cfg.Spam.SettingsDefault)
}

func (a *Admin) handleMinGap(cfg *config.Config, args []string, _ string) ports.ActionType {
	return handleMinGapSettings(args, &cfg.Spam.SettingsDefault)
}

func handleOnOffSettings(cmd string, s *config.SpamSettings) ports.ActionType {
	s.Enabled = cmd == "on"
	return None
}

func handleSimSettings(args []string, s *config.SpamSettings) ports.ActionType {
	if val, ok := parseFloatArg(args, 0, 1); ok {
		s.SimilarityThreshold = val
		return None
	}
	return ErrSimilarityThreshold
}

func handleMsgSettings(args []string, s *config.SpamSettings) ports.ActionType {
	if val, ok := parseIntArg(args, 2, 15); ok {
		s.MessageLimit = val
		return None
	}
	return ErrMessageLimit
}

func handleToSettings(args []string, s *config.SpamSettings) ports.ActionType {
	var timeouts []int
	for i, str := range args {
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
	s.Timeouts = timeouts
	return None
}

func handleRtoSettings(args []string, s *config.SpamSettings) ports.ActionType {
	if val, ok := parseIntArg(args, 1, 86400); ok {
		s.ResetTimeoutSeconds = val
		return None
	}
	return ErrResetTimeoutSeconds
}

func handleMwSettings(args []string, s *config.SpamSettings) ports.ActionType {
	if val, ok := parseIntArg(args, 0, 500); ok {
		s.MaxWordLength = val
		return None
	}
	return ErrMaxWordLength
}

func handleMwtSettings(args []string, s *config.SpamSettings) ports.ActionType {
	if val, ok := parseIntArg(args, 0, 1209600); ok {
		s.MaxWordTimeoutTime = val
		return None
	}
	return ErrMaxWordTimeoutTime
}

func handleMinGapSettings(args []string, s *config.SpamSettings) ports.ActionType {
	if val, ok := parseIntArg(args, 0, 15); ok {
		s.MinGapMessages = val
		return None
	}
	return ErrMinGapMessages
}

func (a *Admin) handleDelayAutomod(cfg *config.Config, args []string, _ string) ports.ActionType {
	if val, ok := parseIntArg(args, 0, 10); ok {
		cfg.Spam.DelayAutomod = val
		return None
	}
	return ErrDelayAutomod
}

func (a *Admin) handleReset(cfg *config.Config, _ []string, _ string) ports.ActionType {
	def := a.manager.GetDefault()
	cfg.Spam = def.Spam
	return None
}

func (a *Admin) handleTime(cfg *config.Config, args []string, _ string) ports.ActionType {
	if val, ok := parseIntArg(args, 1, 300); ok {
		cfg.Spam.CheckWindowSeconds = val
		return None
	}
	return ErrCheckWindowSeconds
}

func (a *Admin) handleAdd(cfg *config.Config, args []string, _ string) ports.ActionType {
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

func (a *Admin) handleDel(cfg *config.Config, args []string, _ string) ports.ActionType {
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

func (a *Admin) handleVip(cfg *config.Config, args []string, _ string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	vipCmd, vipArgs := args[0], args[1:]

	handlers := map[string]func([]string, *config.SpamSettings) ports.ActionType{
		"on": func(_ []string, settings *config.SpamSettings) ports.ActionType {
			return handleOnOffSettings(vipCmd, settings)
		},
		"off": func(_ []string, settings *config.SpamSettings) ports.ActionType {
			return handleOnOffSettings(vipCmd, settings)
		},
		"sim":     handleSimSettings,
		"msg":     handleMsgSettings,
		"to":      handleToSettings,
		"rto":     handleRtoSettings,
		"mw":      handleMwSettings,
		"mwt":     handleMwtSettings,
		"min_gap": handleMinGapSettings,
	}

	if handler, ok := handlers[vipCmd]; ok {
		return handler(vipArgs, &cfg.Spam.SettingsVIP)
	}
	return NotFound
}

func (a *Admin) handleCategory(_ *config.Config, args []string, _ string) ports.ActionType {
	if !a.stream.IsLive() {
		return NoStream
	}

	a.stream.SetCategory(strings.Join(args, " "))
	return Success
}

func (a *Admin) handleOnline(cfg *config.Config, args []string, _ string) ports.ActionType {
	if len(args) > 0 && args[0] == "online" {
		cfg.PunishmentOnline = !cfg.PunishmentOnline
		return Success
	}

	return NotFound
}

func (a *Admin) handleMwg(cfg *config.Config, args []string, _ string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	mwgCmd, mwgArgs := args[0], args[1:]

	handlers := map[string]func([]string, *config.Config) ports.ActionType{
		"list":   a.handleMwgList,
		"create": a.handleMwgCreate,
		"set":    a.handleMwgSet,
		"add":    a.handleMwgAdd,
		"del":    a.handleMwgDel,
		"on":     a.handleMwgOnOff,
		"off":    a.handleMwgOnOff,
	}

	if handler, ok := handlers[mwgCmd]; ok {
		return handler(mwgArgs, cfg)
	}
	return NotFound
}

func (a *Admin) handleMwgList(_ []string, cfg *config.Config) ports.ActionType {
	if len(cfg.MwordGroup) == 0 {
		return ErrNotFoundMwordGroups
	}

	msg := "группы:"
	for name := range cfg.MwordGroup {
		msg += fmt.Sprintf(" %s", name)
	}

	return ports.ActionType(msg)
}

func (a *Admin) handleMwgCreate(args []string, cfg *config.Config) ports.ActionType {
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

	if err := a.manager.Update(func(cfg *config.Config) {
		cfg.MwordGroup[groupName] = &config.MwordGroup{
			Action:   action,
			Duration: duration,
			Enabled:  true,
			Words:    []string{},
		}
	}); err != nil {
		a.log.Error("Failed update config", err)
		return ErrFound
	}

	return Success
}

func (a *Admin) handleMwgSet(args []string, cfg *config.Config) ports.ActionType {
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

	if err := a.manager.Update(func(cfg *config.Config) {
		cfg.MwordGroup[groupName].Action = action
		cfg.MwordGroup[groupName].Duration = duration
	}); err != nil {
		a.log.Error("Failed update config", err)
		return ErrFound
	}

	return Success
}

func (a *Admin) handleMwgAdd(args []string, cfg *config.Config) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	words := strings.Split(args[1], ",")

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
			if err := a.manager.Update(func(cfg *config.Config) {
				cfg.MwordGroup[groupName].Words = append(cfg.MwordGroup[groupName].Words, trimmedWord)
			}); err != nil {
				a.log.Error("Failed update config", err)
				return ErrFound
			}
		}
	}

	return Success
}

func (a *Admin) handleMwgDel(args []string, cfg *config.Config) ports.ActionType {
	if len(args) < 2 {
		return NonParametr
	}

	groupName := args[0]
	target := args[1]

	group, exists := cfg.MwordGroup[groupName]
	if !exists {
		return NotFound
	}

	if target == "all" {
		if err := a.manager.Update(func(cfg *config.Config) {
			delete(cfg.MwordGroup, groupName)
		}); err != nil {
			a.log.Error("Failed update config", err)
			return ErrFound
		}

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

	if err := a.manager.Update(func(cfg *config.Config) {
		cfg.MwordGroup[groupName].Words = newWords
	}); err != nil {
		a.log.Error("Failed update config", err)
		return ErrFound
	}

	return Success
}

func (a *Admin) handleMwgOnOff(args []string, cfg *config.Config) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	groupName := args[0]

	if _, exists := cfg.MwordGroup[groupName]; !exists {
		return ErrNotFoundMwordGroup
	}

	if err := a.manager.Update(func(cfg *config.Config) {
		cfg.MwordGroup[groupName].Enabled = strings.HasPrefix(args[0], "on")
	}); err != nil {
		a.log.Error("Failed update config", err)
		return ErrFound
	}

	return Success
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
