package admin

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

var (
	success = &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
	notFoundCmd = &ports.AnswerType{
		Text:    []string{"команда не найдена!"},
		IsReply: true,
	}
	nonParametr = &ports.AnswerType{
		Text:    []string{"не указан один из параметров!"},
		IsReply: true,
	}
	unknownError = &ports.AnswerType{
		Text:    []string{"неизвестная ошибка!"},
		IsReply: true,
	}
	invalidRegex = &ports.AnswerType{
		Text:    []string{"неверное регулярное выражение!"},
		IsReply: true,
	}
	errorPunishmentParse = &ports.AnswerType{
		Text:    []string{"не удалось распарсить наказания!"},
		IsReply: true,
	}
	errorPunishmentCopy = &ports.AnswerType{
		Text:    []string{"невозможно скопировать наказания!"},
		IsReply: true,
	}
	invalidPunishmentFormat = &ports.AnswerType{
		Text:    []string{"наказания не указаны!"},
		IsReply: true,
	}
	invalidMessageLimitFormat = &ports.AnswerType{
		Text:    []string{"лимит сообщений не указан или указан неверно!"},
		IsReply: true,
	}
	invalidMessageLimitValue = &ports.AnswerType{
		Text:    []string{"значение лимита сообщений должно быть от 2 до 15!"},
		IsReply: true,
	}
	aliasDenied = &ports.AnswerType{
		Text:    []string{"нельзя добавить алиас на эту команду!"},
		IsReply: true,
	}
	incorrectSyntax = &ports.AnswerType{
		Text:    []string{"некорректный синтаксис!"},
		IsReply: true,
	}
	notFoundAliasGroup = &ports.AnswerType{
		Text:    []string{"группа алиасов не найдена!"},
		IsReply: true,
	}
	existsAliasGroup = &ports.AnswerType{
		Text:    []string{"группа алиасов уже существует!"},
		IsReply: true,
	}
	invalidValueRequest = &ports.AnswerType{
		Text:    []string{"не указано корректное количество запросов!"},
		IsReply: true,
	}
	invalidValueInterval = &ports.AnswerType{
		Text:    []string{"не указан корректный интервал!"},
		IsReply: true,
	}
	invalidValueRepeats = &ports.AnswerType{
		Text:    []string{"кол-во повторов не указано или указано неверно!"},
		IsReply: true,
	}
	streamOff = &ports.AnswerType{
		Text:    []string{"стрим выключен!"},
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
	messages ports.StorePort[storage.Message]

	root ports.Command
}

type CompositeCommand struct {
	subcommands map[string]ports.Command
	defaultCmd  ports.Command
	cursor      int
}

func New(log logger.Logger, manager *config.Manager, stream ports.StreamPort, api ports.APIPort, template ports.TemplatePort, fs ports.FileServerPort, timers ports.TimersPort, messages ports.StorePort[storage.Message]) *Admin {
	a := &Admin{
		log:      log,
		manager:  manager,
		stream:   stream,
		fs:       fs,
		api:      api,
		template: template,
		timers:   timers,
		messages: messages,
	}
	a.root = a.buildCommandTree()

	return a
}

var startApp = time.Now()

func (a *Admin) FindMessages(msg *domain.ChatMessage) *ports.AnswerType {
	if (!msg.Chatter.IsBroadcaster && !msg.Chatter.IsMod) || !strings.HasPrefix(msg.Message.Text.Text(domain.LowerOption), "!am") {
		return nil
	}

	words := msg.Message.Text.Words(domain.LowerOption)
	if len(words) < 2 {
		return notFoundCmd
	}

	// дикий костыль, не смотреть - есть шанс лишиться зрения
	markCmd := a.root.(*CompositeCommand).subcommands["mark"].(*CompositeCommand)
	markCmd.defaultCmd.(*AddMarker).username = msg.Chatter.Username
	markCmd.subcommands["add"].(*AddMarker).username = msg.Chatter.Username
	markCmd.subcommands["clear"].(*ClearMarker).username = msg.Chatter.Username
	markCmd.subcommands["list"].(*ListMarker).username = msg.Chatter.Username

	var result *ports.AnswerType
	if err := a.manager.Update(func(cfg *config.Config) {
		result = a.root.Execute(cfg, &msg.Message.Text)
	}); err != nil {
		a.log.Error("Failed update config", err, slog.String("msg", msg.Message.Text.Text()))
		return unknownError
	}

	if result != nil {
		return result
	}
	return success
}

func (c *CompositeCommand) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	words := text.Words(domain.LowerOption)
	if c.cursor >= len(words) {
		if c.defaultCmd != nil {
			return c.defaultCmd.Execute(cfg, text)
		}
		return notFoundCmd
	}

	cmdName := words[c.cursor]
	if cmd, ok := c.subcommands[cmdName]; ok {
		return cmd.Execute(cfg, text)
	}

	if c.defaultCmd != nil {
		return c.defaultCmd.Execute(cfg, text)
	}
	return notFoundCmd
}

