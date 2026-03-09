package rca

import "strconv"

func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
