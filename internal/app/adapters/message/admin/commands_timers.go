package admin

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddCommandTimer struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	t        *AddTimer
}

func (c *AddCommandTimer) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleCommandTimersAdd(cfg, text)
}

func (c *AddCommandTimer) handleCommandTimersAdd(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := c.template.Options().ParseAll(text.Text(), template.TimersOptions)

	// !am cmd timer <кол-во сообщений> <интервал в секундах> <команда>
	// или !am cmd timer add <кол-во сообщений> <интервал в секундах> <команда>
	matches := c.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 4 {
		return nonParametr
	}

	count, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil || count < 1 || count > 10 {
		return invalidValueRequest
	}

	if _, ok := opts["a"]; ok && count > 1 {
		return &ports.AnswerType{
			Text:    []string{"при использовании анонсов можно отправить только 1 сообщение за раз!"},
			IsReply: true,
		}
	}

	interval, err := strconv.Atoi(strings.TrimSpace(matches[2]))
	if err != nil || interval < 5 || interval > 86400 {
		return invalidValueInterval
	}

	name := strings.ToLower(strings.TrimSpace(matches[3]))
	if !strings.HasPrefix(name, "!") {
		name = "!" + name
	}

	cfg.Commands[name].Timer = &config.Timers{
		Enabled:  true,
		Interval: time.Duration(interval) * time.Second,
		Count:    count,
		Options:  c.template.Options().MergeTimer(nil, opts),
	}
	c.t.AddTimer(name, cfg.Commands[name])

	return success
}

type DelCommandTimer struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	timers   ports.TimersPort
}

func (c *DelCommandTimer) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleCommandTimersDel(cfg, text)
}

func (c *DelCommandTimer) handleCommandTimersDel(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Text()) // !am cmd timer del <команды через запятую>
	if len(matches) != 2 {
		return nonParametr
	}

	words := strings.Split(strings.TrimSpace(matches[1]), ",")
	removed, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if !strings.HasPrefix(word, "!") {
			word = "!" + word
		}

		if _, ok := cfg.Commands[word]; !ok {
			notFound = append(notFound, word)
			continue
		}

		c.timers.RemoveTimer(word)
		cfg.Commands[word].Timer = nil
		removed = append(removed, word)
	}

	return buildResponse("команды не указаны", RespArg{Items: removed, Name: "удалены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type OnOffCommandTimer struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	timers   ports.TimersPort
	t        *AddTimer
}

func (c *OnOffCommandTimer) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleCommandTimersOnOff(cfg, text)
}

func (c *OnOffCommandTimer) handleCommandTimersOnOff(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Text()) // !am cmd timer on/off <команды через запятую>
	if len(matches) != 3 {
		return nonParametr
	}

	state := strings.ToLower(strings.TrimSpace(matches[1]))

	words := strings.Split(strings.TrimSpace(matches[2]), ",")
	edited, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if !strings.HasPrefix(word, "!") {
			word = "!" + word
		}

		if _, ok := cfg.Commands[word]; !ok {
			notFound = append(notFound, word)
			continue
		}

		if state != "on" {
			c.timers.RemoveTimer(word)
			cfg.Commands[word].Timer.Enabled = false
			edited = append(edited, word)
			continue
		}

		cmd := cfg.Commands[word]
		cmd.Timer.Enabled = true
		edited = append(edited, word)
		c.t.AddTimer(word, cmd)
	}

	return buildResponse("команды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type SetCommandTimer struct {
	re       *regexp.Regexp
	template ports.TemplatePort
	timers   ports.TimersPort
	t        *AddTimer
}

func (c *SetCommandTimer) Execute(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleCommandTimersSet(cfg, text)
}

func (c *SetCommandTimer) handleCommandTimersSet(cfg *config.Config, text *domain.MessageText) *ports.AnswerType {
	textWithoutOpts, opts := c.template.Options().ParseAll(text.Text(), template.TimersOptions)

	// !am cmd timer set <кол-во сообщений> <интервал в секундах> <команды через запятую>
	// или !am cmd timer set <опции> <команды через запятую>
	matches := c.re.FindStringSubmatch(textWithoutOpts)
	if len(matches) != 4 {
		return nonParametr
	}

	var count, interval int
	var err error
	if strings.TrimSpace(matches[1]) != "" {
		count, err = strconv.Atoi(strings.TrimSpace(matches[1]))
		if err != nil || count < 1 || count > 10 {
			return invalidValueRequest
		}

		if _, ok := opts["a"]; ok && count > 1 {
			return &ports.AnswerType{
				Text:    []string{"при использовании анонсов можно отправить только 1 сообщение за раз!"},
				IsReply: true,
			}
		}
	}

	if strings.TrimSpace(matches[2]) != "" {
		interval, err = strconv.Atoi(strings.TrimSpace(matches[2]))
		if err != nil || interval < 5 || interval > 86400 {
			return invalidValueInterval
		}
	}

	words := strings.Split(strings.TrimSpace(matches[3]), ",")
	edited, notFound := make([]string, 0, len(words)), make([]string, 0, len(words))
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		if !strings.HasPrefix(word, "!") {
			word = "!" + word
		}

		cmd, ok := cfg.Commands[word]
		if !ok || cmd.Timer == nil {
			notFound = append(notFound, word)
			continue
		}

		if count != 0 {
			cmd.Timer.Count = count
		}

		if interval != 0 {
			cmd.Timer.Interval = time.Duration(interval) * time.Second
		}

		c.timers.RemoveTimer(word)
		cfg.Commands[word].Timer.Options = c.template.Options().MergeTimer(cmd.Timer.Options, opts)
		c.t.AddTimer(word, cfg.Commands[word])
		edited = append(edited, word)
	}

	return buildResponse("команды не указаны", RespArg{Items: edited, Name: "изменены"}, RespArg{Items: notFound, Name: "не найдены"})
}

type AddTimer struct {
	Cfg    *config.Config
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
		if !timer.Enabled {
			return
		}

		mode := config.OnlineMode
		if timer.Options != nil && timer.Options.Mode != nil {
			mode = *timer.Options.Mode
		}

		if ((mode == config.OnlineMode || mode == 0) && !a.Stream.IsLive()) || (mode == config.OfflineMode && a.Stream.IsLive()) {
			return
		}

		msg := &ports.AnswerType{}
		for range timer.Count {
			msg.Text = append(msg.Text, args["text"].(string))
		}

		if _, ok := a.Cfg.UsersTokens[a.Stream.ChannelID()]; ok && timer.Options != nil && timer.Options.IsAnnounce != nil && *timer.Options.IsAnnounce {
			a.Api.SendChatAnnouncements(a.Stream.ChannelID(), msg, *timer.Options.ColorAnnounce)
			return
		}

		a.Api.SendChatMessages(a.Stream.ChannelID(), msg)
	})
}
