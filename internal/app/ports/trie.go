package ports

import "twitchspam/internal/app/infrastructure/trie"

type TriePort[T any] interface {
	Update(m map[string]T)
	Root() *trie.Node[T]
}

type NodePort[T any] interface {
	Children() map[string]*trie.Node[T]
	Value() *T
}
