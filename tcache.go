package tcache

import (
	"container/list"
	"sync"
	"time"
)

func (c *Cache[K, V]) OnEvict(fn func(K, V)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onEvict = fn
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

func (c *Cache[K, V]) SetCapacity(capacity int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.capacity = capacity
	for c.list.Len() > c.capacity {
		backItem := c.list.Back()
		if backItem != nil {
			delete(c.items, backItem.Value.(*Item[K, V]).key)
			c.list.Remove(c.list.Back())
			if c.onEvict != nil {
				c.onEvict(backItem.Value.(*Item[K, V]).key, backItem.Value.(*Item[K, V]).value)
			}
		}
	}
}

func (c *Cache[K, V]) Refresh(key K, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item, exists := c.items[key]; exists {
		if item.Value.(*Item[K, V]).timer != nil {
			item.Value.(*Item[K, V]).timer.Stop()
		}
		item.Value.(*Item[K, V]).timer = time.AfterFunc(ttl, func() {
			if c.onDelete != nil {
				c.onDelete(key, item.Value.(*Item[K, V]).value)
			}
			c.Delete(key)
			c.list.Remove(item)
		})
	}
}
func (c *Cache[K, V]) Update(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		item.Value.(*Item[K, V]).value = value
		if c.onUpdate != nil {
			c.onUpdate(key, value, item.Value.(*Item[K, V]).value)
		}
	}
}
func (c *Cache[K, V]) UpdateWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if !exists {
		return
	}
	if item.Value.(*Item[K, V]).timer != nil {
		item.Value.(*Item[K, V]).timer.Stop()
	}

	var timer *time.Timer
	if ttl > 0 {
		timer = time.AfterFunc(ttl, func() {
			if item, exists := c.items[key]; exists {
				c.list.Remove(item)
				delete(c.items, key)
				if c.onExpire != nil {
					c.onExpire(key, value)
				}
			}
		})
	}

	item.Value.(*Item[K, V]).value = value
	item.Value.(*Item[K, V]).timer = timer

	if c.onUpdate != nil {
		c.onUpdate(key, value, item.Value.(*Item[K, V]).value)
	}
}
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if item, exists := c.items[key]; exists {
		if item.Value.(*Item[K, V]).timer != nil {
			item.Value.(*Item[K, V]).timer.Stop()
		}
		delete(c.items, key)
		c.list.Remove(item)
		if c.onDelete != nil {
			c.onDelete(key, item.Value.(*Item[K, V]).value)
		}
	}
}

func (c *Cache[K, V]) DeleteAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, item := range c.items {
		if item.Value.(*Item[K, V]).timer != nil {
			item.Value.(*Item[K, V]).timer.Stop()
			if c.onDelete != nil {
				c.onDelete(key, item.Value.(*Item[K, V]).value)
			}
			c.list.Remove(item)
			delete(c.items, key)
		}
	}
}

func (c *Cache[K, V]) GetAllItems() map[K]V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	items := make(map[K]V, len(c.items))
	for key, item := range c.items {
		items[key] = item.Value.(*Item[K, V]).value
	}
	return items
}

func (c *Cache[K, V]) GetAllValues() []V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	values := make([]V, 0, len(c.items))
	for _, item := range c.items {
		values = append(values, item.Value.(*Item[K, V]).value)
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
		c.list.MoveToFront(item)
		return item.Value.(*Item[K, V]).value, true
	}

	var zero V
	return zero, false
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		if item.Value.(*Item[K, V]).timer != nil {
			item.Value.(*Item[K, V]).timer.Stop()
		}
		c.list.MoveToFront(item)
	}

	if c.list.Len() >= c.capacity {
		backItem := c.list.Back()
		if backItem != nil {
			delete(c.items, backItem.Value.(*Item[K, V]).key)
			c.list.Remove(backItem)
			if c.onEvict != nil {
				c.onEvict(backItem.Value.(*Item[K, V]).key, backItem.Value.(*Item[K, V]).value)
			}
		}
	}
	newItem := &Item[K, V]{key: key, value: value}
	c.items[key] = c.list.PushFront(newItem)

	if c.onInsert != nil {
		c.onInsert(key, value)
	}
}

func (c *Cache[K, V]) Range(fn func(K, V) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, item := range c.items {
		if !fn(key, item.Value.(*Item[K, V]).value) {
			break
		}
	}
}

func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, exists := c.items[key]
	if exists {
		item.Value.(*Item[K, V]).timer.Stop()
		c.list.MoveToFront(item)
	}

	var timer *time.Timer
	if ttl > 0 {
		timer = time.AfterFunc(ttl, func() {
			if item, exists := c.items[key]; exists {
				c.list.Remove(item)
				delete(c.items, key)
				if c.onExpire != nil {
					c.onExpire(key, value)
				}
			}
		})
	}

	if c.list.Len() >= c.capacity {
		backItem := c.list.Back()
		if backItem != nil {
			if backItem.Value.(*Item[K, V]).timer != nil {
				backItem.Value.(*Item[K, V]).timer.Stop()
			}
			delete(c.items, backItem.Value.(*Item[K, V]).key)
			c.list.Remove(backItem)
			if c.onEvict != nil {
				c.onEvict(backItem.Value.(*Item[K, V]).key, backItem.Value.(*Item[K, V]).value)
			}
		}

	}
	newItem := &Item[K, V]{value: value, key: key, timer: timer}
	c.items[key] = c.list.PushFront(newItem)

	if c.onInsert != nil {
		c.onInsert(key, value)
	}
}

func New[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{
		items:    make(map[K]*list.Element),
		list:     list.New(),
		capacity: capacity,
	}
}

type Cache[K comparable, V any] struct {
	items    map[K]*list.Element
	list     *list.List
	mu       sync.RWMutex
	capacity int //max size cache
	onInsert func(K, V)
	onUpdate func(K, V, V)
	onDelete func(K, V)
	onExpire func(K, V)
	onEvict  func(K, V)
}

type Item[K comparable, V any] struct {
	key   K
	value V
	timer *time.Timer
}
