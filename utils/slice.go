package utils

func InSlice[T comparable](item T, slice []T) bool {
	for _, val := range slice {
		if item == val {
			return true
		}
	}

	return false
}
