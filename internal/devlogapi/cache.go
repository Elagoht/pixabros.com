package devlogapi

import (
	"container/list"
	"sync"
)

// searchCache is a tiny LRU cache for the public search endpoint. A query page
// is cheap to compute but the same directory/year/search terms come back on
// every keystroke when a visitor is narrowing a list, so holding the last few
// responses in memory keeps those requests off the database.
//
// The capacity is deliberately small: search results are per-request JSON, not
// shared state, so an eviction only costs the next miss a query.
type searchCache struct {
	mu    sync.Mutex
	items map[string]*list.Element
	lru   *list.List
	cap   int
}

type searchCacheEntry struct {
	key  string
	body []byte
}

// NewSearchCache builds an LRU cache holding up to capacity responses. The
// router hands one instance to both the public search handler and the admin
// handlers, so a mutation clears the same store the search reads from.
func NewSearchCache(capacity int) *searchCache {
	if capacity < 1 {
		capacity = 1
	}
	return &searchCache{
		items: make(map[string]*list.Element, capacity),
		lru:   list.New(),
		cap:   capacity,
	}
}

func (c *searchCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.lru.MoveToFront(element)
	return element.Value.(*searchCacheEntry).body, true
}

func (c *searchCache) Put(key string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		element.Value.(*searchCacheEntry).body = body
		c.lru.MoveToFront(element)
		return
	}

	element := c.lru.PushFront(&searchCacheEntry{key: key, body: body})
	c.items[key] = element

	if c.lru.Len() > c.cap {
		evicted := c.lru.Back()
		c.lru.Remove(evicted)
		delete(c.items, evicted.Value.(*searchCacheEntry).key)
	}
}

// Clear drops every held response. The admin handlers call it on any change so
// a fresh post, rename or delete is visible to the public index immediately
// rather than waiting for the entry to age out.
func (c *searchCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]*list.Element, c.cap)
	c.lru.Init()
}