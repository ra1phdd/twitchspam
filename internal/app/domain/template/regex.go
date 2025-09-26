package template

import (
	"strings"
)

type RegexTemplate struct{}

func NewRegex() *RegexTemplate {
	return &RegexTemplate{}
}

func (rt *RegexTemplate) MatchPhrase(words []string, phrase string) bool {
	phraseParts := strings.Split(phrase, " ")
	if len(phraseParts) == 1 {
		for _, w := range words {
			if w == phrase {
				return true
			}
		}
		return false
	}

	for i := 0; i <= len(words)-len(phraseParts); i++ {
		match := true
		for j := 0; j < len(phraseParts); j++ {
			if words[i+j] != phraseParts[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
