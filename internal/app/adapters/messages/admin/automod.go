package admin

import (
	"regexp"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type OnOffAutomod struct {
	enabled bool
}

func (a *OnOffAutomod) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return a.handleOnOffAutomod(cfg)
}

func (a *OnOffAutomod) handleOnOffAutomod(cfg *config.Config) *ports.AnswerType {
	cfg.Automod.Enabled = a.enabled // !am mod on/off
	return Success
}

type DelayAutomod struct {
	re       *regexp.Regexp
	template ports.TemplatePort
}

func (a *DelayAutomod) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleDelayAutomod(cfg, text)
}

func (a *DelayAutomod) handleDelayAutomod(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	matches := a.re.FindStringSubmatch(text.Original) // !am mod delay <число>
	if len(matches) != 2 {
		return NonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(strings.TrimSpace(matches[1]), 0, 10); ok {
		cfg.Automod.Delay = val
		return Success
	}

	return &ports.AnswerType{
		Text:    []string{"значение задержки срабатывания на автомод должно быть от 0 до 10!"},
		IsReply: true,
	}
}
