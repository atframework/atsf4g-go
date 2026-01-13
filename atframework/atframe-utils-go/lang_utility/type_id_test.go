// Copyright 2025 atframework
// Package libatframe_utils_lang_utility provides utility functions for language-level operations.
package libatframe_utils_lang_utility

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Custom Types for Testing
// ============================================================================

// UserModule represents a user module in a game server
type UserModule struct {
	UserID   uint64
	Username string
	Level    int
}

// ItemModule represents an item module in a game server
type ItemModule struct {
	ItemID   uint64
	Name     string
	Quantity int
}

// BagModule represents a bag/inventory module
type BagModule struct {
	Capacity int
	Items    []ItemModule
}

// TaskModule represents a task/quest module
type TaskModule struct {
	TaskID     uint64
	Progress   int
	IsComplete bool
}

// AnotherUserModule has the same fields as UserModule but is a different type
type AnotherUserModule struct {
	UserID   uint64
	Username string
	Level    int
}

// GenericModule is a generic module with type parameter
type GenericModule[T any] struct {
	Data T
}

// ModuleInterface defines a common interface for modules
type ModuleInterface interface {
	GetModuleID() string
}

func (u *UserModule) GetModuleID() string { return "user" }
func (i *ItemModule) GetModuleID() string { return "item" }
func (b *BagModule) GetModuleID() string  { return "bag" }
func (t *TaskModule) GetModuleID() string { return "task" }

// ============================================================================
// Unit Tests for TypeID
// ============================================================================

func TestGetTypeID_SameType(t *testing.T) {
	// Arrange: two values of the same type
	var x int = 42
	var y int = 100

	// Act
	idX := GetTypeID(x)
	idY := GetTypeID(y)

	// Assert
	assert.Equal(t, idX, idY, "same type should have same TypeID")
	assert.True(t, idX.IsValid(), "TypeID should be valid")
}

func TestGetTypeID_DifferentTypes(t *testing.T) {
	// Arrange: values of different types
	var x int = 42
	var y string = "hello"
	var z float64 = 3.14

	// Act
	idX := GetTypeID(x)
	idY := GetTypeID(y)
	idZ := GetTypeID(z)

	// Assert
	assert.NotEqual(t, idX, idY, "int and string should have different TypeIDs")
	assert.NotEqual(t, idX, idZ, "int and float64 should have different TypeIDs")
	assert.NotEqual(t, idY, idZ, "string and float64 should have different TypeIDs")
}

func TestGetTypeID_NilInterface(t *testing.T) {
	// Arrange
	var i interface{} = nil

	// Act
	id := GetTypeID(i)

	// Assert
	assert.Equal(t, TypeID(0), id, "nil interface should have TypeID 0")
	assert.False(t, id.IsValid(), "nil TypeID should not be valid")
}

func TestGetTypeID_PointerTypes(t *testing.T) {
	// Arrange
	var x int = 42
	var px *int = &x
	var py *int = &x
	var pz *string

	// Act
	idPx := GetTypeID(px)
	idPy := GetTypeID(py)
	idPz := GetTypeID(pz)

	// Assert
	assert.Equal(t, idPx, idPy, "*int should have same TypeID")
	assert.NotEqual(t, idPx, idPz, "*int and *string should have different TypeIDs")
}

func TestGetTypeID_StructTypes(t *testing.T) {
	// Arrange
	type Foo struct{ X int }
	type Bar struct{ X int }

	var foo Foo
	var bar Bar

	// Act
	idFoo := GetTypeID(foo)
	idBar := GetTypeID(bar)

	// Assert: different named types should have different TypeIDs
	assert.NotEqual(t, idFoo, idBar, "Foo and Bar should have different TypeIDs even with same fields")
}

func TestGetTypeID_SliceTypes(t *testing.T) {
	// Arrange
	var intSlice []int
	var strSlice []string

	// Act
	idInt := GetTypeID(intSlice)
	idStr := GetTypeID(strSlice)

	// Assert
	assert.NotEqual(t, idInt, idStr, "[]int and []string should have different TypeIDs")
}

func TestGetTypeID_MapTypes(t *testing.T) {
	// Arrange
	var m1 map[string]int
	var m2 map[string]string

	// Act
	id1 := GetTypeID(m1)
	id2 := GetTypeID(m2)

	// Assert
	assert.NotEqual(t, id1, id2, "map[string]int and map[string]string should have different TypeIDs")
}

