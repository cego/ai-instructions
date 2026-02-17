package registry

import (
	"sync"
	"time"
)

// Cache provides in-memory caching with TTL for registry data.
type Cache struct {
	mu        sync.RWMutex
	ttl       time.Duration
	registry  *cacheEntry[*Registry]
	manifests map[string]*cacheEntry[*StackManifest]
}

type cacheEntry[T any] struct {
	value     T
	expiresAt time.Time
}

// NewCache creates a cache with the given TTL.
func NewCache(ttl time.Duration) *Cache {
	return &Cache{
		ttl:       ttl,
		manifests: make(map[string]*cacheEntry[*StackManifest]),
	}
}

// GetRegistry returns the cached registry if still valid.
func (c *Cache) GetRegistry() (*Registry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.registry == nil || time.Now().After(c.registry.expiresAt) {
		return nil, false
	}
	return c.registry.value, true
}

// SetRegistry caches the registry.
func (c *Cache) SetRegistry(reg *Registry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.registry = &cacheEntry[*Registry]{
		value:     reg,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// GetManifest returns a cached stack manifest if still valid.
func (c *Cache) GetManifest(stackID string) (*StackManifest, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.manifests[stackID]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.value, true
}

// SetManifest caches a stack manifest.
func (c *Cache) SetManifest(stackID string, m *StackManifest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.manifests[stackID] = &cacheEntry[*StackManifest]{
		value:     m,
		expiresAt: time.Now().Add(c.ttl),
	}
}
