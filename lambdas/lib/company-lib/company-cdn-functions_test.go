package Companylib

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateDomainURL(t *testing.T) {
	t.Run("It should generate the correct domain URL", func(t *testing.T) {
		domain := "example.com"
		key := "image.jpg"
		expectedURL := "https://example.com/image.jpg"

		url, err := GenerateDomainURL(domain, key)

		assert.NoError(t, err)
		assert.Equal(t, expectedURL, url)
	})

	t.Run("It should return an error if domain or key is empty", func(t *testing.T) {
		domain := ""
		key := "image.jpg"
		expectedError := fmt.Errorf("domain and key must be provided")

		url, err := GenerateDomainURL(domain, key)

		assert.EqualError(t, err, expectedError.Error())
		assert.Empty(t, url)

		domain = "example.com"
		key = ""
		expectedError = fmt.Errorf("domain and key must be provided")

		url, err = GenerateDomainURL(domain, key)

		assert.EqualError(t, err, expectedError.Error())
		assert.Empty(t, url)
	})
}
