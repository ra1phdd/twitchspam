package admin

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Ping struct{}

func (p *Ping) Execute(_ *config.Config, _ *domain.MessageText) *ports.AnswerType {
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
	enabled  bool
	template ports.TemplatePort
}

func (o *OnOff) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return o.handleOnOff(cfg, o.enabled)
}

func (o *OnOff) handleOnOff(cfg *config.Config, enabled bool) *ports.AnswerType {
	cfg.Enabled = enabled

	metrics.BotEnabled.Set(map[bool]float64{true: 1, false: 0}[enabled])
	o.template.SpamPause().Pause(0)
	return success
}

type Game struct {
	re     *regexp.Regexp
	stream ports.StreamPort
}

func (g *Game) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return g.handleGame(text)
}

func (g *Game) handleGame(text *domain.MessageText) *ports.AnswerType {
	matches := g.re.FindStringSubmatch(text.Text()) // !am game <игра>
	if len(matches) != 2 {
		return nonParametr
	}

	if !g.stream.IsLive() {
		return streamOff
	}

	if g.stream.Category() != "Games + Demos" {
		return &ports.AnswerType{
			Text:    []string{"работает только при категории Games + Demos!"},
			IsReply: true,
		}
	}

	gameName := strings.TrimSpace(matches[1])
	g.stream.SetCategory(gameName)
	return success
}

type Status struct {
	template ports.TemplatePort
}

func (s *Status) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return s.handleStatus(cfg)
}

func (s *Status) handleStatus(cfg *config.Config) *ports.AnswerType {
	if !cfg.Enabled {
		return &ports.AnswerType{Text: []string{"бот выключен!"}, IsReply: true}
	}

	msg := []string{"бот включен"}
	if r := s.template.SpamPause().Remaining(); r > 0 {
		msg = append(msg, fmt.Sprintf("антиспам на паузе (%s)", domain.FormatDuration(r)))
	} else {
		state := "выключен"
		if cfg.Spam.SettingsDefault.Enabled {
			state = "включен"
		}
		msg = append(msg, "антиспам "+state)
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msg, " • ") + "!"},
		IsReply: true,
	}
}

type Reset struct {
	manager *config.Manager
}

func (r *Reset) Execute(cfg *config.Config, _ *domain.MessageText) *ports.AnswerType {
	return r.handleReset(cfg)
}

func (r *Reset) handleReset(cfg *config.Config) *ports.AnswerType {
	cfg.Spam = r.manager.GetDefault().Spam // !am reset
	return success
}

type Say struct {
	re *regexp.Regexp
}

func (s *Say) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return s.handleSay(text)
}

func (s *Say) handleSay(text *domain.MessageText) *ports.AnswerType {
	matches := s.re.FindStringSubmatch(text.Text()) // !am say <текст>
	if len(matches) != 2 {
		return nonParametr
	}

	return &ports.AnswerType{
		Text:    []string{strings.TrimSpace(matches[1])},
		IsReply: false,
	}
}

type Spam struct {
	re *regexp.Regexp
}

func (s *Spam) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return s.handleSpam(text)
}

func (s *Spam) handleSpam(text *domain.MessageText) *ports.AnswerType {
	matches := s.re.FindStringSubmatch(text.Text()) // !am spam <кол-во> <текст>
	if len(matches) != 3 {
		return nonParametr
	}

	count, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil || count <= 0 {
		return invalidValueRepeats
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

type SetCategory struct {
	re     *regexp.Regexp
	log    logger.Logger
	stream ports.StreamPort
	api    ports.APIPort
}

func (c *SetCategory) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleSetCategory(text)
}

func (c *SetCategory) handleSetCategory(text *domain.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Text()) // !am cat <название категории>
	if len(matches) != 2 {
		return nonParametr
	}

	id, name := "0", ""
	match := strings.TrimSpace(matches[1])

	if match != "" {
		var err error
		id, name, err = c.api.SearchCategory(match)
		if err != nil {
			return &ports.AnswerType{
				Text:    []string{"категория не найдена!"},
				IsReply: true,
			}
		}
	}

	err := c.api.UpdateChannelGameID(c.stream.ChannelID(), id)
	if err != nil {
		c.log.Error("Failed to update channel game id", err)
		return unknownError
	}

	if id == "0" {
		return &ports.AnswerType{
			Text:    []string{"категория удалена!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("установлена категория %s!", name)},
		IsReply: true,
	}
}

type SetTitle struct {
	re     *regexp.Regexp
	log    logger.Logger
	stream ports.StreamPort
	api    ports.APIPort
}

func (t *SetTitle) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return t.handleSetTitle(text)
}

func (t *SetTitle) handleSetTitle(text *domain.MessageText) *ports.AnswerType {
	matches := t.re.FindStringSubmatch(text.Text()) // !am title <название>
	if len(matches) != 2 {
		return nonParametr
	}

	if len(strings.TrimSpace(matches[1])) > 140 {
		return &ports.AnswerType{
			Text:    []string{"название стрима не может быть длиннее 140 символов!"},
			IsReply: true,
		}
	}

	err := t.api.UpdateChannelTitle(t.stream.ChannelID(), strings.TrimSpace(matches[1]))
	if err != nil {
		t.log.Error("Failed to update channel title", err)

		if err.Error() == "400" {
			return &ports.AnswerType{
				Text:    []string{"одно из указанных слов находится в бан-листе Twitch!"},
				IsReply: true,
			}
		}
		return unknownError
	}

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("установлено название стрима - %s!", strings.TrimSpace(matches[1]))},
		IsReply: true,
	}
}
