package banwords

import (
	"github.com/dlclark/regexp2"
	"strings"
)

type Banwords struct {
	words map[string]struct{}
	re    []*regexp2.Regexp
}

func New(words []string, re []*regexp2.Regexp) *Banwords {
	bw := &Banwords{
		words: make(map[string]struct{}, len(words)),
		re:    re,
	}

	for _, bword := range words {
		bw.words[strings.TrimSpace(bword)] = struct{}{}
	}

	return bw
}

func (bw *Banwords) CheckMessage(text, textOriginal string) bool {
	for _, re := range bw.re {
		if isMatch, _ := re.MatchString(text); isMatch {
			return true
		}
	}

	words := strings.Fields(textOriginal)
	for _, word := range words {
		if _, ok := bw.words[word]; ok {
			return true
		}
	}

	return false
}
