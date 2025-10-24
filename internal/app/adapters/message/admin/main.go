package admin

import (
	"errors"
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"log/slog"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/adapters/metrics"
	"twitchspam/internal/app/adapters/platform/twitch/api"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
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
	re              *regexp.Regexp
	log             logger.Logger
	stream          ports.StreamPort
	api             ports.APIPort
	cacheCategories *storage.Cache[Category]
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *SetCategory) Execute(_ *config.Config, text *domain.MessageText) *ports.AnswerType {
	return c.handleSetCategory(text)
}

func (c *SetCategory) handleSetCategory(text *domain.MessageText) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(text.Text()) // !am cat <название категории>
	if len(matches) != 2 {
		return nonParametr
	}

	match := strings.TrimSpace(matches[1])
	if match == "" {
		if err := c.api.UpdateChannelCategoryID(c.stream.ChannelID(), "0"); err != nil {
			c.log.Error("Failed to remove Category", err)

			if errors.Is(err, api.ErrUserAuthNotCompleted) {
				return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
			}
			if errors.Is(err, api.ErrBadRequest) {
				return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
			}
			return unknownError
		}
		c.log.Info("Category removed", slog.String("channel_id", c.stream.ChannelID()))
		return &ports.AnswerType{Text: []string{"категория удалена!"}, IsReply: true}
	}

	var id, name string
	key := text.Text(domain.RemovePunctuationOption)
	if cat, ok := c.cacheCategories.Get(key); ok {
		id, name = cat.ID, cat.Name
		c.log.Info("Category found in cache", slog.String("id", id), slog.String("name", name))
	} else {
		var err error
		id, name, err = c.api.SearchCategory(match)
		if err != nil {
			c.log.Error("Category search failed", err, slog.String("query", match))
			if errors.Is(err, api.ErrUserAuthNotCompleted) {
				return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
			}

			if err := c.api.UpdateChannelCategoryID(c.stream.ChannelID(), "66082"); err != nil {
				c.log.Error("Failed to set fallback Category", err)

				if errors.Is(err, api.ErrUserAuthNotCompleted) {
					return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
				}
				if errors.Is(err, api.ErrBadRequest) {
					return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
				}
				return unknownError
			}

			if errors.Is(err, api.ErrBadRequest) {
				return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
			}
			return &ports.AnswerType{Text: []string{"категория не найдена, по умолчанию установлена Games + Demos!"}, IsReply: true}
		}
		c.cacheCategories.Set(key, Category{id, name})
		c.log.Info("Category added to cache", slog.String("id", id), slog.String("name", name))
	}

	if err := c.api.UpdateChannelCategoryID(c.stream.ChannelID(), id); err != nil {
		c.log.Error("Failed to update channel Category id", err, slog.String("channel_id", c.stream.ChannelID()), slog.String("category_id", id))

		if errors.Is(err, api.ErrUserAuthNotCompleted) {
			return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
		}
		if errors.Is(err, api.ErrBadRequest) {
			return &ports.AnswerType{Text: []string{"некорректный запрос!"}, IsReply: true}
		}
		return unknownError
	}

	c.log.Info("Category set successfully", slog.String("channel_id", c.stream.ChannelID()), slog.String("category_name", name), slog.String("category_id", id))
	return &ports.AnswerType{Text: []string{fmt.Sprintf("установлена категория %s!", name)}, IsReply: true}
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

		if errors.Is(err, api.ErrUserAuthNotCompleted) {
			return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
		}
		if errors.Is(err, api.ErrBadRequest) {
			return &ports.AnswerType{Text: []string{"некорректный запрос! Вероятно, одно из указанных слов находится в бан-листе Twitch"}, IsReply: true}
		}
		return unknownError
	}

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("установлено название стрима - %s!", strings.TrimSpace(matches[1]))},
		IsReply: true,
	}
}
