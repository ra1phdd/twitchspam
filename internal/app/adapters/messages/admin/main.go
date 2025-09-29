package admin

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"regexp"
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
	return Success
}

type Game struct {
	re     *regexp.Regexp
	stream ports.StreamPort
}

func (g *Game) Execute(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	return g.handleGame(text)
}

func (g *Game) handleGame(text *ports.MessageText) *ports.AnswerType {
	matches := g.re.FindStringSubmatch(text.Lower()) // !am game <игра>
	if len(matches) != 2 {
		return NonParametr
	}

	if !g.stream.IsLive() {
		return &ports.AnswerType{
			Text:    []string{"стрим выключен!"},
			IsReply: true,
		}
	}

	if g.stream.Category() != "Games + Demos" {
		return &ports.AnswerType{
			Text:    []string{"работает только при категории Games + Demos!"},
			IsReply: true,
		}
	}

	gameName := strings.TrimSpace(matches[1])
	g.stream.SetCategory(gameName)
	return Success
}

type Status struct{}

func (s *Status) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return s.handleStatus(cfg)
}

func (s *Status) handleStatus(cfg *config.Config) *ports.AnswerType {
	if !cfg.Enabled { // !am status
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
	cfg.Spam = r.manager.GetDefault().Spam // !am reset
	return Success
}

type Say struct {
	re *regexp.Regexp
}

func (s *Say) Execute(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	return s.handleSay(text)
}

func (s *Say) handleSay(text *ports.MessageText) *ports.AnswerType {
	matches := s.re.FindStringSubmatch(text.Lower()) // !am say <текст>
	if len(matches) != 2 {
		return NonParametr
	}

	return &ports.AnswerType{
		Text:    []string{strings.TrimSpace(matches[1])},
		IsReply: false,
	}
}

type Spam struct {
	re *regexp.Regexp
}

func (s *Spam) Execute(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	return s.handleSpam(text)
}

func (s *Spam) handleSpam(text *ports.MessageText) *ports.AnswerType {
	matches := s.re.FindStringSubmatch(text.Lower()) // !am spam <кол-во> <текст>
	if len(matches) != 3 {
		return NonParametr
	}

	count, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil || count <= 0 {
		return &ports.AnswerType{
			Text:    []string{"кол-во повторов не указано или указано неверно!"},
			IsReply: true,
		}
	}

	if count > 100 {
		count = 100
	}

	msg := strings.TrimSpace(matches[2])
	answers := make([]string, count)
	for i := range answers {
		answers[i] = msg
	}

	return &ports.AnswerType{
		Text:    answers,
		IsReply: false,
	}
}
