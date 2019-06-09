package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOwnerAndRepository(t *testing.T) {
	tcs := []struct {
		Input         string
		ExpectedOwner string
		ExpectedRepo  string
		ExpecteError  bool
	}{
		{Input: "", ExpectedOwner: "", ExpectedRepo: "", ExpecteError: true},
		{Input: "/", ExpectedOwner: "", ExpectedRepo: "", ExpecteError: true},
		{Input: "/abc", ExpectedOwner: "", ExpectedRepo: "", ExpecteError: true},
		{Input: "abc/def/ghi", ExpectedOwner: "", ExpectedRepo: "", ExpecteError: true},
		{Input: "abc", ExpectedOwner: "abc", ExpectedRepo: "", ExpecteError: false},
		{Input: "abc/", ExpectedOwner: "abc", ExpectedRepo: "", ExpecteError: false},
		{Input: "abc/def", ExpectedOwner: "abc", ExpectedRepo: "def", ExpecteError: false},
	}

	for _, tc := range tcs {
		owner, repo, err := ParseOwnerAndRepository(tc.Input)

		assert.Equal(t, owner, tc.ExpectedOwner)
		assert.Equal(t, repo, tc.ExpectedRepo)
		if tc.ExpecteError {
			assert.Error(t, err, tc.ExpecteError)
		} else {
			assert.NoError(t, err, tc.ExpecteError)
		}
	}
}
