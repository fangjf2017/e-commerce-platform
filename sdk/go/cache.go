package featureflags

import (
	"sync"
	"time"
)

type cacheEntry struct {
	result    *EvaluationResult
	expiresAt time.Time
}

type localCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	ttl     time.Duration
}

func newLocalCache(ttl time.Duration) *localCache {
	c := &localCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
	}
	go c.cleanup()
	return c
}

func (c *localCache) get(key string) (*EvaluationResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.result, true
}

func (c *localCache) set(key string, result *EvaluationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &cacheEntry{result: result, expiresAt: time.Now().Add(c.ttl)}
}

func (c *localCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// cleanup removes expired entries every minute
func (c *localCache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}
