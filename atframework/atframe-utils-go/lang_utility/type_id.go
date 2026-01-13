// Copyright 2025 atframework
// Package libatframe_utils_lang_utility provides utility functions for language-level operations.
package libatframe_utils_lang_utility

import "unsafe"

// TypeID is a unique identifier for a Go type.
// It can be used as a map key and is comparable.
// This is a high-performance alternative to reflect.Type when you only need
// type identity comparison, not the full reflection capabilities.
//
// TypeID is derived from the internal runtime type pointer, which is guaranteed
// to be unique and stable for each distinct type within a program.
type TypeID uintptr

// GetTypeID returns the TypeID for the given interface value.
// The TypeID is based on the dynamic type of the value, not the static type.
//
// Example:
//
//	var x int = 42
//	var y int = 100
//	var z string = "hello"
//	GetTypeID(x) == GetTypeID(y)  // true: same type (int)
//	GetTypeID(x) == GetTypeID(z)  // false: different types
func GetTypeID(i interface{}) TypeID {
	if i == nil {
		return 0
	}
	ef := (*eface)(unsafe.Pointer(&i))
	return TypeID(uintptr(ef._type))
}

// GetTypeIDOf returns a unique TypeID for type T that works for ALL types,
// including interface types. This is achieved by using the pointer type *T internally,
// which always has valid type information even for nil values.
//
// The returned TypeID uniquely identifies the type T. Different types always have
// different TypeIDs, and the same type always has the same TypeID.
//
// Example:
//
//	id := GetTypeIDOf[int]()              // works for primitive types
//	id := GetTypeIDOf[MyStruct]()         // works for struct types
//	id := GetTypeIDOf[io.Reader]()        // works for interface types
func GetTypeIDOf[T any]() TypeID {
	var zero *T
	return GetTypeID(zero)
}

// GetTypeIDOfPointer returns the TypeID for pointer type *T.
// This is equivalent to GetTypeIDOf[*T]() but more convenient.
//
// Example:
//
//	id := GetTypeIDOfPointer[MyStruct]()  // returns TypeID for *MyStruct
//	// equivalent to: GetTypeIDOf[*MyStruct]()
func GetTypeIDOfPointer[T any]() TypeID {
	var zero **T
	return GetTypeID(zero)
}

// IsValid returns true if the TypeID is valid (non-zero).
// A zero TypeID indicates a nil interface.
func (t TypeID) IsValid() bool {
	return t != 0
}

// String returns a string representation of the TypeID.
// Note: This does not return the type name; use reflect.Type for that.
func (t TypeID) String() string {
	if t == 0 {
		return "TypeID(nil)"
	}
	// Use a simple hex representation
	const hexDigits = "0123456789abcdef"
	var buf [25]byte // "TypeID(0x" + 16 hex digits + ")"
	buf[0] = 'T'
	buf[1] = 'y'
	buf[2] = 'p'
	buf[3] = 'e'
	buf[4] = 'I'
	buf[5] = 'D'
	buf[6] = '('
	buf[7] = '0'
	buf[8] = 'x'

	v := uintptr(t)
	for i := 15; i >= 0; i-- {
		buf[9+i] = hexDigits[v&0xf]
		v >>= 4
	}
	buf[25-1] = ')'

	// Trim leading zeros but keep at least one digit
	start := 9
	for start < 24 && buf[start] == '0' {
		start++
	}
	if start == 24 {
		start = 23 // Keep at least "0"
	}

	// Build result: prefix + trimmed hex + ")"
	result := make([]byte, 0, 25)
	result = append(result, buf[:9]...)       // "TypeID(0x"
	result = append(result, buf[start:24]...) // hex digits
	result = append(result, ')')
	return string(result)
}
