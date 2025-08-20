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
	c := &cache{
		data: make(map[string]item),
	}

	go func() {
		for range time.Tick(2 * time.Second) {
			c.mu.Lock()

			for key, item := range c.data {
				if item.isExpired() {
					delete(c.data, key)
				}
			}

			c.mu.Unlock()
		}
	}()

	return c
}

func (c *cache) get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, found := c.data[key]

	if item.isExpired() {
		delete(c.data, key)
		return item.value, false
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
	return time.Now().After(i.expiry)
}
