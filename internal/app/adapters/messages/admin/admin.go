package admin

import (
	"fmt"
	"github.com/dlclark/regexp2"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/file_server"
	"twitchspam/internal/app/domain/regex"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

var (
	NotFoundCmd = &ports.AnswerType{
		Text:    []string{"команда не найдена!"},
		IsReply: true,
	}
	NonParametr = &ports.AnswerType{
		Text:    []string{"не указан один из параметров!"},
		IsReply: true,
	}
	UnknownError = &ports.AnswerType{
		Text:    []string{"неизвестная ошибка!"},
		IsReply: true,
	}
)

type Admin struct {
	log     logger.Logger
	manager *config.Manager
	stream  ports.StreamPort
	regexp  ports.RegexPort
	fs      ports.FileServerPort
	api     ports.APIPort
	aliases ports.AliasesPort
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, regexp *regex.Regex, api ports.APIPort, aliases ports.AliasesPort) *Admin {
	return &Admin{
		log:     log,
		manager: manager,
		stream:  stream,
		regexp:  regexp,
		fs:      file_server.New(),
		api:     api,
		aliases: aliases,
	}
}

var startApp = time.Now()

func (a *Admin) FindMessages(msg *ports.ChatMessage) *ports.AnswerType {
	if !(msg.Chatter.IsBroadcaster || msg.Chatter.IsMod) || !strings.HasPrefix(msg.Message.Text, "!am") {
		return nil
	}

	parts := strings.Fields(msg.Message.Text)
	if len(parts) < 2 {
		return NotFoundCmd
	}

	cmd, args := parts[1], parts[2:]
	if cmd == "ping" {
		return a.handlePing()
	}

	handlers := map[string]func(cfg *config.Config, cmd string, args []string) *ports.AnswerType{
		"on":     a.handleOnOff,
		"off":    a.handleOnOff,
		"info":   a.handleStatus,
		"say":    a.handleSay,
		"spam":   a.handleSpam,
		"as":     a.handleAntiSpam,
		"online": a.handleMode,
		"always": a.handleMode,
		"sim": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleSim(cfg, cmd, args, "default")
		},
		"msg": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMsg(cfg, cmd, args, "default")
		},
		"time": a.handleTime,
		"p": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handlePunishments(cfg, cmd, args, "default")
		},
		"rp": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleDurationResetPunishments(cfg, cmd, args, "default")
		},
		"mwlen": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMaxLen(cfg, cmd, args, "default")
		},
		"mwp": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMaxPunishment(cfg, cmd, args, "default")
		},
		"min_gap": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
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
		"alias": a.handleAliases,
		"mark": func(cfg *config.Config, cmd string, args []string) *ports.AnswerType {
			return a.handleMarkers(cfg, cmd, args, msg.Chatter.Username)
		},
		"cmd": a.handleCommand,
	}

	handler, ok := handlers[cmd]
	if !ok {
		a.log.Info("cmd", slog.String("cmd", cmd))
		return NotFoundCmd
	}

	var result *ports.AnswerType
	if err := a.manager.Update(func(cfg *config.Config) {
		result = handler(cfg, cmd, args)
	}); err != nil {
		a.log.Error("Failed update config", err, slog.String("msg", msg.Message.Text))
		return UnknownError
	}

	if result != nil {
		return result
	}
	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
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

func regexExists(list []*regexp2.Regexp, re *regexp2.Regexp) bool {
	for _, r := range list {
		if r.String() == re.String() {
			return true
		}
	}
	return false
}

func parsePunishment(punishment string, allowInherit bool) (config.Punishment, error) {
	punishment = strings.TrimSpace(punishment)
	if punishment == "-" {
		return config.Punishment{Action: "delete"}, nil
	}

	if allowInherit && punishment == "*" {
		return config.Punishment{Action: "inherit"}, nil
	}

	if punishment == "0" {
		return config.Punishment{Action: "ban"}, nil
	}

	duration, err := strconv.Atoi(punishment)
	if err != nil || duration < 1 || duration > 1209600 {
		return config.Punishment{}, fmt.Errorf("invalid timeout value")
	}

	return config.Punishment{Action: "timeout", Duration: duration}, nil
}

func formatPunishments(punishments []config.Punishment) []string {
	result := make([]string, 0, len(punishments))
	for _, p := range punishments {
		result = append(result, formatPunishment(p))
	}
	return result
}

func formatPunishment(punishment config.Punishment) string {
	var result string
	switch punishment.Action {
	case "delete":
		result = "удаление сообщения"
	case "timeout":
		result = fmt.Sprintf("таймаут (%d)", punishment.Duration)
	case "ban":
		result = "бан"
	}

	return result
}
