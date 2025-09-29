package admin

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddCommandTimer struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	t        *AddTimer
}

func (c *AddCommandTimer) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandTimersAdd(cfg, text)
}

func (c *AddCommandTimer) handleCommandTimersAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := c.template.Options().ParseAll(text.Original, template.TimersOptions)

	// !am cmd timer <интервал в секундах> <кол-во сообщений> <команда>
	// или !am cmd timer add <интервал в секундах> <кол-во сообщений> <команда>
	matches := c.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 4 {
		return NonParametr
	}

	interval, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{"не указан интервал команды!"},
			IsReply: true,
		}
	}

	count, err := strconv.Atoi(strings.TrimSpace(matches[2]))
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{"не указано количество сообщений!"},
			IsReply: true,
		}
	}

	name := strings.TrimSpace(matches[3])
	if !strings.HasPrefix(name, "!") {
		name = "!" + name
	}

	cfg.Commands[name].Timer = &config.Timers{
		Interval: time.Duration(interval) * time.Second,
		Count:    count,
		Options:  c.template.Options().MergeTimer(config.TimerOptions{}, opts),
	}
	c.t.AddTimer(name, cfg.Commands[name])

	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
}

type DelCommandTimer struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	timers   ports.TimersPort
}

func (c *DelCommandTimer) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandTimersDel(cfg, text)
}

func (c *DelCommandTimer) handleCommandTimersDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Original) // !am cmd timer del <команды через запятую>
	if len(matches) != 2 {
		return NonParametr
	}

	var removed, notFound []string
	for _, key := range strings.Split(strings.TrimSpace(matches[1]), ",") {
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

	return buildResponse("команды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type OnOffCommandTimer struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	timers   ports.TimersPort
	t        *AddTimer
}

func (c *OnOffCommandTimer) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandTimersOnOff(cfg, text)
}

func (c *OnOffCommandTimer) handleCommandTimersOnOff(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Original) // !am cmd timer on/off <команды через запятую>
	if len(matches) != 3 {
		return NonParametr
	}

	state := strings.ToLower(strings.TrimSpace(matches[1]))

	var edited, notFound []string
	for _, key := range strings.Split(strings.TrimSpace(matches[2]), ",") {
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

		if state != "on" {
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

	return buildResponse("команды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type SetCommandTimer struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	timers   ports.TimersPort
	t        *AddTimer
}

func (c *SetCommandTimer) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandTimersSet(cfg, text)
}

func (c *SetCommandTimer) handleCommandTimersSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := c.template.Options().ParseAll(text.Original, template.TimersOptions)

	// !am cmd timer set int/count <значение> <команды через запятую> или !am cmd timer set <опции> <команды через запятую>
	matches := c.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) < 2 {
		return NonParametr
	}

	param, idx := 0, 1
	match := strings.ToLower(strings.TrimSpace(matches[1]))
	if len(matches) == 4 && (match == "int" || match == "count") {
		val, err := strconv.Atoi(strings.TrimSpace(matches[2]))
		if err != nil {
			return UnknownError
		}
		param, idx = val, 3
	}

	var edited, notFound, incorrectValue []string
	for _, key := range strings.Split(strings.TrimSpace(matches[idx]), ",") {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		if !strings.HasPrefix(key, "!") {
			key = "!" + key
		}

		cmd, ok := cfg.Commands[key]
		if !ok || cmd.Timer == nil {
			notFound = append(notFound, key)
			continue
		}

		switch {
		case param < 0:
			incorrectValue = append(incorrectValue, key)
		case param > 0:
			if match == "int" {
				cmd.Timer.Interval = time.Duration(param) * time.Second
			} else if match == "count" {
				cmd.Timer.Count = param
			}
		}

		c.timers.RemoveTimer(key)
		cfg.Commands[key].Timer.Options = c.template.Options().MergeTimer(cmd.Timer.Options, opts)
		c.t.AddTimer(key, cfg.Commands[key])
		edited = append(edited, key)
	}

	return buildResponse("команды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"}, RespArg{Items: incorrectValue, Name: "некорректные значения"})
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
			a.Api.SendChatMessages(msg) // FIXME
		} else {
			a.Api.SendChatMessages(msg)
		}
	})
}
