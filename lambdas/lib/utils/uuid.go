package utils

import (
	"crypto/rand"
	"math/big"
)

// Function to generate a random alphanumeric string of a specified length
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var result string
	for i := 0; i < length; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result += string(charset[idx.Int64()])
	}
	return result
}

func GenerateRandomNumber(length int) string {
	const charset = "0123456789"
	var result string
	for i := 0; i < length; i++ {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result += string(charset[idx.Int64()])
	}
	return result
}
