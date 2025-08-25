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

func (bw *Banwords) CheckMessage(text string) bool {
	words := strings.Fields(text)
	for _, word := range words {
		if _, ok := bw.bwords[word]; ok {
			return true
		}
	}

	return false
}
