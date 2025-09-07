package aliases

import (
	"sort"
	"strings"
)

type Aliases struct {
	aliases    map[string]string
	sortedKeys []string
}

func New(aliases map[string]string) *Aliases {
	m := &Aliases{
		aliases: make(map[string]string),
	}
	m.Update(aliases)

	return m
}

func (a *Aliases) Update(newAliases map[string]string) {
	a.aliases = make(map[string]string, len(newAliases))
	a.sortedKeys = make([]string, 0, len(newAliases))
	for k, v := range newAliases {
		a.aliases[k] = v
		a.sortedKeys = append(a.sortedKeys, k)
	}
	sort.Slice(a.sortedKeys, func(i, j int) bool {
		return len(a.sortedKeys[i]) > len(a.sortedKeys[j])
	})
}

func (a *Aliases) ReplaceOne(text string) string {
	for _, alias := range a.sortedKeys {
		if strings.Contains(text, alias) {
			text = strings.Replace(text, alias, a.aliases[alias], 1)
		}
	}
	return text
}
