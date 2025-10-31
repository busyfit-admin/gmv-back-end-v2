package Companylib

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

func Test_PutSkillsToDynamoDB(t *testing.T) {
	t.Run("It should perform batch put operation for skills", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			BatchWriteItemOutputs: []dynamodb.BatchWriteItemOutput{
				{},
			},
			BatchErrors: []error{
				nil,
			},
		}

		svc := TenantEngagementService{
			ctx:               context.TODO(),
			dynamodbClient:    &ddbClient,
			logger:            log.New(logBuffer, "TEST:", 0),
			TenantSkillsTable: "TenantSkillsTable-test",
		}

		skills := []TenantSkillsTable{
			{SkillId: "SKILL-001", SkillName: "Go", SkillDesc: "Programming Language"},
			{SkillId: "SKILL-002", SkillName: "AWS", SkillDesc: "Cloud Platform"},
		}

		err := svc.PutSkillsToDynamoDB(skills, "POST")
		assert.NoError(t, err)

		// Assertion for the PutItems input
		expectedWriteRequests := []types.WriteRequest{
			{
				PutRequest: &types.PutRequest{
					Item: map[string]types.AttributeValue{
						"SkillId":   &types.AttributeValueMemberS{Value: "SKILL-001"},
						"SkillName": &types.AttributeValueMemberS{Value: "Go"},
						"SkillDesc": &types.AttributeValueMemberS{Value: "Programming Language"},
					},
				},
			},
			{
				PutRequest: &types.PutRequest{
					Item: map[string]types.AttributeValue{
						"SkillId":   &types.AttributeValueMemberS{Value: "SKILL-002"},
						"SkillName": &types.AttributeValueMemberS{Value: "AWS"},
						"SkillDesc": &types.AttributeValueMemberS{Value: "Cloud Platform"},
					},
				},
			},
		}

		assert.Equal(t, expectedWriteRequests, ddbClient.BatchWriteItemsInputs[0].RequestItems["TenantSkillsTable-test"])

	})

	t.Run("It should perform batch delete operation for skills", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			BatchWriteItemOutputs: []dynamodb.BatchWriteItemOutput{
				{},
			},
			BatchErrors: []error{
				nil,
			},
		}

		svc := TenantEngagementService{
			ctx:               context.TODO(),
			dynamodbClient:    &ddbClient,
			logger:            log.New(logBuffer, "TEST:", 0),
			TenantSkillsTable: "TenantSkillsTable-test",
		}

		skills := []TenantSkillsTable{
			{SkillId: "SKILL-001"},
			{SkillId: "SKILL-002"},
		}

		err := svc.PutSkillsToDynamoDB(skills, "DELETE")
		assert.NoError(t, err)

		// Assertion for the PutItems input
		expectedWriteRequests := []types.WriteRequest{
			{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"SkillId": &types.AttributeValueMemberS{Value: "SKILL-001"},
					},
				},
			},
			{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"SkillId": &types.AttributeValueMemberS{Value: "SKILL-002"},
					},
				},
			},
		}

		assert.Equal(t, expectedWriteRequests, ddbClient.BatchWriteItemsInputs[0].RequestItems["TenantSkillsTable-test"])

	})

	t.Run("It should return an error for invalid ReqType", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{}

		svc := TenantEngagementService{
			ctx:               context.TODO(),
			dynamodbClient:    &ddbClient,
			logger:            log.New(logBuffer, "TEST:", 0),
			TenantSkillsTable: "TenantSkillsTable-test",
		}

		skills := []TenantSkillsTable{
			{SkillId: "SKILL-001", SkillName: "Go", SkillDesc: "Programming Language"},
		}

		err := svc.PutSkillsToDynamoDB(skills, "INVALID")
		assert.Error(t, err)
		assert.Equal(t, "ReqType Not found", err.Error())
	})
}

