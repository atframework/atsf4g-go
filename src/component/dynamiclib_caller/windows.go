//go:build windows

package dynamiclib_caller

import "syscall"

func openLibrary(name string) (uintptr, error) {
	// Use [syscall.LoadLibrary] here to avoid external dependencies (#270).
	// For actual use cases, [golang.org/x/sys/windows.NewLazySystemDLL] is recommended.
	handle, err := syscall.LoadLibrary(name)
	return uintptr(handle), err
}

func getSymbolAddress(handle uintptr, symbol string) (uintptr, error) {
	// Windows: use GetProcAddress
	addr, err := syscall.GetProcAddress(syscall.Handle(handle), symbol)
	if err != nil {
		return 0, err
	}
	return uintptr(addr), nil
}

func closeLibrary(handle uintptr) error {
	// Windows: use FreeLibrary
	err := syscall.FreeLibrary(syscall.Handle(handle))
	return err
}
