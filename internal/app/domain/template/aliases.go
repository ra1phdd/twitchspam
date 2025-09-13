package template

import (
	"strings"
)

type node struct {
	children map[string]*node // дочерние узлы для каждого слова
	alias    string           // строка замены, если путь до узла совпал с ключом
}

type AliasesTemplate struct {
	root *node
}

func NewAliases(m map[string]string) *AliasesTemplate {
	at := &AliasesTemplate{}
	at.root = at.buildTree(m)

	return at
}

func (at *AliasesTemplate) buildTree(m map[string]string) *node {
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

func (at *AliasesTemplate) update(newAliases map[string]string) {
	at.root = at.buildTree(newAliases)
}

func (at *AliasesTemplate) replace(text string) string {
	var bestAlias string
	var bestStart, bestEnd int

	parts := strings.Fields(text)
	for i := 0; i < len(parts); i++ {
		cur := at.root
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
