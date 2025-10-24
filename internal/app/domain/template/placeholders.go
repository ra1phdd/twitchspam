package template

import (
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"
	"twitchspam/internal/app/domain"
	"twitchspam/internal/app/ports"
)

type PlaceholdersTemplate struct {
	stream ports.StreamPort

	placeholders      map[string]func(args []string) string
	queryRe           *regexp.Regexp
	nonGameCategories map[string]struct{}
}

func NewPlaceholders(stream ports.StreamPort) *PlaceholdersTemplate {
	nonGameCategories := []string{
		"", "Just Chatting", "IRL", "I'm Only Sleeping", "DJs", "Music",
		"Games + Demos", "ASMR", "Special Events", "Art", "Politics",
		"Pools, Hot Tubs, and Beaches", "Slots", "Food & Drink",
		"Science & Technology", "Sports", "Animals, Aquariums, and Zoos",
		"Crypto", "Talk Shows & Podcasts", "Co-working & Studying",
		"Software and Game Development", "Makers & Crafting", "Writing & Reading",
	}

	pt := &PlaceholdersTemplate{
		stream:            stream,
		queryRe:           regexp.MustCompile(`\{query(?: (\d+))?\}`),
		nonGameCategories: make(map[string]struct{}, len(nonGameCategories)),
	}

	for _, c := range nonGameCategories {
		pt.nonGameCategories[c] = struct{}{}
	}

	pt.placeholders = map[string]func(args []string) string{
		"game":     pt.placeholderGame,
		"category": pt.placeholderCategory,
		"channel":  pt.placeholderChannel,
		"randint":  pt.placeholderRandint,
		"countdown": func(args []string) string {
			return pt.placeholderTime(args, false)
		},
		"countup": func(args []string) string {
			return pt.placeholderTime(args, true)
		},
	}

	return pt
}

func (pt *PlaceholdersTemplate) ReplaceAll(text string, parts []string) string {
	phs := pt.parse(text)

	for ph, argsList := range phs {
		if ph == "query" {
			continue
		}

		fn, exists := pt.placeholders[ph]
		if !exists {
			continue
		}

		replacement := fn(argsList)
		if replacement == "" {
			return ""
		}

		text = pt.replace(text, ph, replacement)
	}

	if !strings.Contains(text, "{query") {
		return text
	}

	cmdIdx := -1
	for i, part := range parts {
		if strings.HasPrefix(part, "!") {
			cmdIdx = i
			break
		}
	}

	if cmdIdx != -1 && cmdIdx+1 < len(parts) {
		parts = parts[cmdIdx+1:]
	} else {
		parts = nil
	}

	return pt.placeholderQuery(text, parts)
}

func (pt *PlaceholdersTemplate) parse(s string) map[string][]string {
	result := make(map[string][]string)

	for {
		start := strings.IndexByte(s, '{')
		if start == -1 {
			break
		}

		end := strings.IndexByte(s[start+1:], '}')
		if end == -1 {
			break
		}
		end += start + 1

		parts := strings.Fields(s[start+1 : end])
		if len(parts) > 0 {
			key := parts[0]
			args := parts[1:]
			result[key] = append(result[key], args...)
		}

		s = s[end+1:]
	}

	return result
}

func (pt *PlaceholdersTemplate) replace(s, key, replacement string) string {
	var sb strings.Builder
	start := 0

	for {
		openBracket := strings.IndexByte(s[start:], '{')
		if openBracket == -1 {
			sb.WriteString(s[start:])
			break
		}
		openBracket += start
		sb.WriteString(s[start:openBracket])

		closeBracket := strings.IndexByte(s[openBracket+1:], '}')
		if closeBracket == -1 {
			sb.WriteString(s[openBracket:])
			break
		}
		closeBracket += openBracket + 1

		parts := strings.Fields(s[openBracket+1 : closeBracket])
		if len(parts) > 0 && parts[0] == key {
			sb.WriteString(replacement)
		} else {
			sb.WriteString(s[openBracket : closeBracket+1])
		}

		start = closeBracket + 1
	}

	return sb.String()
}

func (pt *PlaceholdersTemplate) placeholderGame(args []string) string {
	if _, ok := pt.nonGameCategories[pt.stream.Category()]; ok {
		if len(args) > 0 && args[0] == "true" {
			return "игра не найдена"
		}
		return ""
	}

	return pt.stream.Category()
}

func (pt *PlaceholdersTemplate) placeholderCategory(args []string) string {
	category := pt.stream.Category()
	if category == "" {
		if len(args) > 0 && args[0] == "false" {
			return ""
		}
		return "игра не найдена"
	}

	return category
}

func (pt *PlaceholdersTemplate) placeholderChannel(_ []string) string {
	return pt.stream.ChannelName()
}

func (pt *PlaceholdersTemplate) placeholderRandint(args []string) string {
	if len(args) != 2 {
		return "неверные аргументы команды"
	}

	minInt, err1 := strconv.Atoi(args[0])
	maxInt, err2 := strconv.Atoi(args[1])
	if err1 != nil || err2 != nil {
		return "неверные аргументы команды"
	}

	if minInt > maxInt {
		minInt, maxInt = maxInt, minInt
	}

	// #nosec G404
	num := rand.Intn(maxInt-minInt+1) + minInt
	return strconv.Itoa(num)
}

func (pt *PlaceholdersTemplate) placeholderTime(args []string, countUp bool) string {
	if len(args) < 2 {
		return "неверные аргументы команды"
	}

	targetTime, err := domain.ParseDateTime(args[0], args[1])
	if err != nil {
		return "неверные аргументы команды"
	}

	var duration time.Duration
	if countUp {
		duration = time.Since(targetTime)
	} else {
		duration = time.Until(targetTime)
	}

	return domain.FormatDuration(duration)
}

func (pt *PlaceholdersTemplate) placeholderQuery(template string, queryParts []string) string {
	nextIdx := 0
	return pt.queryRe.ReplaceAllStringFunc(template, func(m string) string {
		matches := pt.queryRe.FindStringSubmatch(m)
		if len(matches) == 0 {
			return m
		}

		if matches[1] != "" {
			i, _ := strconv.Atoi(matches[1])
			if i >= 1 && i <= len(queryParts) {
				return queryParts[i-1]
			}
			return ""
		}

		if nextIdx < len(queryParts) {
			val := queryParts[nextIdx]
			nextIdx++
			return val
		}

		return ""
	})
}
