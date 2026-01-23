package Companylib

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func TestCreateCustomAttribute(t *testing.T) {
	t.Run("It should create a custom attribute successfully", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{nil},
		}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		attr := TeamAttribute{
			TeamId:        "TEAM-test123",
			AttributeType: AttributeTypeSkill,
			Name:          "Custom Skill",
			Description:   "A custom skill for testing",
			CreatedBy:     "testuser",
		}

		err := svc.CreateCustomAttribute(attr)

		assert.NoError(t, err)
		assert.Len(t, ddbClient.PutItemInputs, 1)
		assert.Equal(t, "TeamAttributesTable-test", *ddbClient.PutItemInputs[0].TableName)
	})

	t.Run("It should reject invalid attribute type", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		ddbClient := awsclients.MockDynamodbClient{}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		attr := TeamAttribute{
			TeamId:        "TEAM-test123",
			AttributeType: "INVALID",
			Name:          "Custom Skill",
			Description:   "A custom skill for testing",
			CreatedBy:     "testuser",
		}

		err := svc.CreateCustomAttribute(attr)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid attribute type")
	})
}

func TestInitializeDefaultAttributes(t *testing.T) {
	t.Run("It should initialize 12 default attributes for a team", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		ddbClient := awsclients.MockDynamodbClient{
			BatchWriteItemOutputs: []dynamodb.BatchWriteItemOutput{
				{},
			},
			BatchErrors: []error{nil},
		}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		err := svc.InitializeDefaultAttributes("TEAM-abc123", "admin-user")

		assert.NoError(t, err)
		assert.Len(t, ddbClient.BatchWriteItemsInputs, 1)

		// Verify 12 write requests were made
		writeRequests := ddbClient.BatchWriteItemsInputs[0].RequestItems["TeamAttributesTable-test"]
		assert.Len(t, writeRequests, 12)

		// Count each type
		skillCount := 0
		valueCount := 0
		milestoneCount := 0
		metricCount := 0

		for _, req := range writeRequests {
			if req.PutRequest != nil {
				attrType := req.PutRequest.Item["AttributeType"].(*dynamodb_types.AttributeValueMemberS).Value
				switch attrType {
				case "SKILL":
					skillCount++
				case "VALUE":
					valueCount++
				case "MILESTONE":
					milestoneCount++
				case "METRIC":
					metricCount++
				}
			}
		}

		assert.Equal(t, 3, skillCount, "Should have 3 default skills")
		assert.Equal(t, 3, valueCount, "Should have 3 default values")
		assert.Equal(t, 3, milestoneCount, "Should have 3 default milestones")
		assert.Equal(t, 3, metricCount, "Should have 3 default metrics")
	})
}

func TestListTeamAttributes(t *testing.T) {
	t.Run("It should list all attributes for a team", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		skillAttr := TeamAttribute{
			AttributeId:   "ATTR-001",
			TeamId:        "TEAM-test123",
			AttributeType: AttributeTypeSkill,
			Name:          "Leadership",
			Description:   "Leadership skills",
			IsDefault:     true,
			CreatedBy:     "system",
		}

		skillItem, _ := attributevalue.MarshalMap(skillAttr)

		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 1,
					Items: []map[string]dynamodb_types.AttributeValue{
						skillItem,
					},
				},
			},
			QueryErrors: []error{nil},
		}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		attributes, err := svc.ListTeamAttributes("TEAM-test123", nil)

		assert.NoError(t, err)
		assert.Len(t, attributes, 1)
		assert.Equal(t, "Leadership", attributes[0].Name)
		assert.Equal(t, AttributeTypeSkill, attributes[0].AttributeType)
	})

	t.Run("It should filter attributes by type", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
					Items: []map[string]dynamodb_types.AttributeValue{},
				},
			},
			QueryErrors: []error{nil},
		}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		attrType := AttributeTypeSkill
		attributes, err := svc.ListTeamAttributes("TEAM-test123", &attrType)

		assert.NoError(t, err)
		assert.Len(t, attributes, 0)
		assert.Len(t, ddbClient.QueryInputs, 1)

		// Verify the query includes AttributeType filter
		queryInput := ddbClient.QueryInputs[0]
		assert.Contains(t, *queryInput.KeyConditionExpression, "AttributeType")
	})
}

