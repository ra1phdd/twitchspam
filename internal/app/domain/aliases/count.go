package aliases

import (
	"fmt"
	"strings"
	"time"
)

func (a *Aliases) countdown(args string) string {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		return "{неверные аргументы}"
	}

	dateStr := parts[0]
	timeStr := parts[1]

	targetTime, err := parseDateTime(dateStr, timeStr)
	if err != nil {
		return "{ошибка}"
	}

	now := time.Now()
	duration := targetTime.Sub(now)
	sign := ""
	if duration < 0 {
		sign = "-"
		duration = -duration
	}

	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	return fmt.Sprintf("%s%dд %dч %dм %dс", sign, days, hours, minutes, seconds)
}

func (a *Aliases) countup(args string) string {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		return "{неверные аргументы}"
	}

	dateStr := parts[0]
	timeStr := parts[1]

	targetTime, err := parseDateTime(dateStr, timeStr)
	if err != nil {
		return "{ошибка}"
	}

	now := time.Now()
	duration := now.Sub(targetTime) // всегда прошедшее время
	if duration < 0 {
		duration = 0 // если событие в будущем, считаем 0
	}

	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	return fmt.Sprintf("%dд %dч %dм %dс", days, hours, minutes, seconds)
}

func parseDateTime(dateStr, timeStr string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04-07:00",
		"02-01-2006 15:04:05-07:00",
		"02-01-2006 15:04-07:00",
		"2006.01.02 15:04:05-07:00",
		"2006.01.02 15:04-07:00",
		"02.01.2006 15:04:05-07:00",
		"02.01.2006 15:04-07:00",

		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"02-01-2006 15:04:05",
		"02-01-2006 15:04",
		"2006.01.02 15:04:05",
		"2006.01.02 15:04",
		"02.01.2006 15:04:05",
		"02.01.2006 15:04",
	}

	timeStr = strings.ToUpper(timeStr)
	if !strings.Contains(timeStr, ":") && strings.HasSuffix(timeStr, "PM") {
		timeStr = "12:" + timeStr // fallback
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, fmt.Sprintf("%s %s", dateStr, timeStr)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("не удалось распознать дату/время")
}
