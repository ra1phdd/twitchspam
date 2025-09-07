package ports

import "github.com/dlclark/regexp2"

type RegexPort interface {
	Parse(str string) (*regexp2.Regexp, error)
	SplitWords(input string) []string
	SplitWordsBySpace(input string) []string
}
