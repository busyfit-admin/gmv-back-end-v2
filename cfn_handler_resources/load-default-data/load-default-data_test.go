package main

import (
	"bytes"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_handleCreateDDBData(t *testing.T) {
	t.Run("It should Create the Item in DDB table when all input is correctly provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}

		ddbClient := awsclients.MockDynamodbClient{

			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},
		}

		svc := DefaultDataService{
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),
			cfnData: CFNRequestData{
				DDBTableName: "test-table",
				Data:         string(`{"RuleId": "rr-ab23das", "RuleType": "RewardRules"}`),
			},
		}

		err := svc.handleCreateDDBData()

		data := map[string]types.AttributeValue{
			"RuleId":   &types.AttributeValueMemberS{Value: "rr-ab23das"},
			"RuleType": &types.AttributeValueMemberS{Value: "RewardRules"},
		}

		expectedDDBInput := dynamodb.PutItemInput{
			TableName: aws.String("test-table"),
			Item:      data,
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.PutItemInputs[0])
	})
}
