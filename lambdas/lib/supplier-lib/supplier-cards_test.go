package supplierlib

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_GetAllCards(t *testing.T) {
	t.Run("It should return all cards output when called", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"CardId":     &ddb_types.AttributeValueMemberS{Value: "CardId-1"},
							"IsActive":   &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
							"CardName":   &ddb_types.AttributeValueMemberS{Value: "CardName1"},
							"CardType":   &ddb_types.AttributeValueMemberS{Value: "general"},
							"CardDesc":   &ddb_types.AttributeValueMemberS{Value: "desc"},
							"ExpiryDate": &ddb_types.AttributeValueMemberS{Value: "2010-01-01"},
						},
						{
							"CardId":     &ddb_types.AttributeValueMemberS{Value: "CardId-2"},
							"IsActive":   &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
							"CardName":   &ddb_types.AttributeValueMemberS{Value: "CardName2"},
							"CardType":   &ddb_types.AttributeValueMemberS{Value: "health"},
							"CardDesc":   &ddb_types.AttributeValueMemberS{Value: "desc"},
							"ExpiryDate": &ddb_types.AttributeValueMemberS{Value: "2010-01-01"},
						},
					},
				},
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"CardId":     &ddb_types.AttributeValueMemberS{Value: "CardId-3"},
							"IsActive":   &ddb_types.AttributeValueMemberS{Value: "INACTIVE"},
							"CardName":   &ddb_types.AttributeValueMemberS{Value: "CardName3"},
							"CardType":   &ddb_types.AttributeValueMemberS{Value: "climate rewards"},
							"CardDesc":   &ddb_types.AttributeValueMemberS{Value: "desc"},
							"ExpiryDate": &ddb_types.AttributeValueMemberS{Value: "2010-01-01"},
						},
					},
				},
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"CardId":     &ddb_types.AttributeValueMemberS{Value: "CardId-4"},
							"IsActive":   &ddb_types.AttributeValueMemberS{Value: "EXPIRED"},
							"CardName":   &ddb_types.AttributeValueMemberS{Value: "CardName4"},
							"CardType":   &ddb_types.AttributeValueMemberS{Value: "climate rewards"},
							"CardDesc":   &ddb_types.AttributeValueMemberS{Value: "desc"},
							"ExpiryDate": &ddb_types.AttributeValueMemberS{Value: "2010-01-01"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
				nil,
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}
		svc := SupplierCardsService{
			ctx:                              context.Background(),
			logger:                           log.New(logBuffer, "TEST:", 0),
			dynamodbClient:                   &ddbClient,
			SupplierCardsTable:               "SupplierCardsTable",
			SupplierCardsTable_IsActiveIndex: "IsActive_Index",
		}

		expectedDDBInput1 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT CardId, CardDesc, ExpiryDate, IsActive, CardType, CardName FROM \"SupplierCardsTable\".\"IsActive_Index\" WHERE IsActive = 'ACTIVE' ORDER BY CardType ASC"),
			ConsistentRead: aws.Bool(false),
		}
		expectedDDBInput2 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT CardId, CardDesc, ExpiryDate, IsActive, CardType, CardName FROM \"SupplierCardsTable\".\"IsActive_Index\" WHERE IsActive = 'INACTIVE' ORDER BY CardType ASC"),
			ConsistentRead: aws.Bool(false),
		}
		expectedDDBInput3 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT CardId, CardDesc, ExpiryDate, IsActive, CardType, CardName FROM \"SupplierCardsTable\".\"IsActive_Index\" WHERE IsActive = 'EXPIRED' ORDER BY CardType ASC"),
			ConsistentRead: aws.Bool(false),
		}

		expectedItems := AllCards{
			ActiveCards: []SupplierCards{
				{
					CardId:     "CardId-1",
					CardName:   "CardName1",
					CardType:   "general",
					CardDesc:   "desc",
					ExpiryDate: "2010-01-01",
					IsActive: "ACTIVE",
				},
				{
					CardId:     "CardId-2",
					CardName:   "CardName2",
					CardType:   "health",
					CardDesc:   "desc",
					ExpiryDate: "2010-01-01",
					IsActive: "ACTIVE",
				},
			},
			InactiveCards: []SupplierCards{
				{
					CardId:     "CardId-3",
					CardName:   "CardName3",
					CardType:   "climate rewards",
					CardDesc:   "desc",
					ExpiryDate: "2010-01-01",
					IsActive: "INACTIVE",
				},
			},
			ExpiredCards: []SupplierCards{
				{
					CardId:     "CardId-4",
					CardName:   "CardName4",
					CardType:   "climate rewards",
					CardDesc:   "desc",
					ExpiryDate: "2010-01-01",
					IsActive: "EXPIRED",
				},
			},
		}

		output, err := svc.GetAllCards()

		assert.Equal(t, expectedItems, output)
		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput1, ddbClient.ExecuteStatementInputs[0])
		assert.Equal(t, expectedDDBInput2, ddbClient.ExecuteStatementInputs[1])
		assert.Equal(t, expectedDDBInput3, ddbClient.ExecuteStatementInputs[2])
	})
}

