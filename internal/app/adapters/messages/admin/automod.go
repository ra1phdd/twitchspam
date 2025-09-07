package admin

import (
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handleDelayAutomod(cfg *config.Config, _ string, args []string) *ports.AnswerType {
	if val, ok := parseIntArg(args, 0, 10); ok {
		cfg.Spam.DelayAutomod = val
		return nil
	}
	return &ports.AnswerType{
		Text:    []string{"значение задержки срабатывания на автомод должно быть от 0 до 10!"},
		IsReply: true,
	}
}
