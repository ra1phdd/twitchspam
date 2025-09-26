package admin

import (
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
	cfg.Automod.Enabled = a.enabled
	return nil
}

type DelayAutomod struct {
	template ports.TemplatePort
}

func (a *DelayAutomod) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleDelayAutomod(cfg, text)
}

func (a *DelayAutomod) handleDelayAutomod(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am mod delay <значение>
		return NonParametr
	}

	if val, ok := a.template.Parser().ParseIntArg(words[3], 0, 10); ok {
		cfg.Automod.Delay = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение задержки срабатывания на автомод должно быть от 0 до 10!"},
		IsReply: true,
	}
}
