package admin

import (
	"fmt"
	"log/slog"
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

	root ports.Command
}

type CompositeCommand struct {
	subcommands map[string]ports.Command
	defaultCmd  ports.Command
	cursor      int
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, api ports.APIPort, template ports.TemplatePort, fs ports.FileServerPort, timers ports.TimersPort) *Admin {
	a := &Admin{
		log:      log,
		manager:  manager,
		stream:   stream,
		fs:       fs,
		api:      api,
		template: template,
		timers:   timers,
	}
	a.root = a.buildCommandTree()

	return a
}

var startApp = time.Now()

func (a *Admin) buildCommandTree() ports.Command {
	timer := &AddTimer{
		Timers: a.timers,
		Stream: a.stream,
		Api:    a.api,
	}

	return &CompositeCommand{
		subcommands: map[string]ports.Command{
			"ping":   &Ping{},
			"on":     &OnOff{enabled: true},
			"off":    &OnOff{enabled: false},
			"status": &Status{},
			"say":    &Say{},
			"spam":   &Spam{},
			"reset":  &Reset{manager: a.manager},
			"game":   &Category{stream: a.stream},
			"alias": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"add":  &AddAlias{template: a.template},
					"del":  &DelAlias{template: a.template},
					"list": &ListAlias{fs: a.fs},
				},
				cursor: 2,
			},
			"as": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":   &OnOffAntispam{enabled: true, typeSpam: "default"},
					"off":  &OnOffAntispam{enabled: false, typeSpam: "default"},
					"info": &InfoAntispam{template: a.template, fs: a.fs},
				},
				cursor: 2,
			},
			"online": &ModeAntispam{mode: "online"},
			"always": &ModeAntispam{mode: "always"},
			"time":   &TimeAntispam{template: a.template},
			"add":    &AddAntispam{},
			"del":    &DelAntispam{},
			"sim":    &SimAntispam{template: a.template, typeSpam: "default"},
			"msg":    &MsgAntispam{template: a.template, typeSpam: "default"},
			"p":      &PunishmentsAntispam{template: a.template, typeSpam: "default"},
			"rp":     &ResetPunishmentsAntispam{template: a.template, typeSpam: "default"},
			"mwlen":  &MaxLenAntispam{template: a.template, typeSpam: "default"},
			"mwp":    &MaxPunishmentAntispam{template: a.template, typeSpam: "default"},
			"mg":     &MinGapAntispam{template: a.template, typeSpam: "default"},
			"vip": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":    &OnOffAntispam{enabled: true, typeSpam: "vip"},
					"off":   &OnOffAntispam{enabled: false, typeSpam: "vip"},
					"sim":   &SimAntispam{template: a.template, typeSpam: "vip"},
					"msg":   &MsgAntispam{template: a.template, typeSpam: "vip"},
					"p":     &PunishmentsAntispam{template: a.template, typeSpam: "vip"},
					"rp":    &ResetPunishmentsAntispam{template: a.template, typeSpam: "vip"},
					"mwlen": &MaxLenAntispam{template: a.template, typeSpam: "vip"},
					"mwp":   &MaxPunishmentAntispam{template: a.template, typeSpam: "vip"},
					"mg":    &MinGapAntispam{template: a.template, typeSpam: "vip"},
				},
				cursor: 2,
			},
			"emote": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":    &OnOffAntispam{enabled: true, typeSpam: "emote"},
					"off":   &OnOffAntispam{enabled: false, typeSpam: "emote"},
					"msg":   &MsgAntispam{template: a.template, typeSpam: "emote"},
					"p":     &PunishmentsAntispam{template: a.template, typeSpam: "emote"},
					"rp":    &ResetPunishmentsAntispam{template: a.template, typeSpam: "emote"},
					"melen": &MaxLenAntispam{template: a.template, typeSpam: "emote"},
					"mep":   &MaxPunishmentAntispam{template: a.template, typeSpam: "emote"},
				},
				cursor: 2,
			},
			"mod": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":    &OnOffAutomod{enabled: true},
					"off":   &OnOffAutomod{enabled: false},
					"delay": &DelayAutomod{template: a.template},
				},
			},
			"cmd": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"add":  &AddCommand{},
					"del":  &DelCommand{},
					"list": &ListCommand{fs: a.fs},
					"timer": &CompositeCommand{
						subcommands: map[string]ports.Command{
							"on":  &OnOffCommandTimer{template: a.template, timers: a.timers, t: timer},
							"off": &OnOffCommandTimer{template: a.template, timers: a.timers, t: timer},
							"set": &SetCommandTimer{template: a.template, timers: a.timers, t: timer},
							"del": &DelCommandTimer{template: a.template, timers: a.timers},
						},
						defaultCmd: &AddCommandTimer{template: a.template, t: timer},
						cursor:     3,
					},
				},
				cursor: 2,
			},
			"ex": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"set":  &SetExcept{template: a.template, typeExcept: "default"},
					"del":  &DelExcept{typeExcept: "default"},
					"list": &ListExcept{template: a.template, fs: a.fs, typeExcept: "default"},
					"on":   &OnOffExcept{template: a.template, typeExcept: "default"},
					"off":  &OnOffExcept{template: a.template, typeExcept: "default"},
				},
				defaultCmd: &AddExcept{template: a.template, typeExcept: "default"},
				cursor:     2,
			},
			"emx": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"set":  &SetExcept{template: a.template, typeExcept: "emote"},
					"del":  &DelExcept{typeExcept: "emote"},
					"list": &ListExcept{template: a.template, fs: a.fs, typeExcept: "emote"},
					"on":   &OnOffExcept{template: a.template, typeExcept: "emote"},
					"off":  &OnOffExcept{template: a.template, typeExcept: "emote"},
				},
				defaultCmd: &AddExcept{template: a.template, typeExcept: "emote"},
				cursor:     2,
			},
			"mark": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"clear": &ClearMarker{stream: a.stream, username: ""},
					"list":  &ListMarker{stream: a.stream, api: a.api, fs: a.fs, username: ""},
				},
				defaultCmd: &AddMarker{log: a.log, stream: a.stream, api: a.api, username: ""},
				cursor:     2,
			},
			"mw": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"del":  &DelMword{template: a.template},
					"list": &ListMword{template: a.template, fs: a.fs},
				},
				defaultCmd: &AddMword{template: a.template},
				cursor:     2,
			},
			"mwg": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":     &OnOffMwordGroup{template: a.template},
					"off":    &OnOffMwordGroup{template: a.template},
					"create": &CreateMwordGroup{template: a.template},
					"set":    &SetMwordGroup{template: a.template},
					"add":    &AddMwordGroup{template: a.template},
					"del":    &DelMwordGroup{template: a.template},
					"list":   &ListMwordGroup{template: a.template, fs: a.fs},
				},
				cursor: 2,
			},
			"timers": &ListTimers{fs: a.fs},
		},
		cursor: 1,
	}
}

