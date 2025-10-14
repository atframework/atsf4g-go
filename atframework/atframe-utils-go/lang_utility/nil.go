// Copyright 2025 atframework
package libatframe_utils_lang_utility

import "reflect"

func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}
