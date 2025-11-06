package admin

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
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
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/storage"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type Ping struct{}

func (p *Ping) Execute(_ *config.Config, _ string, _ *message.ChatMessage) *ports.AnswerType {
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

func (o *OnOff) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
	cfg.Channels[channel].Enabled = o.enabled

	metrics.BotEnabled.With(prometheus.Labels{"channel": channel}).Set(map[bool]float64{true: 1, false: 0}[o.enabled])
	o.template.SpamPause().Pause(0)
	return success
}

type Game struct {
	re     *regexp.Regexp
	stream ports.StreamPort
}

func (g *Game) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := g.re.FindStringSubmatch(msg.Message.Text.Text()) // !am game <игра>
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

func (s *Status) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
	if !cfg.Channels[channel].Enabled {
		return &ports.AnswerType{Text: []string{"бот выключен!"}, IsReply: true}
	}

	msg := []string{"бот включен"}
	if r := s.template.SpamPause().Remaining(); r > 0 {
		msg = append(msg, fmt.Sprintf("антиспам на паузе (%s)", domain.FormatDuration(r)))
	} else {
		state := "выключен"
		if cfg.Channels[channel].Spam.SettingsDefault.Enabled {
			state = "включен"
		}
		msg = append(msg, "антиспам "+state)
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(msg, " • ") + "!"},
		IsReply: true,
	}
}

type Auth struct {
	log    logger.Logger
	stream ports.StreamPort
	api    ports.APIPort
}

func (a *Auth) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
	token, ok := cfg.UsersTokens[a.stream.ChannelID()]
	if !ok {
		return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
	}

	err := a.api.ValidateToken(context.Background(), token.AccessToken)
	if err == nil {
		return &ports.AnswerType{Text: []string{"авторизация пройдена!"}, IsReply: true}
	}
	a.log.Error("Failed to validate access token", err, slog.String("channel_id", channel))

	resp, err := a.api.RefreshToken(context.Background(), a.stream.ChannelID(), token)
	if err != nil {
		return &ports.AnswerType{Text: []string{"авторизация не пройдена!"}, IsReply: true}
	}

	newToken := &config.UserTokens{
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    resp.ExpiresIn,
		ObtainedAt:   time.Now(),
	}
	cfg.UsersTokens[a.stream.ChannelID()] = newToken

	return &ports.AnswerType{Text: []string{"авторизация пройдена!"}, IsReply: true}
}

type Reset struct {
	manager *config.Manager
}

func (r *Reset) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
	cfg.Channels[channel].Spam = r.manager.GetChannel().Spam // !am reset
	return success
}

type Say struct {
	re *regexp.Regexp
}

func (s *Say) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := s.re.FindStringSubmatch(msg.Message.Text.Text()) // !am say <текст>
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

func (s *Spam) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := s.re.FindStringSubmatch(msg.Message.Text.Text()) // !am spam <кол-во> <текст>
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

	answer := strings.TrimSpace(matches[2])
	answers := make([]string, count)
	for i := range answers {
		answers[i] = answer
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

func (c *SetCategory) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := c.re.FindStringSubmatch(msg.Message.Text.Text()) // !am cat <название категории>
	if len(matches) != 2 {
		return nonParametr
	}

	match := strings.TrimSpace(matches[1])
	if match == "" {
		if _, ok := template.NonGameCategories[c.stream.Category()]; ok {
			return nil
		}
		return &ports.AnswerType{Text: []string{fmt.Sprintf("игра - %s!", c.stream.Category())}, IsReply: true}
	}

	if match == "-" {
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
		c.stream.SetCategory("")
		c.log.Info("Category removed", slog.String("channel_id", c.stream.ChannelID()))
		return &ports.AnswerType{Text: []string{"категория удалена!"}, IsReply: true}
	}

	var id, name string
	key := msg.Message.Text.Text(message.RemovePunctuationOption)
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

	c.stream.SetCategory(name)
	c.log.Info("Category set successfully", slog.String("channel_id", c.stream.ChannelID()), slog.String("category_name", name), slog.String("category_id", id))
	return &ports.AnswerType{Text: []string{fmt.Sprintf("установлена категория %s!", name)}, IsReply: true}
}

type SetTitle struct {
	re     *regexp.Regexp
	log    logger.Logger
	stream ports.StreamPort
	api    ports.APIPort
}

func (t *SetTitle) Execute(_ *config.Config, _ string, msg *message.ChatMessage) *ports.AnswerType {
	matches := t.re.FindStringSubmatch(msg.Message.Text.Text()) // !am title <название>
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