func (a *Admin) buildCommandTree() ports.Command {
	timer := &AddTimer{
		Cfg:    a.manager.Get(),
		Timers: a.timers,
		Stream: a.stream,
		Api:    a.api,
	}

	return &CompositeCommand{
		subcommands: map[string]ports.Command{
			"ping":   &Ping{},
			"on":     &OnOff{enabled: true, template: a.template},
			"off":    &OnOff{enabled: false, template: a.template},
			"status": &Status{},
			"say":    &Say{re: regexp.MustCompile(`(?i)^!am\s+say\s+(.+)$`)},
			"spam":   &Spam{re: regexp.MustCompile(`(?i)^!am\s+spam\s+(.+)\s+(.+)$`)},
			"reset":  &Reset{manager: a.manager},
			"game":   &Game{re: regexp.MustCompile(`(?i)^!am\s+game\s+(.+)$`), stream: a.stream},
			"as": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":      &OnOffAntispam{enabled: true, typeSpam: "default", template: a.template},
					"off":     &OnOffAntispam{enabled: false, typeSpam: "default", template: a.template},
					"info":    &InfoAntispam{template: a.template, fs: a.fs},
					"online":  &ModeAntispam{mode: config.OnlineMode},
					"offline": &ModeAntispam{mode: config.OfflineMode},
					"always":  &ModeAntispam{mode: config.AlwaysMode},
				},
				defaultCmd: &PauseAntispam{re: regexp.MustCompile(`(?i)^!am\s+as\s+(.+)$`), template: a.template},
				cursor:     2,
			},
			"add":  &AddAntispam{re: regexp.MustCompile(`(?i)^!am\s+add\s+(.+)$`)},
			"del":  &DelAntispam{re: regexp.MustCompile(`(?i)^!am\s+del\s+(.+)$`)},
			"sim":  &SimAntispam{re: regexp.MustCompile(`(?i)^!am\s+sim\s+(.+)$`), template: a.template, messages: a.messages, typeSpam: "default"},
			"msg":  &MsgAntispam{re: regexp.MustCompile(`(?i)^!am\s+msg\s+(.+)$`), template: a.template, messages: a.messages, typeSpam: "default"},
			"p":    &PunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+p\s+(.+)$`), template: a.template, typeSpam: "default"},
			"rp":   &ResetPunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+rp\s+(.+)$`), template: a.template, typeSpam: "default"},
			"mlen": &MaxLenAntispam{re: regexp.MustCompile(`(?i)^!am\s+mlen\s+(.+)$`), template: a.template, typeSpam: "default"},
			"mp":   &MaxPunishmentAntispam{re: regexp.MustCompile(`(?i)^!am\s+mp\s+(.+)$`), template: a.template, typeSpam: "default"},
			"mg":   &MinGapAntispam{re: regexp.MustCompile(`(?i)^!am\s+mg\s+(.+)$`), template: a.template, messages: a.messages, typeSpam: "default"},
			"nuke": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"stop": &NukeStop{template: a.template},
				},
				defaultCmd: &Nuke{re: regexp.MustCompile(`(?i)^!am nuke(?:\s+(\S+))?(?:\s+(\S+))?(?:\s+(\S+))?\s+(.+)$`),
					reWords: regexp.MustCompile(`(?i)r'(.*?)'|r"(.*?)"|'(.*?)'|"(.*?)"|([^,'"\s]+)`),
					log:     a.log, api: a.api, template: a.template, stream: a.stream, messages: a.messages},
				cursor: 2,
			},
			"mod": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":    &OnOffAutomod{enabled: true},
					"off":   &OnOffAutomod{enabled: false},
					"delay": &DelayAutomod{re: regexp.MustCompile(`(?i)^!am\s+mod\s+delay\s+(.+)$`), template: a.template},
				},
				cursor: 2,
			},
			"vip": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":   &OnOffAntispam{enabled: true, typeSpam: "vip", template: a.template},
					"off":  &OnOffAntispam{enabled: false, typeSpam: "vip", template: a.template},
					"sim":  &SimAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+sim\s+(.+)$`), template: a.template, messages: a.messages, typeSpam: "vip"},
					"msg":  &MsgAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+msg\s+(.+)$`), template: a.template, messages: a.messages, typeSpam: "vip"},
					"p":    &PunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+p\s+(.+)$`), template: a.template, typeSpam: "vip"},
					"rp":   &ResetPunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+rp\s+(.+)$`), template: a.template, typeSpam: "vip"},
					"mlen": &MaxLenAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+mlen\s+(.+)$`), template: a.template, typeSpam: "vip"},
					"mp":   &MaxPunishmentAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+mp\s+(.+)$`), template: a.template, typeSpam: "vip"},
					"mg":   &MinGapAntispam{re: regexp.MustCompile(`(?i)^!am\s+vip\s+mg\s+(.+)$`), template: a.template, messages: a.messages, typeSpam: "vip"},
				},
				cursor: 2,
			},
			"emote": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":   &OnOffAntispam{enabled: true, typeSpam: "emote", template: a.template},
					"off":  &OnOffAntispam{enabled: false, typeSpam: "emote", template: a.template},
					"msg":  &MsgAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+msg\s+(.+)$`), template: a.template, messages: a.messages, typeSpam: "emote"},
					"p":    &PunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+p\s+(.+)$`), template: a.template, typeSpam: "emote"},
					"rp":   &ResetPunishmentsAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+rp\s+(.+)$`), template: a.template, typeSpam: "emote"},
					"mlen": &MaxLenAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+mlen\s+(.+)$`), template: a.template, typeSpam: "emote"},
					"mp":   &MaxPunishmentAntispam{re: regexp.MustCompile(`(?i)^!am\s+emote\s+mp\s+(.+)$`), template: a.template, typeSpam: "emote"},
					"ex": &CompositeCommand{
						subcommands: map[string]ports.Command{
							"list": &ListExcept{template: a.template, fs: a.fs, typeExcept: "emote"},
							"add":  &AddExcept{re: regexp.MustCompile(`(?i)^!am\s+emote\s+ex(?:\s+add)?\s+(\d+)\s+(\S+)\s*(?:\s*(re)\s+(\S+)\s+(.+)|\s+(.+))$`), template: a.template, typeExcept: "emote"},
							"set":  &SetExcept{re: regexp.MustCompile(`(?i)^!am\s+emote\s+ex\s+set(?:\s+(ml|p)\s+([^ ]+))?\s+(.+)$`), template: a.template, typeExcept: "emote"},
							"del":  &DelExcept{re: regexp.MustCompile(`(?i)^!am\s+emote\s+ex\s+del\s+(.+)$`), typeExcept: "emote"},
							"on":   &OnOffExcept{re: regexp.MustCompile(`(?i)^!am\s+emote\s+ex\s+(on)\s+(.+)$`), template: a.template, typeExcept: "emote"},
							"off":  &OnOffExcept{re: regexp.MustCompile(`(?i)^!am\s+emote\s+ex\s+(off)\s+(.+)$`), template: a.template, typeExcept: "emote"},
						},
						defaultCmd: &AddExcept{re: regexp.MustCompile(`(?i)^!am\s+emote\s+ex(?:\s+add)?\s+(\d+)\s+(\S+)\s*(?:\s*(re)\s+(\S+)\s+(.+)|\s+(.+))$`), template: a.template, typeExcept: "emote"},
						cursor:     3,
					},
				},
				cursor: 2,
			},
			"mark": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"add":   &AddMarker{re: regexp.MustCompile(`(?i)^!am\s+mark(?:\s+add)?\s+(\S+)$`), log: a.log, stream: a.stream, api: a.api},
					"clear": &ClearMarker{re: regexp.MustCompile(`(?i)^!am\s+mark\s+clear(?:\s+(\S+))?$`), stream: a.stream},
					"list":  &ListMarker{re: regexp.MustCompile(`(?i)^!am\s+mark\s+list(?:\s+(\S+))?$`), stream: a.stream, api: a.api, fs: a.fs},
				},
				defaultCmd: &AddMarker{re: regexp.MustCompile(`(?i)^!am\s+mark(?:\s+add)?\s+(\S+)$`), log: a.log, stream: a.stream, api: a.api},
				cursor:     2,
			},
			"ex": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"list": &ListExcept{template: a.template, fs: a.fs, typeExcept: "default"},
					"add":  &AddExcept{re: regexp.MustCompile(`(?i)^!am\s+ex(?:\s+add)?\s+(\d+)\s+(\S+)\s*(?:\s*(re)\s+(\S+)\s+(.+)|\s+(.+))$`), template: a.template, typeExcept: "default"},
					"set":  &SetExcept{re: regexp.MustCompile(`(?i)^!am\s+ex\s+set(?:\s+(ml|p)\s+([^ ]+))?\s+(.+)$`), template: a.template, typeExcept: "default"},
					"del":  &DelExcept{re: regexp.MustCompile(`(?i)^!am\s+ex\s+del\s+(.+)$`), typeExcept: "default"},
					"on":   &OnOffExcept{re: regexp.MustCompile(`(?i)^!am\s+ex\s+(on)\s+(.+)$`), template: a.template, typeExcept: "default"},
					"off":  &OnOffExcept{re: regexp.MustCompile(`(?i)^!am\s+ex\s+(off)\s+(.+)$`), template: a.template, typeExcept: "default"},
				},
				defaultCmd: &AddExcept{re: regexp.MustCompile(`(?i)^!am\s+ex(?:\s+add)?\s+(\d+)\s+(\S+)\s*(?:\s*(re)\s+(\S+)\s+(.+)|\s+(.+))$`), template: a.template, typeExcept: "default"},
				cursor:     2,
			},
			"mw": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"add":  &AddMword{re: regexp.MustCompile(`(?i)^!am\s+mw(?:\s+add)?\s+(\S+)\s*(?:(re)\s+(\S+)\s+(.+)|(.+))$`), template: a.template},
					"set":  &SetMword{re: regexp.MustCompile(`(?i)^!am\s+mw\s+set\s+(?:([^ ]+)\s+)?(.+)$`), template: a.template},
					"del":  &DelMword{re: regexp.MustCompile(`(?i)^!am\s+mw\s+del\s+(.+)$`), template: a.template},
					"list": &ListMword{template: a.template, fs: a.fs},
				},
				defaultCmd: &AddMword{re: regexp.MustCompile(`(?i)^!am\s+mw(?:\s+add)?\s+(\S+)\s*(?:(re)\s+(\S+)\s+(.+)|(.+))$`), template: a.template},
				cursor:     2,
			},
			"mwg": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":     &OnOffMwordGroup{re: regexp.MustCompile(`(?i)^!am\s+mwg\s+(on)\s+(.+)$`), template: a.template},
					"off":    &OnOffMwordGroup{re: regexp.MustCompile(`(?i)^!am\s+mwg\s+(off)\s+(.+)$`), template: a.template},
					"create": &CreateMwordGroup{re: regexp.MustCompile(`(?i)^!am\s+mwg\s+create\s+(\S+)\s+(.+)$`), template: a.template},
					"set":    &SetMwordGroup{re: regexp.MustCompile(`(?i)^!am\s+mwg\s+set\s+(\S+)(?:\s+(.+))?$`), template: a.template},
					"add":    &AddMwordGroup{re: regexp.MustCompile(`(?i)^!am\s+mwg(?:\s+add)?\s+(\S+)\s*(?:(re)\s+(\S+)\s+(.+)|(.+))$`), template: a.template},
					"del":    &DelMwordGroup{re: regexp.MustCompile(`(?i)^!am\s+mwg\s+del\s+(\S+)(?:\s+(.+))?$`), template: a.template},
					"list":   &ListMwordGroup{template: a.template, fs: a.fs},
				},
				cursor: 2,
			},
			"timers": &ListTimers{fs: a.fs},
			"cmd": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"list":    &ListCommand{fs: a.fs},
					"add":     &AddCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd(?:\s+add)?\s+(\S+)\s+(.+)$`), template: a.template},
					"set":     &SetCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+set\s+(\S+)(?:\s+(.+))?$`), template: a.template},
					"del":     &DelCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+del\s+(.+)$`)},
					"aliases": &AliasesCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+aliases\s+(.+)$`)},
					"timer": &CompositeCommand{
						subcommands: map[string]ports.Command{
							"on":  &OnOffCommandTimer{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+timer\s+(on)\s+(.+)$`), template: a.template, timers: a.timers, t: timer},
							"off": &OnOffCommandTimer{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+timer\s+(off)\s+(.+)$`), template: a.template, timers: a.timers, t: timer},
							"add": &AddCommandTimer{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+timer(?:\s+add)?\s+(.+)\s+(.+)\s+(.+)$`), template: a.template, t: timer},
							"set": &SetCommandTimer{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+timer\s+set(?:\s+(\d*)\s+(\d*)\s+)?(.+)$`), template: a.template, timers: a.timers, t: timer},
							"del": &DelCommandTimer{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+timer\s+del\s+(.+)$`), template: a.template, timers: a.timers},
						},
						defaultCmd: &AddCommandTimer{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+timer(?:\s+add)?\s+(.+)\s+(.+)\s+(.+)$`), template: a.template, t: timer},
						cursor:     3,
					},
					"lim": &CompositeCommand{
						subcommands: map[string]ports.Command{
							"add": &AddCommandLimiter{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+lim(?:\s+add)?\s+(.+)\s+(.+)\s+(.+)$`)},
							"set": &SetCommandLimiter{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+lim\s+set\s+(.+)\s+(.+)\s+(.+)$`)},
							"del": &DelCommandLimiter{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+lim\s+del\s+(.+)$`)},
						},
						defaultCmd: &AddCommandLimiter{re: regexp.MustCompile(`(?i)^!am\s+cmd\s+lim(?:\s+add)?\s+(.+)\s+(.+)\s+(.+)$`)},
						cursor:     3,
					},
				},
				defaultCmd: &AddCommand{re: regexp.MustCompile(`(?i)^!am\s+cmd(?:\s+add)?\s+(.+)$`), template: a.template},
				cursor:     2,
			},
			"al": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"list": &ListAlias{fs: a.fs},
					"add":  &AddAlias{re: regexp.MustCompile(`(?i)^!am\s+al(?:\s+add)?\s+(.+)\s+from\s+(.+)$`), template: a.template},
					"del":  &DelAlias{re: regexp.MustCompile(`(?i)^!am\s+al\s+del\s+(.+)$`), template: a.template},
				},
				defaultCmd: &AddAlias{re: regexp.MustCompile(`(?i)^!am\s+al(?:\s+add)?\s+(.+)\s+from\s+(.+)$`), template: a.template},
				cursor:     2,
			},
			"alg": &CompositeCommand{
				subcommands: map[string]ports.Command{
					"on":     &OnOffAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+(on)\s+(\S+)$`), template: a.template},
					"off":    &OnOffAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+(off)\s+(\S+)$`), template: a.template},
					"create": &CreateAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+create\s+(\S+)\s+(.+)$`), template: a.template},
					"add":    &AddAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+add\s+(\S+)\s+(.+)$`), template: a.template},
					"del":    &DelAliasGroup{re: regexp.MustCompile(`(?i)^!am\s+alg\s+del\s+(\S+)(?:\s+(.+))?$`), template: a.template},
				},
				cursor: 2,
			},
			"cat":   &SetCategory{re: regexp.MustCompile(`(?i)^!am\s+cat\s*(.*)$`), log: a.log, stream: a.stream, api: a.api},
			"title": &SetTitle{re: regexp.MustCompile(`(?i)^!am\s+title\s+(.+)$`), log: a.log, stream: a.stream, api: a.api},
		},
		cursor: 1,
	}
}

type RespArg struct {
	Items []string
	Name  string
}

func buildResponse(errMsg string, args ...RespArg) *ports.AnswerType {
	var msgParts []string

	for _, a := range args {
		if len(a.Items) > 0 {
			msgParts = append(msgParts, fmt.Sprintf("%s: %s", a.Name, strings.Join(a.Items, ", ")))
		}
	}

	if len(msgParts) == 0 {
		return &ports.AnswerType{
			Text:    []string{errMsg + "!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msgParts, " • ") + "."},
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

	parts := make([]string, 0, len(items))
	for key, value := range items {
		parts = append(parts, formatFunc(key, value))
	}

	msg := prefix + ":\n" + strings.Join(parts, "\n")

	key, err := fs.UploadToHaste(msg)
	if err != nil {
		return unknownError
	}

	return &ports.AnswerType{
		Text:    []string{fs.GetURL(key)},
		IsReply: true,
	}
}
