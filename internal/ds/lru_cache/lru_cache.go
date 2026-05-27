package cache

import (
	"errors"
	"monk-db/internal/constants"
)

type Node struct {
	Key  string
	Val  interface{}
	Prev *Node
	Next *Node
}

func NewNode(key string, val interface{}) *Node {
	return &Node{
		Key:  key,
		Val:  val,
		Prev: nil,
		Next: nil,
	}
}

type lruCache struct {
	Capacity int
	Map      map[string]*Node
	Head     *Node
	Tail     *Node
}

var Cache *lruCache

func NewLRUCache(capacity int) {
	var head = NewNode("", "")
	var tail = NewNode("", "")
	head.Next = tail
	tail.Prev = head

	Cache = &lruCache{
		Capacity: capacity,
		Map:      make(map[string]*Node),
		Head:     head,
		Tail:     tail,
	}
}

func (c *lruCache) Get(key string) (interface{}, error) {
	if key == constants.EMPTYSTRING {
		return "", nil
	}

	if node, ok := c.Map[key]; ok {
		c._delete(node)
		c._insert(node)
		return node.Val, nil
	}

	return constants.EMPTYSTRING, errors.New(constants.ERRNOTFOUND)
}

func (c *lruCache) PUT(key string, val interface{}) {
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

func (c *lruCache) _insert(node *Node) {
	var nextNode = c.Head.Next
	var prevNode = c.Head

	node.Next = nextNode
	node.Prev = prevNode

	prevNode.Next = node
	nextNode.Prev = node
}

func (c *lruCache) _delete(node *Node) {
	var prevNode = node.Prev
	var nextNode = node.Next

	prevNode.Next = nextNode
	nextNode.Prev = prevNode
}
