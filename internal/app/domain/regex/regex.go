package regex

import (
	"errors"
	"regexp"
	"strings"
)

type Regex struct{}

func New() *Regex {
	return &Regex{}
}

var InvalidRegex = errors.New("невалидное регулярное выражение")

func (r *Regex) Parse(str string) (*regexp.Regexp, error) {
	if (strings.HasPrefix(str, `r"`) && strings.HasSuffix(str, `"`)) ||
		(strings.HasPrefix(str, `r'`) && strings.HasSuffix(str, `'`)) {
		pattern := str[2 : len(str)-1]
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, InvalidRegex
		}
		return re, nil
	}

	return nil, nil
}

func (r *Regex) SplitWords(input string) []string {
	var words []string
	var buf strings.Builder
	inRegex := false
	quoteChar := rune(0)

	for i, r := range input {
		// Начало регулярки
		if !inRegex && (strings.HasPrefix(input[i:], `r"`) || strings.HasPrefix(input[i:], `r'`)) {
			inRegex = true
			quoteChar = rune(input[i+1]) // " или '
			buf.WriteRune(r)             // пишем 'r'
			continue
		}

		// Конец регулярки
		if inRegex && r == quoteChar {
			inRegex = false
			buf.WriteRune(r)
			continue
		}

		// Разделитель запятая, только если мы не внутри регулярки
		if r == ',' && !inRegex {
			words = append(words, strings.TrimSpace(buf.String()))
			buf.Reset()
			continue
		}

		buf.WriteRune(r)
	}

	if buf.Len() > 0 {
		words = append(words, strings.TrimSpace(buf.String()))
	}

	return words
}
