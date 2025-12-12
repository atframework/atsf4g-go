// Copyright 2025 atframework
package libatframe_utils_lang_utility

import (
	"unsafe"
)

// //////////////// 转换产物不可写 ////////////////////////
func StringtoBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
func BytestoString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
