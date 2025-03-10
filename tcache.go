package tcache

import (
	"sync"
	"time"
)

type Cache[K comparable, V any] struct {
	items    map[K]*Item[V]
	mu       sync.RWMutex
	onInsert func(K, V)
	onUpdate func(K, V, V)
	onDelete func(K, V)
	onExpire func(K, V)
}

type Item[V any] struct {
	value V
	timer *time.Timer
}

func (c *Cache[K, V]) OnInsert(fn func(K, V)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onInsert = fn
}

func (c *Cache[K, V]) OnUpdate(fn func(K, V, V)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onUpdate = fn
}

func (c *Cache[K, V]) OnDelete(fn func(K, V)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onDelete = fn
}

func (c *Cache[K, V]) OnExpire(fn func(K, V)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onExpire = fn
}

func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

func (c *Cache[K, V]) Has(key K) (exists bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists = c.items[key]
	return exists
}

func (c *Cache[K, V]) Refresh(key K, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item, exists := c.items[key]; exists {
		if item.timer != nil {
			item.timer.Stop()
		}
		item.timer = time.AfterFunc(ttl, func() {
			if c.onDelete != nil {
				c.onDelete(key, item.value)
			}
			c.Delete(key)
		})
	}
}

func (c *Cache[K, V]) Update(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item := c.items[key]
	c.items[key] = &Item[V]{value: value, timer: item.timer}
	if c.onUpdate != nil {
		c.onUpdate(key, value, item.value)
	}
}
func (c *Cache[K, V]) UpdateWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item := c.items[key]
	if item.timer != nil {
		item.timer.Stop()
	}

	var timer *time.Timer
	if ttl > 0 {
		timer = time.AfterFunc(ttl, func() {
			if c.onExpire != nil {
				c.onExpire(key, value)
			}
			c.Delete(key)
		})
	}

	c.items[key] = &Item[V]{value: value, timer: timer}
	if c.onUpdate != nil {
		c.onUpdate(key, value, item.value)
	}
}

func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		if item.timer != nil {
			item.timer.Stop()
		}

		delete(c.items, key)

		if c.onDelete != nil {
			c.onDelete(key, item.value)
		}
	}
}

func (c *Cache[K, V]) DeleteAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, item := range c.items {
		if item.timer != nil {
			item.timer.Stop()
			delete(c.items, key)
		}
	}
}

func (c *Cache[K, V]) GetAllItems() map[K]V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := make(map[K]V, len(c.items))
	for key, item := range c.items {
		items[key] = item.value
	}
	return items
}

func (c *Cache[K, V]) GetAllValues() []V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	values := make([]V, 0, len(c.items))
	for _, item := range c.items {
		values = append(values, item.value)
	}
	return values
}

func (c *Cache[K, V]) GetAllKeys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]K, 0, len(c.items))

	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item, exists := c.items[key]; exists {
		return item.value, true
	}

	var zero V
	return zero, false
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		if item.timer != nil {
			item.timer.Stop()
		}
	}

	c.items[key] = &Item[V]{
		value: value,
		timer: nil,
	}
	if c.onInsert != nil {
		c.onInsert(key, value)
	}
}

func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		item.timer.Stop()
	}

	var timer *time.Timer

	if ttl > 0 {
		timer = time.AfterFunc(ttl, func() {
			if c.onExpire != nil {
				c.onExpire(key, value)
			}
			c.Delete(key)

		})
	}

	c.items[key] = &Item[V]{
		value: value,
		timer: timer,
	}
	if c.onInsert != nil {
		c.onInsert(key, value)
	}
}

func New[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{items: make(map[K]*Item[V])}
}
