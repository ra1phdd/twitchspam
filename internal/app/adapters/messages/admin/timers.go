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
	return buildList(cfg.Commands, "таймеры", "таймеры не найдены!",
		func(key string, cmd *config.Commands) string {
			options := []string{"-noa", "-online"}
			if cmd.Timer.Options != nil {
				options[0] = map[bool]string{true: "-a", false: "-noa"}[cmd.Timer.Options.IsAnnounce]
				options[1] = map[bool]string{true: "-always", false: "-online"}[cmd.Timer.Options.IsAlways]
			}
			return fmt.Sprintf("- %s (включен: %v, интервал: %s, кол-во сообщений: %d, опции: %s)",
				key, cmd.Timer.Enabled, cmd.Timer.Interval, cmd.Timer.Count, strings.Join(options, " "))
		}, t.fs)
}
