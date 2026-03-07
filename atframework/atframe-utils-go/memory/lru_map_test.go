// Copyright 2025 atframework
package libatframe_utils_memory

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Basic functionality tests with int keys and string values
// ============================================================================

// TestNewLRUMap_ValidCapacity tests creating a new LRUMap with valid capacity
func TestNewLRUMap_ValidCapacity(t *testing.T) {
	// Arrange & Act
	lru := NewLRUMap[int, string](10)

	// Assert
	assert.NotNil(t, lru)
	assert.Equal(t, 0, lru.Len())
	assert.Equal(t, 10, lru.Cap())
}

// TestNewLRUMap_ZeroCapacity tests creating a new LRUMap with zero capacity (unlimited)
func TestNewLRUMap_ZeroCapacity(t *testing.T) {
	// Arrange & Act
	lru := NewLRUMap[int, string](0)

	// Assert
	assert.NotNil(t, lru)
	assert.Equal(t, 0, lru.Cap())

	// Should allow unlimited items when capacity is 0
	for i := 0; i < 100; i++ {
		lru.Put(i, "value")
	}
	assert.Equal(t, 100, lru.Len())
}

// TestNewLRUMap_NegativeCapacity tests creating a new LRUMap with negative capacity (unlimited)
func TestNewLRUMap_NegativeCapacity(t *testing.T) {
	// Arrange & Act
	lru := NewLRUMap[int, string](-5)

	// Assert
	assert.NotNil(t, lru)
	assert.Equal(t, -5, lru.Cap())

	// Should allow unlimited items when capacity is negative
	for i := 0; i < 50; i++ {
		lru.Put(i, "value")
	}
	assert.Equal(t, 50, lru.Len())
}

// ============================================================================
// Put and Get tests
// ============================================================================

// TestLRUMap_Put_NewKey tests adding a new key to the cache
func TestLRUMap_Put_NewKey(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act
	lru.Put(1, "one")

	// Assert
	assert.Equal(t, 1, lru.Len())
	val, ok := lru.Get(1, false)
	assert.True(t, ok)
	assert.Equal(t, "one", val)
}

// TestLRUMap_Put_UpdateExistingKey tests updating an existing key
func TestLRUMap_Put_UpdateExistingKey(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")

	// Act
	lru.Put(1, "ONE")

	// Assert
	assert.Equal(t, 1, lru.Len())
	val, ok := lru.Get(1, false)
	assert.True(t, ok)
	assert.Equal(t, "ONE", val)
}

// TestLRUMap_Put_EvictsLRU tests that the least recently used item is evicted when capacity is reached
func TestLRUMap_Put_EvictsLRU(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act - adding a 4th item should evict key 1 (least recently used)
	lru.Put(4, "four")

	// Assert
	assert.Equal(t, 3, lru.Len())
	_, ok := lru.Get(1, false)
	assert.False(t, ok, "key 1 should be evicted")
	val, ok := lru.Get(4, false)
	assert.True(t, ok)
	assert.Equal(t, "four", val)
}

// TestLRUMap_Put_EvictsCorrectItemAfterAccess tests that access updates LRU order
func TestLRUMap_Put_EvictsCorrectItemAfterAccess(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Access key 1 to make it most recently used
	lru.Get(1, true)

	// Act - adding a 4th item should evict key 2 (now least recently used)
	lru.Put(4, "four")

	// Assert
	assert.Equal(t, 3, lru.Len())
	_, ok := lru.Get(2, false)
	assert.False(t, ok, "key 2 should be evicted")
	val, ok := lru.Get(1, false)
	assert.True(t, ok, "key 1 should still exist")
	assert.Equal(t, "one", val)
}

// TestLRUMap_Get_NonExistentKey tests getting a non-existent key
func TestLRUMap_Get_NonExistentKey(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act
	val, ok := lru.Get(999, false)

	// Assert
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

// TestLRUMap_Get_UpdateVisitFalse tests that updateVisit=false doesn't change order
func TestLRUMap_Get_UpdateVisitFalse(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act - access key 1 without updating visit
	lru.Get(1, false)
	lru.Put(4, "four")

	// Assert - key 1 should still be evicted because visit wasn't updated
	_, ok := lru.Get(1, false)
	assert.False(t, ok, "key 1 should be evicted")
}

// ============================================================================
// Front and Back tests
// ============================================================================

// TestLRUMap_Front_Empty tests Front on an empty cache
func TestLRUMap_Front_Empty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act
	key, val, ok := lru.Front()

	// Assert
	assert.False(t, ok)
	assert.Equal(t, 0, key)
	assert.Equal(t, "", val)
}

