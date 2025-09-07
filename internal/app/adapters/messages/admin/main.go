package admin

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"runtime"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handlePing() *ports.AnswerType {
	uptime := time.Since(startApp)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	percent, _ := cpu.Percent(0, false)
	if len(percent) == 0 {
		percent = append(percent, 0)
	}

	return &ports.AnswerType{
		Text:    []string{fmt.Sprintf("бот работает %v • загрузка CPU %.2f%% • потребление ОЗУ %v MB", uptime.Truncate(time.Second), percent[0], m.Sys/1024/1024)},
		IsReply: true,
	}
}

func (a *Admin) handleOnOff(cfg *config.Config, cmd string, _ []string) *ports.AnswerType {
	cfg.Enabled = cmd == "on"
	return nil
}

func (a *Admin) handleCategory(_ *config.Config, _ string, args []string) *ports.AnswerType {
	if !a.stream.IsLive() {
		return &ports.AnswerType{
			Text:    []string{"стрим выключен!"},
			IsReply: true,
		}
	}

	a.stream.SetCategory(strings.Join(args, " "))
	return nil
}

func (a *Admin) handleStatus(cfg *config.Config, _ string, _ []string) *ports.AnswerType {
	var parts []string
	if cfg.Enabled {
		parts = append(parts, "бот включён")
	} else {
		parts = append(parts, "бот выключен")
	}

	if cfg.Spam.SettingsDefault.Enabled {
		parts = append(parts, "антиспам включён")
	} else {
		parts = append(parts, "антиспам выключен")
	}

	return &ports.AnswerType{
		Text:    []string{strings.Join(parts, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleReset(cfg *config.Config, _ string, _ []string) *ports.AnswerType {
	cfg.Spam = a.manager.GetDefault().Spam
	return nil
}

func (a *Admin) handleSay(_ *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) == 0 {
		return &ports.AnswerType{
			Text:    []string{"не указан текст сообщения!"},
			IsReply: true,
		}
	}
	text := strings.Join(args, " ")

	return &ports.AnswerType{
		Text:    []string{text},
		IsReply: false,
	}
}

func (a *Admin) handleSpam(_ *config.Config, _ string, args []string) *ports.AnswerType {
	if len(args) < 2 {
		return NonParametr
	}

	count := args[0]
	countInt, err := strconv.Atoi(count)
	if err != nil {
		return &ports.AnswerType{
			Text:    []string{"кол-во повторов не указано или указано неверно!"},
			IsReply: true,
		}
	}

	var text []string
	for i := 0; i < countInt; i++ {
		text = append(text, strings.Join(args[1:], " "))
	}

	return &ports.AnswerType{
		Text:    text,
		IsReply: false,
	}
}
