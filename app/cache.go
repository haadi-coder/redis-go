package main

import "sync"

type cache struct {
	data map[string]string
	mu   sync.RWMutex
}

var redisCache = newCache()

func newCache() *cache {
	return &cache{
		data: make(map[string]string),
	}
}

func (c *cache) get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, found := c.data[key]

	return value, found
}

func (c *cache) set(key string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = value
}
