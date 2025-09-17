package trie

import "strings"

type Node[T any] struct {
	children map[string]*Node[T]
	value    *T
}

type Trie[T any] struct {
	root *Node[T]
}

func NewTrie[T any](m map[string]T) *Trie[T] {
	t := &Trie[T]{root: &Node[T]{children: make(map[string]*Node[T])}}
	t.Update(m)
	return t
}

func (t *Trie[T]) Update(m map[string]T) {
	t.root = &Node[T]{children: make(map[string]*Node[T])}
	for k, v := range m {
		cur := t.root
		for _, w := range strings.Fields(k) {
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
