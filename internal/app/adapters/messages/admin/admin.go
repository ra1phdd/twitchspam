package admin

import (
	"fmt"
	"log/slog"
	"regexp"
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
			"say":    &Say{re: regexp.MustCompile(`(?i)^!am\s+say\s+(.+)$`)},
			"spam":   &Spam{re: regexp.MustCompile(`(?i)^!am\s+spam\s+(\d+)\s+(.+)$`)},
			"reset":  &Reset{manager: a.manager},
			"game":   &Game{re: regexp.MustCompile(`(?i)^!am\s+game\s+(.+)$`), stream: a.stream},
			"al": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"add":  &AddAlias{re: regexp.MustCompile(`(?i)^!am\s+al(?:\s+add)?\s+(.+)\s+from\s+(.+)$`), template: a.template},
					"del":  &DelAlias{re: regexp.MustCompile(`(?i)^!am\s+al\s+del\s+(.+)$`), template: a.template},
					"list": &ListAlias{fs: a.fs},
				},
				defaultCmd: &AddAlias{re: regexp.MustCompile(`(?i)^!am\s+al(?:\s+add)?\s+(.+)\s+from\s+(.+)$`), template: a.template},
				cursor:     2,
			},
			"alg": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":     &OnOffAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+on\s+(\S+)$`), template: a.template},
					"off":    &OnOffAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+off\s+(\S+)$`), template: a.template},
					"create": &CreateAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+create\s+(\S+)\s+(.+)$`), template: a.template},
					"add":    &AddAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+add\s+(\S+)\s+(.+)$`), template: a.template},
					"del":    &DelAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+del\s+(\S+)(?:\s+(.+))?$`), template: a.template},
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
			"time":   &TimeAntispam{re: regexp.MustCompile(`(?i)^!am\s+time\s+([0-9]+)$`), template: a.template},
			"add":    &AddAntispam{re: regexp.MustCompile(`(?i)^!am\s+add\s+(.+)$`)},
			"del":    &DelAntispam{re: regexp.MustCompile(`(?i)^!am\s+del\s+(.+)$`)},
			"sim":    &SimAntispam{re: regexp.MustCompile(`(?i)^!am\s+sim\s+([0-9]*\.?[0-9]+)$`), template: a.template, typeSpam: "default"},
			"msg":    &MsgAntispam{re: regexp.MustCompile(`(?i)^!am\s+msg\s+([0-9]+)$`), template: a.template, typeSpam: "default"},
			"p":      &PunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+p\s+(.+)$`), template: a.template, typeSpam: "default"},
			"rp":     &ResetPunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+rp\s+([0-9]+)$`), template: a.template, typeSpam: "default"},
			"mlen":   &MaxLenAntispam{re: regexp.MustCompile(`(?i)^!am\s+mlen\s+([0-9]+)$`), template: a.template, typeSpam: "default"},
			"mp":     &MaxPunishmentAntispam{re: regexp.MustCompile(`(?i)^!am\s+mp\s+(.+)$`), template: a.template, typeSpam: "default"},
			"mg":     &MinGapAntispam{re: regexp.MustCompile(`(?i)^!am\s+mg\s+([0-9]+)$`), template: a.template, typeSpam: "default"},
			"vip": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":   &OnOffAntispam{enabled: true, typeSpam: "vip"},
					"off":  &OnOffAntispam{enabled: false, typeSpam: "vip"},
					"sim":  &SimAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+sim\s+([0-9]*\.?[0-9]+)$`), template: a.template, typeSpam: "vip"},
					"msg":  &MsgAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+msg\s+([0-9]+)$`), template: a.template, typeSpam: "vip"},
					"p":    &PunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+p\s+(.+)$`), template: a.template, typeSpam: "vip"},
					"rp":   &ResetPunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+rp\s+([0-9]+)$`), template: a.template, typeSpam: "vip"},
					"mlen": &MaxLenAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+mlen\s+([0-9]+)$`), template: a.template, typeSpam: "vip"},
					"mp":   &MaxPunishmentAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+mp\s+(.+)$`), template: a.template, typeSpam: "vip"},
					"mg":   &MinGapAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+mg\s+([0-9]+)$`), template: a.template, typeSpam: "vip"},
				},
				cursor: 2,
			},
			"emote": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":   &OnOffAntispam{enabled: true, typeSpam: "emote"},
					"off":  &OnOffAntispam{enabled: false, typeSpam: "emote"},
					"msg":  &MsgAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+msg\s+([0-9]+)$`), template: a.template, typeSpam: "emote"},
					"p":    &PunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+p\s+(.+)$`), template: a.template, typeSpam: "emote"},
					"rp":   &ResetPunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+rp\s+([0-9]+)$`), template: a.template, typeSpam: "emote"},
					"mlen": &MaxLenAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+mlen\s+([0-9]+)$`), template: a.template, typeSpam: "emote"},
					"mp":   &MaxPunishmentAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+mp\s+(.+)$`), template: a.template, typeSpam: "emote"},
				},
				cursor: 2,
			},
			"mod": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":    &OnOffAutomod{enabled: true},
					"off":   &OnOffAutomod{enabled: false},
					"delay": &DelayAutomod{re: regexp.MustCompile(`(?i)^!am\s+mod\s+delay\s+(\d+)$`), template: a.template},
				},
			},
			"cmd": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"list":    &ListCommand{fs: a.fs},
					"add":     &AddCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd(?:\s+add)?\s+(.+)$`)},
					"del":     &DelCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+del\s+(.+)$`)},
					"aliases": &AliasesCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+aliases\s+(.+)$`)},
					"timer": &CompositeCommand{
						subcommands: map[string]ports.Command{
							"on":  &OnOffCommandTimer{template: a.template, timers: a.timers, t: timer},
							"off": &OnOffCommandTimer{template: a.template, timers: a.timers, t: timer},
							"add": &AddCommandTimer{template: a.template, t: timer},
							"set": &SetCommandTimer{template: a.template, timers: a.timers, t: timer},
							"del": &DelCommandTimer{template: a.template, timers: a.timers},
						},
						defaultCmd: &AddCommandTimer{template: a.template, t: timer},
						cursor:     3,
					},
					"lim": &CompositeCommand{
						subcommands: map[string]ports.Command{
							"add": &AddCommandLimiter{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+lim(?:\s+add)?\s+(\d+)\s+(\d+)\s+(.+)$`)},
							"set": &SetCommandLimiter{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+lim\s+set\s+(\d+)\s+(\d+)\s+(.+)$`)},
							"del": &DelCommandLimiter{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+lim\s+del\s+(.+)$`)},
						},
						defaultCmd: &AddCommandLimiter{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+lim(?:\s+add)?\s+(\d+)\s+(\d+)\s+(.+)$`)},
						cursor:     3,
					},
				},
				defaultCmd: &AddCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd(?:\s+add)?\s+(.+)$`)},
				cursor:     2,
			},
			"ex": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"add":  &AddExcept{template: a.template, typeExcept: "default"},
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
					"add":  &AddExcept{template: a.template, typeExcept: "emote"},
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
					"add":   &AddMarker{log: a.log, stream: a.stream, api: a.api, username: ""},
					"clear": &ClearMarker{stream: a.stream, username: ""},
					"list":  &ListMarker{stream: a.stream, api: a.api, fs: a.fs, username: ""},
				},
				defaultCmd: &AddMarker{log: a.log, stream: a.stream, api: a.api, username: ""},
				cursor:     2,
			},
			"mw": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"add":  &AddMword{template: a.template},
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

var Success = &ports.AnswerType{
	Text:    []string{"успешно!"},
	IsReply: true,
}

func (a *Admin) FindMessages(msg *ports.ChatMessage) *ports.AnswerType {
	if !(msg.Chatter.IsBroadcaster || msg.Chatter.IsMod) || !strings.HasPrefix(msg.Message.Text.Original, "!am") {
		return nil
	}

	words := msg.Message.Text.WordsLower()
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
	return Success
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

func mergeTimerOptions(dst config.TimerOptions, src map[string]bool) config.TimerOptions {
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
