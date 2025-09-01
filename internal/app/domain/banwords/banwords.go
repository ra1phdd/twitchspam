package banwords

import (
	"strings"
	"twitchspam/internal/app/domain"
)

type Banwords struct {
	bwords map[string]struct{}
}

func New(list []string) *Banwords {
	bw := &Banwords{
		bwords: make(map[string]struct{}, len(list)),
	}

	for _, bword := range list {
		bw.bwords[domain.NormalizeText(strings.ToLower(bword))] = struct{}{}
	}

	return bw
}

func (bw *Banwords) CheckMessage(words []string) bool {
	for _, word := range words {
		if _, ok := bw.bwords[word]; ok {
			return true
		}
	}

	return false
}

func (bw *Banwords) CheckOnline(text string) bool {
	queries := []string{
		"че с онлайном",
		"чё с онлайном",
		"что с онлайном",
		"где онлайн",
		"че со зрителями",
		"чё со зрителями",
		"что со зрителями",
		"че по онлайну",
		"чё по онлайну",
		"что по онлайну",
		"где зрители",
		"где зрилы",
		"че так мало онлайна",
		"чё так мало онлайна",
		"почему так мало онлайна",
	}

	for _, q := range queries {
		if strings.Contains(text, q) {
			return true
		}
	}

	return false
}
