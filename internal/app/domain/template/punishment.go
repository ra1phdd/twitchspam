package template

import (
	"errors"
	"fmt"
	"regexp"
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

	switch punishment {
	case "none", "n":
		return config.Punishment{Action: "none"}, nil
	case "delete", "d":
		return config.Punishment{Action: "delete"}, nil
	case "warn", "w":
		return config.Punishment{Action: "warn"}, nil
	case "ban", "b", "0":
		return config.Punishment{Action: "ban"}, nil
	}

	units := map[string]int{
		"s": 1,
		"m": 60,
		"h": 3600,
		"d": 86400,
		"w": 604800,

		"с": 1,
		"м": 60,
		"ч": 3600,
		"д": 86400,
		"н": 604800,
	}

	re := regexp.MustCompile(`(?i)(\d+)([smhdwсмчдн])`)
	matches := re.FindAllStringSubmatch(punishment, -1)

	if len(matches) > 0 {
		total := 0
		for _, m := range matches {
			valueStr := m[1]
			unitStr := strings.ToLower(m[2])

			value, err := strconv.Atoi(valueStr)
			if err != nil {
				return config.Punishment{}, errors.New("invalid numeric value")
			}

			multiplier, ok := units[unitStr]
			if !ok {
				return config.Punishment{}, errors.New("invalid time unit")
			}

			total += value * multiplier
		}

		if total < 1 {
			total = 1
		}

		if total > 1209600 {
			total = 1209600
		}

		return config.Punishment{Action: "timeout", Duration: total}, nil
	}

	duration, err := strconv.Atoi(punishment)
	if err != nil {
		return config.Punishment{}, err
	}

	if duration < 1 {
		duration = 1
	}

	if duration > 1209600 {
		duration = 1209600
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