func Test_PutValuesToDynamoDB(t *testing.T) {
	t.Run("It should perform batch put operation for values", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			BatchWriteItemOutputs: []dynamodb.BatchWriteItemOutput{
				{},
			},
			BatchErrors: []error{
				nil,
			},
		}

		svc := TenantEngagementService{
			ctx:               context.TODO(),
			dynamodbClient:    &ddbClient,
			logger:            log.New(logBuffer, "TEST:", 0),
			TenantValuesTable: "TenantValuesTable-test",
		}

		values := []TenantValuesTable{
			{ValueId: "VALUE-001", ValueName: "Integrity", ValueDesc: "Doing the right thing"},
			{ValueId: "VALUE-002", ValueName: "Excellence", ValueDesc: "Striving for the best"},
		}

		err := svc.PutValuesToDynamoDB(values, "POST")
		assert.NoError(t, err)

		// Assertion for the PutItems input
		expectedWriteRequests := []types.WriteRequest{
			{
				PutRequest: &types.PutRequest{
					Item: map[string]types.AttributeValue{
						"ValueId":   &types.AttributeValueMemberS{Value: "VALUE-001"},
						"ValueName": &types.AttributeValueMemberS{Value: "Integrity"},
						"ValueDesc": &types.AttributeValueMemberS{Value: "Doing the right thing"},
					},
				},
			},
			{
				PutRequest: &types.PutRequest{
					Item: map[string]types.AttributeValue{
						"ValueId":   &types.AttributeValueMemberS{Value: "VALUE-002"},
						"ValueName": &types.AttributeValueMemberS{Value: "Excellence"},
						"ValueDesc": &types.AttributeValueMemberS{Value: "Striving for the best"},
					},
				},
			},
		}

		assert.Equal(t, expectedWriteRequests, ddbClient.BatchWriteItemsInputs[0].RequestItems["TenantValuesTable-test"])

	})

	t.Run("It should perform batch delete operation for values", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			BatchWriteItemOutputs: []dynamodb.BatchWriteItemOutput{
				{},
			},
			BatchErrors: []error{
				nil,
			},
		}

		svc := TenantEngagementService{
			ctx:               context.TODO(),
			dynamodbClient:    &ddbClient,
			logger:            log.New(logBuffer, "TEST:", 0),
			TenantValuesTable: "TenantValuesTable-test",
		}

		values := []TenantValuesTable{
			{ValueId: "VALUE-001"},
			{ValueId: "VALUE-002"},
		}

		err := svc.PutValuesToDynamoDB(values, "DELETE")
		assert.NoError(t, err)

		// Assertion for the PutItems input
		expectedWriteRequests := []types.WriteRequest{
			{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"ValueId": &types.AttributeValueMemberS{Value: "VALUE-001"},
					},
				},
			},
			{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"ValueId": &types.AttributeValueMemberS{Value: "VALUE-002"},
					},
				},
			},
		}

		assert.Equal(t, expectedWriteRequests, ddbClient.BatchWriteItemsInputs[0].RequestItems["TenantValuesTable-test"])

	})

	t.Run("It should return an error for invalid ReqType", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{}

		svc := TenantEngagementService{
			ctx:               context.TODO(),
			dynamodbClient:    &ddbClient,
			logger:            log.New(logBuffer, "TEST:", 0),
			TenantValuesTable: "TenantValuesTable-test",
		}

		values := []TenantValuesTable{
			{ValueId: "VALUE-001", ValueName: "Integrity", ValueDesc: "Doing the right thing"},
		}

		err := svc.PutValuesToDynamoDB(values, "INVALID")
		assert.Error(t, err)
		assert.Equal(t, "ReqType Not found", err.Error())
	})
}

func Test_CreateEngagement(t *testing.T) {
	t.Run("It should create an appreciation", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},
		}

		svc := TenantEngagementService{
			ctx:                   context.TODO(),
			dynamodbClient:        &ddbClient,
			logger:                log.New(logBuffer, "TEST:", 0),
			TenantEngagementTable: "TenantEngagementTable-test",
		}

		appreciation := TenantEngagementTable{
			EntityId:   "USER-123",
			ProvidedBy: "USER-456",
			Message:    "Great work on the project!",
			Skill:      []string{"Go"},
			Value:      []string{"Integrity"},
			Timestamp:  "2024-08-27T12:34:56Z",
		}

		err := svc.CreateEngagement(appreciation)
		assert.NoError(t, err)
	})
}

func Test_GetAllSkills(t *testing.T) {
	t.Run("It should get all skills", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ScanOutputs: []dynamodb.ScanOutput{
				{
					Items: []map[string]types.AttributeValue{
						{
							"SkillName": &types.AttributeValueMemberS{Value: "Go"},
						},
						{
							"SkillName": &types.AttributeValueMemberS{Value: "AWS"},
						},
					},
					Count: 2,
				},
			},
			ScanErrors: []error{
				nil,
			},
		}

		svc := TenantEngagementService{
			ctx:               context.TODO(),
			dynamodbClient:    &ddbClient,
			logger:            log.New(logBuffer, "TEST:", 0),
			TenantSkillsTable: "TenantSkillsTable-test",
		}

		skills, err := svc.GetAllSkills()
		assert.NoError(t, err)
		assert.ElementsMatch(t, []AppreciationObject{
			{
				ObjectType: "Skill",
				Value:      "Go",
			},
			{
				ObjectType: "Skill",
				Value:      "AWS",
			},
		}, skills)
	})
}

func Test_GetAllValues(t *testing.T) {
	t.Run("It should get all values", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ScanOutputs: []dynamodb.ScanOutput{
				{
					Items: []map[string]types.AttributeValue{
						{
							"ValueName": &types.AttributeValueMemberS{Value: "Integrity"},
						},
						{
							"ValueName": &types.AttributeValueMemberS{Value: "Excellence"},
						},
					},
					Count: 2,
				},
			},
			ScanErrors: []error{
				nil,
			},
		}

		svc := TenantEngagementService{
			ctx:               context.TODO(),
			dynamodbClient:    &ddbClient,
			logger:            log.New(logBuffer, "TEST:", 0),
			TenantValuesTable: "TenantValuesTable-test",
		}

		values, err := svc.GetAllValues()
		assert.NoError(t, err)
		assert.ElementsMatch(t, []AppreciationObject{
			{
				ObjectType: "Value",
				Value:      "Integrity",
			},
			{
				ObjectType: "Value",
				Value:      "Excellence",
			},
		}, values)
	})
}
