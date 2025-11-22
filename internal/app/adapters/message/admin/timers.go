package admin

import (
	"fmt"
	"strings"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

type ListTimers struct {
	fs ports.FileServerPort
}

func (t *ListTimers) Execute(cfg *config.Config, channel string, _ *message.ChatMessage) *ports.AnswerType {
	timers := make(map[string]*config.Timers)
	for _, cmd := range cfg.Channels[channel].Commands {
		if cmd.Timer == nil {
			continue
		}
		timers[cmd.Text] = cmd.Timer
	}

	return buildList(timers, "таймеры", "таймеры не найдены!",
		func(key string, timer *config.Timers) string {
			if timer == nil {
				return ""
			}

			options := make([]string, 2)
			isAnnounce := timer.Options != nil && timer.Options.IsAnnounce != nil && *timer.Options.IsAnnounce
			options[0] = map[bool]string{
				true:  "-a",
				false: "-noa",
			}[isAnnounce]

			mode := config.OnlineMode
			if timer.Options != nil && timer.Options.Mode != nil {
				mode = *timer.Options.Mode
			}

			options[1] = map[int]string{
				0:                  "-online",
				config.OnlineMode:  "-online",
				config.OfflineMode: "-offline",
				config.AlwaysMode:  "-always",
			}[mode]

			return fmt.Sprintf("- %s (включен: %v, интервал: %s, кол-во сообщений: %d, опции: %s)",
				key, timer.Enabled, timer.Interval, timer.Count, strings.Join(options, " "))
		}, t.fs)
}

type AddTimer struct {
	cfg    *config.Config
	timers ports.TimersPort
	stream ports.StreamPort
	api    ports.APIPort
}

func (a *AddTimer) AddTimer(key string, cmd *config.Commands) {
	a.timers.AddTimer(key, cmd.Timer.Interval, true, map[string]any{
		"text":  cmd.Text,
		"timer": cmd.Timer,
	}, func(args map[string]any) {
		timer := args["timer"].(*config.Timers)
		if !timer.Enabled {
			return
		}

		mode := config.OnlineMode
		if timer.Options != nil && timer.Options.Mode != nil {
			mode = *timer.Options.Mode
		}

		if ((mode == config.OnlineMode || mode == 0) && !a.stream.IsLive()) || (mode == config.OfflineMode && a.stream.IsLive()) {
			return
		}

		msg := &ports.AnswerType{}
		for range timer.Count {
			msg.Text = append(msg.Text, args["text"].(string))
		}

		if _, ok := a.cfg.UsersTokens[a.stream.ChannelID()]; ok && timer.Options != nil && timer.Options.IsAnnounce != nil && *timer.Options.IsAnnounce {
			a.api.SendChatAnnouncements(a.stream.ChannelID(), msg, *timer.Options.ColorAnnounce)
			return
		}

		a.api.SendChatMessages(a.stream.ChannelID(), msg)
	})
}
