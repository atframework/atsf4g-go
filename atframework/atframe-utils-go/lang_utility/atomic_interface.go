// Copyright 2025 atframework
package libatframe_utils_lang_utility

import (
	"sync/atomic"
)

// AtomicInterface is a lock-free holder dedicated to interface values, providing nil-safe helpers.
type AtomicInterface[T comparable] struct {
	ptr atomic.Pointer[atomicInterfaceHolder[T]]
}

// NewAtomicInterface creates an AtomicInterface initialized with the provided value.
// The initial value may be nil.
func NewAtomicInterface[T comparable](initial T) *AtomicInterface[T] {
	av := &AtomicInterface[T]{}
	av.Store(initial)
	return av
}

type atomicInterfaceHolder[T comparable] struct {
	value T
	isNil bool
}

// Store saves v atomically. It accepts typed nil values.
func (av *AtomicInterface[T]) Store(v T) {
	if av == nil {
		return
	}
	av.ptr.Store(newAtomicInterfaceHolder(v))
}

// Load returns the stored value or the zero value if nothing has been stored yet.
func (av *AtomicInterface[T]) Load() T {
	var zero T
	if av == nil {
		return zero
	}
	holder := av.ptr.Load()
	if holder == nil || holder.isNil {
		return zero
	}
	return holder.value
}

// Swap atomically stores newV and returns the previously stored value.
func (av *AtomicInterface[T]) Swap(newV T) T {
	var zero T
	if av == nil {
		return zero
	}
	newHolder := newAtomicInterfaceHolder(newV)
	value := av.ptr.Swap(newHolder)
	if value == nil || value.isNil {
		return zero
	}
	return value.value
}

// CompareAndSwap atomically replaces the stored value with newV when the current value matches oldV.
// It returns true on success.
func (av *AtomicInterface[T]) CompareAndSwap(oldV, newV T) bool {
	if av == nil {
		return false
	}
	newHolder := newAtomicInterfaceHolder(newV)
	expectedNil := IsNil(any(oldV))

	for {
		current := av.ptr.Load()
		if current == nil {
			if !expectedNil {
				return false
			}
			if av.ptr.CompareAndSwap(nil, newHolder) {
				return true
			}
			continue
		}

		if !holderEquals(current, oldV) {
			return false
		}

		if av.ptr.CompareAndSwap(current, newHolder) {
			return true
		}
	}
}

func newAtomicInterfaceHolder[T comparable](v T) *atomicInterfaceHolder[T] {
	return &atomicInterfaceHolder[T]{
		value: v,
		isNil: IsNil(any(v)),
	}
}

func holderEquals[T comparable](holder *atomicInterfaceHolder[T], expected T) bool {
	if holder == nil || holder.isNil {
		return IsNil(any(expected))
	}
	if IsNil(any(expected)) {
		return false
	}
	return holder.value == expected
}
