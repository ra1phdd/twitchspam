package template

import (
	"fmt"
	"strconv"
	"strings"
	"twitchspam/internal/app/infrastructure/config"
)

type PunishmentTemplate struct{}

func NewPunishment() *PunishmentTemplate {
	return &PunishmentTemplate{}
}

func (p *PunishmentTemplate) Parse(punishment string, allowInherit bool) (config.Punishment, error) {
	punishment = strings.TrimSpace(punishment)
	if punishment == "-" {
		return config.Punishment{Action: "delete"}, nil
	}

	if allowInherit && punishment == "*" {
		return config.Punishment{Action: "inherit"}, nil
	}

	if punishment == "0" {
		return config.Punishment{Action: "ban"}, nil
	}

	duration, err := strconv.Atoi(punishment)
	if err != nil || duration < 1 || duration > 1209600 {
		return config.Punishment{}, fmt.Errorf("invalid timeout value")
	}

	return config.Punishment{Action: "timeout", Duration: duration}, nil
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
	case "delete":
		result = "удаление сообщения"
	case "timeout":
		result = fmt.Sprintf("таймаут (%d)", punishment.Duration)
	case "ban":
		result = "бан"
	}

	return result
}
