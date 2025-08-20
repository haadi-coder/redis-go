package main

import (
	"sync"
	"time"
)

type cache struct {
	data map[string]item
	mu   sync.RWMutex
}

type item struct {
	value  string
	expiry time.Time
}

func newCache() *cache {
	return &cache{
		data: make(map[string]item),
	}
}

func (c *cache) get(key string) (string, bool) {
	c.mu.RLock()
	item, found := c.data[key]
	c.mu.RUnlock()

	if found && item.isExpired() {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()

		return "", false
	}

	return item.value, found
}

func (c *cache) set(key string, value string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiry := time.Time{}
	if ttl > 0 {
		expiry = time.Now().Add(ttl)
	}

	c.data[key] = item{
		value:  value,
		expiry: expiry,
	}
}

func (i *item) isExpired() bool {
	return !i.expiry.IsZero() && time.Now().After(i.expiry)
}
