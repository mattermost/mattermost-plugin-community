package util

import (
	"errors"
	"strings"
)

// ParseOwnerAndRepository parses a combination of owner and repository into the components
func ParseOwnerAndRepository(input string) (string, string, error) {
	splitStr := strings.Split(input, "/")

	if splitStr[0] == "" || len(splitStr) > 2 {
		return "", "", errors.New("invalid input")
	}

	if len(splitStr) == 1 {
		return splitStr[0], "", nil
	}

	return splitStr[0], splitStr[1], nil
}
