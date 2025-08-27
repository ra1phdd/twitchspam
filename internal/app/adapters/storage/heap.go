package storage

type ItemHeap[T any] []*Item[T]

func (h *ItemHeap[T]) Len() int { return len(*h) }
func (h *ItemHeap[T]) Less(i, j int) bool {
	return (*h)[i].TTL < (*h)[j].TTL
}
func (h *ItemHeap[T]) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

func (h *ItemHeap[T]) Push(x interface{}) {
	*h = append(*h, x.(*Item[T]))
}

func (h *ItemHeap[T]) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
