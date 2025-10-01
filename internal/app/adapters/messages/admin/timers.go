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
	timers := make(map[string]*config.Timers)
	for _, cmd := range cfg.Commands {
		if cmd.Timer == nil {
			continue
		}
		timers[cmd.Text] = cmd.Timer
	}

	return buildList(timers, "таймеры", "таймеры не найдены!",
		func(key string, timer *config.Timers) string {
			options := make([]string, 2)
			options[0] = map[bool]string{true: "-a", false: "-noa"}[timer.Options.IsAnnounce]
			options[1] = map[bool]string{true: "-always", false: "-online"}[timer.Options.IsAlways]

			return fmt.Sprintf("- %s (включен: %v, интервал: %s, кол-во сообщений: %d, опции: %s)",
				key, timer.Enabled, timer.Interval, timer.Count, strings.Join(options, " "))
		}, t.fs)
}
