package admin

import (
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/domain/regex"
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
	regexp  ports.RegexPort
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, regexp *regex.Regex) *Admin {
	return &Admin{
		log:     log,
		manager: manager,
		stream:  stream,
		regexp:  regexp,
	}
}

type cmdHandler func(cfg *config.Config, cmd string, args []string) ports.ActionType

var startApp = time.Now()

func (a *Admin) FindMessages(msg *ports.ChatMessage) ports.ActionType {
	if !(msg.Chatter.IsBroadcaster || msg.Chatter.IsMod) || !strings.HasPrefix(msg.Message.Text, "!am") {
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
		"on":     a.handleOnOff,
		"off":    a.handleOnOff,
		"spam":   a.handleSpam,
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
		"mwlen": func(cfg *config.Config, cmd string, args []string) ports.ActionType {
			return a.handleMwLen(cfg, cmd, args, "default")
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
		"mw":    a.handleMw,
		"ex":    a.handleEx,
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

func regexExists(list []*regexp.Regexp, re *regexp.Regexp) bool {
	for _, r := range list {
		if r.String() == re.String() {
			return true
		}
	}
	return false
}
