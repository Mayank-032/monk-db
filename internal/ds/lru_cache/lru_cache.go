package cache

import (
	"errors"
	"monk-db/internal/constants"
)

type Node[T any] struct {
	Key  string
	Val  T
	Prev *Node[T]
	Next *Node[T]
}

func NewNode[T any](key string, val T) *Node[T] {
	return &Node[T]{
		Key:  key,
		Val:  val,
		Prev: nil,
		Next: nil,
	}
}

type Cache[T any] struct {
	Capacity int
	Map      map[string]*Node[T]
	Head     *Node[T]
	Tail     *Node[T]
}

func NewLRUCache[T any](capacity int) *Cache[T] {
	var zero T
	var head = NewNode("", zero)
	var tail = NewNode("", zero)
	head.Next = tail
	tail.Prev = head

	return &Cache[T]{
		Capacity: capacity,
		Map:      make(map[string]*Node[T]),
		Head:     head,
		Tail:     tail,
	}
}

func (c *Cache[T]) Get(key string) (T, error) {
	var zero T

	if key == constants.EMPTYSTRING {
		return zero, nil
	}

	if node, ok := c.Map[key]; ok {
		c._delete(node)
		c._insert(node)
		return node.Val, nil
	}

	return zero, errors.New(constants.ERRNOTFOUND)
}

func (c *Cache[T]) PUT(key string, val T) {
	if key == constants.EMPTYSTRING {
		return
	}

	if node, ok := c.Map[key]; ok {
		c._delete(node)
	}

	var node = NewNode(key, val)
	c.Map[key] = node
	c._insert(node)

	if len(c.Map) > c.Capacity {
		node = c.Tail.Prev
		c._delete(node)
		delete(c.Map, node.Key)
	}
}

func (c *Cache[T]) _insert(node *Node[T]) {
	var nextNode = c.Head.Next
	var prevNode = c.Head

	node.Next = nextNode
	node.Prev = prevNode

	prevNode.Next = node
	nextNode.Prev = node
}

func (c *Cache[T]) _delete(node *Node[T]) {
	var prevNode = node.Prev
	var nextNode = node.Next

	prevNode.Next = nextNode
	nextNode.Prev = prevNode
}
