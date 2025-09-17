package admin

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"runtime"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type Ping struct{}

func (p *Ping) Execute(_ *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return p.handlePing()
}

func (p *Ping) handlePing() *ports.AnswerType {
	uptime := time.Since(startApp)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	percent, _ := cpu.Percent(0, false)
	if len(percent) == 0 {
		percent = append(percent, 0)
	}

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("бот работает %v • загрузка CPU %.2f%% • потребление ОЗУ %v MB", uptime.Truncate(time.Second), percent[0], m.Sys/1024/1024)},
		IsReply: true,
	}
}

type OnOff struct {
	enabled bool
}

func (o *OnOff) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return o.handleOnOff(cfg, o.enabled)
}

func (o *OnOff) handleOnOff(cfg *config.Config, enabled bool) *ports.AnswerType {
	cfg.Enabled = enabled
	return nil
}

type Category struct {
	stream ports.StreamPort
}

func (c *Category) Execute(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	return c.handleCategory(text)
}

func (c *Category) handleCategory(text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am game <игра>
		return NonParametr
	}

	if !c.stream.IsLive() {
		return &ports.AnswerType{
			Text:    []string{"стрим выключен!"},
			IsReply: true,
		}
	}

	if c.stream.Category() != "Games + Demos" {
		return &ports.AnswerType{
			Text:    []string{"работает только при категории Games + Demos!"},
			IsReply: true,
		}
	}

	c.stream.SetCategory(text.Tail(2))
	return nil
}

type Status struct{}

func (s *Status) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return s.handleStatus(cfg)
}

func (s *Status) handleStatus(cfg *config.Config) *ports.AnswerType {
	if !cfg.Enabled {
		return &ports.AnswerType{
			Text:    []string{"бот выключен!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text: []string{strings.Join([]string{
			"бот включён", map[bool]string{true: "антиспам включён", false: "антиспам выключен"}[cfg.Spam.SettingsDefault.Enabled],
		}, " • ") + "!"},
		IsReply: true,
	}
}

type Reset struct {
	manager *config.Manager
}

func (r *Reset) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return r.handleReset(cfg)
}

func (r *Reset) handleReset(cfg *config.Config) *ports.AnswerType {
	cfg.Spam = r.manager.GetDefault().Spam
	return nil
}

type Say struct{}

func (s *Say) Execute(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	return s.handleSay(text)
}

func (s *Say) handleSay(text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am say <текст>
		return NonParametr
	}

	return &ports.AnswerType{
		Text:    []string{text.Tail(2)},
		IsReply: false,
	}
}

type Spam struct{}

func (s *Spam) Execute(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	return s.handleSpam(text)
}

func (a *Spam) handleSpam(text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am spam <кол-во раз> <текст>
		return NonParametr
	}

	count, err := strconv.Atoi(words[2])
	if err != nil || count <= 0 {
		return &ports.AnswerType{
			Text:    []string{"кол-во повторов не указано или указано неверно!"},
			IsReply: true,
		}
	}

	if count > 100 {
		count = 100
	}

	msg := text.Tail(3)
	answers := make([]string, count)
	for i := range answers {
		answers[i] = msg
	}

	return &ports.AnswerType{
		Text:    answers,
		IsReply: false,
	}
}
