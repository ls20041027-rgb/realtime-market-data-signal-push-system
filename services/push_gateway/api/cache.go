package api

import (
	"sync"
	"time"
)

// cacheEntry holds a cached value with expiration.
type cacheEntry struct {
	data      any
	expiresAt time.Time
}

// MemCache is a simple in-memory TTL cache (sharded by key).
type MemCache struct {
	mu      sync.RWMutex
	items   map[string]cacheEntry
	ttl     time.Duration
	maxSize int
}

func NewMemCache(ttl time.Duration, maxSize int) *MemCache {
	c := &MemCache{
		items:   make(map[string]cacheEntry, 256),
		ttl:     ttl,
		maxSize: maxSize,
	}
	// Background eviction every ttl interval
	go func() {
		ticker := time.NewTicker(ttl)
		defer ticker.Stop()
		for range ticker.C {
			c.evict()
		}
	}()
	return c
}

func (c *MemCache) Get(key string) (any, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *MemCache) Set(key string, data any) {
	c.mu.Lock()
	// Simple eviction: if over max size, clear all (cheap for short TTL cache)
	if len(c.items) >= c.maxSize {
		c.items = make(map[string]cacheEntry, 256)
	}
	c.items[key] = cacheEntry{data: data, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *MemCache) evict() {
	now := time.Now()
	c.mu.Lock()
	for k, v := range c.items {
		if now.After(v.expiresAt) {
			delete(c.items, k)
		}
	}
	c.mu.Unlock()
}
