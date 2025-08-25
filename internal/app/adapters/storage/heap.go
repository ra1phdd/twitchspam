package storage

type MsgHeap []*Message

func (h *MsgHeap) Len() int { return len(*h) }
func (h *MsgHeap) Less(i, j int) bool {
	return (*h)[i].TTL < (*h)[j].TTL
}
func (h *MsgHeap) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}

func (h *MsgHeap) Push(x interface{}) {
	*h = append(*h, x.(*Message))
}

func (h *MsgHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}
