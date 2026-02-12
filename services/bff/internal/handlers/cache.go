package handlers

import (
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type cacheItem struct {
	val       interface{}
	expiresAt time.Time
}

var (
	bffCache = struct {
		mu    sync.RWMutex
		items map[string]cacheItem
		ttl   time.Duration
	}{
		items: make(map[string]cacheItem),
		ttl:   60 * time.Second,
	}
)

// InitCache initializes the in-memory TTL cache and subscribes to NATS subject for invalidation.
func InitCache(ttlSec int, nc *nats.Conn, subj string) {
	if ttlSec <= 0 {
		ttlSec = 60
	}
	bffCache.mu.Lock()
	bffCache.ttl = time.Duration(ttlSec) * time.Second
	if bffCache.items == nil {
		bffCache.items = make(map[string]cacheItem)
	}
	bffCache.mu.Unlock()

	if nc == nil || subj == "" {
		return
	}
	_, _ = nc.Subscribe(subj, func(m *nats.Msg) {
		key := string(m.Data)
		bffCache.mu.Lock()
		defer bffCache.mu.Unlock()
		if key == "" || key == "ALL" || key == "all" {
			bffCache.items = make(map[string]cacheItem)
			return
		}
		delete(bffCache.items, key)
	})
}

func cacheGet(key string) (interface{}, bool) {
	bffCache.mu.RLock()
	it, ok := bffCache.items[key]
	bffCache.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(it.expiresAt) {
		bffCache.mu.Lock()
		if cur, ok2 := bffCache.items[key]; ok2 {
			if time.Now().After(cur.expiresAt) {
				delete(bffCache.items, key)
			}
		}
		bffCache.mu.Unlock()
		return nil, false
	}
	return it.val, true
}

func cacheSet(key string, v interface{}) {
	bffCache.mu.Lock()
	bffCache.items[key] = cacheItem{val: v, expiresAt: time.Now().Add(bffCache.ttl)}
	bffCache.mu.Unlock()
}
