package handlers

import (
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Cache is the minimal read/write interface for the BFF response cache.
// Implementations must be safe for concurrent use.
type Cache interface {
	Get(key string) (any, bool)
	Set(key string, v any)
}

type cacheItem struct {
	val       any
	expiresAt time.Time
}

// TTLCache is an in-memory Cache with per-entry expiry and optional NATS invalidation.
type TTLCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
	ttl   time.Duration
}

// NewTTLCache creates a TTLCache and wires up NATS key-level invalidation when nc is non-nil.
func NewTTLCache(ttlSec int, nc *nats.Conn, subj string) *TTLCache {
	if ttlSec <= 0 {
		ttlSec = 60
	}
	c := &TTLCache{
		items: make(map[string]cacheItem),
		ttl:   time.Duration(ttlSec) * time.Second,
	}
	if nc != nil && subj != "" {
		_, _ = nc.Subscribe(subj, func(m *nats.Msg) {
			key := string(m.Data)
			c.mu.Lock()
			defer c.mu.Unlock()
			if key == "" || strings.EqualFold(key, "ALL") {
				c.items = make(map[string]cacheItem)
				return
			}
			delete(c.items, key)
		})
	}
	return c
}

func (c *TTLCache) Get(key string) (any, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(it.expiresAt) {
		c.mu.Lock()
		if cur, ok2 := c.items[key]; ok2 && time.Now().After(cur.expiresAt) {
			delete(c.items, key)
		}
		c.mu.Unlock()
		return nil, false
	}
	return it.val, true
}

func (c *TTLCache) Set(key string, v any) {
	c.mu.Lock()
	c.items[key] = cacheItem{val: v, expiresAt: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}
