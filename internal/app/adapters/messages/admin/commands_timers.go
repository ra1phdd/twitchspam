package admin

import (
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddCommandTimer struct {
	template ports.TemplatePort
	t        *AddTimer
}

func (c *AddCommandTimer) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandTimersAdd(cfg, text)
}

func (c *AddCommandTimer) handleCommandTimersAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := c.template.Options().ParseAll(&words, template.TimersOptions) // ParseOptions удаляет опции из слайса words

	idx := 3 // id параметра, с которого начинаются аргументы команды
	if words[3] == "add" {
		idx = 4
	}

	// !am cmd timer <интервал в секундах> <кол-во сообщений> <команда>
	// или !am cmd timer add <интервал в секундах> <кол-во сообщений> <команда>
	if len(words) < idx+3 {
		return NonParametr
	}

	interval, err := strconv.Atoi(words[3])
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{"не указан интервал команды!"},
			IsReply: true,
		}
	}

	count, err := strconv.Atoi(words[4])
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{"не указано количество сообщений!"},
			IsReply: true,
		}
	}

	name := words[5]
	if !strings.HasPrefix(name, "!") {
		name = "!" + name
	}

	cmd := cfg.Commands[name]
	cmd.Timer = &config.Timers{
		Interval: time.Duration(interval) * time.Second,
		Count:    count,
		Options:  mergeTimerOptions(cmd.Timer.Options, opts),
	}
	c.t.AddTimer(words[5], cmd)

	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
}

type DelCommandTimer struct {
	template ports.TemplatePort
	timers   ports.TimersPort
}

func (c *DelCommandTimer) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandTimersDel(cfg, text)
}

func (c *DelCommandTimer) handleCommandTimersDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 5 { // !am cmd timer del <команды через запятую>
		return NonParametr
	}

	var removed, notFound []string
	for _, key := range strings.Split(text.Tail(4), ",") {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		if !strings.HasPrefix(key, "!") {
			key = "!" + key
		}

		if _, ok := cfg.Commands[key]; !ok {
			notFound = append(notFound, key)
			continue
		}

		c.timers.RemoveTimer(key)
		cfg.Commands[key].Timer = nil
		removed = append(removed, key)
	}

	return buildResponse(removed, "удалены", notFound, "не найдены", "таймеры не указаны")
}

type OnOffCommandTimer struct {
	template ports.TemplatePort
	timers   ports.TimersPort
	t        *AddTimer
}

func (c *OnOffCommandTimer) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandTimersOnOff(cfg, text)
}

func (c *OnOffCommandTimer) handleCommandTimersOnOff(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 5 { // !am cmd timer on/off <команды через запятую>
		return NonParametr
	}

	var edited, notFound []string
	for _, key := range strings.Split(words[4], ",") {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		if _, ok := cfg.Commands[key]; !ok {
			notFound = append(notFound, key)
			continue
		}

		if !strings.HasPrefix(key, "!") {
			key = "!" + key
		}

		if words[3] != "on" {
			c.timers.RemoveTimer(key)
			cfg.Commands[key].Timer.Enabled = false
			edited = append(edited, key)
			continue
		}

		cmd := cfg.Commands[key]
		cmd.Timer.Enabled = true
		edited = append(edited, key)
		c.t.AddTimer(key, cmd)
	}

	return buildResponse(edited, "изменены", notFound, "не найдены", "таймеры не указаны")
}

type SetCommandTimer struct {
	template ports.TemplatePort
	timers   ports.TimersPort
	t        *AddTimer
}

func (c *SetCommandTimer) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandTimersSet(cfg, text)
}

func (c *SetCommandTimer) handleCommandTimersSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := c.template.Options().ParseAll(&words, template.TimersOptions) // ParseOptions удаляет опции из слайса words

	// !am cmd timer set int/count <значение> <команды через запятую> или !am cmd timer set <опции>
	idx := 4
	param := 0
	if words[4] == "int" || words[4] == "count" {
		var err error
		param, err = strconv.Atoi(words[5])
		if err != nil {
			return UnknownError
		}

		idx = 6
	}

	var edited, notFound []string
	for _, key := range strings.Split(words[idx], ",") {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		if !strings.HasPrefix(key, "!") {
			key = "!" + key
		}

		cmd, ok := cfg.Commands[key]
		if !ok {
			notFound = append(notFound, key)
			continue
		}

		if param != 0 {
			switch words[4] {
			case "int":
				cmd.Timer.Interval = time.Duration(param) * time.Second
			case "count":
				cmd.Timer.Count = param
			}
		}

		c.timers.RemoveTimer(key)
		cfg.Commands[key].Timer.Options = mergeTimerOptions(cfg.Commands[key].Timer.Options, opts)
		c.t.AddTimer(key, cmd)
		edited = append(edited, key)
	}

	return buildResponse(edited, "изменены", notFound, "не найдены", "таймеры не указаны")
}

type AddTimer struct {
	Timers ports.TimersPort
	Stream ports.StreamPort
	Api    ports.APIPort
}

func (a *AddTimer) AddTimer(key string, cmd *config.Commands) {
	a.Timers.AddTimer(key, cmd.Timer.Interval, true, map[string]any{
		"text":  cmd.Text,
		"timer": cmd.Timer,
	}, func(args map[string]any) {
		timer := args["timer"].(*config.Timers)
		if !timer.Enabled || (!timer.Options.IsAlways && !a.Stream.IsLive()) {
			return
		}

		msg := &ports.AnswerType{}
		for i := 0; i < timer.Count; i++ {
			msg.Text = append(msg.Text, args["text"].(string))
		}

		if timer.Options.IsAnnounce {
			a.Api.SendChatMessages(msg) //FIXME
		} else {
			a.Api.SendChatMessages(msg)
		}
	})
}
