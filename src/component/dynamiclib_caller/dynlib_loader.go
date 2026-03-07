package dynamiclib_caller

import (
	"fmt"
	"os"
)

// DynlibLoader 用于加载动态库文件的组件
type DynlibLoader struct {
	libPath   string
	libHandle uintptr
	fnCache   map[string]uintptr
}

// NewDynlibLoader 创建一个新的动态库加载器
func NewDynlibLoader(libPath string, requiredFuncs []string) (*DynlibLoader, error) {
	if _, err := os.Stat(libPath); err != nil {
		return nil, fmt.Errorf("library path not found: %s, error: %w", libPath, err)
	}

	handle, err := openLibrary(libPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open library: %s - %w", libPath, err)
	}

	loader := &DynlibLoader{
		libPath:   libPath,
		libHandle: handle,
		fnCache:   make(map[string]uintptr),
	}

	for _, funcName := range requiredFuncs {
		fnPtr, err := loader.validateLibrary(funcName)
		if err != nil {
			return nil, err
		}
		loader.fnCache[funcName] = fnPtr
	}

	return loader, nil
}

func (d *DynlibLoader) validateLibrary(requiredFunc string) (uintptr, error) {
	fnPtr, err := getSymbolAddress(d.libHandle, requiredFunc)
	if err != nil {
		return 0, fmt.Errorf("function %s not found in library %s: %w", requiredFunc, d.libPath, err)
	}
	return fnPtr, nil
}

func (d *DynlibLoader) GetFunctionPointer(funcName string) (uintptr, error) {
	if ptr, exists := d.fnCache[funcName]; exists {
		return ptr, nil
	}
	return 0, fmt.Errorf("function %s not found in cache", funcName)
}

func (d *DynlibLoader) Release() error {
	return closeLibrary(d.libHandle)
}
