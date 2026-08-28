package api

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// responseCache is a small L1 cache for public catalogue reads. It prevents
// repeated navigation by many LAN clients from re-running the same database
// query, while keeping the data lifetime intentionally short.
type responseCache struct {
	mu      sync.RWMutex
	entries map[string]cachedResponse
}

type cachedResponse struct {
	status      int
	body        []byte
	contentType string
	expiresAt   time.Time
}

func newResponseCache() *responseCache {
	return &responseCache{entries: make(map[string]cachedResponse)}
}

func (c *responseCache) get(key string) (cachedResponse, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			c.mu.Lock()
			delete(c.entries, key)
			c.mu.Unlock()
		}
		return cachedResponse{}, false
	}
	return entry, true
}

func (c *responseCache) set(key string, entry cachedResponse) {
	c.mu.Lock()
	// Bound memory even when clients make many unique searches.
	if len(c.entries) >= 256 {
		for expiredKey, candidate := range c.entries {
			if time.Now().After(candidate.expiresAt) {
				delete(c.entries, expiredKey)
				break
			}
		}
		if len(c.entries) >= 256 {
			for oldestKey := range c.entries {
				delete(c.entries, oldestKey)
				break
			}
		}
	}
	c.entries[key] = entry
	c.mu.Unlock()
}

func (c *responseCache) clear() {
	c.mu.Lock()
	c.entries = make(map[string]cachedResponse)
	c.mu.Unlock()
}

func catalogueCacheTTL(path string) time.Duration {
	if path == "/api/categories" || path == "/api/dashboard/stats" {
		return 20 * time.Second
	}
	if path == "/api/media" && strings.Contains(path, "/stream") {
		return 0
	}
	if strings.HasPrefix(path, "/api/media") || strings.HasPrefix(path, "/api/hubs") || strings.HasPrefix(path, "/api/franchises") || strings.HasPrefix(path, "/api/people") || path == "/api/showcases" {
		return 45 * time.Second
	}
	return 0
}

func cacheKey(r *http.Request) string {
	return r.URL.Path + "?" + r.URL.RawQuery
}
