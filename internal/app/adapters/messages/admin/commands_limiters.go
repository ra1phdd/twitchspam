package admin

import (
	"golang.org/x/time/rate"
	"regexp"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type AddCommandLimiter struct {
	re *regexp.Regexp
}

func (c *AddCommandLimiter) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandLimiterAdd(cfg, text)
}

func (c *AddCommandLimiter) handleCommandLimiterAdd(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	// !am cmd lim <кол-во запросов> <интервал в секундах> <команды через запятую>
	// или !am cmd lim add <кол-во запросов> <интервал в секундах> <команды через запятую>
	matches := c.re.FindStringSubmatch(text.Original)
	if len(matches) != 4 {
		return NonParametr
	}

	requests, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil || requests <= 0 {
		return &ports.AnswerType{
			Text:    []string{"не указано корректное количество запросов!"},
			IsReply: true,
		}
	}

	seconds, err := strconv.Atoi(strings.TrimSpace(matches[2]))
	if err != nil || seconds <= 0 {
		return &ports.AnswerType{
			Text:    []string{"не указан корректный интервал!"},
			IsReply: true,
		}
	}

	var added, notFound []string
	for _, key := range strings.Split(strings.TrimSpace(matches[3]), ",") {
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

type SetCommandLimiter struct {
	re *regexp.Regexp
}

func (c *SetCommandLimiter) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandLimiterSet(cfg, text)
}

func (c *SetCommandLimiter) handleCommandLimiterSet(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Original) // !am cmd lim set <кол-во запросов> <интервал в секундах> <команды через запятую>
	if len(matches) != 4 {
		return NonParametr
	}

	requests, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil || requests <= 0 {
		return &ports.AnswerType{
			Text:    []string{"не указано корректное количество запросов!"},
			IsReply: true,
		}
	}

	seconds, err := strconv.Atoi(strings.TrimSpace(matches[2]))
	if err != nil || seconds <= 0 {
		return &ports.AnswerType{
			Text:    []string{"не указан корректный интервал!"},
			IsReply: true,
		}
	}

	var edited, notFound []string
	for _, key := range strings.Split(strings.TrimSpace(matches[3]), ",") {
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

type DelCommandLimiter struct {
	re *regexp.Regexp
}

func (c *DelCommandLimiter) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCommandLimiterDel(cfg, text)
}

func (c *DelCommandLimiter) handleCommandLimiterDel(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Original) // !am cmd lim del <команды через запятую>
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
