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
	"twitchspam/config"
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
)

type Admin struct {
	log     logger.Logger
	manager *config.Manager
}

func New(log logger.Logger, manager *config.Manager) *Admin {
	return &Admin{
		log:     log,
		manager: manager,
	}
}

type cmdHandler func(cfg *config.Config, args []string, cmd string) ports.ActionType

var startApp = time.Now()

func (a *Admin) FindMessages(irc *ports.IRCMessage) ports.ActionType {
	if !irc.IsMod || !strings.HasPrefix(irc.Text, "!am") {
		return None
	}

	parts := strings.Fields(irc.Text)
	if len(parts) < 2 {
		return NonParametr
	}

	cmd, args := parts[1], parts[2:]
	if cmd == "ping" {
		return a.handlePing()
	}

	handlers := map[string]cmdHandler{
		"on":      a.handleOnOff,
		"off":     a.handleOnOff,
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
		a.log.Error("Failed update config", err, slog.String("msg", irc.Text))
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

func (a *Admin) handleOnOff(cfg *config.Config, _ []string, cmd string) ports.ActionType {
	cfg.Spam.SettingsDefault.Enabled = cmd == "on"
	return None
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
	for _, str := range args {
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
	for _, u := range strings.Split(args[0], ",") {
		u = strings.TrimSpace(u)
		if u != "" && !slices.Contains(cfg.Spam.WhitelistUsers, u) {
			cfg.Spam.WhitelistUsers = append(cfg.Spam.WhitelistUsers, u)
		}
	}
	return None
}

func (a *Admin) handleDel(cfg *config.Config, args []string, _ string) ports.ActionType {
	users := strings.Split(args[0], ",")
	cfg.Spam.WhitelistUsers = slices.DeleteFunc(cfg.Spam.WhitelistUsers, func(w string) bool {
		for _, u := range users {
			if strings.TrimSpace(u) == w {
				return true
			}
		}
		return false
	})
	return None
}

func (a *Admin) handleVip(cfg *config.Config, args []string, _ string) ports.ActionType {
	if len(args) < 1 {
		return NonParametr
	}
	vipCmd, vipArgs := args[0], args[1:]

	handlers := map[string]func([]string, *config.SpamSettings) ports.ActionType{
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
