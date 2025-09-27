package admin

import (
	"golang.org/x/time/rate"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddCommandLimiter struct{}

func (c *AddCommandLimiter) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandLimiterAdd(cfg, text)
}

func (c *AddCommandLimiter) handleCommandLimiterAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()

	idx := 3 // id параметра, с которого начинаются аргументы команды
	if words[3] == "add" {
		idx = 4
	}

	// !am cmd lim <команды через запятую> <кол-во запросов> <интервал в секундах>
	// или !am cmd lim add <команды через запятую> <кол-во запросов> <интервал в секундах>
	if len(words) < idx+3 {
		return NonParametr
	}

	cmds := strings.Split(words[idx], ",")
	requests, err := strconv.Atoi(words[idx+1])
	if err != nil || requests <= 0 {
		return &ports.AnswerType{
			Text:    []string{"не указано корректное количество запросов!"},
			IsReply: true,
		}
	}

	seconds, err := strconv.Atoi(words[idx+2])
	if err != nil || seconds <= 0 {
		return &ports.AnswerType{
			Text:    []string{"не указан корректный интервал!"},
			IsReply: true,
		}
	}

	var added, notFound []string
	for _, key := range cmds {
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

		cmd.Limiter = &config.Limiter{
			Requests: requests,
			Per:      time.Duration(seconds) * time.Second,
			Rate:     rate.NewLimiter(rate.Every(time.Duration(seconds)*time.Second), requests),
		}
		added = append(added, key)
	}

	return buildResponse(added, "добавлены", notFound, "не найдены", "команды не указаны")
}

type DelCommandLimiter struct{}

func (c *DelCommandLimiter) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandLimiterDel(cfg, text)
}

func (c *DelCommandLimiter) handleCommandLimiterDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 5 { // !am cmd lim del <команды через запятую>
		return NonParametr
	}

	var removed, notFound []string
	for _, key := range strings.Split(words[4], ",") {
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

		cmd.Limiter = nil
		removed = append(removed, key)
	}

	return buildResponse(removed, "удалены", notFound, "не найдены", "лимитеры не указаны")
}

type SetCommandLimiter struct{}

func (c *SetCommandLimiter) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandLimiterSet(cfg, text)
}

func (c *SetCommandLimiter) handleCommandLimiterSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()

	if len(words) < 7 { // !am cmd lim set <команды через запятую> <кол-во запросов> <интервал>
		return NonParametr
	}

	cmds := strings.Split(words[4], ",")
	requests, err := strconv.Atoi(words[5])
	if err != nil || requests <= 0 {
		return &ports.AnswerType{
			Text:    []string{"не указано корректное количество запросов!"},
			IsReply: true,
		}
	}

	seconds, err := strconv.Atoi(words[6])
	if err != nil || seconds <= 0 {
		return &ports.AnswerType{
			Text:    []string{"не указан корректный интервал!"},
			IsReply: true,
		}
	}

	var edited, notFound []string
	for _, key := range cmds {
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

		cmd.Limiter = &config.Limiter{
			Requests: requests,
			Per:      time.Duration(seconds) * time.Second,
			Rate:     rate.NewLimiter(rate.Every(time.Duration(seconds)*time.Second/time.Duration(requests)), requests),
		}
		edited = append(edited, key)
	}

	return buildResponse(edited, "изменены", notFound, "не найдены", "лимитеры не указаны")
}
