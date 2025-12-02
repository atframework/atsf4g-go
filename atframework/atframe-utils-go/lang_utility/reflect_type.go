package libatframe_utils_lang_utility

import "reflect"

type ReflectType interface {
	GetReflectType() reflect.Type
}

func GetReflectType[T ReflectType]() reflect.Type {
	var zero T
	return zero.GetReflectType()
}

func GetStaticReflectType[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}
