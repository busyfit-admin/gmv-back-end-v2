package utils

import "strings"

// ConvertToArray takes a string with a specific delimiter and returns an array of strings.
func ConvertToArray(input string, delimiter string) []string {
	// Use strings.Split to break the input string into an array.
	return strings.Split(input, delimiter)
}
