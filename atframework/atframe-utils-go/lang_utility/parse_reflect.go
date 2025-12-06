package libatframe_utils_lang_utility

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// Assign 根据参数类型字符串将值转换为对应的reflect.Value.
func assignBasicValue(pt reflect.Type, p string) (reflect.Value, error) {
	var nilValue = reflect.Value{}
	switch pt.Kind() { //nolint:exhaustive
	case reflect.Bool:
		// Only accept explicit bool strings: true, false (case-insensitive: true, True, TRUE, false, False, FALSE)
		// Reject numeric 0, 1 and short forms (t, f, T, F)
		switch p {
		case "true", "True", "TRUE":
			pv := reflect.New(pt).Elem()
			pv.SetBool(true)
			return pv, nil
		case "false", "False", "FALSE":
			pv := reflect.New(pt).Elem()
			pv.SetBool(false)
			return pv, nil
		default:
			return nilValue, fmt.Errorf("parse bool failed: invalid bool string '%s'", p)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		a, err := strconv.ParseInt(p, 0, 64)
		if err != nil {
			return nilValue, fmt.Errorf("parse int failed: %w", err)
		}
		pv := reflect.New(pt).Elem()
		pv.SetInt(a)
		return pv, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		a, err := strconv.ParseUint(p, 0, 64)
		if err != nil {
			return nilValue, fmt.Errorf("parse uint failed: %w", err)
		}
		pv := reflect.New(pt).Elem()
		pv.SetUint(a)
		return pv, nil
	case reflect.Float32, reflect.Float64:
		a, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nilValue, fmt.Errorf("parse float failed: %w", err)
		}
		pv := reflect.New(pt).Elem()
		pv.SetFloat(a)
		return pv, nil
	case reflect.String:
		pv := reflect.New(pt).Elem()
		pv.SetString(p)
		return pv, nil
	default:
		return nilValue, fmt.Errorf("unsupported basic type: %s", pt.Kind())
	}
}

func assignSliceValue(pt reflect.Type, p string) (reflect.Value, error) {
	var nilValue = reflect.Value{}
	var tmp []interface{}
	if err := json.Unmarshal([]byte(p), &tmp); err != nil {
		return nilValue, fmt.Errorf("parse slice failed: %w", err)
	}
	pv := reflect.MakeSlice(pt, len(tmp), len(tmp))
	for idx, v := range tmp {
		elemBuf, err := json.Marshal(v)
		if err != nil {
			return nilValue, fmt.Errorf("marshal slice elem failed: %w", err)
		}
		kv, err := AssignValue(pt.Elem(), string(elemBuf))
		if err != nil {
			return nilValue, fmt.Errorf("assign slice elem failed: %w", err)
		}
		pv.Index(idx).Set(kv)
	}
	return pv, nil
}

func assignArrayValue(pt reflect.Type, p string) (reflect.Value, error) {
	var nilValue = reflect.Value{}
	var tmp []interface{}
	if err := json.Unmarshal([]byte(p), &tmp); err != nil {
		return nilValue, fmt.Errorf("parse array failed: %w", err)
	}
	arrayType := reflect.ArrayOf(len(tmp), pt.Elem())
	pv := reflect.New(arrayType).Elem()
	for idx, v := range tmp {
		kv, err := AssignValue(pt.Elem(), fmt.Sprintf("%v", v))
		if err != nil {
			return nilValue, fmt.Errorf("assign array elem failed: %w", err)
		}
		pv.Index(idx).Set(kv)
	}
	return pv, nil
}

func assignMapValue(pt reflect.Type, p string) (reflect.Value, error) {
	var nilValue = reflect.Value{}
	tmp := make(map[string]interface{})
	if err := json.Unmarshal([]byte(p), &tmp); err != nil {
		return nilValue, fmt.Errorf("parse map failed: %w", err)
	}
	pv := reflect.MakeMap(pt)
	for k, v := range tmp {
		kk, err := AssignValue(pt.Key(), k)
		if err != nil {
			return nilValue, fmt.Errorf("assign map key failed: %w", err)
		}
		kv, err := AssignValue(pt.Elem(), fmt.Sprintf("%v", v))
		if err != nil {
			return nilValue, fmt.Errorf("assign map value failed: %w", err)
		}
		pv.SetMapIndex(kk, kv)
	}
	return pv, nil
}

func assignStructValue(pt reflect.Type, p string) (reflect.Value, error) {
	var nilValue = reflect.Value{}
	pv := reflect.New(pt)
	if err := json.Unmarshal([]byte(p), pv.Interface()); err != nil {
		return nilValue, fmt.Errorf("parse struct failed: %w", err)
	}
	return pv.Elem(), nil
}

func assignPtrValue(pt reflect.Type, p string) (reflect.Value, error) {
	var nilValue = reflect.Value{}

	// Handle null/nil pointer
	if p == "nil" {
		return reflect.Zero(pt), nil // Return nil pointer
	}

	pv, err := AssignValue(pt.Elem(), p)
	if err != nil {
		return nilValue, fmt.Errorf("assign ptr failed: %w", err)
	}
	ptr := reflect.New(pt.Elem())
	ptr.Elem().Set(pv)
	return ptr, nil
}

// 不支持 Map of Structs ，Nested Maps.
func AssignValue(pt reflect.Type, p string) (reflect.Value, error) {
	switch pt.Kind() { //nolint:exhaustive
	case reflect.Slice:
		return assignSliceValue(pt, p)
	case reflect.Array:
		return assignArrayValue(pt, p)
	case reflect.Map:
		return assignMapValue(pt, p)
	case reflect.Struct:
		return assignStructValue(pt, p)
	case reflect.Ptr:
		return assignPtrValue(pt, p)
	default:
		return assignBasicValue(pt, p)
	}
}

func FormatValues(values []reflect.Value) []string {
	result := make([]string, 0)

	for _, rv := range values {
		switch rv.Kind() { //nolint:exhaustive
		case reflect.String:
			result = append(result, rv.String())
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			result = append(result, fmt.Sprintf("%d", rv.Int()))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			result = append(result, fmt.Sprintf("%d", rv.Uint()))
		case reflect.Float32, reflect.Float64:
			result = append(result, fmt.Sprintf("%f", rv.Float()))
		case reflect.Bool:
			result = append(result, fmt.Sprintf("%v", rv.Bool()))
		case reflect.Interface:
			// 处理 interface{} 类型，例如 error
			if !rv.IsNil() {
				if err, ok := rv.Interface().(error); ok {
					result = append(result, fmt.Sprintf("error: %v", err))
				} else {
					result = append(result, fmt.Sprintf("%v", rv.Interface()))
				}
			}
		default:
			result = append(result, fmt.Sprintf("%v", rv.Interface()))
		}
	}

	return result
}
