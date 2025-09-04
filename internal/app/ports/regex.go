package ports

import "regexp"

type RegexPort interface {
	Parse(str string) (*regexp.Regexp, error)
	SplitWords(input string) []string
}
