package template

import (
	"strings"
	"twitchspam/internal/app/infrastructure/trie"
	"twitchspam/internal/app/ports"
)

type AliasesTemplate struct {
	trie ports.TriePort[string]
}

func NewAliases(m map[string]string) *AliasesTemplate {
	return &AliasesTemplate{
		trie: trie.NewTrie(m),
	}
}

func (at *AliasesTemplate) update(newAliases map[string]string) {
	at.trie.Update(newAliases)
}

func (at *AliasesTemplate) replace(parts []string) (string, bool) {
	var bestAlias string
	var bestStart, bestEnd int

	for i := 0; i < len(parts); i++ {
		cur := at.trie.Root()
		j := i

		for j < len(parts) {
			next, ok := cur.Children()[parts[j]]
			if !ok {
				break
			}

			cur = next
			j++

			if cur.Value() != nil && *cur.Value() != "" {
				bestAlias = *cur.Value()
				bestStart = i
				bestEnd = j
			}
		}
	}

	if bestAlias == "" {
		return "", false
	}

	// это можно было сделать в одну строку через append, но это +лишние аллокации памяти
	var sb strings.Builder
	for k := 0; k < bestStart; k++ {
		sb.WriteString(parts[k])
		sb.WriteByte(' ')
	}
	sb.WriteString(bestAlias)
	if bestEnd < len(parts) {
		sb.WriteByte(' ')
		for k := bestEnd; k < len(parts); k++ {
			sb.WriteString(parts[k])
			if k+1 < len(parts) {
				sb.WriteByte(' ')
			}
		}
	}

	return sb.String(), true
}