func TestDeleteAttribute(t *testing.T) {
	t.Run("It should delete a custom attribute", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		customAttr := TeamAttribute{
			AttributeId:   "ATTR-custom-001",
			TeamId:        "TEAM-test123",
			AttributeType: AttributeTypeSkill,
			Name:          "Custom Skill",
			IsDefault:     false,
			CreatedBy:     "testuser",
		}

		attrItem, _ := attributevalue.MarshalMap(customAttr)

		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: attrItem,
				},
			},
			GetItemErrors: []error{nil},
			DeleteItemOutputs: []dynamodb.DeleteItemOutput{
				{},
			},
			DeleteItemErrors: []error{nil},
		}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		err := svc.DeleteAttribute("ATTR-custom-001", "TEAM-test123")

		assert.NoError(t, err)
		assert.Len(t, ddbClient.DeleteItemInputs, 1)
	})

	t.Run("It should not delete a default attribute", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		defaultAttr := TeamAttribute{
			AttributeId:   "ATTR-default-001",
			TeamId:        "TEAM-test123",
			AttributeType: AttributeTypeSkill,
			Name:          "Leadership",
			IsDefault:     true,
			CreatedBy:     "system",
		}

		attrItem, _ := attributevalue.MarshalMap(defaultAttr)

		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: attrItem,
				},
			},
			GetItemErrors: []error{nil},
		}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		err := svc.DeleteAttribute("ATTR-default-001", "TEAM-test123")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default attributes")
		assert.Len(t, ddbClient.DeleteItemInputs, 0)
	})
}

func TestUpdateAttribute(t *testing.T) {
	t.Run("It should update an attribute", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{nil},
		}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		err := svc.UpdateAttribute("ATTR-001", "TEAM-test123", "Updated Name", "Updated Description")

		assert.NoError(t, err)
		assert.Len(t, ddbClient.UpdateItemInputs, 1)

		updateInput := ddbClient.UpdateItemInputs[0]
		assert.Equal(t, "TeamAttributesTable-test", *updateInput.TableName)
		assert.Equal(t, "ATTR-001", updateInput.Key["AttributeId"].(*dynamodb_types.AttributeValueMemberS).Value)
	})
}

func TestAttributeTypes(t *testing.T) {
	// Test all attribute type constants
	assert.Equal(t, TeamAttributeType("SKILL"), AttributeTypeSkill)
	assert.Equal(t, TeamAttributeType("VALUE"), AttributeTypeValue)
	assert.Equal(t, TeamAttributeType("MILESTONE"), AttributeTypeMilestone)
	assert.Equal(t, TeamAttributeType("METRIC"), AttributeTypeMetric)
}

func TestGroupedAttributesStructure(t *testing.T) {
	t.Run("It should properly group attributes by type", func(t *testing.T) {
		grouped := GroupedAttributes{
			Skills: []TeamAttribute{
				{Name: "Leadership", AttributeType: AttributeTypeSkill},
			},
			Values: []TeamAttribute{
				{Name: "Integrity", AttributeType: AttributeTypeValue},
			},
			Milestones: []TeamAttribute{
				{Name: "Q1 Achievement", AttributeType: AttributeTypeMilestone},
			},
			Metrics: []TeamAttribute{
				{Name: "Productivity", AttributeType: AttributeTypeMetric},
			},
		}

		assert.Len(t, grouped.Skills, 1)
		assert.Len(t, grouped.Values, 1)
		assert.Len(t, grouped.Milestones, 1)
		assert.Len(t, grouped.Metrics, 1)

		assert.Equal(t, "Leadership", grouped.Skills[0].Name)
		assert.Equal(t, "Integrity", grouped.Values[0].Name)
		assert.Equal(t, "Q1 Achievement", grouped.Milestones[0].Name)
		assert.Equal(t, "Productivity", grouped.Metrics[0].Name)
	})
}

func TestGetAttributesByType(t *testing.T) {
	t.Run("It should retrieve and group attributes by type", func(t *testing.T) {
		ctx := context.Background()
		logBuffer := &bytes.Buffer{}

		skillAttr := TeamAttribute{
			AttributeId:   "ATTR-001",
			TeamId:        "TEAM-test123",
			AttributeType: AttributeTypeSkill,
			Name:          "Leadership",
		}

		valueAttr := TeamAttribute{
			AttributeId:   "ATTR-002",
			TeamId:        "TEAM-test123",
			AttributeType: AttributeTypeValue,
			Name:          "Integrity",
		}

		skillItem, _ := attributevalue.MarshalMap(skillAttr)
		valueItem, _ := attributevalue.MarshalMap(valueAttr)

		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 2,
					Items: []map[string]dynamodb_types.AttributeValue{
						skillItem,
						valueItem,
					},
				},
			},
			QueryErrors: []error{nil},
		}

		svc := TeamAttributeServiceV2{
			ctx:                       ctx,
			dynamodbClient:            &ddbClient,
			logger:                    log.New(logBuffer, "TEST:", 0),
			TeamAttributesTable:       "TeamAttributesTable-test",
			TeamAttributesTeamIdIndex: "TeamId-AttributeType-index",
		}

		grouped, err := svc.GetAttributesByType("TEAM-test123")

		assert.NoError(t, err)
		assert.Len(t, grouped.Skills, 1)
		assert.Len(t, grouped.Values, 1)
		assert.Equal(t, "Leadership", grouped.Skills[0].Name)
		assert.Equal(t, "Integrity", grouped.Values[0].Name)
	})
}
