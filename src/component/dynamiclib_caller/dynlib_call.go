package dynamiclib_caller

import (
	"fmt"
	"reflect"

	purego "github.com/ebitengine/purego"
)

// DynlibCaller 用于调用动态库中的函数
// 主要职责：连接 loader 和 function binding
type DynlibCaller struct {
	loader *DynlibLoader
}

// NewDynlibCaller 创建一个新的动态库函数调用器
func NewDynlibCaller(libPath string, funsName []string) (*DynlibCaller, error) {
	loader, err := NewDynlibLoader(libPath, funsName)
	if err != nil {
		return nil, fmt.Errorf("failed to create DynlibLoader: %w", err)
	}

	return &DynlibCaller{
		loader: loader,
	}, nil
}

// BindFunction 绑定动态库中的函数到 Go 函数指针
// dynFunName 是动态库中的函数名
// fn 是指向函数类型的指针，例如：var addFunc func(int, int) int; BindFunction("add", &addFunc)
func (d *DynlibCaller) BindFunction(dynFunName string, fn any) error {
	if d.loader == nil {
		return fmt.Errorf("loader not initialized")
	}

	if dynFunName == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	// 检查 fn 是否是函数类型的指针
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Ptr {
		return fmt.Errorf("fn must be a pointer to a function")
	}

	fnElem := fnValue.Elem()
	if fnElem.Kind() != reflect.Func {
		return fmt.Errorf("fn must be a pointer to a function type, got %v", fnElem.Kind())
	}

	// 验证函数在动态库中存在
	_, err := d.loader.validateLibrary(dynFunName)
	if err != nil {
		return fmt.Errorf("failed to bind function %s: %w", dynFunName, err)
	}

	// 创建绑定结果
	purego.RegisterLibFunc(fn, d.loader.libHandle, dynFunName)

	return nil
}

// InjectFunction 注入 Go 函数作为回调到动态库中
// recvFuncName 是动态库中用于接收回调函数的函数名
// fn 是要注入的 Go 函数
func (d *DynlibCaller) InjectFunction(recvFuncName string, fn any) error {
	if d.loader == nil {
		return fmt.Errorf("loader not initialized")
	}

	if recvFuncName == "" {
		return fmt.Errorf("function name cannot be empty")
	}

	// 验证函数在动态库中存在
	_, err := d.loader.validateLibrary(recvFuncName)
	if err != nil {
		return fmt.Errorf("failed to bind function %s: %w", recvFuncName, err)
	}

	callbackPtr := purego.NewCallback(fn)
	if callbackPtr == 0 {
		return fmt.Errorf("failed to create callback for function")
	}

	ptr, err := d.loader.GetFunctionPointer(recvFuncName)
	if err != nil {
		return fmt.Errorf("failed to get function pointer for %s: %w", recvFuncName, err)
	}

	_, _, errno := purego.SyscallN(ptr, uintptr(callbackPtr))
	if errno != 0 {
		return fmt.Errorf("failed to inject function %s: errno %d", recvFuncName, errno)
	}

	return nil
}

func (d *DynlibCaller) Release() error {
	return d.loader.Release()
}