// TestLRUMap_Front_NonEmpty tests Front returns most recently used item
func TestLRUMap_Front_NonEmpty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	key, val, ok := lru.Front()

	// Assert - key 3 is most recently used
	assert.True(t, ok)
	assert.Equal(t, 3, key)
	assert.Equal(t, "three", val)
	assert.Equal(t, 3, lru.Len(), "Front should not remove the item")
}

// TestLRUMap_Back_Empty tests Back on an empty cache
func TestLRUMap_Back_Empty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act
	key, val, ok := lru.Back()

	// Assert
	assert.False(t, ok)
	assert.Equal(t, 0, key)
	assert.Equal(t, "", val)
}

// TestLRUMap_Back_NonEmpty tests Back returns least recently used item
func TestLRUMap_Back_NonEmpty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	key, val, ok := lru.Back()

	// Assert - key 1 is least recently used
	assert.True(t, ok)
	assert.Equal(t, 1, key)
	assert.Equal(t, "one", val)
	assert.Equal(t, 3, lru.Len(), "Back should not remove the item")
}

// ============================================================================
// PopFront and PopBack tests
// ============================================================================

// TestLRUMap_PopFront_Empty tests PopFront on an empty cache
func TestLRUMap_PopFront_Empty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act
	key, val, ok := lru.PopFront()

	// Assert
	assert.False(t, ok)
	assert.Equal(t, 0, key)
	assert.Equal(t, "", val)
}

// TestLRUMap_PopFront_NonEmpty tests PopFront removes and returns most recently used item
func TestLRUMap_PopFront_NonEmpty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	key, val, ok := lru.PopFront()

	// Assert
	assert.True(t, ok)
	assert.Equal(t, 3, key)
	assert.Equal(t, "three", val)
	assert.Equal(t, 2, lru.Len())
	assert.False(t, lru.Contains(3))
}

// TestLRUMap_PopBack_Empty tests PopBack on an empty cache
func TestLRUMap_PopBack_Empty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act
	key, val, ok := lru.PopBack()

	// Assert
	assert.False(t, ok)
	assert.Equal(t, 0, key)
	assert.Equal(t, "", val)
}

// TestLRUMap_PopBack_NonEmpty tests PopBack removes and returns least recently used item
func TestLRUMap_PopBack_NonEmpty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	key, val, ok := lru.PopBack()

	// Assert
	assert.True(t, ok)
	assert.Equal(t, 1, key)
	assert.Equal(t, "one", val)
	assert.Equal(t, 2, lru.Len())
	assert.False(t, lru.Contains(1))
}

// ============================================================================
// Delete tests
// ============================================================================

// TestLRUMap_Delete_ExistingKey tests deleting an existing key
func TestLRUMap_Delete_ExistingKey(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")

	// Act
	deleted := lru.Delete(1)

	// Assert
	assert.True(t, deleted)
	assert.Equal(t, 1, lru.Len())
	assert.False(t, lru.Contains(1))
}

// TestLRUMap_Delete_NonExistentKey tests deleting a non-existent key
func TestLRUMap_Delete_NonExistentKey(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")

	// Act
	deleted := lru.Delete(999)

	// Assert
	assert.False(t, deleted)
	assert.Equal(t, 1, lru.Len())
}

// ============================================================================
// Clear tests
// ============================================================================

// TestLRUMap_Clear tests clearing all items from the cache
func TestLRUMap_Clear(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	lru.Clear()

	// Assert
	assert.Equal(t, 0, lru.Len())
	assert.False(t, lru.Contains(1))
	assert.False(t, lru.Contains(2))
	assert.False(t, lru.Contains(3))
}

// ============================================================================
// Contains tests
// ============================================================================

// TestLRUMap_Contains_ExistingKey tests Contains with an existing key
func TestLRUMap_Contains_ExistingKey(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")

	// Act & Assert
	assert.True(t, lru.Contains(1))
}

// TestLRUMap_Contains_NonExistentKey tests Contains with a non-existent key
func TestLRUMap_Contains_NonExistentKey(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act & Assert
	assert.False(t, lru.Contains(999))
}

// ============================================================================
// Keys and Values tests
// ============================================================================

// TestLRUMap_Keys_ReturnsOrderedKeys tests that Keys returns keys in MRU order
func TestLRUMap_Keys_ReturnsOrderedKeys(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](5)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	keys := lru.Keys()

	// Assert - order should be 3, 2, 1 (most to least recently used)
	assert.Equal(t, []int{3, 2, 1}, keys)
}

