package adminlib

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
	"github.com/stretchr/testify/assert"
)

func Test_GetAllTenantStages(t *testing.T) {
	t.Run("It should return all the stages data when Data is found in DDB Table", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 2,
					Items: []map[string]types.AttributeValue{
						{
							"TenantId":      &types.AttributeValueMemberS{Value: "TenantId-1"},
							"StageId":       &types.AttributeValueMemberS{Value: "STG01"},
							"StageStatus":   &types.AttributeValueMemberS{Value: "COMPLETED"},
							"CommentsCount": &types.AttributeValueMemberN{Value: "2"},
							"StageComments": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Comment":         &types.AttributeValueMemberS{Value: "This is a Test Comment"},
										"CommentBy":       &types.AttributeValueMemberS{Value: "Person1"},
										"UpdateTimeStamp": &types.AttributeValueMemberS{Value: "2024-04-01"},
									},
								},
								&types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Comment":         &types.AttributeValueMemberS{Value: "This is a Test Comment2"},
										"CommentBy":       &types.AttributeValueMemberS{Value: "Person2"},
										"UpdateTimeStamp": &types.AttributeValueMemberS{Value: "2024-04-02"},
									},
								},
							}},
						},
						{
							"TenantId":      &types.AttributeValueMemberS{Value: "TenantId-1"},
							"StageId":       &types.AttributeValueMemberS{Value: "STG03"},
							"StageStatus":   &types.AttributeValueMemberS{Value: "INPROG"},
							"CommentsCount": &types.AttributeValueMemberN{Value: "2"},
							"StageComments": &types.AttributeValueMemberL{Value: []types.AttributeValue{
								&types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Comment":         &types.AttributeValueMemberS{Value: "This is a Test Comment"},
										"CommentBy":       &types.AttributeValueMemberS{Value: "Person1"},
										"UpdateTimeStamp": &types.AttributeValueMemberS{Value: "2024-04-01"},
									},
								},
								&types.AttributeValueMemberM{
									Value: map[string]types.AttributeValue{
										"Comment":         &types.AttributeValueMemberS{Value: "This is a Test Comment2"},
										"CommentBy":       &types.AttributeValueMemberS{Value: "Person2"},
										"UpdateTimeStamp": &types.AttributeValueMemberS{Value: "2024-04-02"},
									},
								},
							}},
						},
					},
				},
			},
			QueryErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}
		tenantSVC := TenantStageService{
			ctx:            context.TODO(),
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),

			TenantStagesTable:          "TestStageTable",
			TenantStages_TenantIdIndex: "TestStageTable_Index",
		}

		expectedOuptput := []TenantStages{
			{
				TenantId:      "TenantId-1",
				StageId:       "STG01",
				StageStatus:   "COMPLETED",
				CommentsCount: 2,
				StageComments: []CommentData{
					{
						Comment:         "This is a Test Comment",
						CommentBy:       "Person1",
						UpdateTimeStamp: "2024-04-01",
					},
					{
						Comment:         "This is a Test Comment2",
						CommentBy:       "Person2",
						UpdateTimeStamp: "2024-04-02",
					},
				},
			},
			{
				TenantId:      "TenantId-1",
				StageId:       "STG03",
				StageStatus:   "INPROG",
				CommentsCount: 2,
				StageComments: []CommentData{
					{
						Comment:         "This is a Test Comment",
						CommentBy:       "Person1",
						UpdateTimeStamp: "2024-04-01",
					},
					{
						Comment:         "This is a Test Comment2",
						CommentBy:       "Person2",
						UpdateTimeStamp: "2024-04-02",
					},
				},
			},
		}

		output, err := tenantSVC.GetAllTenantStages("TenantId-1")

		assert.NoError(t, err)
		assert.Equal(t, expectedOuptput, output)

	})
	t.Run("It should return an empty stage data with nil error if no data is found", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
					Items: []map[string]types.AttributeValue{},
				},
			},
			QueryErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}
		tenantSVC := TenantStageService{
			ctx:            context.TODO(),
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),

			TenantStagesTable:          "TestStageTable",
			TenantStages_TenantIdIndex: "TestStageTable_Index",
		}

		expectedOutput := []TenantStages{}

		output, err := tenantSVC.GetAllTenantStages("TenantId-1")

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)

	})

	t.Run("It should return an error when DDB Query fails", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{},
			},
			QueryErrors: []error{
				fmt.Errorf("Failed to query DDB"),
			},
		}
		logBuffer := &bytes.Buffer{}
		tenantSVC := TenantStageService{
			ctx:            context.TODO(),
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),

			TenantStagesTable:          "TestStageTable",
			TenantStages_TenantIdIndex: "TestStageTable_Index",
		}

		expectedOuptput := []TenantStages{}

		output, err := tenantSVC.GetAllTenantStages("TenantId-1")

		assert.Equal(t, fmt.Errorf("Failed to query DDB"), err)
		assert.Equal(t, expectedOuptput, output)

	})
}

