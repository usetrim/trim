package cache

import (
	"container/list"
	"sync"

	"github.com/cespare/xxhash/v2"
)

// LRU is a small in-memory least-recently-used string cache for AST outputs.
// Keys are xxhash64 of source content (bible: sub-microsecond cache lookups).
type LRU struct {
	mu       sync.Mutex
	capacity int
	ll       *list.List
	items    map[uint64]*list.Element
}

type entry struct {
	key   uint64
	value string
}

func NewLRU(capacity int) *LRU {
	if capacity < 1 {
		capacity = 256
	}
	return &LRU{
		capacity: capacity,
		ll:       list.New(),
		items:    make(map[uint64]*list.Element, capacity),
	}
}

// Hash returns xxhash64 of s (faster and lower collision risk than FNV for AST cache keys).
func Hash(s string) uint64 {
	return xxhash.Sum64String(s)
}

func (c *LRU) Get(key uint64) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return "", false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*entry).value, true
}

func (c *LRU) Set(key uint64, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		el.Value.(*entry).value = value
		return
	}
	el := c.ll.PushFront(&entry{key: key, value: value})
	c.items[key] = el
	if c.ll.Len() > c.capacity {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			delete(c.items, oldest.Value.(*entry).key)
		}
	}
}
