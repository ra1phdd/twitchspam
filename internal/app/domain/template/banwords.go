package template

import (
	"github.com/dlclark/regexp2"
	"strings"
)

type BanwordsTemplate struct {
	words map[string]struct{}
	re    []*regexp2.Regexp
}

func NewBanwords(banWords []string, banRegexps []*regexp2.Regexp) *BanwordsTemplate {
	bt := &BanwordsTemplate{
		words: make(map[string]struct{}, len(banWords)),
		re:    banRegexps,
	}

	for _, bword := range banWords {
		bt.words[strings.TrimSpace(bword)] = struct{}{}
	}

	return bt
}

func (bt *BanwordsTemplate) checkMessage(text, textOriginal string) bool {
	for _, re := range bt.re {
		if isMatch, _ := re.MatchString(text); isMatch {
			return true
		}
	}

	words := strings.Fields(textOriginal)
	for _, word := range words {
		if _, ok := bt.words[word]; ok {
			return true
		}
	}

	return false
}
