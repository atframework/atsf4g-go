// Copyright 2025 atframework
package libatframe_utils_lang_utility

func Mutable[T any](v **T) *T {
	if v == nil {
		return nil
	}

	if *v == nil {
		*v = new(T)
	}

	return *v
}
