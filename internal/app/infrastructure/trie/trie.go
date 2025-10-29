package trie

import "strings"

type Mode int

const (
	WordMode Mode = iota
	CharMode
)

type Node[T any] struct {
	children map[string]*Node[T]
	value    *T
}

type Trie[T any] struct {
	root *Node[T]
	mode Mode
}

func NewTrie[T any](m map[string]T, mode Mode) *Trie[T] {
	t := &Trie[T]{root: &Node[T]{children: make(map[string]*Node[T])}, mode: mode}
	t.Update(m)
	return t
}

func (t *Trie[T]) Update(m map[string]T) {
	t.root = &Node[T]{children: make(map[string]*Node[T])}
	for k, v := range m {
		cur := t.root
		var keys []string

		if t.mode == WordMode {
			keys = strings.Fields(k)
		} else {
			keys = make([]string, 0, len([]rune(k)))
			for _, r := range k {
				keys = append(keys, string(r))
			}
		}

		for _, w := range keys {
			if cur.children[w] == nil {
				cur.children[w] = &Node[T]{children: make(map[string]*Node[T])}
			}
			cur = cur.children[w]
		}
		cur.value = new(T)
		*cur.value = v
	}
}

func (t *Trie[T]) Root() *Node[T]                { return t.root }
func (n *Node[T]) Children() map[string]*Node[T] { return n.children }
func (n *Node[T]) Value() *T                     { return n.value }

func (t *Trie[T]) Contains(runes []rune) bool {
	for i := range runes {
		cur := t.root
		for j := i; j < len(runes); j++ {
			r := runes[j]
			next, ok := cur.children[string(r)]
			if !ok {
				break
			}
			cur = next
			if cur.value != nil {
				return true
			}
		}
	}
	return false
}

func (t *Trie[T]) Match(runes []rune) bool {
	cur := t.root
	for _, r := range runes {
		next, ok := cur.children[string(r)]
		if !ok {
			return false
		}
		cur = next
	}
	return cur.value != nil
}
