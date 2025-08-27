package domain

import (
	"strings"
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

func GetByIndexOrLast(arr []int, idx int) int {
	if len(arr) == 0 {
		return 600
	}

	if idx >= len(arr) {
		return arr[len(arr)-1]
	}

	if idx < 0 {
		return arr[0]
	}
	return arr[idx]
}