func TestGetTypeIDOf_Generic(t *testing.T) {
	// Act
	idInt := GetTypeIDOf[int]()
	idStr := GetTypeIDOf[string]()

	// Assert
	assert.True(t, idInt.IsValid(), "TypeID for int should be valid")
	assert.True(t, idStr.IsValid(), "TypeID for string should be valid")
	assert.NotEqual(t, idInt, idStr, "int and string should have different TypeIDs")

	// Compare with GetTypeID
	assert.Equal(t, idInt, GetTypeID(42), "GetTypeIDOf[int]() should match GetTypeID(42)")
	assert.Equal(t, idStr, GetTypeID("hello"), "GetTypeIDOf[string]() should match GetTypeID(\"hello\")")
}

func TestGetTypeIDOfPointer_Generic(t *testing.T) {
	// Act
	idPtrInt := GetTypeIDOfPointer[int]()
	idPtrStr := GetTypeIDOfPointer[string]()

	// Assert
	assert.True(t, idPtrInt.IsValid(), "TypeID for *int should be valid")
	assert.NotEqual(t, idPtrInt, idPtrStr, "*int and *string should have different TypeIDs")

	// Compare with GetTypeID
	var p *int
	assert.Equal(t, idPtrInt, GetTypeID(p), "GetTypeIDOfPointer[int]() should match GetTypeID((*int)(nil))")
}

func TestTypeID_AsMapKey(t *testing.T) {
	// Arrange: create a map using TypeID as key
	typeNames := make(map[TypeID]string)

	// Act: populate the map
	typeNames[GetTypeIDOf[int]()] = "int"
	typeNames[GetTypeIDOf[string]()] = "string"
	typeNames[GetTypeIDOf[float64]()] = "float64"

	// Assert: retrieve values
	assert.Equal(t, "int", typeNames[GetTypeID(42)])
	assert.Equal(t, "string", typeNames[GetTypeID("hello")])
	assert.Equal(t, "float64", typeNames[GetTypeID(3.14)])

	// Non-existent type should return empty string
	assert.Equal(t, "", typeNames[GetTypeIDOf[bool]()])
}

func TestTypeID_String(t *testing.T) {
	// Arrange
	id := GetTypeIDOf[int]()
	nilID := TypeID(0)

	// Act
	str := id.String()
	nilStr := nilID.String()

	// Assert
	assert.Contains(t, str, "TypeID(0x", "TypeID string should contain prefix")
	assert.Equal(t, "TypeID(nil)", nilStr, "nil TypeID should have specific string")
}

func TestTypeID_ConsistencyWithReflect(t *testing.T) {
	// This test verifies that TypeID behaves consistently with reflect.Type
	// for type identity comparison

	testCases := []struct {
		name string
		v1   interface{}
		v2   interface{}
		same bool
	}{
		{"same int values", 1, 2, true},
		{"int vs int64", int(1), int64(1), false},
		{"int vs uint", int(1), uint(1), false},
		{"same string values", "a", "b", true},
		{"string vs []byte", "a", []byte("a"), false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id1 := GetTypeID(tc.v1)
			id2 := GetTypeID(tc.v2)

			rt1 := reflect.TypeOf(tc.v1)
			rt2 := reflect.TypeOf(tc.v2)

			// TypeID equality should match reflect.Type equality
			typeIDEqual := id1 == id2
			reflectEqual := rt1 == rt2

			assert.Equal(t, tc.same, typeIDEqual, "TypeID comparison mismatch")
			assert.Equal(t, tc.same, reflectEqual, "reflect.Type comparison mismatch")
			assert.Equal(t, typeIDEqual, reflectEqual, "TypeID and reflect.Type should agree")
		})
	}
}

// ============================================================================
// Custom Type Tests - Ensure same type instances have same TypeID
// ============================================================================

func TestTypeID_CustomType_SameTypeMultipleInstances(t *testing.T) {
	// Arrange: create multiple instances of the same custom type
	user1 := UserModule{UserID: 1, Username: "Alice", Level: 10}
	user2 := UserModule{UserID: 2, Username: "Bob", Level: 20}
	user3 := UserModule{UserID: 3, Username: "Charlie", Level: 30}

	// Act
	id1 := GetTypeID(user1)
	id2 := GetTypeID(user2)
	id3 := GetTypeID(user3)

	// Assert: all instances of the same type should have the same TypeID
	assert.Equal(t, id1, id2, "same type (UserModule) instances should have same TypeID")
	assert.Equal(t, id2, id3, "same type (UserModule) instances should have same TypeID")
	assert.Equal(t, id1, id3, "same type (UserModule) instances should have same TypeID")
	assert.True(t, id1.IsValid(), "TypeID should be valid")
}

