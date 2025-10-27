package admin

import (
	"regexp"
	"strings"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type OnOffAutomod struct {
	enabled bool
}

func (a *OnOffAutomod) Execute(cfg *config.Config, channel string, _ *domain.MessageText) *ports.AnswerType {
	return a.handleOnOffAutomod(cfg, channel)
}

func (a *OnOffAutomod) handleOnOffAutomod(cfg *config.Config, channel string) *ports.AnswerType {
	cfg.Channels[channel].Automod.Enabled = a.enabled // !am mod on/off
	return success
}

type DelayAutomod struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *DelayAutomod) Execute(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	return a.handleDelayAutomod(cfg, channel, text)
}

func (a *DelayAutomod) handleDelayAutomod(cfg *config.Config, channel string, text *domain.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Text()) // !am mod delay <число>
	if len(matches) != 2 {
		return nonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 0, 10); ok {
		cfg.Channels[channel].Automod.Delay = val
		return success
	}

	return &ports.AnswerType{
		Text:    []string{"значение задержки срабатывания на автомод должно быть от 0 до 10!"},
		IsReply: true,
	}
}