func Test_AddNewStageData(t *testing.T) {
	t.Run("It should add new data to the Dynamodb table when the data is correctly sent", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}
		tenantSVC := TenantStageService{
			ctx:            context.TODO(),
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),

			TenantStagesTable:          "TestStageTable",
			TenantStages_TenantIdIndex: "TestStageTable_Index",
		}

		err := tenantSVC.AddNewStageData(PostReqNewStageData{
			TenantId:    "test-tenantId-1",
			StageId:     "STG01",
			StageStatus: "Assigned",
			Comment:     "this is test comment",
			CommentBy:   "Person1",
		})

		timeNow := utils.GenerateTimestamp()

		expectedDDBInput := []dynamodb.UpdateItemInput{
			{
				Key: map[string]dynamodb_types.AttributeValue{
					"StageId": &dynamodb_types.AttributeValueMemberS{
						Value: "STG01",
					},
					"TenantId": &dynamodb_types.AttributeValueMemberS{
						Value: "test-tenantId-1",
					},
				},
				TableName:        aws.String("TestStageTable"),
				UpdateExpression: aws.String("ADD CommentsCount :One SET StageComments = list_append(if_not_exists(StageComments, :EmptyList), :Comment), StageStatus = :StageStatus, LastModifiedDate = :LastModifiedDate"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":Comment": &dynamodb_types.AttributeValueMemberL{
						Value: []dynamodb_types.AttributeValue{
							&dynamodb_types.AttributeValueMemberM{
								Value: map[string]dynamodb_types.AttributeValue{
									"Comment": &dynamodb_types.AttributeValueMemberS{
										Value: "this is test comment",
									},
									"CommentBy": &dynamodb_types.AttributeValueMemberS{
										Value: "Person1",
									},
									"UpdateTimeStamp": &dynamodb_types.AttributeValueMemberS{
										Value: timeNow,
									},
								},
							},
						},
					},
					":EmptyList": &dynamodb_types.AttributeValueMemberL{},
					":One":       &dynamodb_types.AttributeValueMemberN{Value: "1"},
					":LastModifiedDate": &dynamodb_types.AttributeValueMemberS{
						Value: timeNow,
					},
					":StageStatus": &dynamodb_types.AttributeValueMemberS{Value: "Assigned"},
				},
				ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs)
	})

	t.Run("It should send an error if ddb failed to update", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				fmt.Errorf("internal ddb error"),
			},
		}
		logBuffer := &bytes.Buffer{}
		tenantSVC := TenantStageService{
			ctx:            context.TODO(),
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),

			TenantStagesTable:          "TestStageTable",
			TenantStages_TenantIdIndex: "TestStageTable_Index",
		}

		err := tenantSVC.AddNewStageData(PostReqNewStageData{
			TenantId:    "test-tenantId-1",
			StageId:     "STG01",
			StageStatus: "Assigned",
			Comment:     "this is test comment",
			CommentBy:   "Person1",
		})

		assert.Equal(t, fmt.Errorf("internal ddb error"), err)

	})

}
