package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortSlice(t *testing.T) {
	s := []string{"z", "carlos1", "a", "Carlos2", "cArlOs3"}
	SortSlice(s)
	assert.Equal(t, []string{"a", "carlos1", "Carlos2", "cArlOs3", "z"}, s)
}

func TestContains(t *testing.T) {
	tcs := []struct {
		Slice   []string
		Element string
		Found   bool
	}{
		{Slice: []string{}, Element: "", Found: false},
		{Slice: []string{""}, Element: "", Found: true},
		{Slice: []string{"abc"}, Element: "abc", Found: true},
		{Slice: []string{"abc", "def"}, Element: "def", Found: true},
	}

	for _, tc := range tcs {
		found := Contains(tc.Slice, tc.Element)

		assert.Equal(t, found, tc.Found)
	}
}
