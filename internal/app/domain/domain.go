package domain

import (
	"strings"
	"time"
	"twitchspam/internal/app/infrastructure/config"
	"unicode"
)

func JaccardSimilarity(a, b string) float64 {
	wa := strings.Fields(a)
	wb := strings.Fields(b)
	setA := make(map[string]struct{}, len(wa))
	setB := make(map[string]struct{}, len(wb))
	for _, w := range wa {
		setA[w] = struct{}{}
	}
	for _, w := range wb {
		setB[w] = struct{}{}
	}

	intersection := 0
	union := make(map[string]struct{}, len(setA)+len(setB))
	for w := range setA {
		if _, ok := setB[w]; ok {
			intersection++
		}
		union[w] = struct{}{}
	}
	for w := range setB {
		union[w] = struct{}{}
	}
	if len(union) == 0 {
		return 0
	}
	return float64(intersection) / float64(len(union))
}

func NormalizeText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	var prev rune
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) && !unicode.IsSpace(r) {
			continue
		}
		if r == prev && unicode.IsLetter(r) {
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return strings.ToLower(b.String())
}

func GetPunishment(arr []config.Punishment, idx int) (string, time.Duration) {
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