// TestLRUMap_Keys_Empty tests Keys on an empty cache
func TestLRUMap_Keys_Empty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act
	keys := lru.Keys()

	// Assert
	assert.Empty(t, keys)
}

// TestLRUMap_Values_ReturnsOrderedValues tests that Values returns values in MRU order
func TestLRUMap_Values_ReturnsOrderedValues(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](5)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	values := lru.Values()

	// Assert - order should be "three", "two", "one" (most to least recently used)
	assert.Equal(t, []string{"three", "two", "one"}, values)
}

// TestLRUMap_Values_Empty tests Values on an empty cache
func TestLRUMap_Values_Empty(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)

	// Act
	values := lru.Values()

	// Assert
	assert.Empty(t, values)
}

// ============================================================================
// Range and RangeReverse tests
// ============================================================================

// TestLRUMap_Range_IteratesInMRUOrder tests Range iterates from MRU to LRU
func TestLRUMap_Range_IteratesInMRUOrder(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](5)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	var keys []int
	var values []string

	// Act
	lru.Range(func(k int, v string) bool {
		keys = append(keys, k)
		values = append(values, v)
		return true
	})

	// Assert - order should be 3, 2, 1 (most to least recently used)
	assert.Equal(t, []int{3, 2, 1}, keys)
	assert.Equal(t, []string{"three", "two", "one"}, values)
}

// TestLRUMap_Range_StopsOnFalse tests Range stops when callback returns false
func TestLRUMap_Range_StopsOnFalse(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](5)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	var keys []int

	// Act - stop after first item
	lru.Range(func(k int, v string) bool {
		keys = append(keys, k)
		return false
	})

	// Assert - should only have one item
	assert.Equal(t, []int{3}, keys)
}

// TestLRUMap_RangeReverse_IteratesInLRUOrder tests RangeReverse iterates from LRU to MRU
func TestLRUMap_RangeReverse_IteratesInLRUOrder(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](5)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	var keys []int
	var values []string

	// Act
	lru.RangeReverse(func(k int, v string) bool {
		keys = append(keys, k)
		values = append(values, v)
		return true
	})

	// Assert - order should be 1, 2, 3 (least to most recently used)
	assert.Equal(t, []int{1, 2, 3}, keys)
	assert.Equal(t, []string{"one", "two", "three"}, values)
}

// TestLRUMap_RangeReverse_StopsOnFalse tests RangeReverse stops when callback returns false
func TestLRUMap_RangeReverse_StopsOnFalse(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](5)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	var keys []int

	// Act - stop after first item
	lru.RangeReverse(func(k int, v string) bool {
		keys = append(keys, k)
		return false
	})

	// Assert - should only have one item (the LRU)
	assert.Equal(t, []int{1}, keys)
}

// ============================================================================
// Resize tests
// ============================================================================

// TestLRUMap_Resize_Increase tests increasing capacity
func TestLRUMap_Resize_Increase(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	lru.Resize(5)

	// Assert
	assert.Equal(t, 5, lru.Cap())
	assert.Equal(t, 3, lru.Len())

	// Should be able to add more items without eviction
	lru.Put(4, "four")
	lru.Put(5, "five")
	assert.Equal(t, 5, lru.Len())
	assert.True(t, lru.Contains(1))
}

// TestLRUMap_Resize_Decrease tests decreasing capacity evicts LRU items
func TestLRUMap_Resize_Decrease(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](5)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")
	lru.Put(4, "four")
	lru.Put(5, "five")

	// Act
	lru.Resize(2)

	// Assert
	assert.Equal(t, 2, lru.Cap())
	assert.Equal(t, 2, lru.Len())

	// Only most recently used items should remain (4 and 5)
	assert.False(t, lru.Contains(1))
	assert.False(t, lru.Contains(2))
	assert.False(t, lru.Contains(3))
	assert.True(t, lru.Contains(4))
	assert.True(t, lru.Contains(5))
}

// TestLRUMap_Resize_ToZero tests resizing to zero (unlimited)
func TestLRUMap_Resize_ToZero(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act
	lru.Resize(0)

	// Assert - all items should remain and no limit on new items
	assert.Equal(t, 0, lru.Cap())
	assert.Equal(t, 3, lru.Len())

	// Should be able to add unlimited items
	for i := 4; i <= 20; i++ {
		lru.Put(i, "value")
	}
	assert.Equal(t, 20, lru.Len())
}