func (a *Admin) FindMessages(msg *ports.ChatMessage) *ports.AnswerType {
	if !(msg.Chatter.IsBroadcaster || msg.Chatter.IsMod) || !strings.HasPrefix(msg.Message.Text.Original, "!am") {
		return nil
	}

	words := msg.Message.Text.Words()
	if len(words) < 2 {
		return NotFoundCmd
	}

	// дикий костыль, не смотреть - есть шанс лишиться зрения
	markCmd := a.root.(*CompositeCommand).subcommands["mark"].(*CompositeCommand)
	markCmd.defaultCmd.(*AddMarker).username = msg.Chatter.Username
	markCmd.subcommands["clear"].(*ClearMarker).username = msg.Chatter.Username
	markCmd.subcommands["list"].(*ListMarker).username = msg.Chatter.Username

	var result *ports.AnswerType
	if err := a.manager.Update(func(cfg *config.Config) {
		result = a.root.Execute(cfg, &msg.Message.Text)
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

func (c *CompositeCommand) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if c.cursor >= len(words) {
		if c.defaultCmd != nil {
			return c.defaultCmd.Execute(cfg, text)
		}
		return NotFoundCmd
	}

	cmdName := words[c.cursor]
	if cmd, ok := c.subcommands[cmdName]; ok {
		return cmd.Execute(cfg, text)
	}

	if c.defaultCmd != nil {
		return c.defaultCmd.Execute(cfg, text)
	}
	return NotFoundCmd
}

func mergeSpamOptions(dst *config.SpamOptions, src map[string]bool) *config.SpamOptions {
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

func mergeTimerOptions(dst *config.TimerOptions, src map[string]bool) *config.TimerOptions {
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

func buildResponse(arg1 []string, nameArg1 string, arg2 []string, nameArg2, err string) *ports.AnswerType {
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

func buildList[T any](
	items map[string]T,
	prefix string,
	notFoundMsg string,
	formatFunc func(key string, value T) string,
	fs ports.FileServerPort,
) *ports.AnswerType {
	if len(items) == 0 {
		return &ports.AnswerType{
			Text:    []string{notFoundMsg},
			IsReply: true,
		}
	}

	var parts []string
	for key, value := range items {
		parts = append(parts, formatFunc(key, value))
	}

	msg := prefix + ":\n" + strings.Join(parts, "\n")

	key, err := fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}

	return &ports.AnswerType{
		Text:    []string{fs.GetURL(key)},
		IsReply: true,
	}
}