func TestTypeID_CustomType_PointerInstances(t *testing.T) {
	// Arrange: create multiple pointer instances of the same type
	user1 := &UserModule{UserID: 1, Username: "Alice", Level: 10}
	user2 := &UserModule{UserID: 2, Username: "Bob", Level: 20}
	user3 := &UserModule{UserID: 3, Username: "Charlie", Level: 30}

	// Act
	id1 := GetTypeID(user1)
	id2 := GetTypeID(user2)
	id3 := GetTypeID(user3)

	// Assert: all pointer instances of the same type should have the same TypeID
	assert.Equal(t, id1, id2, "*UserModule instances should have same TypeID")
	assert.Equal(t, id2, id3, "*UserModule instances should have same TypeID")
}

func TestTypeID_CustomType_ValueVsPointer(t *testing.T) {
	// Arrange: value type vs pointer type
	userValue := UserModule{UserID: 1, Username: "Alice", Level: 10}
	userPtr := &UserModule{UserID: 2, Username: "Bob", Level: 20}

	// Act
	idValue := GetTypeID(userValue)
	idPtr := GetTypeID(userPtr)

	// Assert: value type and pointer type should have different TypeIDs
	assert.NotEqual(t, idValue, idPtr, "UserModule and *UserModule should have different TypeIDs")
}

func TestTypeID_CustomType_DifferentTypes(t *testing.T) {
	// Arrange: instances of different custom types
	user := UserModule{UserID: 1, Username: "Alice", Level: 10}
	item := ItemModule{ItemID: 100, Name: "Sword", Quantity: 1}
	bag := BagModule{Capacity: 50, Items: nil}
	task := TaskModule{TaskID: 1000, Progress: 50, IsComplete: false}

	// Act
	idUser := GetTypeID(user)
	idItem := GetTypeID(item)
	idBag := GetTypeID(bag)
	idTask := GetTypeID(task)

	// Assert: different types must have different TypeIDs
	assert.NotEqual(t, idUser, idItem, "UserModule and ItemModule should have different TypeIDs")
	assert.NotEqual(t, idUser, idBag, "UserModule and BagModule should have different TypeIDs")
	assert.NotEqual(t, idUser, idTask, "UserModule and TaskModule should have different TypeIDs")
	assert.NotEqual(t, idItem, idBag, "ItemModule and BagModule should have different TypeIDs")
	assert.NotEqual(t, idItem, idTask, "ItemModule and TaskModule should have different TypeIDs")
	assert.NotEqual(t, idBag, idTask, "BagModule and TaskModule should have different TypeIDs")
}

func TestTypeID_CustomType_SameFieldsDifferentTypes(t *testing.T) {
	// Arrange: two types with identical field layout but different type names
	user := UserModule{UserID: 1, Username: "Alice", Level: 10}
	anotherUser := AnotherUserModule{UserID: 1, Username: "Alice", Level: 10}

	// Act
	idUser := GetTypeID(user)
	idAnother := GetTypeID(anotherUser)

	// Assert: different named types should have different TypeIDs even with same fields
	assert.NotEqual(t, idUser, idAnother,
		"UserModule and AnotherUserModule should have different TypeIDs despite same fields")
}

func TestTypeID_CustomType_GenericTypeDifferentParams(t *testing.T) {
	// Arrange: generic types with different type parameters
	intModule := GenericModule[int]{Data: 42}
	strModule := GenericModule[string]{Data: "hello"}
	userModule := GenericModule[UserModule]{Data: UserModule{}}

	// Act
	idInt := GetTypeID(intModule)
	idStr := GetTypeID(strModule)
	idUser := GetTypeID(userModule)

	// Assert: generic types with different type parameters should have different TypeIDs
	assert.NotEqual(t, idInt, idStr, "GenericModule[int] and GenericModule[string] should differ")
	assert.NotEqual(t, idInt, idUser, "GenericModule[int] and GenericModule[UserModule] should differ")
	assert.NotEqual(t, idStr, idUser, "GenericModule[string] and GenericModule[UserModule] should differ")
}

