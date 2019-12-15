package utils

// check if string in slice
func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// check if two slice equal
func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	// 忽略顺序
	for _, v := range a {
		if !Contains(b, v) {
			return false
		}
	}

	return true
}

func BoolToInt(v bool) (n int) {
	n = 0
	if v {
		n = 1
	}
	return n
}
