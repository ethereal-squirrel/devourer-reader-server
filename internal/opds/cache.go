package opds

import (
	"sync"
	"time"
)

const cacheTTL = 5 * time.Minute

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

type Cache struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
}

func NewCache() *Cache {
	return &Cache{entries: make(map[string]*cacheEntry)}
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		delete(c.entries, key)
		return nil, false
	}
	return e.data, true
}

func (c *Cache) Set(key string, data []byte) {
	c.mu.Lock()
	c.entries[key] = &cacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
	c.mu.Unlock()
}

func (c *Cache) Invalidate(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}
