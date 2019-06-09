package util

import (
	"sort"
	"strings"
)

// Contains returns true if a given slice contains a given element
func Contains(slice []string, element string) bool {
	for _, e := range slice {
		if e == element {
			return true
		}
	}
	return false
}

// SortSlice sorts a given slice
func SortSlice(s []string) {
	sort.Slice(s, func(i, j int) bool {
		return strings.ToLower(s[i]) < strings.ToLower(s[j])
	})
}
