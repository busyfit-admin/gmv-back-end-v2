package Companylib

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_PerformRewardsTransfer(t *testing.T) {

	t.Run("It should perform the transaction when the input is provided correctly", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			TransactWriteItemsOutput: []dynamodb.TransactWriteItemsOutput{
				{},
			},
			TransactWriteItemsErrors: []error{
				nil,
			},
		}

		svc := RewardsTransferService{
			ctx:            context.TODO(),
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),

			RewardsTransferLogsTable: "Test-rewardTransferLogs",
			EmployeeTable:            "test-employee-table",
		}

		err := svc.PerformRewardsTransfer(RewardsTransferInput{
			TxId:                "test-id",
			SourceUserName:      "source-username-1",
			DestinationUserName: "dest-username-1",
			TransferPoints:      10,
			RewardType:          "RD00",
		})

		expectedDDBInput := dynamodb.TransactWriteItemsInput{
			TransactItems: []dynamodb_types.TransactWriteItem{
				{
					Update: &dynamodb_types.Update{
						TableName: aws.String("test-employee-table"),
						Key: map[string]dynamodb_types.AttributeValue{
							"UserName": &dynamodb_types.AttributeValueMemberS{Value: "source-username-1"},
						},
						ConditionExpression: aws.String("RewardsData.#REWID.RewardPoints > :TXPOINTS"),
						UpdateExpression:    aws.String("SET RewardsData.#REWID.RewardPoints = RewardsData.#REWID.RewardPoints - :TXPOINTS"),
						ExpressionAttributeNames: map[string]string{
							"#REWID": *aws.String("RD00"),
						},
						ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
							":TXPOINTS": &dynamodb_types.AttributeValueMemberN{Value: "10"},
						},
					},
				},
				{
					Update: &dynamodb_types.Update{
						TableName: aws.String("test-employee-table"),
						Key: map[string]dynamodb_types.AttributeValue{
							"UserName": &dynamodb_types.AttributeValueMemberS{Value: "dest-username-1"},
						},
						UpdateExpression: aws.String("SET RewardsData.#REWID.RewardPoints = RewardsData.#REWID.RewardPoints + :TXPOINTS"),
						ExpressionAttributeNames: map[string]string{
							"#REWID": *aws.String("RD00"),
						},
						ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
							":TXPOINTS": &dynamodb_types.AttributeValueMemberN{Value: "10"},
						},
					},
				},
			},
			ClientRequestToken: aws.String("test-id"),
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.TransactWriteItemsInputs[0])
	})
}
