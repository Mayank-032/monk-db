package utils

import "errors"

type Comparator[T any] func(a, b T) bool

type PriorityQueue[T any] struct {
	data []T
	comp Comparator[T]
}

func NewPriorityQueue[T any](comp Comparator[T]) *PriorityQueue[T] {
	return &PriorityQueue[T]{
		data: make([]T, 0),
		comp: comp,
	}
}

func (pq *PriorityQueue[T]) Push(val T) error {
	if pq == nil || pq.data == nil {
		return errors.New("priority queue not initialized")
	}

	pq.data = append(pq.data, val)
	pq.upHeapify(pq.Size() - 1)

	return nil
}

func (pq *PriorityQueue[T]) Pop() (T, error) {
	var val T

	if pq == nil || pq.data == nil {
		return val, errors.New("priority queue not initialized")
	}

	if pq.Size() == 0 {
		return val, errors.New("queue underflow")
	}

	val = pq.data[0]
	pq.data = pq.data[1:]

	pq.downHeapify(0)

	return val, nil
}

func (pq *PriorityQueue[T]) Peek() (T, error) {
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

func (pq *PriorityQueue[T]) upHeapify(idx int) {
	if idx < 0 || idx >= len(pq.data) {
		return
	}

	var parentIdx = (idx - 1) / 2

	if !pq.comp(pq.data[idx], pq.data[parentIdx]) {
		return
	}

	var temp = pq.data[idx]
	pq.data[idx] = pq.data[parentIdx]
	pq.data[parentIdx] = temp

	pq.upHeapify(parentIdx)
}

func (pq *PriorityQueue[T]) downHeapify(idx int) {
	if idx < 0 || idx >= len(pq.data) {
		return
	}

	var lChildIdx int = 2*idx + 1
	var rChildIdx int = 2*idx + 2
	if pq.comp(pq.data[idx], pq.data[lChildIdx]) && pq.comp(pq.data[idx], pq.data[rChildIdx]) {
		return
	}

	var temp = pq.data[idx]
	if pq.comp(pq.data[idx], pq.data[lChildIdx]) {
		pq.data[idx] = pq.data[lChildIdx]
		pq.data[lChildIdx] = temp

		pq.downHeapify(lChildIdx)
	} else {
		pq.data[idx] = pq.data[rChildIdx]
		pq.data[rChildIdx] = temp

		pq.downHeapify(rChildIdx)
	}
}

func (pq *PriorityQueue[T]) Size() int {
	return len(pq.data)
}
