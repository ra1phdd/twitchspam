package admin

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"
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
	log      logger.Logger
	manager  *config.Manager
	stream   ports.StreamPort
	fs       ports.FileServerPort
	api      ports.APIPort
	template ports.TemplatePort
	timers   ports.TimersPort
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, api ports.APIPort, template ports.TemplatePort, fs ports.FileServerPort, timers ports.TimersPort) *Admin {
	return &Admin{
		log:      log,
		manager:  manager,
		stream:   stream,
		fs:       fs,
		api:      api,
		template: template,
		timers:   timers,
	}
}

var startApp = time.Now()

func (a *Admin) FindMessages(msg *ports.ChatMessage) *ports.AnswerType {
	if !(msg.Chatter.IsBroadcaster || msg.Chatter.IsMod) || !strings.HasPrefix(msg.Message.Text.Original, "!am") {
		return nil
	}

	words := msg.Message.Text.Words()
	if len(words) < 2 {
		return NotFoundCmd
	}

	cmd := words[1]
	if cmd == "ping" {
		return a.handlePing()
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"on": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleOnOff(cfg, true)
		},
		"off": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleOnOff(cfg, false)
		},
		"status": a.handleStatus,
		"say":    a.handleSay,
		"spam":   a.handleSpam,
		"as":     a.handleAntiSpam,
		"online": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMode(cfg, "online")
		},
		"always": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMode(cfg, "always")
		},
		"sim": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleSim(cfg, text, "default")
		},
		"msg": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMsg(cfg, text, "default")
		},
		"time": a.handleTime,
		"p": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handlePunishments(cfg, text, "default")
		},
		"rp": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleDurationResetPunishments(cfg, text, "default")
		},
		"mwlen": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMaxLen(cfg, text, "default")
		},
		"mwp": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMaxPunishment(cfg, text, "default")
		},
		"min_gap": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMinGap(cfg, text, "default")
		},
		"da":    a.handleDelayAutomod,
		"reset": a.handleReset,
		"add":   a.handleAdd,
		"del":   a.handleDel,
		"vip":   a.handleVip,
		"emote": a.handleEmote,
		"game":  a.handleCategory,
		"mwg":   a.handleMwg,
		"mw":    a.handleMw,
		"ex":    a.handleExcept,
		"alias": a.handleAliases,
		"mark": func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
			return a.handleMarkers(cfg, text, msg.Chatter.Username)
		},
		"cmd":    a.handleCommand,
		"timers": a.handleTimersList,
	}

	handler, ok := handlers[cmd]
	if !ok {
		a.log.Info("cmd", slog.String("cmd", cmd))
		return NotFoundCmd
	}

	var result *ports.AnswerType
	if err := a.manager.Update(func(cfg *config.Config) {
		result = handler(cfg, &msg.Message.Text)
	}); err != nil {
		a.log.Error("Failed update config", err, slog.String("msg", msg.Message.Text.Original))
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

func parseIntArg(valStr string, min, max int) (int, bool) {
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, false
	}
	if (min != -1 && val < min) || (max != -1 && val > max) {
		return 0, false
	}
	return val, true
}

func parseFloatArg(valStr string, min, max float64) (float64, bool) {
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil || val < min || val > max {
		return 0, false
	}

	return math.Round(val*100) / 100, true
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

func (a *Admin) mergeSpamOptions(dst *config.SpamOptions, src map[string]bool) *config.SpamOptions {
	if dst == nil {
		dst = &config.SpamOptions{}
	}

	if _, ok := src["-nofirst"]; ok {
		dst.IsFirst = false
	}

	if _, ok := src["-first"]; ok {
		dst.IsFirst = true
	}

	if _, ok := src["-nosub"]; ok {
		dst.NoSub = true
	}

	if _, ok := src["-sub"]; ok {
		dst.NoSub = false
	}

	if _, ok := src["-novip"]; ok {
		dst.NoVip = true
	}

	if _, ok := src["-vip"]; ok {
		dst.NoVip = false
	}

	if _, ok := src["-norepeat"]; ok {
		dst.NoRepeat = true
	}

	if _, ok := src["-repeat"]; ok {
		dst.NoRepeat = false
	}

	if _, ok := src["-oneword"]; ok {
		dst.OneWord = true
	}

	if _, ok := src["-noontains"]; ok {
		dst.Contains = false
	}

	if _, ok := src["-contains"]; ok {
		dst.Contains = true
	}

	return dst
}

func (a *Admin) mergeTimerOptions(dst *config.TimerOptions, src map[string]bool) *config.TimerOptions {
	if dst == nil {
		dst = &config.TimerOptions{}
	}

	if _, ok := src["-noa"]; ok {
		dst.IsAnnounce = false
	}

	if _, ok := src["-a"]; ok {
		dst.IsAnnounce = true
	}

	if _, ok := src["-online"]; ok {
		dst.IsAlways = false
	}

	if _, ok := src["-always"]; ok {
		dst.IsAlways = true
	}

	return dst
}

func (a *Admin) buildResponse(arg1 []string, nameArg1 string, arg2 []string, nameArg2, err string) *ports.AnswerType {
	var msgParts []string
	if len(arg1) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%s: %s", nameArg1, strings.Join(arg1, ", ")))
	}
	if len(arg2) > 0 {
		msgParts = append(msgParts, fmt.Sprintf("%s: %s", nameArg2, strings.Join(arg2, ", ")))
	}

	if len(msgParts) == 0 {
		return &ports.AnswerType{
			Text:    []string{err + "!"},
			IsReply: true,
		}
	}

	if len(arg2) == 0 {
		return nil
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "!"},
		IsReply: true,
	}
}
