package tcache

import (
	"container/list"
	"encoding/json"
	"os"
	"sync"
	"time"
	"unsafe"
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
		// stop timer if it exists
		if timer := item.Value.(*Item[K, V]).timer; timer != nil {
			timer.Stop()
		}

		if ttl > 0 {
			item.Value.(*Item[K, V]).timer = time.AfterFunc(ttl, func() {
				if c.onDelete != nil {
					c.onDelete(key, item.Value.(*Item[K, V]).value)
				}
				c.Delete(key)
				c.list.Remove(item)
			})
		}

		item.Value.(*Item[K, V]).ttl = ttl
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

		if item.Value.(*Item[K, V]).onWatch != nil {
			item.Value.(*Item[K, V]).onWatch(key, DELETE, item.Value.(*Item[K, V]).value, item.Value.(*Item[K, V]).value)
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
	item, exists := c.items[key]
	if !exists {
		c.mu.RUnlock()
		var zero V
		return zero, false
	}

	c.mu.RUnlock()
	c.Refresh(key, item.Value.(*Item[K, V]).ttl)
	c.list.MoveToFront(item)
	return item.Value.(*Item[K, V]).value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, exists := c.items[key]
	if exists {
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
			if exists {
				if item.Value.(*Item[K, V]).onWatch != nil {
					item.Value.(*Item[K, V]).onWatch(key, EVICT, value, value)
				}
			}
		}
	}

	var newItem *Item[K, V]
	if exists {
		if item.Value.(*Item[K, V]).onWatch != nil {
			newItem = &Item[K, V]{value: value, key: key, onWatch: item.Value.(*Item[K, V]).onWatch}
		}
	} else {
		newItem = &Item[K, V]{value: value, key: key}
	}

	c.items[key] = c.list.PushFront(newItem)
	if c.items[key].Value.(*Item[K, V]).onWatch != nil {
		item.Value.(*Item[K, V]).onWatch(key, UPDATE, item.Value.(*Item[K, V]).value, newItem.value)
	}

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
	} else {

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
			if exists {
				if item.Value.(*Item[K, V]).onWatch != nil {
					item.Value.(*Item[K, V]).onWatch(key, EVICT, value, value)
				}
			}

		}

	}
	var newItem *Item[K, V]
	if exists {
		if item.Value.(*Item[K, V]).onWatch != nil {
			newItem = &Item[K, V]{value: value, key: key, timer: timer, onWatch: item.Value.(*Item[K, V]).onWatch}
		}
	} else {
		newItem = &Item[K, V]{value: value, key: key, timer: timer, ttl: ttl}
	}

	c.items[key] = c.list.PushFront(newItem)
	if c.items[key].Value.(*Item[K, V]).onWatch != nil {
		item.Value.(*Item[K, V]).onWatch(key, UPDATE, item.Value.(*Item[K, V]).value, newItem.value)
	}

	if c.onInsert != nil {
		c.onInsert(key, value)
	}
}

func (c *Cache[K, V]) SizeInBytes() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var size uint64

	for _, items := range c.items {
		size += uint64(unsafe.Sizeof(items.Value.(*Item[K, V]).value))
	}

	return size
}

func (c *Cache[K, V]) ExportToFile(filename, exp string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	itemsMap := make(map[K]V, len(c.items))

	for _, item := range c.items {
		itemTyped := item.Value.(*Item[K, V])
		itemsMap[itemTyped.key] = itemTyped.value
	}

	data, err := json.Marshal(itemsMap)

	if err != nil {
		return err
	}
	return os.WriteFile(filename+exp, data, 0644)
}

func New[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{
		items:    make(map[K]*list.Element),
		list:     list.New(),
		capacity: capacity,
	}
}

func (c *Cache[K, V]) OnWatch(key K, fn func(key K, op Operation, oldValue V, newValue V)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, exists := c.items[key]; exists {
		item.Value.(*Item[K, V]).onWatch = fn

	} else {
		newItem := &Item[K, V]{key: key, onWatch: fn}
		c.items[key] = c.list.PushFront(newItem)
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
	key         K
	value       V
	timer       *time.Timer
	ttl         time.Duration
	onWatch, fn func(K, Operation, V, V)
}

type Operation uint8

const (
	INSERT Operation = iota
	UPDATE
	DELETE
	EVICT
	EXPIRE
)
