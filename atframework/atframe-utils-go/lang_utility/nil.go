// Copyright 2025 atframework
// Package libatframe_utils_lang_utility provides utility functions for language-level operations.
package libatframe_utils_lang_utility

import (
	"unsafe"
)

// Kind constants matching reflect.Kind values.
// These are stable across Go versions as they are part of the ABI.
const (
	kindPtr           = 22 // reflect.Ptr
	kindUnsafePointer = 26 // reflect.UnsafePointer
	kindSlice         = 23 // reflect.Slice
	kindMap           = 21 // reflect.Map
	kindChan          = 18 // reflect.Chan
	kindFunc          = 19 // reflect.Func
	kindInterface     = 20 // reflect.Interface

	// kindMask is used to extract the actual kind from the kind byte,
	// which may have additional flags in the high bits.
	kindMask = (1 << 5) - 1
)

// rtype mirrors the beginning of runtime._type / reflect.rtype.
// We only need to access the 'kind' field to determine the type category.
// Layout (64-bit): size(8) + ptrdata(8) + hash(4) + tflag(1) + align(1) + fieldAlign(1) + kind(1)
// Layout (32-bit): size(4) + ptrdata(4) + hash(4) + tflag(1) + align(1) + fieldAlign(1) + kind(1)
type rtype struct {
	size       uintptr
	ptrdata    uintptr
	hash       uint32
	tflag      uint8
	align      uint8
	fieldAlign uint8
	kind       uint8
	// ... more fields follow but we don't need them
}

// sliceHeader mirrors the internal structure of a Go slice.
type sliceHeader struct {
	Data unsafe.Pointer
	Len  int
	Cap  int
}

// IsNil reports whether the interface value i is nil.
//
// Unlike a simple "i == nil" comparison, this function correctly handles
// the case where an interface contains a nil pointer (e.g., a nil *T
// assigned to an interface{} is not equal to nil, but IsNil returns true).
//
// This implementation uses unsafe pointer manipulation for maximum performance,
// completely avoiding reflection overhead. It directly accesses the interface's
// internal structure and the runtime type information.
//
// Supported nil-able types: pointer, slice, map, channel, function, interface, unsafe.Pointer.
// For non-nil-able types (int, string, struct, etc.), returns false.
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}

	// Get the eface structure
	ef := (*eface)(unsafe.Pointer(&i))

	// Get the kind from the type information
	typ := (*rtype)(ef._type)
	kind := typ.kind & kindMask

	// Check based on kind
	switch kind {
	case kindPtr, kindUnsafePointer:
		// For pointers:
		// - If data is nil, it means the pointer value is nil
		// - If data is not nil, it points to the actual pointer value
		//   (Go stores pointer values directly in data for types that fit in a word)
		// We need to check: is the POINTED-TO value nil?
		// But wait - for a pointer to nil, data itself is nil.
		// For a pointer to non-nil, data IS the pointer value.
		// Actually in eface, for pointer types, data IS the pointer value directly.
		return ef.data == nil

	case kindSlice:
		// For nil slice, data is nil
		if ef.data == nil {
			return true
		}
		// Slice is stored as sliceHeader, check if Data pointer is nil
		header := (*sliceHeader)(ef.data)
		return header.Data == nil

	case kindMap, kindChan, kindFunc:
		// For these types, the pointer value is stored directly in data
		// nil map/chan/func => data is nil
		// non-nil map/chan/func => data is the hmap/hchan/funcval pointer
		return ef.data == nil

	case kindInterface:
		// For nil interface value inside, data would be nil
		if ef.data == nil {
			return true
		}
		// Nested interface: check if the inner interface's data is nil
		innerEface := (*eface)(ef.data)
		return innerEface._type == nil

	default:
		// Non-nil-able types (int, string, struct, array, etc.)
		return false
	}
}
