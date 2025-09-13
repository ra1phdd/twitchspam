package admin

import (
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleCommandTimers(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am cmd timer add/del/list/...
		return NonParametr
	}

	handlers := map[string]func(cfg *config.Config, text *ports.MessageText) *ports.AnswerType{
		"set": a.handleCommandTimersSet,
		"del": a.handleCommandTimersDel,
		"on":  a.handleCommandTimersOnOff,
		"off": a.handleCommandTimersOnOff,
	}

	linkCmd := words[3]
	if handler, ok := handlers[linkCmd]; ok {
		return handler(cfg, text)
	}
	return a.handleCommandTimersAdd(cfg, text)
}

func (a *Admin) handleCommandTimersAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := a.template.ParseOptions(&words, template.TimersOptions) // ParseOptions удаляет опции из слайса words

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

	cmd := cfg.Commands[words[5]]
	cmd.Timer = &config.Timers{
		Interval: time.Duration(interval) * time.Second,
		Count:    count,
		Options:  a.mergeTimerOptions(cmd.Timer.Options, opts),
	}
	a.addTimer(words[5], cmd)

	return &ports.AnswerType{
		Text:    []string{"успешно!"},
		IsReply: true,
	}
}

func (a *Admin) handleCommandTimersDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
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

		if _, ok := cfg.Commands[key]; !ok {
			notFound = append(notFound, key)
			continue
		}

		a.timers.RemoveTimer(key)
		cfg.Commands[key].Timer = nil
		removed = append(removed, key)
	}

	return a.buildResponse(removed, "удалены", notFound, "не найдены", "таймеры не указаны")
}

func (a *Admin) handleCommandTimersOnOff(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
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

		if words[3] != "on" {
			a.timers.RemoveTimer(key)
			cfg.Commands[key].Timer.Enabled = false
			edited = append(edited, key)
			continue
		}

		cmd := cfg.Commands[key]
		cmd.Timer.Enabled = true
		edited = append(edited, key)
		a.addTimer(key, cmd)
	}

	return a.buildResponse(edited, "изменены", notFound, "не найдены", "таймеры не указаны")
}

func (a *Admin) handleCommandTimersSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	opts := a.template.ParseOptions(&words, template.TimersOptions) // ParseOptions удаляет опции из слайса words

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

		a.timers.RemoveTimer(key)
		cfg.Commands[key].Timer.Options = a.mergeTimerOptions(cfg.Commands[key].Timer.Options, opts)
		a.addTimer(key, cmd)
		edited = append(edited, key)
	}

	return a.buildResponse(edited, "изменены", notFound, "не найдены", "таймеры не указаны")
}

func (a *Admin) addTimer(key string, cmd *config.Commands) {
	a.timers.AddTimer(key, cmd.Timer.Interval, true, map[string]any{
		"text":  cmd.Text,
		"count": cmd.Timer.Count,
		"opts":  cmd.Timer.Options,
	}, func(args map[string]any) {
		msg := &ports.AnswerType{}
		for i := 0; i < args["count"].(int); i++ {
			msg.Text = append(msg.Text, args["text"].(string))
		}

		a.api.SendChatMessages(msg)
	})
}