// ============================================================================
// Nil receiver tests
// ============================================================================

// TestLRUMap_NilReceiver tests all methods handle nil receiver gracefully
func TestLRUMap_NilReceiver(t *testing.T) {
	// Arrange
	var lru *LRUMap[int, string]

	// Assert - all methods should handle nil gracefully
	assert.Equal(t, 0, lru.Len())
	assert.Equal(t, 0, lru.Cap())
	assert.False(t, lru.Contains(1))

	val, ok := lru.Get(1, false)
	assert.False(t, ok)
	assert.Equal(t, "", val)

	key, val, ok := lru.Front()
	assert.False(t, ok)
	assert.Equal(t, 0, key)
	assert.Equal(t, "", val)

	key, val, ok = lru.Back()
	assert.False(t, ok)
	assert.Equal(t, 0, key)
	assert.Equal(t, "", val)

	key, val, ok = lru.PopFront()
	assert.False(t, ok)
	assert.Equal(t, 0, key)
	assert.Equal(t, "", val)

	key, val, ok = lru.PopBack()
	assert.False(t, ok)
	assert.Equal(t, 0, key)
	assert.Equal(t, "", val)

	assert.False(t, lru.Delete(1))
	assert.Nil(t, lru.Keys())
	assert.Nil(t, lru.Values())

	// These should not panic
	lru.Put(1, "one")
	lru.Clear()
	lru.Resize(10)
	lru.Range(func(k int, v string) bool { return true })
	lru.RangeReverse(func(k int, v string) bool { return true })
}

// ============================================================================
// Generic type tests
// ============================================================================

// TestLRUMap_StringKey tests LRUMap with string keys
func TestLRUMap_StringKey(t *testing.T) {
	// Arrange
	lru := NewLRUMap[string, int](3)

	// Act
	lru.Put("one", 1)
	lru.Put("two", 2)
	lru.Put("three", 3)

	// Assert
	val, ok := lru.Get("two", false)
	assert.True(t, ok)
	assert.Equal(t, 2, val)
}

// TestLRUMap_StructKey tests LRUMap with struct keys
func TestLRUMap_StructKey(t *testing.T) {
	// Arrange
	type Point struct {
		X, Y int
	}
	lru := NewLRUMap[Point, string](3)

	// Act
	lru.Put(Point{1, 2}, "origin")
	lru.Put(Point{3, 4}, "destination")

	// Assert
	val, ok := lru.Get(Point{1, 2}, false)
	assert.True(t, ok)
	assert.Equal(t, "origin", val)
}

// TestLRUMap_PointerValue tests LRUMap with pointer values
func TestLRUMap_PointerValue(t *testing.T) {
	// Arrange
	type Data struct {
		Value string
	}
	lru := NewLRUMap[int, *Data](3)

	data1 := &Data{Value: "first"}
	data2 := &Data{Value: "second"}

	// Act
	lru.Put(1, data1)
	lru.Put(2, data2)

	// Assert
	val, ok := lru.Get(1, false)
	assert.True(t, ok)
	assert.Same(t, data1, val)
	assert.Equal(t, "first", val.Value)
}

// TestLRUMap_SliceValue tests LRUMap with slice values
func TestLRUMap_SliceValue(t *testing.T) {
	// Arrange
	lru := NewLRUMap[string, []int](3)

	// Act
	lru.Put("primes", []int{2, 3, 5, 7, 11})
	lru.Put("fibonacci", []int{1, 1, 2, 3, 5, 8})

	// Assert
	val, ok := lru.Get("primes", false)
	assert.True(t, ok)
	assert.Equal(t, []int{2, 3, 5, 7, 11}, val)
}

// TestLRUMap_MapValue tests LRUMap with map values
func TestLRUMap_MapValue(t *testing.T) {
	// Arrange
	lru := NewLRUMap[string, map[string]int](3)

	data := map[string]int{"a": 1, "b": 2}

	// Act
	lru.Put("letters", data)

	// Assert
	val, ok := lru.Get("letters", false)
	assert.True(t, ok)
	assert.Equal(t, 1, val["a"])
	assert.Equal(t, 2, val["b"])
}

// ============================================================================
// Concurrency tests
// ============================================================================

