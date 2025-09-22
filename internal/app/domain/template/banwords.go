package template

import (
	"regexp"
	"strings"
	"twitchspam/internal/app/infrastructure/trie"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type BanwordsTemplate struct {
	trie ports.TriePort[bool]
	re   *regexp.Regexp
}

func NewBanwords(log logger.Logger, banWords []string, banRegexps []*regexp.Regexp) *BanwordsTemplate {
	m := make(map[string]bool, len(banWords))
	for _, w := range banWords {
		w = strings.TrimSpace(w)
		if w != "" {
			m[w] = true
		}
	}

	bt := &BanwordsTemplate{
		trie: trie.NewTrie(m),
	}

	if len(banRegexps) != 0 {
		patterns := make([]string, len(banRegexps))
		for i, r := range banRegexps {
			patterns[i] = r.String()
		}
		combinedPattern := "(?i)(" + strings.Join(patterns, "|") + ")"

		var err error
		bt.re, err = regexp.Compile(combinedPattern)
		if err != nil {
			log.Error("Failed to compile regexp on banwords", err)
		}
	}

	return bt
}

func (bt *BanwordsTemplate) checkMessage(textLower string, wordsOriginal []string) bool {
	if bt.re != nil && bt.re.MatchString(textLower) {
		return true
	}

	for i := 0; i < len(wordsOriginal); i++ {
		cur := bt.trie.Root()
		j := i
		for j < len(wordsOriginal) {
			next, ok := cur.Children()[wordsOriginal[j]]
			if !ok {
				break
			}
			cur = next
			j++
			if cur.Value() != nil {
				return true
			}
		}
	}
	return false
}
