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

var startApp = time.Now()

func (a *Admin) FindMessages(irc *ports.IRCMessage) ports.ActionType {
	if !irc.IsMod || !strings.HasPrefix(irc.Text, "!am") {
		return None
	}

	parts := strings.Fields(irc.Text)
	if len(parts) < 2 {
		return NonParametr
	}

	cmd := parts[1]
	args := parts[2:]

	if cmd == "ping" {
		uptime := time.Since(startApp)

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		percent, _ := cpu.Percent(0, false)
		return ports.ActionType(fmt.Sprintf("бот работает %v • загрузка CPU %.2f%% • потребление ОЗУ %v MB", uptime.Truncate(time.Second), percent[0], m.Sys/1024/1024))
	}

	actionUpdate := None
	if err := a.manager.Update(func(cfg *config.Config) {
		switch cmd {
		case "on", "off":
			cfg.Spam.Enabled = cmd == "on"

		case "online", "always":
			cfg.Spam.Mode = cmd

		case "sim":
			if val, ok := parseFloatArg(args, 0, 1); ok {
				cfg.Spam.SimilarityThreshold = val
			} else {
				actionUpdate = ErrSimilarityThreshold
			}

		case "msg":
			if val, ok := parseIntArg(args, 2, 15); ok {
				cfg.Spam.MessageLimit = val
			} else {
				actionUpdate = ErrMessageLimit
			}

		case "time":
			if val, ok := parseIntArg(args, 1, 300); ok {
				cfg.Spam.CheckWindowSeconds = val
			} else {
				actionUpdate = ErrCheckWindowSeconds
			}

		case "to":
			var timeouts []int
			for _, s := range args {
				if t, err := strconv.Atoi(s); err == nil {
					timeouts = append(timeouts, t)
				} else {
					actionUpdate = NonValue
					break
				}
			}

			if len(timeouts) > 0 {
				cfg.Spam.Timeouts = timeouts
			} else {
				actionUpdate = NonValue
			}

		case "rto":
			if val, ok := parseIntArg(args, 1, 86400); ok {
				cfg.Spam.ResetTimeoutSeconds = val
			} else {
				actionUpdate = ErrResetTimeoutSeconds
			}

		case "mw":
			if val, ok := parseIntArg(args, 0, 500); ok {
				cfg.Spam.MaxWordLength = val
			} else {
				actionUpdate = ErrMaxWordLength
			}

		case "mwt":
			if val, ok := parseIntArg(args, 0, 1209600); ok {
				cfg.Spam.MaxWordTimeoutTime = val
			} else {
				actionUpdate = ErrMaxWordTimeoutTime
			}

		case "min_gap":
			if val, ok := parseIntArg(args, 0, 15); ok {
				cfg.Spam.MinGapMessages = val
			} else {
				actionUpdate = ErrMinGapMessages
			}

		case "da":
			if val, ok := parseIntArg(args, 0, 10); ok {
				cfg.Spam.DelayAutomod = val
			} else {
				actionUpdate = ErrDelayAutomod
			}

		case "reset":
			err := a.manager.Update(func(cfg *config.Config) {
				def := a.manager.GetDefault()
				cfg.Spam = def.Spam
			})
			if err != nil {
				a.log.Error("Failed update config", err, slog.String("msg", irc.Text))
				actionUpdate = ErrFound
			}

		case "add":
			for _, u := range strings.Split(args[0], ",") {
				u = strings.TrimSpace(u)
				if u == "" {
					continue
				}
				if !slices.Contains(cfg.Spam.WhitelistUsers, u) {
					cfg.Spam.WhitelistUsers = append(cfg.Spam.WhitelistUsers, u)
				}
			}

		case "del":
			users := strings.Split(args[0], ",")
			cfg.Spam.WhitelistUsers = slices.DeleteFunc(cfg.Spam.WhitelistUsers, func(w string) bool {
				for _, u := range users {
					if strings.TrimSpace(u) == w {
						return true
					}
				}
				return false
			})

		default:
			a.log.Info("cmd", slog.String("cmd", cmd))
			actionUpdate = NotFound
		}
	}); err != nil {
		a.log.Error("Failed update config", err, slog.String("msg", irc.Text))
		return None
	}

	if actionUpdate != None {
		return actionUpdate
	}

	return Success
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
