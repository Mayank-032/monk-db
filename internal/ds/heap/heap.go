package heap

import (
	"errors"
)

type Comparator[T any] func(a, b T) (comp bool)

type Heap[T any] struct {
	data []T
	comp Comparator[T]
}

func NewHeap[T any](comp Comparator[T]) *Heap[T] {
	return &Heap[T]{
		data: make([]T, 0),
		comp: comp,
	}
}

func (pq *Heap[T]) Push(val T) error {
	if pq == nil || pq.data == nil {
		return errors.New("priority queue not initialized")
	}

	pq.data = append(pq.data, val)

	if pq.Size() == 1 {
		return nil
	}

	pq.upHeapify(pq.Size() - 1)

	return nil
}

func (pq *Heap[T]) Pop() (T, error) {
	var val T

	if pq == nil || pq.data == nil {
		return val, errors.New("priority queue not initialized")
	}

	if pq.Size() == 0 {
		return val, errors.New("queue underflow")
	}

	if pq.Size() == 1 {
		val = pq.data[0]
		pq.data = make([]T, 0)

		return val, nil
	}

	val = pq.data[0]
	pq.Swap(0, len(pq.data)-1)
	pq.data = pq.data[:len(pq.data)-1]

	pq.downHeapify(0)

	return val, nil
}

func (pq *Heap[T]) Peek() (T, error) {
	var val T

	if pq == nil || pq.data == nil {
		return val, errors.New("priority queue not initialized")
	}

	if pq.Size() == 0 {
		return val, errors.New("queue underflow")
	}

	val = pq.data[0]
	return val, nil
}

func (pq *Heap[T]) upHeapify(idx int) {
	if idx < 0 || idx >= pq.Size() {
		return
	}

	var parentIdx = (idx - 1) / 2

	if parentIdx < 0 || parentIdx >= pq.Size() {
		return
	}

	comp := pq.comp(pq.data[idx], pq.data[parentIdx])
	if !comp {
		return
	}

	pq.Swap(idx, parentIdx)
	if parentIdx == 0 {
		return
	}

	pq.upHeapify(parentIdx)
}

func (pq *Heap[T]) downHeapify(idx int) {
	if idx < 0 || idx >= pq.Size() {
		return
	}

	var lChildIdx int = 2*idx + 1
	var rChildIdx int = 2*idx + 2

	if (lChildIdx < 0 || lChildIdx >= pq.Size()) && (rChildIdx < 0 || rChildIdx >= pq.Size()) {
		return
	} else if !(lChildIdx < 0 || lChildIdx >= pq.Size()) && (rChildIdx < 0 || rChildIdx >= pq.Size()) {
		if comp := pq.comp(pq.data[idx], pq.data[lChildIdx]); comp {
			return
		}

		pq.Swap(idx, lChildIdx)
		pq.downHeapify(lChildIdx)
	} else if (lChildIdx < 0 || lChildIdx >= pq.Size()) && !(rChildIdx < 0 || rChildIdx >= pq.Size()) {
		if comp := pq.comp(pq.data[idx], pq.data[rChildIdx]); comp {
			return
		}

		pq.Swap(idx, rChildIdx)
		pq.downHeapify(rChildIdx)
	} else {
		var swapIdx = idx
		if lChildIdx < pq.Size() && pq.comp(pq.data[lChildIdx], pq.data[swapIdx]) {
			swapIdx = lChildIdx
		}

		if rChildIdx < pq.Size() && pq.comp(pq.data[rChildIdx], pq.data[swapIdx]) {
			swapIdx = rChildIdx
		}

		if swapIdx == idx {
			return
		}

		pq.Swap(idx, swapIdx)
		pq.downHeapify(swapIdx)
	}
}

func (pq *Heap[T]) Size() int {
	return len(pq.data)
}

func (pq *Heap[T]) IsEmpty() bool {
	if len(pq.data) == 0 {
		return true
	}

	return false
}

func (pq *Heap[T]) GetData() []T {
	return pq.data
}

func (pq *Heap[T]) Swap(idx1, idx2 int) {
	var temp = pq.data[idx1]
	pq.data[idx1] = pq.data[idx2]
	pq.data[idx2] = temp
}
