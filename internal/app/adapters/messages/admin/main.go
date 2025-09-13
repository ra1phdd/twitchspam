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

func (a *Admin) handleOnOff(cfg *config.Config, enabled bool) *ports.AnswerType {
	cfg.Enabled = enabled
	return nil
}

func (a *Admin) handleCategory(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am game <игра>
		return NonParametr
	}

	if !a.stream.IsLive() {
		return &ports.AnswerType{
			Text:    []string{"стрим выключен!"},
			IsReply: true,
		}
	}

	if a.stream.Category() != "Games + Demos" {
		return &ports.AnswerType{
			Text:    []string{"работает только при категории Games + Demos!"},
			IsReply: true,
		}
	}

	a.stream.SetCategory(text.Tail(2))
	return nil
}

func (a *Admin) handleStatus(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	if !cfg.Enabled {
		return &ports.AnswerType{
			Text:    []string{"бот выключен!"},
			IsReply: true,
		}
	}

	return &ports.AnswerType{
		Text: []string{strings.Join([]string{
			"бот включён", map[bool]string{true: "антиспам включён", false: "антиспам выключен"}[cfg.Spam.SettingsDefault.Enabled],
		}, " • ") + "!"},
		IsReply: true,
	}
}

func (a *Admin) handleReset(cfg *config.Config, _ *ports.MessageText) *ports.AnswerType {
	cfg.Spam = a.manager.GetDefault().Spam
	return nil
}

func (a *Admin) handleSay(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 3 { // !am say <текст>
		return NonParametr
	}

	return &ports.AnswerType{
		Text:    []string{text.Tail(2)},
		IsReply: false,
	}
}

func (a *Admin) handleSpam(_ *config.Config, text *ports.MessageText) *ports.AnswerType {
	words := text.Words()
	if len(words) < 4 { // !am spam <кол-во раз> <текст>
		return NonParametr
	}

	count, err := strconv.Atoi(words[2])
	if err != nil || count <= 0 {
		return &ports.AnswerType{
			Text:    []string{"кол-во повторов не указано или указано неверно!"},
			IsReply: true,
		}
	}

	if count > 100 {
		count = 100
	}

	msg := text.Tail(3)
	answers := make([]string, count)
	for i := range answers {
		answers[i] = msg
	}

	return &ports.AnswerType{
		Text:    answers,
		IsReply: false,
	}
}