// TestLRUMap_ConcurrentRead tests concurrent read access
// Note: LRUMap is NOT thread-safe by default; this test demonstrates the need for external synchronization
func TestLRUMap_ConcurrentReadWithMutex(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](100)
	var mu sync.RWMutex

	// Pre-populate the cache
	for i := 0; i < 100; i++ {
		lru.Put(i, "value")
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	readsPerGoroutine := 100

	// Act - concurrent reads with mutex protection
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < readsPerGoroutine; i++ {
				key := i % 100
				mu.RLock()
				val, ok := lru.Get(key, false)
				mu.RUnlock()
				if ok {
					assert.Equal(t, "value", val)
				}
			}
		}()
	}

	wg.Wait()
}

// TestLRUMap_ConcurrentReadWriteWithMutex tests concurrent read/write access with mutex
func TestLRUMap_ConcurrentReadWriteWithMutex(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, int](100)
	var mu sync.RWMutex

	var wg sync.WaitGroup
	numWriters := 5
	numReaders := 10
	opsPerGoroutine := 100

	// Writers
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := (writerID * opsPerGoroutine) + i
				mu.Lock()
				lru.Put(key, key*2)
				mu.Unlock()
			}
		}(w)
	}

	// Readers
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := i % (numWriters * opsPerGoroutine)
				mu.RLock()
				val, ok := lru.Get(key, false)
				mu.RUnlock()
				if ok {
					assert.Equal(t, key*2, val)
				}
			}
		}()
	}

	wg.Wait()
}

// ============================================================================
// Edge case tests
// ============================================================================

// TestLRUMap_SingleCapacity tests LRUMap with capacity of 1
func TestLRUMap_SingleCapacity(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](1)

	// Act
	lru.Put(1, "one")
	assert.Equal(t, 1, lru.Len())

	lru.Put(2, "two")

	// Assert - only key 2 should remain
	assert.Equal(t, 1, lru.Len())
	assert.False(t, lru.Contains(1))
	val, ok := lru.Get(2, false)
	assert.True(t, ok)
	assert.Equal(t, "two", val)
}

// TestLRUMap_PopAllItems tests popping all items from the cache
func TestLRUMap_PopAllItems(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act & Assert
	k, v, ok := lru.PopFront()
	assert.True(t, ok)
	assert.Equal(t, 3, k)
	assert.Equal(t, "three", v)

	k, v, ok = lru.PopFront()
	assert.True(t, ok)
	assert.Equal(t, 2, k)
	assert.Equal(t, "two", v)

	k, v, ok = lru.PopFront()
	assert.True(t, ok)
	assert.Equal(t, 1, k)
	assert.Equal(t, "one", v)

	// Cache should be empty now
	_, _, ok = lru.PopFront()
	assert.False(t, ok)
	assert.Equal(t, 0, lru.Len())
}

// TestLRUMap_UpdateExistingMovesToFront tests that updating existing key moves it to front
func TestLRUMap_UpdateExistingMovesToFront(t *testing.T) {
	// Arrange
	lru := NewLRUMap[int, string](3)
	lru.Put(1, "one")
	lru.Put(2, "two")
	lru.Put(3, "three")

	// Act - update key 1
	lru.Put(1, "ONE")

	// Assert - key 1 should now be at front
	k, v, ok := lru.Front()
	assert.True(t, ok)
	assert.Equal(t, 1, k)
	assert.Equal(t, "ONE", v)

	// And key 2 should be at back
	k, _, ok = lru.Back()
	assert.True(t, ok)
	assert.Equal(t, 2, k)
}

// ============================================================================
// Benchmark tests
// ============================================================================

// BenchmarkLRUMap_Put benchmarks Put operation
func BenchmarkLRUMap_Put(b *testing.B) {
	lru := NewLRUMap[int, string](1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Put(i%1000, "value")
	}
}

// BenchmarkLRUMap_Get benchmarks Get operation
func BenchmarkLRUMap_Get(b *testing.B) {
	lru := NewLRUMap[int, string](1000)
	for i := 0; i < 1000; i++ {
		lru.Put(i, "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Get(i%1000, false)
	}
}

// BenchmarkLRUMap_GetWithUpdate benchmarks Get operation with updateVisit=true
func BenchmarkLRUMap_GetWithUpdate(b *testing.B) {
	lru := NewLRUMap[int, string](1000)
	for i := 0; i < 1000; i++ {
		lru.Put(i, "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Get(i%1000, true)
	}
}

// BenchmarkLRUMap_PutWithEviction benchmarks Put with eviction
func BenchmarkLRUMap_PutWithEviction(b *testing.B) {
	lru := NewLRUMap[int, string](100)
	for i := 0; i < 100; i++ {
		lru.Put(i, "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lru.Put(i+100, "value")
	}
}
