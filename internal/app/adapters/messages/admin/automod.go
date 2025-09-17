package admin

import (
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type DelayAutomod struct {
	template ports.TemplatePort
}

func (a *DelayAutomod) Execute(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	return a.handleDelayAutomod(cfg, text)
}

func (a *DelayAutomod) handleDelayAutomod(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am da <значение>
		return NonParametr
	}

	if val, ok := a.template.ParseIntArg(words[2], 0, 10); ok {
		cfg.Spam.DelayAutomod = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение задержки срабатывания на автомод должно быть от 0 до 10!"},
		IsReply: true,
	}
}
