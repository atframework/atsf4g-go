// Copyright 2025 atframework
package libatframe_utils_memory

import (
	"container/list"
)

// LRUMap is a generic LRU (Least Recently Used) cache with map-like access.
// K must be comparable (used as map key), V can be any type.
type LRUMap[K comparable, V any] struct {
	capacity int
	cache    map[K]*list.Element
	list     *list.List
}

// entry is a generic key-value pair stored in the LRU cache.
type entry[K comparable, V any] struct {
	key   K
	value V
}

// NewLRUMap creates a new LRUMap with the specified capacity.
// If capacity <= 0, the cache will have unlimited capacity.
func NewLRUMap[K comparable, V any](capacity int) *LRUMap[K, V] {
	return &LRUMap[K, V]{
		capacity: capacity,
		cache:    make(map[K]*list.Element),
		list:     list.New(),
	}
}

// Len returns the number of items in the cache.
func (c *LRUMap[K, V]) Len() int {
	if c == nil {
		return 0
	}

	return len(c.cache)
}

// Cap returns the capacity of the cache.
func (c *LRUMap[K, V]) Cap() int {
	if c == nil {
		return 0
	}

	return c.capacity
}

// Clear removes all items from the cache.
func (c *LRUMap[K, V]) Clear() {
	if c == nil {
		return
	}

	c.cache = make(map[K]*list.Element)
	c.list.Init()
}

// Contains checks if a key exists in the cache without updating its access time.
func (c *LRUMap[K, V]) Contains(key K) bool {
	if c == nil {
		return false
	}

	_, ok := c.cache[key]
	return ok
}

// Front returns the most recently used key-value pair without removing it.
// Returns the zero values and false if the cache is empty.
func (c *LRUMap[K, V]) Front() (K, V, bool) {
	var zeroK K
	var zeroV V

	if c == nil || c.list.Len() == 0 {
		return zeroK, zeroV, false
	}

	frontElem := c.list.Front()
	if frontElem == nil {
		return zeroK, zeroV, false
	}

	e := frontElem.Value.(*entry[K, V])
	return e.key, e.value, true
}

// Back returns the least recently used key-value pair without removing it.
// Returns the zero values and false if the cache is empty.
func (c *LRUMap[K, V]) Back() (K, V, bool) {
	var zeroK K
	var zeroV V

	if c == nil || c.list.Len() == 0 {
		return zeroK, zeroV, false
	}

	backElem := c.list.Back()
	if backElem == nil {
		return zeroK, zeroV, false
	}

	e := backElem.Value.(*entry[K, V])
	return e.key, e.value, true
}

// PopFront removes and returns the most recently used key-value pair.
// Returns the zero values and false if the cache is empty.
func (c *LRUMap[K, V]) PopFront() (K, V, bool) {
	var zeroK K
	var zeroV V

	if c == nil || c.list.Len() == 0 {
		return zeroK, zeroV, false
	}

	frontElem := c.list.Front()
	if frontElem == nil {
		return zeroK, zeroV, false
	}

	c.list.Remove(frontElem)
	e := frontElem.Value.(*entry[K, V])
	delete(c.cache, e.key)
	return e.key, e.value, true
}

// PopBack removes and returns the least recently used key-value pair.
// Returns the zero values and false if the cache is empty.
func (c *LRUMap[K, V]) PopBack() (K, V, bool) {
	var zeroK K
	var zeroV V

	if c == nil || c.list.Len() == 0 {
		return zeroK, zeroV, false
	}

	backElem := c.list.Back()
	if backElem == nil {
		return zeroK, zeroV, false
	}

	c.list.Remove(backElem)
	e := backElem.Value.(*entry[K, V])
	delete(c.cache, e.key)
	return e.key, e.value, true
}

// Get retrieves a value from the cache by key.
// If updateVisit is true, the item is moved to the front (most recently used).
// Returns the zero value and false if the key is not found.
func (c *LRUMap[K, V]) Get(key K, updateVisit bool) (V, bool) {
	var zeroV V

	if c == nil {
		return zeroV, false
	}

	if elem, ok := c.cache[key]; ok {
		if updateVisit {
			c.list.MoveToFront(elem)
		}
		return elem.Value.(*entry[K, V]).value, true
	}

	return zeroV, false
}

// Put adds or updates a key-value pair in the cache.
// If the key already exists, its value is updated and moved to the front.
// If the cache is at capacity, the least recently used item is evicted.
func (c *LRUMap[K, V]) Put(key K, value V) {
	if c == nil {
		return
	}

	if elem, ok := c.cache[key]; ok {
		// Key exists, update it
		elem.Value.(*entry[K, V]).value = value
		c.list.MoveToFront(elem)
	} else {
		// Key does not exist, add new entry
		if c.capacity > 0 && c.list.Len() >= c.capacity {
			// Cache is full, remove the least recently used item
			backElem := c.list.Back()
			if backElem != nil {
				c.list.Remove(backElem)
				delete(c.cache, backElem.Value.(*entry[K, V]).key)
			}
		}
		newElem := c.list.PushFront(&entry[K, V]{key, value})
		c.cache[key] = newElem
	}
}

// Delete removes a key from the cache.
// Returns true if the key was found and removed, false otherwise.
func (c *LRUMap[K, V]) Delete(key K) bool {
	if c == nil {
		return false
	}

	if elem, ok := c.cache[key]; ok {
		c.list.Remove(elem)
		delete(c.cache, key)
		return true
	}

	return false
}

// Keys returns all keys in the cache, ordered from most to least recently used.
func (c *LRUMap[K, V]) Keys() []K {
	if c == nil {
		return nil
	}

	keys := make([]K, 0, c.list.Len())
	for elem := c.list.Front(); elem != nil; elem = elem.Next() {
		keys = append(keys, elem.Value.(*entry[K, V]).key)
	}
	return keys
}

// Values returns all values in the cache, ordered from most to least recently used.
func (c *LRUMap[K, V]) Values() []V {
	if c == nil {
		return nil
	}

	values := make([]V, 0, c.list.Len())
	for elem := c.list.Front(); elem != nil; elem = elem.Next() {
		values = append(values, elem.Value.(*entry[K, V]).value)
	}
	return values
}

// Range iterates over all key-value pairs in the cache, from most to least recently used.
// If the callback returns false, iteration stops.
func (c *LRUMap[K, V]) Range(fn func(key K, value V) bool) {
	if c == nil {
		return
	}

	for elem := c.list.Front(); elem != nil; elem = elem.Next() {
		e := elem.Value.(*entry[K, V])
		if !fn(e.key, e.value) {
			break
		}
	}
}

// RangeReverse iterates over all key-value pairs in the cache, from least to most recently used.
// If the callback returns false, iteration stops.
func (c *LRUMap[K, V]) RangeReverse(fn func(key K, value V) bool) {
	if c == nil {
		return
	}

	for elem := c.list.Back(); elem != nil; elem = elem.Prev() {
		e := elem.Value.(*entry[K, V])
		if !fn(e.key, e.value) {
			break
		}
	}
}

// Resize changes the capacity of the cache.
// If the new capacity is smaller than the current size, least recently used items are evicted.
// If newCapacity <= 0, the cache will have unlimited capacity.
func (c *LRUMap[K, V]) Resize(newCapacity int) {
	if c == nil {
		return
	}

	c.capacity = newCapacity
	if newCapacity <= 0 {
		return
	}

	// Evict items if necessary
	for c.list.Len() > newCapacity {
		backElem := c.list.Back()
		if backElem != nil {
			c.list.Remove(backElem)
			delete(c.cache, backElem.Value.(*entry[K, V]).key)
		}
	}
}
