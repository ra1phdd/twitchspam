package template

import (
	"github.com/dlclark/regexp2"
	"strings"
	"testing"
	"twitchspam/internal/app/infrastructure/trie"
)

func BenchmarkCheckMessage(b *testing.B) {
	words := []string{
		"неоЖИДанно",
		"НЕГРамотный",
		"асПИДОРАС",
		"асПИДОРАСЫ",
		"асПИДОР",
		"асПИДОРЫ",
		"ПЕДИКюр",
		"ХАЧу",
		"НИГЕРмания",
		"СИМПл",
	}

	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[strings.TrimSpace(w)] = true
	}

	bt := &BanwordsTemplate{
		trie: trie.NewTrie(m),
		re:   regexp2.MustCompile("(?i)пидр|(?<!ас)пидор|пидар|(?<!\\p{L})педик(?!ю)|негр(?!амотн)|(?\u003c!\\p{L})хач(?:а|ем|е|и|ей|ам|ами|ах|ик|ику|ике|иках|ики)?(?!\\p{L})|(?\u003c!\\p{L})жид(?:а|ом|у|ов|е|ы|ам|ами|ах|ик|ику|ике|иках|ики)?(?!\\p{L})|(?:хохляцк\\p{L}*|хох(?:ол|лы|лу|лам|ле|лом|лами|лов|лах))|(?:хахляцк\\p{L}*|хах(?:ол|лы|лу|лам|ле|лом|лами|лов|лах))|(?\u003c!\\p{L})(?:русня\\p{L}*|русн(?:ей|е|ю|ёй|и))(?!\\p{L})", regexp2.None),
	}

	message := "Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, хач and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum."
	message = strings.ToLower(message)
	parts := strings.Fields(message)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bt.checkMessage(message, parts)
	}
}