func Test_GetCardDetails(t *testing.T) {
	t.Run("It should return card details for a given cardId", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: map[string]ddb_types.AttributeValue{
						"CardId":     &ddb_types.AttributeValueMemberS{Value: "CardId-1"},
						"CardName":   &ddb_types.AttributeValueMemberS{Value: "CardName1"},
						"CardType":   &ddb_types.AttributeValueMemberS{Value: "general"},
						"CardDesc":   &ddb_types.AttributeValueMemberS{Value: "desc"},
						"ExpiryDate": &ddb_types.AttributeValueMemberS{Value: "2010-01-01"},
						"IsActive": &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
					},
				},
			},
			GetItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}
		svc := SupplierCardsService{
			ctx:                context.Background(),
			logger:             log.New(logBuffer, "TEST:", 0),
			dynamodbClient:     &ddbClient,
			SupplierCardsTable: "SupplierCardsTable",
		}

		expectedGetItemInput := dynamodb.GetItemInput{
			Key: map[string]ddb_types.AttributeValue{
				"CardId": &ddb_types.AttributeValueMemberS{Value: "CardId-1"},
			},
			TableName:      aws.String("SupplierCardsTable"),
			ConsistentRead: aws.Bool(true),
		}

		expectedItem := SupplierCards{
			CardId:     "CardId-1",
			CardName:   "CardName1",
			CardType:   "general",
			CardDesc:   "desc",
			ExpiryDate: "2010-01-01",
			IsActive: "ACTIVE",
		}

		output, err := svc.GetCardDetails("CardId-1")

		assert.Equal(t, expectedItem, output)
		assert.NoError(t, err)
		assert.Equal(t, expectedGetItemInput, ddbClient.GetItemInputs[0])
	})
}

func Test_CreateSupplierCards(t *testing.T) {
	t.Run("It should create a new supplier card", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}
		svc := SupplierCardsService{
			ctx:                context.Background(),
			logger:             log.New(logBuffer, "TEST:", 0),
			dynamodbClient:     &ddbClient,
			SupplierCardsTable: "SupplierCardsTable",
		}

		card := SupplierCards{
			CardId:     "CardId-1",
			CardName:   "CardName1",
			CardType:   "general",
			CardDesc:   "desc",
			ExpiryDate: "2010-01-01",
			IsActive: "ACTIVE",
		}

		err := svc.CreateSupplierCards(card)

		expectedPutItemInput := dynamodb.PutItemInput{
			TableName: aws.String("SupplierCardsTable"),
			Item: map[string]ddb_types.AttributeValue{
				"CardId":     &ddb_types.AttributeValueMemberS{Value: "CardId-1"},
				"CardName":   &ddb_types.AttributeValueMemberS{Value: "CardName1"},
				"CardType":   &ddb_types.AttributeValueMemberS{Value: "general"},
				"CardDesc":   &ddb_types.AttributeValueMemberS{Value: "desc"},
				"ExpiryDate": &ddb_types.AttributeValueMemberS{Value: "2010-01-01"},
				"IsActive": &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
			},
			ConditionExpression: aws.String("attribute_not_exists(CardId)"),
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedPutItemInput, ddbClient.PutItemInputs[0])
	})
}

func Test_UpdateSupplierCard(t *testing.T) {
	t.Run("It should update an existing supplier card", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}
		svc := SupplierCardsService{
			ctx:                context.Background(),
			logger:             log.New(logBuffer, "TEST:", 0),
			dynamodbClient:     &ddbClient,
			SupplierCardsTable: "SupplierCardsTable",
		}

		card := SupplierCards{
			CardId:     "CardId-1",
			CardName:   "CardName1",
			CardType:   "general",
			CardDesc:   "desc",
			ExpiryDate: "2010-01-01",
			IsActive: "ACTIVE",
		}

		err := svc.UpdateSupplierCard(card)

		expectedPutItemInput := dynamodb.PutItemInput{
			TableName: aws.String("SupplierCardsTable"),
			Item: map[string]ddb_types.AttributeValue{
				"CardId":     &ddb_types.AttributeValueMemberS{Value: "CardId-1"},
				"CardName":   &ddb_types.AttributeValueMemberS{Value: "CardName1"},
				"CardType":   &ddb_types.AttributeValueMemberS{Value: "general"},
				"CardDesc":   &ddb_types.AttributeValueMemberS{Value: "desc"},
				"ExpiryDate": &ddb_types.AttributeValueMemberS{Value: "2010-01-01"},
				"IsActive": &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
			},
			ConditionExpression: aws.String("attribute_exists(CardId)"),
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedPutItemInput, ddbClient.PutItemInputs[0])
	})
}

func Test_DeleteSupplierCard(t *testing.T) {
	t.Run("It should delete an existing supplier card", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			DeleteItemOutputs: []dynamodb.DeleteItemOutput{
				{},
			},
			DeleteItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}
		svc := SupplierCardsService{
			ctx:                context.Background(),
			logger:             log.New(logBuffer, "TEST:", 0),
			dynamodbClient:     &ddbClient,
			SupplierCardsTable: "SupplierCardsTable",
		}

		cardId := "CardId-1"

		err := svc.DeleteSupplierCard(cardId)

		expectedDeleteItemInput := dynamodb.DeleteItemInput{
			TableName: aws.String("SupplierCardsTable"),
			Key: map[string]ddb_types.AttributeValue{
				"CardId": &ddb_types.AttributeValueMemberS{Value: cardId},
			},
			ConditionExpression: aws.String("IsActive = :InActive"),
			ExpressionAttributeValues: map[string]ddb_types.AttributeValue{
				"InActive": &ddb_types.AttributeValueMemberS{Value: CARD_ISACTIVE_FALSE},
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedDeleteItemInput, ddbClient.DeleteItemInputs[0])
	})
}
