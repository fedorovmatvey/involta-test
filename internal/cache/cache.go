package cache

import (
	"sync"
	"time"

	"github.com/fedorovmatvey/involta-test/internal/model"
)

type cacheItem struct {
	document  *model.Document
	expiresAt time.Time
}

type Cache struct {
	mu              sync.RWMutex
	items           map[string]*cacheItem
	ttl             time.Duration
	capacity        int
	cleanupInterval time.Duration
	stopCleanup     chan struct{}
}

func New(ttl, cleanupInterval time.Duration, capacity int) *Cache {
	c := &Cache{
		items:           make(map[string]*cacheItem),
		ttl:             ttl,
		capacity:        capacity,
		cleanupInterval: cleanupInterval,
		stopCleanup:     make(chan struct{}),
	}

	go c.startCleanup()

	return c
}

func (c *Cache) Get(id string) (*model.Document, bool) {
	c.mu.RLock()
	item, exists := c.items[id]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		item, exists = c.items[id]
		if exists && time.Now().After(item.expiresAt) {
			delete(c.items, id)
		}
		c.mu.Unlock()
		return nil, false
	}

	return item.document, true
}

func (c *Cache) Set(id string, doc *model.Document) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.items[id]; !exists && c.capacity > 0 && len(c.items) >= c.capacity {
		c.evictRandom()
	}

	c.items[id] = &cacheItem{
		document:  doc,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *Cache) evictRandom() {
	for key := range c.items {
		delete(c.items, key)
		return
	}
}

func (c *Cache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, id)
}

func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheItem)
}

func (c *Cache) startCleanup() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *Cache) cleanup() {
	keysToDelete := make([]string, 0)
	now := time.Now()

	c.mu.RLock()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	c.mu.RUnlock()

	if len(keysToDelete) > 0 {
		c.mu.Lock()
		for _, key := range keysToDelete {
			item, exists := c.items[key]
			if exists && now.After(item.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *Cache) Stop() {
	close(c.stopCleanup)
}

func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.items)
}
