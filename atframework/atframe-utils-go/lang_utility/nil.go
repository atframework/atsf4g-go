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

func Compare(l interface{}, r interface{}) bool {
	if l == r {
		return true
	}

	leftNil := IsNil(l)
	rightNil := IsNil(r)
	if leftNil || rightNil {
		return leftNil && rightNil
	}

	lv := unwrapInterface(reflect.ValueOf(l))
	rv := unwrapInterface(reflect.ValueOf(r))
	if !lv.IsValid() || !rv.IsValid() {
		return lv.IsValid() == rv.IsValid()
	}

	if lv.Kind() != rv.Kind() {
		return false
	}

	switch lv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Chan, reflect.Func, reflect.Slice, reflect.UnsafePointer:
		if lv.IsNil() || rv.IsNil() {
			return lv.IsNil() && rv.IsNil()
		}
		return lv.Pointer() == rv.Pointer()
	default:
		if lv.Type().Comparable() {
			return lv.Interface() == rv.Interface()
		}
		return reflect.DeepEqual(lv.Interface(), rv.Interface())
	}
}

func unwrapInterface(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}
