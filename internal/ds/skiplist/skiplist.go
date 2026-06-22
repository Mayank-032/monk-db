package skiplist

import (
	"golang.org/x/exp/constraints"
)

type Node[T constraints.Ordered] struct {
	val  T
	next []*Node[T]
}

func NewNode[T constraints.Ordered](v T) *Node[T] {
	return &Node[T]{
		val:  v,
		next: make([]*Node[T], 0),
	}
}

var (
	currentMaxLevel = 0
)

func New[T constraints.Ordered](level int) *Node[T] {
	currentMaxLevel = level

	var zero T
	var head = NewNode(zero)
	head.next[0] = nil

	return head
}

func (n *Node[T]) Search(val T) *Node[T] {
	// Start at topmost level of current node (here it means rightmost index)
	var path = make([]*Node[T], currentMaxLevel)
	var level = n._search_(val, path, currentMaxLevel)

	if level == -1 {
		return nil
	}

	return path[level]
}

func (n *Node[T]) _search_(val T, path []*Node[T], currLevel int) int {
	/*
		1) Base condition if level is 0 and element is not found, simply return the path with current
			element's index (if not found return -1)
		2) Loop over the current level of nodes
			1.1) If element is found, return node
			1.2) if not, go one level down recursively (keep track of path traversed)
	*/

	if n == nil {
		return -1
	}

	if currLevel < 0 {
		return -1
	}

	var currNode = n
	for currNode.next[currLevel] != nil {
		if currNode.val == val {
			path[currLevel] = currNode
			return currLevel
		}

		if val < currNode.next[currLevel].val {
			path[currLevel] = currNode
			break
		}

		currNode = currNode.next[currLevel]
	}

	n = currNode

	return n._search_(val, path, currLevel-1)
}
