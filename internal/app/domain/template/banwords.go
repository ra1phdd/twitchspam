package template

import (
	"github.com/dlclark/regexp2"
	"strings"
	"twitchspam/internal/app/infrastructure/trie"
	"twitchspam/internal/app/ports"
	"twitchspam/pkg/logger"
)

type BanwordsTemplate struct {
	trie ports.TriePort[bool]
	re   *regexp2.Regexp
}

func NewBanwords(log logger.Logger, banWords []string, banRegexps []*regexp2.Regexp) *BanwordsTemplate {
	m := make(map[string]bool, len(banWords))
	for _, w := range banWords {
		w = strings.TrimSpace(w)
		if w != "" {
			m[w] = true
		}
	}

	patterns := make([]string, len(banRegexps))
	for i, r := range banRegexps {
		patterns[i] = r.String()
	}
	combinedPattern := "(?i)(" + strings.Join(patterns, "|") + ")"
	re, err := regexp2.Compile(combinedPattern, regexp2.IgnoreCase)
	if err != nil {
		log.Error("Failed to compile regexp on banwords", err)
	}

	return &BanwordsTemplate{
		trie: trie.NewTrie(m),
		re:   re,
	}
}

func (bt *BanwordsTemplate) checkMessage(textLower string, wordsOriginal []string) bool {
	if isMatch, _ := bt.re.MatchString(textLower); isMatch {
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
