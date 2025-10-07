package template

import (
	"strings"
	"twitchspam/internal/app/infrastructure/config"
	"twitchspam/internal/app/infrastructure/trie"
	"twitchspam/internal/app/ports"
)

type AliasesTemplate struct {
	trie ports.TriePort[string]
}

func NewAliases(aliases map[string]string, aliasGroups map[string]*config.AliasGroups, globalAliases map[string]string) *AliasesTemplate {
	at := &AliasesTemplate{
		trie: trie.NewTrie(map[string]string{}),
	}
	at.Update(aliases, aliasGroups, globalAliases)

	return at
}

func (at *AliasesTemplate) Update(newAliases map[string]string, newAliasGroups map[string]*config.AliasGroups, globalAliases map[string]string) {
	als := make(map[string]string)
	for k, v := range globalAliases {
		als[k] = v
	}

	for k, v := range newAliases {
		als[k] = v
	}

	for _, alg := range newAliasGroups {
		if !alg.Enabled {
			continue
		}

		for alias := range alg.Aliases {
			als[alias] = alg.Original
		}
	}

	at.trie.Update(als)
}

func (at *AliasesTemplate) Replace(parts []string) (string, bool) {
	var bestAlias string
	var bestStart, bestEnd int

	for i := 0; i < len(parts); i++ {
		cur := at.trie.Root()
		j := i

		for j < len(parts) {
			next, ok := cur.Children()[strings.ToLower(parts[j])]
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
