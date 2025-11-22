package template_test

import (
	"regexp"
	"strings"
	"testing"
	"time"
	"twitchspam/internal/app/domain/message"
	"twitchspam/internal/app/domain/template"
	"twitchspam/internal/app/infrastructure/config"
)

func TestNuke(t *testing.T) {
	// !am nuke <*наказание> <*длительность> <*scrollback> <слова/фразы через запятую или regex>
	tmpl := template.New()
	msg := &message.Text{Original: "!am nuke 84300 дрейк, дреб"}

	reAll := regexp.MustCompile(`(?i)^!am nuke(?:\s+(\S+))?(?:\s+(\S+))?(?:\s+(\S+))?\s+(.+)$`)
	matches := reAll.FindStringSubmatch(msg.Text(message.NormalizeCommaSpacesOption))
	if len(matches) != 5 {
		t.Error("Expected 5 matches, got ", len(matches))
	}

	var globalErrs []string
	punishment := config.Punishment{
		Action:   "timeout",
		Duration: 60,
	}
	duration := 5 * time.Minute
	scrollback := 60 * time.Second

	if strings.TrimSpace(matches[1]) != "" {
		p, err := tmpl.Punishment().Parse(strings.TrimSpace(matches[1]), false)
		if err != nil {
			globalErrs = append(globalErrs, "не удалось распарсить наказание, применено дефолтное (60))")
		} else {
			punishment = p
		}
	}

	if strings.TrimSpace(matches[2]) != "" {
		if val, ok := tmpl.Parser().ParseIntArg(strings.TrimSpace(matches[2]), 1, 3600); ok {
			duration = time.Duration(val) * time.Second
		}
	}

	if strings.TrimSpace(matches[3]) != "" {
		if val, ok := tmpl.Parser().ParseIntArg(strings.TrimSpace(matches[3]), 1, 180); ok {
			scrollback = time.Duration(val) * time.Second
		}
	}

	if strings.TrimSpace(matches[4]) == "" {
		t.Error("not text")
	}

	reWords := regexp.MustCompile(`(?i)r'(.*?)'|r"(.*?)"|'(.*?)'|"(.*?)"|([^,'"\s]+)`)
	wordsMatches := reWords.FindAllStringSubmatch(strings.TrimSpace(matches[4]), -1)

	var containsWords, words []string
	var re *regexp.Regexp
	for _, m := range wordsMatches {
		switch {
		case strings.TrimSpace(m[1]) != "": // r'...'
			re, _ = regexp.Compile(strings.TrimSpace(m[1]))
		case strings.TrimSpace(m[2]) != "": // r"..."
			re, _ = regexp.Compile(strings.TrimSpace(m[2]))
		case strings.TrimSpace(m[3]) != "": // '...'
			words = append(words, strings.TrimSpace(m[3]))
		case strings.TrimSpace(m[4]) != "": // "..."
			words = append(words, strings.TrimSpace(m[4]))
		case strings.TrimSpace(m[5]) != "": // bareword
			containsWords = append(containsWords, strings.TrimSpace(m[5]))
		}
	}

	t.Log(punishment, duration, scrollback, re, words, containsWords)
}
