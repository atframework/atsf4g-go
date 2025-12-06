package libatframe_utils_lang_utility

func IsDeduplicate[T comparable](elem []T) bool {
	seen := make(map[T]struct{})
	for _, v := range elem {
		if _, ok := seen[v]; ok {
			return true
		}
		seen[v] = struct{}{}
	}
	return false
}

func IsExist[T comparable](elem []T, target T) bool {
	for _, v := range elem {
		if v == target {
			return true
		}
	}
	return false
}