func TestTypeID_CustomType_GenericTypeSameParams(t *testing.T) {
	// Arrange: multiple instances of generic type with same type parameter
	mod1 := GenericModule[int]{Data: 1}
	mod2 := GenericModule[int]{Data: 2}
	mod3 := GenericModule[int]{Data: 3}

	// Act
	id1 := GetTypeID(mod1)
	id2 := GetTypeID(mod2)
	id3 := GetTypeID(mod3)

	// Assert: same generic type with same type parameter should have same TypeID
	assert.Equal(t, id1, id2, "GenericModule[int] instances should have same TypeID")
	assert.Equal(t, id2, id3, "GenericModule[int] instances should have same TypeID")
}

func TestTypeID_CustomType_InterfaceImplementors(t *testing.T) {
	// Arrange: different types implementing the same interface
	var mod1 ModuleInterface = &UserModule{UserID: 1}
	var mod2 ModuleInterface = &ItemModule{ItemID: 100}
	var mod3 ModuleInterface = &BagModule{Capacity: 50}
	var mod4 ModuleInterface = &UserModule{UserID: 2} // another UserModule

	// Act
	id1 := GetTypeID(mod1)
	id2 := GetTypeID(mod2)
	id3 := GetTypeID(mod3)
	id4 := GetTypeID(mod4)

	// Assert: different concrete types should have different TypeIDs
	assert.NotEqual(t, id1, id2, "*UserModule and *ItemModule should have different TypeIDs")
	assert.NotEqual(t, id1, id3, "*UserModule and *BagModule should have different TypeIDs")
	assert.NotEqual(t, id2, id3, "*ItemModule and *BagModule should have different TypeIDs")

	// Same concrete type should have same TypeID
	assert.Equal(t, id1, id4, "*UserModule instances should have same TypeID")
}

func TestTypeID_CustomType_SliceOfCustomTypes(t *testing.T) {
	// Arrange: slices of different custom types
	var userSlice []UserModule
	var itemSlice []ItemModule
	var userSlice2 []UserModule

	// Act
	idUserSlice := GetTypeID(userSlice)
	idItemSlice := GetTypeID(itemSlice)
	idUserSlice2 := GetTypeID(userSlice2)

	// Assert
	assert.NotEqual(t, idUserSlice, idItemSlice, "[]UserModule and []ItemModule should differ")
	assert.Equal(t, idUserSlice, idUserSlice2, "[]UserModule instances should have same TypeID")
}

func TestTypeID_CustomType_MapOfCustomTypes(t *testing.T) {
	// Arrange: maps with custom type values
	var userMap map[uint64]UserModule
	var itemMap map[uint64]ItemModule
	var userMap2 map[uint64]UserModule

	// Act
	idUserMap := GetTypeID(userMap)
	idItemMap := GetTypeID(itemMap)
	idUserMap2 := GetTypeID(userMap2)

	// Assert
	assert.NotEqual(t, idUserMap, idItemMap,
		"map[uint64]UserModule and map[uint64]ItemModule should differ")
	assert.Equal(t, idUserMap, idUserMap2,
		"map[uint64]UserModule instances should have same TypeID")
}

func TestTypeID_CustomType_AllDifferentGuarantee(t *testing.T) {
	// This test ensures that ALL different types have different TypeIDs
	// by collecting all TypeIDs and checking for uniqueness

	types := []interface{}{
		// Primitive types
		int(0), int8(0), int16(0), int32(0), int64(0),
		uint(0), uint8(0), uint16(0), uint32(0), uint64(0),
		float32(0), float64(0),
		complex64(0), complex128(0),
		true, "",
		// Custom struct types
		UserModule{},
		ItemModule{},
		BagModule{},
		TaskModule{},
		AnotherUserModule{},
		// Pointer types
		(*UserModule)(nil), (*ItemModule)(nil), (*BagModule)(nil),
		// Slice types
		[]int(nil), []string(nil), []UserModule(nil), []ItemModule(nil),
		// Map types
		map[string]int(nil), map[uint64]UserModule(nil), map[uint64]ItemModule(nil),
		// Generic types
		GenericModule[int]{},
		GenericModule[string]{},
		GenericModule[UserModule]{},
		// Channel types
		(chan int)(nil), (chan string)(nil), (chan UserModule)(nil),
	}

	// Collect all TypeIDs
	typeIDSet := make(map[TypeID]int) // TypeID -> index in types slice
	for i, v := range types {
		id := GetTypeID(v)
		if existingIdx, exists := typeIDSet[id]; exists {
			t.Errorf("TypeID collision detected: types[%d] (%T) and types[%d] (%T) have same TypeID",
				existingIdx, types[existingIdx], i, v)
		}
		typeIDSet[id] = i
	}

	// Verify we have the expected number of unique TypeIDs
	assert.Equal(t, len(types), len(typeIDSet),
		"all different types should have unique TypeIDs")
}

