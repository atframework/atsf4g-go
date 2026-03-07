//go:build linux
// +build linux

package dynamiclib_caller

import (
	"testing"
)

// TestBasicFunctionBinding 测试基本的函数绑定功能
func TestBasicFunctionBinding(t *testing.T) {
	// 创建 DynlibCaller，加载 libtest.so 并指定需要的函数
	caller, err := NewDynlibCaller("./test/libtest.so", []string{"add", "sub"})
	if err != nil {
		t.Fatalf("failed to create DynlibCaller: %v", err)
	}
	defer caller.Release()

	// 绑定 add 函数 - 使用 int32 匹配 C 的 int 类型
	var addFunc func(int32, int32) int32
	err = caller.BindFunction("add", &addFunc)
	if err != nil {
		t.Fatalf("failed to bind add function: %v", err)
	}

	// 测试 add 函数
	result := addFunc(3, 5)
	if result != 8 {
		t.Errorf("add(3, 5) = %d, want 8", result)
	}

	// 绑定 sub 函数 - 使用 int32 匹配 C 的 int 类型
	var subFunc func(int32, int32) int32
	err = caller.BindFunction("sub", &subFunc)
	if err != nil {
		t.Fatalf("failed to bind sub function: %v", err)
	}

	// 测试 sub 函数
	result = subFunc(10, 3)
	if result != 7 {
		t.Errorf("sub(10, 3) = %d, want 7", result)
	}
}

// TestFunctionBinding_ADD 单独测试 add 函数
func TestFunctionBinding_ADD(t *testing.T) {
	caller, err := NewDynlibCaller("./test/libtest.so", []string{"add"})
	if err != nil {
		t.Fatalf("failed to create DynlibCaller: %v", err)
	}
	defer caller.Release()

	// 使用 int32 匹配 C 的 int 类型
	var addFunc func(int32, int32) int32
	err = caller.BindFunction("add", &addFunc)
	if err != nil {
		t.Fatalf("failed to bind add function: %v", err)
	}

	testCases := []struct {
		name     string
		a, b     int32
		expected int32
	}{
		{"正数相加", 1, 2, 3},
		{"零相加", 0, 0, 0},
		{"随机整数相加", 34, 21, 55},
		{"正负数相加", 5, -3, 2},
		{"负数相加", -5, -3, -8},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := addFunc(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("add(%d, %d) = %d, want %d", tc.a, tc.b, result, tc.expected)
			}
		})
	}
}

// TestFunctionBinding_SUB 单独测试 sub 函数
func TestFunctionBinding_SUB(t *testing.T) {
	caller, err := NewDynlibCaller("./test/libtest.so", []string{"sub"})
	if err != nil {
		t.Fatalf("failed to create DynlibCaller: %v", err)
	}
	defer caller.Release()

	// 使用 int32 匹配 C 的 int 类型
	var subFunc func(int32, int32) int32
	err = caller.BindFunction("sub", &subFunc)
	if err != nil {
		t.Fatalf("failed to bind sub function: %v", err)
	}

	testCases := []struct {
		name     string
		a, b     int32
		expected int32
	}{
		{"正数相减", 10, 3, 7},
		{"零相减", 0, 0, 0},
		{"正数减负数", 5, -3, 8},
		{"负数相减", -5, -3, -2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := subFunc(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("sub(%d, %d) = %d, want %d", tc.a, tc.b, result, tc.expected)
			}
		})
	}
}

// 动态库现在跟随程序卸载 无效用例
func TestMultipleCaller(t *testing.T) {
	caller1, err := NewDynlibCaller("./test/libtest.so", []string{"add"})
	if err != nil {
		t.Fatalf("failed to create first DynlibCaller: %v", err)
	}
	defer caller1.Release()

	caller2, err := NewDynlibCaller("./test/libtest.so", []string{"sub"})
	if err != nil {
		t.Fatalf("failed to create second DynlibCaller: %v", err)
	}
	defer caller2.Release()

	// 使用 int32 匹配 C 的 int 类型
	var addFunc func(int32, int32) int32
	var subFunc func(int32, int32) int32

	err = caller1.BindFunction("add", &addFunc)
	if err != nil {
		t.Fatalf("failed to bind add: %v", err)
	}

	err = caller2.BindFunction("sub", &subFunc)
	if err != nil {
		t.Fatalf("failed to bind sub: %v", err)
	}

	// 两个实例应该能独立工作
	result1 := addFunc(2, 3)
	result2 := subFunc(5, 2)

	if result1 != 5 {
		t.Errorf("addFunc(2, 3) = %d, want 5", result1)
	}
	if result2 != 3 {
		t.Errorf("subFunc(5, 2) = %d, want 3", result2)
	}
}

// TestFunctionInjection 测试函数注入功能
func TestFunctionInjection(t *testing.T) {
	caller, err := NewDynlibCaller("./test/libtest.so", []string{"injector_func", "add"})
	if err != nil {
		t.Fatalf("failed to create DynlibCaller: %v", err)
	}
	defer caller.Release()

	// 首先获取 add 函数 - 使用 int32 匹配 C 的 int 类型
	var addFunc func(int32, int32) int32
	err = caller.BindFunction("add", &addFunc)
	if err != nil {
		t.Fatalf("failed to bind add function: %v", err)
	}

	// 测试注入前的 add(3, 5) - 应该是 8
	result1 := addFunc(3, 5)
	if result1 != 8 {
		t.Errorf("before injection: add(3, 5) = %d, want 8", result1)
	}

	// 定义一个返回 10 的回调函数 - 使用 int32 匹配 C 的 int
	callbackFunc := func() int32 {
		return 10
	}

	// 注入回调函数
	err = caller.InjectFunction("injector_func", callbackFunc)
	if err != nil {
		t.Fatalf("failed to inject function: %v", err)
	}

	// 测试注入后的 add(3, 5) - 应该是 18（3 + 5 + 10）
	result2 := addFunc(3, 5)
	if result2 != 18 {
		t.Errorf("after injection: add(3, 5) = %d, want 18", result2)
	}
}
