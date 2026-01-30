package main

import (
	"testing"

	// complib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
	// "github.com/stretchr/testify/assert"
)

// Test case function
func TestCalculateNumBatches(t *testing.T) {
	// Test cases with different input values
	testCases := []struct {
		cardsOrderQuantity int
		cardsPerBatch      int
		expectedNumBatches int
	}{
		{10, 3, 4},    // (10 + 3 - 1) / 3 = 12 / 3 = 4
		{15, 5, 3},    // (15 + 5 - 1) / 5 = 19 / 5 = 3
		{7, 2, 4},     // (7 + 2 - 1) / 2 = 8 / 2 = 4
		{100, 10, 10}, // (100 + 10 - 1) / 10 = 109 / 10 = 10
		{1500, 500, 3},
		{1510, 500, 4},
	}

	// Iterate over test cases
	for _, testCase := range testCases {
		// Call the function to calculate numBatches
		result := calculateNumBatches(testCase.cardsOrderQuantity, testCase.cardsPerBatch)

		// Check if the result matches the expected value
		if result != testCase.expectedNumBatches {
			t.Errorf("For cardsOrderQuantity=%d, cardsPerBatch=%d, got %d, want %d",
				testCase.cardsOrderQuantity, testCase.cardsPerBatch, result, testCase.expectedNumBatches)
		}
	}
}

// func TestGenerateCardCreationBatches(t *testing.T) {

// 	testCase := complib.CardsMetaDataInput{
// 		TrackingId:         "trackingId1",
// 		CardsOrderQuantity: 1510,
// 		CompanyId:          "companyId-1",
// 		CardId:             "23445",
// 	}

// 	cardPrefix := "33300"
// 	cardType := "Default"

// 	result := generateCardCreationBatches(testCase, cardPrefix, cardType, 4)

// 	expectedResult := []complib.CardCreationBatch{
// 		{BatchId: "batch1", NumberOfCards: 500, CardPrefix: "33300", CardId: "23445", TrackingId: "trackingId1", CardType: "Default"},
// 		{BatchId: "batch2", NumberOfCards: 500, CardPrefix: "33300", CardId: "23445", TrackingId: "trackingId1", CardType: "Default"},
// 		{BatchId: "batch3", NumberOfCards: 500, CardPrefix: "33300", CardId: "23445", TrackingId: "trackingId1", CardType: "Default"},
// 		{BatchId: "batch4", NumberOfCards: 10, CardPrefix: "33300", CardId: "23445", TrackingId: "trackingId1", CardType: "Default"},
// 	}

// 	assert.Equal(t, expectedResult, result)
// }
