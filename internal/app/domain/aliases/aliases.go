package aliases

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Aliases struct {
	aliases    map[string]string
	sortedKeys []string
}

var queryRe = regexp.MustCompile(`\{query(\d*)\}`)

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
			if strings.Contains(a.aliases[alias], "{query") {
				parts := strings.Fields(text)
				aliasParts := strings.Fields(alias)

				return a.replaceQueryPlaceholders(a.aliases[alias], parts[len(aliasParts):])
			}

			return strings.Replace(text, alias, a.aliases[alias], 1)
		}
	}
	return text
}

func (a *Aliases) replaceQueryPlaceholders(template string, args []string) string {
	nextIdx := 0
	result := queryRe.ReplaceAllStringFunc(template, func(ph string) string {
		match := queryRe.FindStringSubmatch(ph)
		if match[1] == "" {
			// {query} — берём следующий по порядку
			if nextIdx < len(args) {
				val := args[nextIdx]
				nextIdx++
				return val
			}
			return ""
		}

		// {queryN} — берём конкретный аргумент
		i, err := strconv.Atoi(match[1])
		if err != nil || i < 1 || i > len(args) {
			return ""
		}
		return args[i-1]
	})

	if nextIdx < len(args) {
		result += " " + strings.Join(args[nextIdx:], " ")
	}

	return result
}
