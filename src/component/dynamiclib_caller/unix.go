//go:build darwin || freebsd || linux || netbsd
package dynamiclib_caller

import (
	purego "github.com/ebitengine/purego"
)

func openLibrary(name string) (uintptr, error) {
	// Linux: use purego.Dlopen
	handle, err := purego.Dlopen(name, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	return handle, err
}

func getSymbolAddress(handle uintptr, symbol string) (uintptr, error) {
	// Linux: use purego.Dlsym
	addr, err := purego.Dlsym(handle, symbol)
	return addr, err
}

func closeLibrary(handle uintptr) error {
	// Linux: use purego.Dlclose
	err := purego.Dlclose(handle)
	return err
}
