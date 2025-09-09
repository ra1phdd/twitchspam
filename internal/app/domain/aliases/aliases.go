package aliases

import (
	"regexp"
	"strings"
	"twitchspam/internal/app/ports"
)

type node struct {
	children map[string]*node // дочерние узлы для каждого слова
	alias    string           // строка замены, если путь до узла совпал с ключом
}

type Aliases struct {
	root   *node
	stream ports.StreamPort
}

var queryRe = regexp.MustCompile(`\{query(\d*)}`)

var nonGameCategories = map[string]struct{}{
	"Just Chatting":                 {},
	"IRL":                           {},
	"I'm Only Sleeping":             {},
	"DJs":                           {},
	"Music":                         {},
	"Games + Demos":                 {},
	"ASMR":                          {},
	"Special Events":                {},
	"Art":                           {},
	"Politics":                      {},
	"Pools, Hot Tubs, and Beaches":  {},
	"Slots":                         {},
	"Food & Drink":                  {},
	"Science & Technology":          {},
	"Sports":                        {},
	"Animals, Aquariums, and Zoos":  {},
	"Crypto":                        {},
	"Talk Shows & Podcasts":         {},
	"Co-working & Studying":         {},
	"Software and Game Development": {},
	"Makers & Crafting":             {},
	"Writing & Reading":             {},
}

func New(m map[string]string, stream ports.StreamPort) *Aliases {
	return &Aliases{
		root:   buildTree(m),
		stream: stream,
	}
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

func (a *Aliases) ReplacePlaceholders(text string, parts []string) string {
	text = a.replaceGamePlaceholder(text)
	text = a.replaceCategoryPlaceholder(text)
	text = a.replaceChannelPlaceholder(text)
	text = a.replaceCountdownPlaceholder(text)
	text = a.replaceCountupPlaceholder(text)
	text = a.replaceRandintPlaceholder(text)

	if strings.Index(text, "{query") == -1 {
		return text
	}

	cmdIdx := -1
	for i, part := range parts {
		if strings.Contains(part, "!") {
			cmdIdx = i
			break
		}
	}

	if cmdIdx != -1 {
		parts = parts[cmdIdx+1:]
	}

	return a.replaceQueryPlaceholders(text, parts)
}
