## tcache is a high performance and easy-to-use cache with TTL (Time To Live)

[![Go Reference](https://pkg.go.dev/badge/github.com/pashkov256/tcache/v1.svg)](https://pkg.go.dev/github.com/pashkov256/tcache)

---

## Features
- 🚀 **Simple API**: Easy-to-use methods for managing cache items.
- 🕒 **TTL Support**: Automatically expire items after a specified duration.
- 🔒 **Thread-Safe**: Safe for concurrent use across multiple goroutines.
- 🧩 **Generics Support**: Works with any key and value types.
- 🛠️ **Flexible**: Manually refresh TTL, update values, or delete items.

## Installation
```
go get github.com/pashkov256/tcache
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
    "github.com/pashkov256/tcache"
)

func main() {
   // Create a new cache with string keys and integer values
	cache := tcache.New[string, int]()
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