package template

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
)

type PunishmentTemplate struct {
}

func NewPunishment() *PunishmentTemplate {
	return &PunishmentTemplate{}
}

func (p *PunishmentTemplate) Parse(punishment string, allowInherit bool) (config.Punishment, error) {
	punishment = strings.TrimSpace(punishment)
	if allowInherit && punishment == "*" {
		return config.Punishment{Action: "inherit"}, nil
	}

	if punishment == "none" || punishment == "n" {
		return config.Punishment{Action: "none"}, nil
	}

	if punishment == "delete" || punishment == "d" {
		return config.Punishment{Action: "delete"}, nil
	}

	if punishment == "warn" || punishment == "w" {
		return config.Punishment{Action: "warn"}, nil
	}

	if punishment == "ban" || punishment == "b" || punishment == "0" {
		return config.Punishment{Action: "ban"}, nil
	}

	duration, err := strconv.Atoi(punishment)
	if err != nil || duration < 1 || duration > 1209600 {
		return config.Punishment{}, errors.New("invalid timeout value")
	}

	return config.Punishment{Action: "timeout", Duration: duration}, nil
}

func (p *PunishmentTemplate) Get(arr []config.Punishment, idx int) (string, time.Duration) {
	if len(arr) == 0 {
		return "timeout", 600 * time.Second
	}

	if idx >= len(arr) {
		return arr[len(arr)-1].Action, time.Duration(arr[len(arr)-1].Duration) * time.Second
	}

	if idx < 0 {
		return arr[0].Action, time.Duration(arr[0].Duration) * time.Second
	}
	return arr[idx].Action, time.Duration(arr[idx].Duration) * time.Second
}

func (p *PunishmentTemplate) FormatAll(punishments []config.Punishment) []string {
	result := make([]string, 0, len(punishments))
	for _, punish := range punishments {
		result = append(result, p.Format(punish))
	}
	return result
}

func (p *PunishmentTemplate) Format(punishment config.Punishment) string {
	var result string
	switch punishment.Action {
	case "none":
		result = "скип"
	case "delete":
		result = "удаление сообщения"
	case "timeout":
		result = fmt.Sprintf("таймаут (%d)", punishment.Duration)
	case "ban":
		result = "бан"
	}

	return result
}
