package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func JaccardHashSimilarity(a, b []string) float64 {
	if len(a) > len(b) {
		a, b = b, a
	}

	set := make(map[string]struct{}, len(a))
	for _, h := range a {
		set[h] = struct{}{}
	}

	intersection := 0
	for _, h := range b {
		if _, ok := set[h]; ok {
			intersection++
		}
	}

	unionSize := len(a) + len(b) - intersection
	if unionSize == 0 {
		return 0
	}
	return float64(intersection) / float64(unionSize)
}

func ParseDateTime(dateStr, timeStr string) (time.Time, error) {
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
	return time.Time{}, errors.New("не удалось распознать дату/время")
}

func pluralize(n int, forms [3]string) string {
	n %= 100
	if n >= 11 && n <= 19 {
		return forms[2]
	}
	switch n % 10 {
	case 1:
		return forms[0]
	case 2, 3, 4:
		return forms[1]
	default:
		return forms[2]
	}
}

func FormatDuration(d time.Duration) string {
	negative := d < 0
	if negative {
		d = -d
	}

	totalSec := int(d.Seconds())
	days := totalSec / 86400
	hours := (totalSec % 86400) / 3600
	minutes := (totalSec % 3600) / 60
	seconds := totalSec % 60

	var result []string
	if days > 0 {
		result = append(result, fmt.Sprintf("%d %s", days, pluralize(days, [3]string{"день", "дня", "дней"})))
	}
	if hours > 0 {
		result = append(result, fmt.Sprintf("%d %s", hours, pluralize(hours, [3]string{"час", "часа", "часов"})))
	}
	if minutes > 0 {
		result = append(result, fmt.Sprintf("%d %s", minutes, pluralize(minutes, [3]string{"минута", "минуты", "минут"})))
	}
	if seconds > 0 || len(result) == 0 {
		result = append(result, fmt.Sprintf("%d %s", seconds, pluralize(seconds, [3]string{"секунда", "секунды", "секунд"})))
	}

	res := strings.Join(result, " ")
	if negative {
		res = "-" + res
	}
	return res
}