func TestTypeID_CustomType_GetTypeIDOf_Generic(t *testing.T) {
	// Test GetTypeIDOf with custom types
	idUser := GetTypeIDOf[UserModule]()
	idItem := GetTypeIDOf[ItemModule]()
	idBag := GetTypeIDOf[BagModule]()

	// Verify they match instances
	assert.Equal(t, idUser, GetTypeID(UserModule{}), "GetTypeIDOf[UserModule]() should match instance")
	assert.Equal(t, idItem, GetTypeID(ItemModule{}), "GetTypeIDOf[ItemModule]() should match instance")
	assert.Equal(t, idBag, GetTypeID(BagModule{}), "GetTypeIDOf[BagModule]() should match instance")

	// Verify they are different
	assert.NotEqual(t, idUser, idItem)
	assert.NotEqual(t, idUser, idBag)
	assert.NotEqual(t, idItem, idBag)
}

func TestTypeID_CustomType_ModuleRegistry(t *testing.T) {
	// Simulate a module registry use case
	type ModuleFactory func() ModuleInterface

	registry := make(map[TypeID]ModuleFactory)

	// Register factories
	registry[GetTypeIDOf[UserModule]()] = func() ModuleInterface { return &UserModule{} }
	registry[GetTypeIDOf[ItemModule]()] = func() ModuleInterface { return &ItemModule{} }
	registry[GetTypeIDOf[BagModule]()] = func() ModuleInterface { return &BagModule{} }

	// Test retrieval with different instances
	user1 := UserModule{UserID: 1}
	user2 := UserModule{UserID: 999}
	item := ItemModule{ItemID: 100}

	// Same type should retrieve same factory
	factory1, ok1 := registry[GetTypeID(user1)]
	factory2, ok2 := registry[GetTypeID(user2)]
	factoryItem, okItem := registry[GetTypeID(item)]

	assert.True(t, ok1 && ok2, "should find UserModule factory")
	assert.True(t, okItem, "should find ItemModule factory")

	// Verify factories create correct types
	assert.Equal(t, "user", factory1().GetModuleID())
	assert.Equal(t, "user", factory2().GetModuleID())
	assert.Equal(t, "item", factoryItem().GetModuleID())
}

// ============================================================================
// Benchmark Tests - Compare with reflect.TypeOf
// ============================================================================

func BenchmarkGetTypeID_Int(b *testing.B) {
	var x interface{} = 42
	for n := 0; n < b.N; n++ {
		_ = GetTypeID(x)
	}
}

func BenchmarkReflectTypeOf_Int(b *testing.B) {
	var x interface{} = 42
	for n := 0; n < b.N; n++ {
		_ = reflect.TypeOf(x)
	}
}

func BenchmarkGetTypeIDOf_Generic(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = GetTypeIDOf[int]()
	}
}

func BenchmarkGetStaticReflectType_Generic(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = GetStaticReflectType[int]()
	}
}

func BenchmarkTypeID_MapLookup(b *testing.B) {
	m := make(map[TypeID]int)
	m[GetTypeIDOf[int]()] = 1
	m[GetTypeIDOf[string]()] = 2
	m[GetTypeIDOf[float64]()] = 3

	var x interface{} = 42
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = m[GetTypeID(x)]
	}
}

func BenchmarkReflectType_MapLookup(b *testing.B) {
	m := make(map[reflect.Type]int)
	m[reflect.TypeOf(int(0))] = 1
	m[reflect.TypeOf(string(""))] = 2
	m[reflect.TypeOf(float64(0))] = 3

	var x interface{} = 42
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_ = m[reflect.TypeOf(x)]
	}
}

func BenchmarkGetTypeID_Struct(b *testing.B) {
	type TestStruct struct {
		A int
		B string
		C float64
	}
	var x interface{} = TestStruct{}
	for n := 0; n < b.N; n++ {
		_ = GetTypeID(x)
	}
}

func BenchmarkReflectTypeOf_Struct(b *testing.B) {
	type TestStruct struct {
		A int
		B string
		C float64
	}
	var x interface{} = TestStruct{}
	for n := 0; n < b.N; n++ {
		_ = reflect.TypeOf(x)
	}
}
