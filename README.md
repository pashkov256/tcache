# tcache - high Performance Cache with TTL and LRU

[![Go Reference](https://pkg.go.dev/badge/github.com/pashkov256/tcache/v1.svg)](https://pkg.go.dev/github.com/pashkov256/tcache/v2)

## Features
- 🚀 **Simple API**: Easy-to-use methods for managing cache items.
- 🕒 **TTL Support**: Automatically expire items after a specified duration.
- ♻️ **LRU (Least Recently Used) Eviction**: Automatically removes the least recently used items when capacity is exceeded.
- 🔒 **Thread-Safe**: Safe for concurrent use across multiple goroutines.
- 🧩 **Generics Support**: Works with any key and value types.
- 🔔 **Event Hooks**: Supports callbacks on insert, update, delete, and expiration.


## Installation
```sh
go get github.com/pashkov256/tcache/v2
```

## Usage
The main type of `tcache` is `Cache`. It represents a single 
in-memory data store.

To create a new instance of `tcache.Cache`, the `tcache.New()` function 
should be called:
```go
package main

import (
    "fmt"
    "time"
    "github.com/pashkov256/tcache/v2"
)

func main() {
    // Create a new cache with string keys and integer values, capacity 100
    cache := tcache.New[string, int](100)
}
```


## Adding Items
You can add items to the cache with or without a TTL:

```go
// Add an item without TTL (stored indefinitely)
cache.Set("key1", 42)

// Add an item with a TTL of 10 minutes
cache.SetWithTTL("key2", 100, 10 * time.Minute)
```

## Retrieving Items
Retrieve items from the cache using the Get method:
```go
if value, ok := cache.Get("key1"); ok {
    fmt.Println("Value for key1:", value)
} else {
    fmt.Println("Key1 not found or expired")
}
```

## Checking for Existence
Check if a key exists in the cache:
```go
if cache.Has("key2") {
    fmt.Println("Key2 exists in the cache")
} else {
    fmt.Println("Key2 does not exist")
}
```

## Updating Items
Update the value of an existing item:

```go
cache.Update("key1", 200)
```

## Refreshing TTL
Refresh the TTL of an existing item:
```go
cache.Refresh("key2", 5 * time.Minute)
//TTL for key2 refreshed to 5 minutes
```
## Deleting Items
Delete items from the cache:

```go
// Delete a single item
cache.Delete("key1")

// Delete all items
cache.DeleteAll()
```
## Get the number of items in the cache

```go
size := cache.Len()
fmt.Println("Number of items in cache:", size)
```


## LRU Eviction
```go
cache := tcache.New(2)  // Capacity of 2
cache.Set("a", 1)
cache.Set("b", 2)
cache.Set("c", 3) // "a" is evicted because it is the least recently used

fmt.Println(cache.Has("a")) // false
fmt.Println(cache.Has("b")) // true
fmt.Println(cache.Has("c")) // true
```

## Change capacity
If the cache is full after installing capacity, the most unused items are automatically deleted.
```go
cache.SetCapacity(2)
```

## Getting  All Keys,Values,Elements
Get all keys or values stored in the cache:

```go
keys := cache.GetAllKeys()
fmt.Println("Keys in cache:", keys)

values := cache.GetAllValues()
fmt.Println("Values in cache:", values)

elements := cache.GetAllItems()
fmt.Println("Map in cache:", elements)
```

## Range 
Method that will allow iterating over all cache items
```go
cache.Range(func(k string, v int) bool {
	fmt.Println(k, v)
	return true
})
```

## SizeInBytes  
Method for estimating the total cache size in bytes. This can be useful for monitoring memory usage.
```go
cache.SizeInBytes()
```

## ExportToFile  
Methods for exporting the current cache state to a file and importing from a file.
```go
cache.ExportToFile("cache_data", ".json")
```

## OnWatch 
You can track which operations are taking place with an item in the cache `UPDATE`,`DELETE`,`EVICT`
```go
	cache.OnWatch("your_key", func(key string, op tcache.Operation, oldVal int, newVal int) {
		switch op {
		case tcache.UPDATE:
			fmt.Printf("Key '%s' UPDATE from %#v to %v\n", key, oldVal, newVal)
		case tcache.DELETE:
			fmt.Printf("Key '%s' was DELETED (had value %d)\n", key, oldVal)
		case tcache.EVICT:
			fmt.Printf("Key '%s' was EVICTED (had value %#v)\n", key, oldVal)
		}
	})
```

## Event Hooks (OnInsert, OnUpdate, OnDelete, OnExpire)
`tcache` supports event hooks to execute custom logic when items are inserted, updated, deleted, expired or evict.
```go

// OnInsert: Called when an item is added
cache.OnInsert(func(key string, value int) {
    fmt.Printf("Inserted: %s -> %d\n", key, value)
})

// OnUpdate: Called when an item is updated
cache.OnUpdate(func(key string, oldValue, newValue int) {
    fmt.Printf("Updated: %s from %d to %d\n", key, oldValue, newValue)
})

// OnDelete: Called when an item is deleted
cache.OnDelete(func(key string, value int) {
    fmt.Printf("Deleted: %s -> %d\n", key, value)
})

// OnExpire: Called when an item expires
cache.OnExpire(func(key string, value int) {
    fmt.Printf("Expired: %s -> %d\n", key, value)
})

// OnEvict: Called when the cache is full and the most unused item is deleted.
cache.OnEvict(func(key string, value int) {
    fmt.Printf("Evict: %s -> %d\n", key, value)
})

// Adding and modifying items to trigger events
cache.Set("example", 123)
cache.Update("example", 456)
cache.Delete("example")
cache.SetWithTTL("temp", 999, 2*time.Second)

// Wait to see expiration event
 time.Sleep(3 * time.Second)
```