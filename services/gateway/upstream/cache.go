package upstream

import (
	"container/list"
	"time"
)

type cacheEntry struct {
	key       string
	response  *cachedResponse
	expiresAt time.Time
	element   *list.Element
}

type cacheSnapshot struct {
	response  *cachedResponse
	expiresAt time.Time
}

func (c *Client) getCacheSnapshot(key string) *cacheSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.cache[key]
	if entry == nil {
		return nil
	}

	c.order.MoveToFront(entry.element)

	return &cacheSnapshot{
		response:  entry.response,
		expiresAt: entry.expiresAt,
	}
}

func (c *Client) storeCache(key string, response *cachedResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if entry, ok := c.cache[key]; ok {
		entry.response = response
		entry.expiresAt = c.now().Add(c.cacheTTL)
		c.order.MoveToFront(entry.element)
		return
	}

	entry := &cacheEntry{
		key:       key,
		response:  response,
		expiresAt: c.now().Add(c.cacheTTL),
		element:   nil,
	}
	entry.element = c.order.PushFront(entry)
	c.cache[key] = entry

	for len(c.cache) > c.maxEntries {
		c.evictOldest()
	}
}

func (c *Client) deleteCache(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.cache[key]
	if entry == nil {
		return
	}

	delete(c.cache, key)
	c.order.Remove(entry.element)
}

func (c *Client) evictOldest() {
	back := c.order.Back()
	if back == nil {
		return
	}

	victim, ok := back.Value.(*cacheEntry)
	if !ok {
		c.order.Remove(back)
		return
	}

	delete(c.cache, victim.key)
	c.order.Remove(back)
	c.log.Debug("evicted cached response", "key", victim.key)
}
