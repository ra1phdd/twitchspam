package template

import (
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/trie"
	"twitchspam/internal/app/ports"
)

type BanwordsTemplate struct {
	trieWords    ports.TriePort[struct{}]
	trieCase     ports.TriePort[struct{}]
	trieContains ports.TriePort[struct{}]
	trieExclude  ports.TriePort[struct{}]
}

func NewBanwords(banwords config.Banwords) *BanwordsTemplate {
	mWords := make(map[string]struct{})
	mCase := make(map[string]struct{})
	mContains := make(map[string]struct{})
	mExclude := make(map[string]struct{})

	for _, word := range banwords.Words {
		mWords[word] = struct{}{}
		mWords[transliterateRuToEnKeyboard(word)] = struct{}{}
	}

	for _, word := range banwords.ContainsWords {
		mContains[word] = struct{}{}
		mContains[transliterateRuToEnKeyboard(word)] = struct{}{}
	}

	for _, word := range banwords.CaseSensitiveWords {
		mCase[word] = struct{}{}
		mCase[transliterateRuToEnKeyboard(word)] = struct{}{}
	}

	for _, word := range banwords.ExcludeWords {
		mExclude[word] = struct{}{}
		mExclude[transliterateRuToEnKeyboard(word)] = struct{}{}
	}

	bt := &BanwordsTemplate{
		trieWords:    trie.NewTrie(mWords, trie.CharMode),
		trieCase:     trie.NewTrie(mCase, trie.CharMode),
		trieContains: trie.NewTrie(mContains, trie.CharMode),
		trieExclude:  trie.NewTrie(mExclude, trie.CharMode),
	}

	return bt
}

func (bt *BanwordsTemplate) CheckMessage(wordsOriginal, wordsLower []string) bool {
	check := func(word []rune, trie ports.TriePort[struct{}]) bool {
		if trie.Contains(word) {
			return !bt.trieExclude.Contains(word)
		}
		return false
	}

	for _, word := range wordsOriginal {
		if check([]rune(word), bt.trieCase) {
			return true
		}
	}

	for _, word := range wordsLower {
		runes := []rune(word)
		if check(runes, bt.trieContains) {
			return true
		}

		if bt.trieWords.Match(runes) {
			return true
		}
	}

	return false
}

var ruToEnKeyboard = map[rune]rune{
	'а': 'f', 'б': ',', 'в': 'd', 'г': 'u', 'д': 'l',
	'е': 't', 'ё': '`', 'ж': ';', 'з': 'p', 'и': 'b',
	'й': 'q', 'к': 'r', 'л': 'k', 'м': 'v', 'н': 'y',
	'о': 'j', 'п': 'g', 'р': 'h', 'с': 'c', 'т': 'n',
	'у': 'e', 'ф': 'a', 'х': '[', 'ц': 'w', 'ч': 'x',
	'ш': 'i', 'щ': 'o', 'ъ': ']', 'ы': 's', 'ь': 'm',
	'э': '\'', 'ю': '.', 'я': 'z',
}

func transliterateRuToEnKeyboard(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if mapped, ok := ruToEnKeyboard[r]; ok {
			out = append(out, mapped)
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
