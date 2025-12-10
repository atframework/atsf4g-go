package libatframe_utils_lang_utility

import (
	"testing"
	"unsafe"
)

func TestInterfaceLayoutCompatibility(t *testing.T) {
	// 1. 验证大小
	if unsafe.Sizeof(eface{}) != unsafe.Sizeof(interface{}(nil)) {
		t.Fatal("interface{} size changed")
	}

	// 2. 验证指针提取逻辑
	target := 12345
	var i interface{} = &target

	// 模拟业务逻辑中的转换
	myFace := GetDataPointerOfInterface(i)

	// 验证提取出的地址是否正确
	if myFace != unsafe.Pointer(&target) {
		t.Fatalf("Layout mismatch! Expected data ptr %p, got %p", &target, myFace)
	}
}

func TestGetDataPointerOfInterfaceNil(t *testing.T) {
	// Arrange: nil interface
	var i interface{} = nil

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert
	if result != nil {
		t.Fatalf("Expected nil for nil interface, got %p", result)
	}
}

func TestGetDataPointerOfInterfaceMap(t *testing.T) {
	// Arrange: 创建一个 map
	target := make(map[string]int)
	target["key"] = 42
	var i interface{} = target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert: map 本身就是引用类型，interface 中存储的是 map 的底层指针
	if result == nil {
		t.Fatal("Expected non-nil pointer for map, got nil")
	}

	// 验证通过指针可以访问到正确的数据
	// map 在 interface{} 中存储的是 *hmap 指针
	recoveredMap := *(*map[string]int)(unsafe.Pointer(&result))
	if recoveredMap["key"] != 42 {
		t.Fatalf("Expected map value 42, got %d", recoveredMap["key"])
	}
}

func TestGetDataPointerOfInterfaceMapPointer(t *testing.T) {
	// Arrange: 创建一个 map 指针
	target := make(map[string]int)
	target["key"] = 100
	var i interface{} = &target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert
	if result != unsafe.Pointer(&target) {
		t.Fatalf("Layout mismatch for map pointer! Expected data ptr %p, got %p", &target, result)
	}
}

func TestGetDataPointerOfInterfaceChan(t *testing.T) {
	// Arrange: 创建一个 channel
	target := make(chan int, 1)
	target <- 123
	var i interface{} = target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert: channel 是引用类型，interface 中存储的是 channel 的底层指针
	if result == nil {
		t.Fatal("Expected non-nil pointer for channel, got nil")
	}

	// 验证通过指针可以访问到正确的数据
	recoveredChan := *(*chan int)(unsafe.Pointer(&result))
	val := <-recoveredChan
	if val != 123 {
		t.Fatalf("Expected channel value 123, got %d", val)
	}
}

func TestGetDataPointerOfInterfaceChanPointer(t *testing.T) {
	// Arrange: 创建一个 channel 指针
	target := make(chan int, 1)
	var i interface{} = &target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert
	if result != unsafe.Pointer(&target) {
		t.Fatalf("Layout mismatch for channel pointer! Expected data ptr %p, got %p", &target, result)
	}
}

type testStruct struct {
	Name  string
	Value int
	Flag  bool
}

func TestGetDataPointerOfInterfaceStructPointer(t *testing.T) {
	// Arrange: 创建一个 struct 指针
	target := testStruct{
		Name:  "test",
		Value: 999,
		Flag:  true,
	}
	var i interface{} = &target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert
	if result != unsafe.Pointer(&target) {
		t.Fatalf("Layout mismatch for struct pointer! Expected data ptr %p, got %p", &target, result)
	}

	// 验证通过指针可以访问到正确的数据
	recoveredStruct := (*testStruct)(result)
	if recoveredStruct.Name != "test" || recoveredStruct.Value != 999 || !recoveredStruct.Flag {
		t.Fatalf("Struct data mismatch! Got: %+v", recoveredStruct)
	}
}

func TestGetDataPointerOfInterfaceStructValue(t *testing.T) {
	// Arrange: 直接存储 struct 值（非指针）
	// 注意：当 struct 值存入 interface{} 时，会发生逃逸，运行时会分配内存并拷贝
	target := testStruct{
		Name:  "value_test",
		Value: 888,
		Flag:  false,
	}
	var i interface{} = target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert: 应该返回非空指针（指向 interface 内部拷贝的数据）
	if result == nil {
		t.Fatal("Expected non-nil pointer for struct value, got nil")
	}

	// 验证通过指针可以访问到正确的数据
	recoveredStruct := (*testStruct)(result)
	if recoveredStruct.Name != "value_test" || recoveredStruct.Value != 888 || recoveredStruct.Flag {
		t.Fatalf("Struct data mismatch! Got: %+v", recoveredStruct)
	}
}

func TestGetDataPointerOfInterfaceSlice(t *testing.T) {
	// Arrange: 创建一个 slice
	target := []int{1, 2, 3, 4, 5}
	var i interface{} = target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert: slice 是引用类型，interface 中存储的是 slice header
	if result == nil {
		t.Fatal("Expected non-nil pointer for slice, got nil")
	}
}

func TestGetDataPointerOfInterfaceSlicePointer(t *testing.T) {
	// Arrange: 创建一个 slice 指针
	target := []int{10, 20, 30}
	var i interface{} = &target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert
	if result != unsafe.Pointer(&target) {
		t.Fatalf("Layout mismatch for slice pointer! Expected data ptr %p, got %p", &target, result)
	}

	// 验证通过指针可以访问到正确的数据
	recoveredSlice := *(*[]int)(result)
	if len(recoveredSlice) != 3 || recoveredSlice[0] != 10 {
		t.Fatalf("Slice data mismatch! Got: %v", recoveredSlice)
	}
}

func TestGetDataPointerOfInterfaceFunc(t *testing.T) {
	// Arrange: 创建一个函数
	target := func(x int) int { return x * 2 }
	var i interface{} = target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert: 函数是引用类型
	if result == nil {
		t.Fatal("Expected non-nil pointer for function, got nil")
	}
}

func TestGetDataPointerOfInterfaceFuncPointer(t *testing.T) {
	// Arrange: 创建一个函数指针
	target := func(x int) int { return x * 2 }
	var i interface{} = &target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert
	if result != unsafe.Pointer(&target) {
		t.Fatalf("Layout mismatch for function pointer! Expected data ptr %p, got %p", &target, result)
	}
}

func TestGetDataPointerOfInterfaceString(t *testing.T) {
	// Arrange: 创建一个 string
	target := "hello world"
	var i interface{} = target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert: string 值存入 interface 时会有 string header 的拷贝
	if result == nil {
		t.Fatal("Expected non-nil pointer for string, got nil")
	}
}

func TestGetDataPointerOfInterfaceStringPointer(t *testing.T) {
	// Arrange: 创建一个 string 指针
	target := "hello pointer"
	var i interface{} = &target

	// Act
	result := GetDataPointerOfInterface(i)

	// Assert
	if result != unsafe.Pointer(&target) {
		t.Fatalf("Layout mismatch for string pointer! Expected data ptr %p, got %p", &target, result)
	}

	// 验证通过指针可以访问到正确的数据
	recoveredString := *(*string)(result)
	if recoveredString != "hello pointer" {
		t.Fatalf("String data mismatch! Got: %s", recoveredString)
	}
}
