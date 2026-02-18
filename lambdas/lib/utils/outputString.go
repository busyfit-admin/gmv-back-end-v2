package utils

import (
	"encoding/json"
	"log"
)

func LogAsJSON(logger *log.Logger, label string, v interface{}) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		logger.Printf("%s: <failed to marshal: %v>", label, err)
		return
	}
	logger.Printf("%s:\n%s", label, string(b))
}
