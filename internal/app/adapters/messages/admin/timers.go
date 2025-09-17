package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type ListTimers struct {
	fs ports.FileServerPort
}

func (t *ListTimers) Execute(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	return t.handleTimersList(cfg)
}

func (t *ListTimers) handleTimersList(cfg *config.Config) *ports.AnswerType {
	if len(cfg.Commands) == 0 {
		return &ports.AnswerType{
			Text:    []string{"таймеры не найдены!"},
			IsReply: true,
		}
	}

	var parts []string
	for key, cmd := range cfg.Commands {
		options := []string{"-noa", "-online"}
		if cmd.Timer.Options != nil {
			options[0] = map[bool]string{true: "-a", false: "-noa"}[cmd.Timer.Options.IsAnnounce]
			options[1] = map[bool]string{true: "-always", false: "-online"}[cmd.Timer.Options.IsAlways]
		}

		parts = append(parts, fmt.Sprintf("- %s (включен: %v, интервал: %s, кол-во сообщений: %d, опции: %s",
			key, cmd.Timer.Enabled, cmd.Timer.Interval, cmd.Timer.Count, strings.Join(options, " ")))
	}
	msg := "таймеры: \n" + strings.Join(parts, "\n")

	if len(parts) == 0 {
		return &ports.AnswerType{
			Text:    []string{"таймеры не найдены!"},
			IsReply: true,
		}
	}

	key, err := t.fs.UploadToHaste(msg)
	if err != nil {
		return UnknownError
	}
	return &ports.AnswerType{
		Text:    []string{t.fs.GetURL(key)},
		IsReply: true,
	}
}
