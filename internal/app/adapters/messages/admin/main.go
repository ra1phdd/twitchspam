package admin

import (
	"fmt"
	"github.com/shirou/gopsutil/cpu"
	"runtime"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/ports"
)

func (a *Admin) handlePing() ports.ActionType {
	uptime := time.Since(startApp)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	percent, _ := cpu.Percent(0, false)
	if len(percent) == 0 {
		percent = append(percent, 0)
	}

	return ports.ActionType(fmt.Sprintf("бот работает %v • загрузка CPU %.2f%% • потребление ОЗУ %v MB", uptime.Truncate(time.Second), percent[0], m.Sys/1024/1024))
}

func (a *Admin) handleOnOff(cfg *config.Config, cmd string, _ []string) ports.ActionType {
	cfg.Enabled = cmd == "on"
	return None
}

func (a *Admin) handleReset(cfg *config.Config, _ string, _ []string) ports.ActionType {
	cfg.Spam = a.manager.GetDefault().Spam
	return None
}

func (a *Admin) handleCategory(_ *config.Config, _ string, args []string) ports.ActionType {
	if !a.stream.IsLive() {
		return NoStream
	}

	a.stream.SetCategory(strings.Join(args, " "))
	return Success
}
