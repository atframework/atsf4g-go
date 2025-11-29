package libatframe_utils_lang_utility

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type atomicValueTestStruct struct {
	value string
}

func (s *atomicValueTestStruct) Name() string { return s.value }

type atomicValueDemo interface {
	Name() string
}

type atomicValueDemoImpl struct{}

func (*atomicValueDemoImpl) Name() string { return "demo" }

func TestAtomicInterfaceLoadBeforeStore(t *testing.T) {
	var av AtomicInterface[atomicValueDemo]

	result := av.Load()
	assert.Nil(t, result)
}

func TestAtomicInterfaceStoreAndLoadNil(t *testing.T) {
	var av AtomicInterface[atomicValueDemo]

	av.Store(nil)

	result := av.Load()
	assert.Nil(t, result)

	expected := &atomicValueTestStruct{value: "hello"}
	av.Store(expected)

	result = av.Load()
	assert.Equal(t, expected, result)
}

func TestNewAtomicInterfaceWithInitialValue(t *testing.T) {
	initial := &atomicValueTestStruct{value: "init"}
	av := NewAtomicInterface[atomicValueDemo](initial)

	result := av.Load()
	assert.Equal(t, initial, result)
}

func TestNewAtomicInterfaceWithNil(t *testing.T) {
	av := NewAtomicInterface[atomicValueDemo](nil)

	result := av.Load()
	assert.Nil(t, result)
}

func TestAtomicInterfaceSwap(t *testing.T) {
	var av AtomicInterface[atomicValueDemo]

	first := &atomicValueTestStruct{value: "first"}
	prev := av.Swap(first)
	assert.Nil(t, prev)

	second := &atomicValueTestStruct{value: "second"}
	prev = av.Swap(second)
	assert.Equal(t, first, prev)

	result := av.Load()
	assert.Equal(t, second, result)
}

func TestAtomicInterfaceStoreTypedNilInterface(t *testing.T) {
	var av AtomicInterface[atomicValueDemo]

	var typedNil atomicValueDemo = (*atomicValueDemoImpl)(nil)
	av.Store(typedNil)

	result := av.Load()
	assert.Nil(t, result)

	av.Store(&atomicValueDemoImpl{})
	result = av.Load()
	assert.NotNil(t, result)
	assert.Equal(t, "demo", result.Name())
}

func TestAtomicInterfaceCompareAndSwapInitialize(t *testing.T) {
	var av AtomicInterface[atomicValueDemo]

	first := &atomicValueTestStruct{value: "first"}
	swapped := av.CompareAndSwap(nil, first)

	assert.True(t, swapped)
	result := av.Load()
	assert.Equal(t, first, result)
}

func TestAtomicInterfaceCompareAndSwapMismatch(t *testing.T) {
	var av AtomicInterface[atomicValueDemo]
	current := &atomicValueTestStruct{value: "current"}
	av.Store(current)

	other := &atomicValueTestStruct{value: "current"}
	swapped := av.CompareAndSwap(other, &atomicValueTestStruct{value: "next"})

	assert.False(t, swapped)
	result := av.Load()
	assert.Equal(t, current, result)
}

func TestAtomicInterfaceCompareAndSwapToNil(t *testing.T) {
	var av AtomicInterface[atomicValueDemo]
	current := &atomicValueTestStruct{value: "current"}
	av.Store(current)

	swapped := av.CompareAndSwap(current, nil)

	assert.True(t, swapped)
	result := av.Load()
	assert.Nil(t, result)
}

func TestAtomicInterfaceCompareAndSwapFromStoredNil(t *testing.T) {
	var av AtomicInterface[atomicValueDemo]
	av.Store(nil)
	second := &atomicValueTestStruct{value: "second"}

	swapped := av.CompareAndSwap(nil, second)

	assert.True(t, swapped)
	result := av.Load()
	assert.Equal(t, second, result)
}

// Uncovered scenarios: None – every public method (Load/Store/Swap/CompareAndSwap/New) is exercised above.
