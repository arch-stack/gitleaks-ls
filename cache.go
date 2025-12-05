package main

import (
	"crypto/sha256"
	"sync"
)

// Cache stores scan results keyed by content hash
type Cache struct {
	mu      sync.RWMutex
	entries map[[32]byte][]Finding
}

// NewCache creates a new result cache
func NewCache() *Cache {
	return &Cache{
		entries: make(map[[32]byte][]Finding),
	}
}

// Get retrieves cached findings for content
func (c *Cache) Get(content string) ([]Finding, bool) {
	hash := sha256.Sum256([]byte(content))
	c.mu.RLock()
	defer c.mu.RUnlock()
	findings, ok := c.entries[hash]
	return findings, ok
}

// Put stores findings for content
func (c *Cache) Put(content string, findings []Finding) {
	hash := sha256.Sum256([]byte(content))
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[hash] = findings
}

// Clear empties the cache (e.g., on config reload)
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[[32]byte][]Finding)
}

// Size returns the number of entries in the cache
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
