package admin

import (
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleDelayAutomod(cfg *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am da <значение>
		return NonParametr
	}

	if val, ok := parseIntArg(words[2], 0, 10); ok {
		cfg.Spam.DelayAutomod = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение задержки срабатывания на автомод должно быть от 0 до 10!"},
		IsReply: true,
	}
}
