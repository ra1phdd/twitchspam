package aliases

import (
	"regexp"
	"strconv"
	"strings"
)

type node struct {
	children map[string]*node // дочерние узлы для каждого слова
	alias    string           // строка замены, если путь до узла совпал с ключом
}

type Aliases struct {
	root *node
}

var queryRe = regexp.MustCompile(`\{query(\d*)}`)

func New(m map[string]string) *Aliases {
	return &Aliases{root: buildTree(m)}
}

func (a *Aliases) Update(newAliases map[string]string) {
	a.root = buildTree(newAliases)
}

func buildTree(m map[string]string) *node {
	root := &node{children: make(map[string]*node)}
	for k, v := range m {
		cur := root
		for _, w := range strings.Fields(k) {
			if cur.children[w] == nil {
				cur.children[w] = &node{children: make(map[string]*node)}
			}
			cur = cur.children[w]
		}
		cur.alias = v
	}
	return root
}

func (a *Aliases) ReplaceOne(text string) string {
	var bestAlias string
	var bestStart, bestEnd int
	parts := strings.Fields(text)

	for i := 0; i < len(parts); i++ {
		cur := a.root
		j := i

		for j < len(parts) {
			next, ok := cur.children[parts[j]]
			if !ok {
				break
			}

			cur = next
			j++

			if cur.alias != "" {
				bestAlias = cur.alias
				bestStart = i
				bestEnd = j
			}
		}
	}

	if bestAlias == "" {
		return text
	}

	if strings.Index(bestAlias, "{query") != -1 {
		return a.replaceQueryPlaceholders(bestAlias, parts[bestEnd:])
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
	return sb.String()
}

func (a *Aliases) replaceQueryPlaceholders(template string, queryParts []string) string {
	var sb strings.Builder
	nextIdx := 0
	last := 0

	for _, loc := range queryRe.FindAllStringSubmatchIndex(template, -1) {
		sb.WriteString(template[last:loc[0]])
		matchNum := template[loc[2]:loc[3]]
		if matchNum == "" {
			if nextIdx < len(queryParts) {
				sb.WriteString(queryParts[nextIdx])
				nextIdx++
			}
		} else {
			i, _ := strconv.Atoi(matchNum)
			if i >= 1 && i <= len(queryParts) {
				sb.WriteString(queryParts[i-1])
			}
		}
		last = loc[1]
	}

	sb.WriteString(template[last:])
	if nextIdx < len(queryParts) {
		sb.WriteByte(' ')
		for k := nextIdx; k < len(queryParts); k++ {
			sb.WriteString(queryParts[k])
			if k+1 < len(queryParts) {
				sb.WriteByte(' ')
			}
		}
	}

	return sb.String()
}
