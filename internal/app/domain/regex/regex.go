package regex

import (
	"errors"
	"github.com/dlclark/regexp2"
	"strings"
)

type Regex struct{}

func New() *Regex {
	return &Regex{}
}

var InvalidRegex = errors.New("невалидное регулярное выражение")

func (r *Regex) Parse(str string) (*regexp2.Regexp, error) {
	if (strings.HasPrefix(str, `r"`) && strings.HasSuffix(str, `"`)) ||
		(strings.HasPrefix(str, `r'`) && strings.HasSuffix(str, `'`)) {
		pattern := str[2 : len(str)-1]
		re, err := regexp2.Compile(pattern, regexp2.None)
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

func (r *Regex) SplitWordsBySpace(input string) []string {
	var words []string
	var buf strings.Builder
	inRegex := false
	quoteChar := rune(0)

	for i, ch := range input {
		// Начало регулярки
		if !inRegex && (strings.HasPrefix(input[i:], `r"`) || strings.HasPrefix(input[i:], `r'`)) {
			inRegex = true
			quoteChar = rune(input[i+1]) // " или '
			buf.WriteRune(ch)            // пишем 'r'
			continue
		}

		// Конец регулярки
		if inRegex && ch == quoteChar {
			inRegex = false
			buf.WriteRune(ch)
			continue
		}

		// Разделитель пробел/таб/новая строка, если не внутри регулярки
		if !inRegex && (ch == ' ' || ch == '\t' || ch == '\n') {
			if buf.Len() > 0 {
				words = append(words, buf.String())
				buf.Reset()
			}
			continue
		}

		buf.WriteRune(ch)
	}

	if buf.Len() > 0 {
		words = append(words, buf.String())
	}

	return words
}
