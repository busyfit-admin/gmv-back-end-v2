package utils

import "time"

func GenerateTimestamp() string {
	// Get the current time in UTC
	currentTime := time.Now().UTC()

	// Format the time according to the desired layout
	timestamp := currentTime.Format("2006-01-02T15:04:05.000Z")

	return timestamp
}

func GenerateDate() string {
	// Get the current time in UTC
	currentTime := time.Now().UTC()

	// Format the time according to the desired layout
	timestamp := currentTime.Format("2006-01-02")

	return timestamp
}
