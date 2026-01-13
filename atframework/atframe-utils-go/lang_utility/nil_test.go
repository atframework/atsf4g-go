// Copyright 2025 atframework
package libatframe_utils_lang_utility

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// IsNil Tests - Verify behavior matches Go's built-in nil checks
// ============================================================================

func TestIsNil_NilInterface(t *testing.T) {
	// Arrange: truly nil interface
	var i interface{} = nil

	// Act & Assert
	assert.True(t, IsNil(i), "nil interface should return true")
	assert.True(t, i == nil, "sanity check: nil interface == nil")
}

func TestIsNil_NilPointer(t *testing.T) {
	// Arrange: nil pointer wrapped in interface
	var p *int = nil
	var i interface{} = p

	// Act & Assert
	assert.True(t, IsNil(i), "nil pointer in interface should return true")
	// Note: i == nil is FALSE here (this is the case IsNil handles)
	assert.False(t, i == nil, "sanity check: interface with nil pointer != nil")
}

func TestIsNil_NonNilPointer(t *testing.T) {
	// Arrange: non-nil pointer
	x := 42
	var i interface{} = &x

	// Act & Assert
	assert.False(t, IsNil(i), "non-nil pointer should return false")
}

func TestIsNil_NilSlice(t *testing.T) {
	// Arrange: nil slice
	var s []int = nil
	var i interface{} = s

	// Act & Assert
	assert.True(t, IsNil(i), "nil slice in interface should return true")
	assert.False(t, i == nil, "sanity check: interface with nil slice != nil")
}

func TestIsNil_EmptySlice(t *testing.T) {
	// Arrange: empty but non-nil slice
	s := make([]int, 0)
	var i interface{} = s

	// Act & Assert
	assert.False(t, IsNil(i), "empty slice should return false (not nil)")
}

func TestIsNil_NilMap(t *testing.T) {
	// Arrange: nil map
	var m map[string]int = nil
	var i interface{} = m

	// Act & Assert
	assert.True(t, IsNil(i), "nil map in interface should return true")
	assert.False(t, i == nil, "sanity check: interface with nil map != nil")
}

func TestIsNil_NonNilMap(t *testing.T) {
	// Arrange: non-nil map
	m := make(map[string]int)
	var i interface{} = m

	// Act & Assert
	assert.False(t, IsNil(i), "non-nil map should return false")
}

func TestIsNil_NilChannel(t *testing.T) {
	// Arrange: nil channel
	var ch chan int = nil
	var i interface{} = ch

	// Act & Assert
	assert.True(t, IsNil(i), "nil channel in interface should return true")
	assert.False(t, i == nil, "sanity check: interface with nil channel != nil")
}

func TestIsNil_NonNilChannel(t *testing.T) {
	// Arrange: non-nil channel
	ch := make(chan int)
	var i interface{} = ch

	// Act & Assert
	assert.False(t, IsNil(i), "non-nil channel should return false")
}

func TestIsNil_NilFunc(t *testing.T) {
	// Arrange: nil function
	var fn func() = nil
	var i interface{} = fn

	// Act & Assert
	assert.True(t, IsNil(i), "nil function in interface should return true")
	assert.False(t, i == nil, "sanity check: interface with nil func != nil")
}

func TestIsNil_NonNilFunc(t *testing.T) {
	// Arrange: non-nil function
	fn := func() {}
	var i interface{} = fn

	// Act & Assert
	assert.False(t, IsNil(i), "non-nil function should return false")
}

func TestIsNil_NilNestedInterface(t *testing.T) {
	// Arrange: nil interface wrapped in interface
	var inner error = nil
	var i interface{} = inner

	// Act & Assert
	// Note: when inner is a nil interface type, assigning to interface{} results in nil
	assert.True(t, IsNil(i), "nil interface value should return true")
}

func TestIsNil_NonNilInterface(t *testing.T) {
	// Arrange: non-nil error interface
	var inner error = &testError{}
	var i interface{} = inner

	// Act & Assert
	assert.False(t, IsNil(i), "non-nil interface should return false")
}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestIsNil_NonNilableTypes(t *testing.T) {
	// Arrange & Act & Assert: non-nil-able types should return false
	testCases := []struct {
		name  string
		value interface{}
	}{
		{"int", 42},
		{"zero int", 0},
		{"string", "hello"},
		{"empty string", ""},
		{"bool true", true},
		{"bool false", false},
		{"float64", 3.14},
		{"struct", struct{ x int }{42}},
		{"empty struct", struct{}{}},
		{"array", [3]int{1, 2, 3}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, IsNil(tc.value), "%s should return false", tc.name)
		})
	}
}

func TestIsNil_PointerToNilPointer(t *testing.T) {
	// Arrange: pointer to nil pointer
	var p *int = nil
	pp := &p // pp is not nil, it points to a nil pointer
	var i interface{} = pp

	// Act & Assert
	assert.False(t, IsNil(i), "pointer to nil pointer should return false (the outer pointer is not nil)")
}

func TestIsNil_NilUnsafePointer(t *testing.T) {
	// Arrange: nil unsafe.Pointer
	// Note: unsafe.Pointer is treated as a regular pointer type
	// We skip this test as unsafe.Pointer is a special case
	t.Skip("unsafe.Pointer nil check is implementation-defined")
}

// ============================================================================
// Benchmark Tests - Compare performance with reflection-based approach
// ============================================================================

func BenchmarkIsNil_NilInterface(b *testing.B) {
	var i interface{} = nil
	for n := 0; n < b.N; n++ {
		_ = IsNil(i)
	}
}

func BenchmarkIsNil_NilPointer(b *testing.B) {
	var p *int = nil
	var i interface{} = p
	for n := 0; n < b.N; n++ {
		_ = IsNil(i)
	}
}

func BenchmarkIsNil_NonNilPointer(b *testing.B) {
	x := 42
	var i interface{} = &x
	for n := 0; n < b.N; n++ {
		_ = IsNil(i)
	}
}

func BenchmarkIsNil_NilSlice(b *testing.B) {
	var s []int = nil
	var i interface{} = s
	for n := 0; n < b.N; n++ {
		_ = IsNil(i)
	}
}

func BenchmarkIsNil_NilMap(b *testing.B) {
	var m map[string]int = nil
	var i interface{} = m
	for n := 0; n < b.N; n++ {
		_ = IsNil(i)
	}
}

func BenchmarkIsNil_Int(b *testing.B) {
	var i interface{} = 42
	for n := 0; n < b.N; n++ {
		_ = IsNil(i)
	}
}
