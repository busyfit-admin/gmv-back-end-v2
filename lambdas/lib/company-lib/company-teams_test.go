package Companylib

import (
	"bytes"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_GetAllTenantTeams(t *testing.T) {
	t.Run("It should Get the tenant teams data based on the query provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "TEAM--ID-001"},
							"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "TEAM-DEFAULT"},
							"TeamName":        &dynamodb_types.AttributeValueMemberS{Value: "test"},
							"TeamDesc":        &dynamodb_types.AttributeValueMemberS{Value: "desc"},
							"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: "Active"},
						},
					},
				},
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "TEAM-ID-002"},
							"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "DEFAULT"},
							"TeamName":        &dynamodb_types.AttributeValueMemberS{Value: "test2"},
							"TeamDesc":        &dynamodb_types.AttributeValueMemberS{Value: "desc2"},
							"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: "Inactive"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
				nil,
			},
		}
		svc := TenantTeamsService{
			dynamodbClient:        &ddbClient,
			logger:                log.New(logBuffer, "TEST:", 0),
			TenantTeamsTable:      "TenantTeamsTable-test",
			TenantTeams_TeamIndex: "RelatedEntityId-EntityId_Index",
		}

		expectedOutput := TenantTeams{
			Active: []TenantTeamsTable{
				{
					EntityId:        "TEAM--ID-001",
					RelatedEntityId: "TEAM-DEFAULT",
					TeamName:        "test",
					TeamDesc:        "desc",
					IsActive:        "Active",
				},
			},
			Draft: []TenantTeamsTable{
				{
					EntityId:        "TEAM-ID-002",
					RelatedEntityId: "DEFAULT",
					TeamName:        "test2",
					TeamDesc:        "desc2",
					IsActive:        "Inactive",
				},
			},
		}

		output, err := svc.GetAllTenantTeams()

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)

	})
}

func Test_GetTeamDetails(t *testing.T) {
	t.Run("It should get the tenant teams data when teamId is passed", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: map[string]dynamodb_types.AttributeValue{
						"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "TEAM--ID-001"},
						"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "TEAM-DEFAULT"},
						"TeamName":        &dynamodb_types.AttributeValueMemberS{Value: "test"},
						"TeamDesc":        &dynamodb_types.AttributeValueMemberS{Value: "desc"},
						"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: "Active"},
					},
				},
			},
			GetItemErrors: []error{
				nil,
			},
		}

		svc := TenantTeamsService{
			dynamodbClient:        &ddbClient,
			logger:                log.New(logBuffer, "TEST:", 0),
			TenantTeamsTable:      "TenantTeamsTable-test",
			TenantTeams_TeamIndex: "RelatedEntityId-EntityId_Index",
		}

		expectedOutput := TenantTeamsTable{
			EntityId:        "TEAM--ID-001",
			RelatedEntityId: "TEAM-DEFAULT",
			TeamName:        "test",
			TeamDesc:        "desc",
			IsActive:        "Active",
		}

		expectedDDBInput := dynamodb.GetItemInput{
			Key: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "TEAM--ID-001"},
				"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "TEAM-DEFAULT"},
			},
			TableName:      aws.String("TenantTeamsTable-test"),
			ConsistentRead: aws.Bool(true),
		}

		output, err := svc.GetTeamDetails("TEAM--ID-001", "TEAM-DEFAULT")

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)
		assert.Equal(t, expectedDDBInput, ddbClient.GetItemInputs[0])

	})
}

func Test_GetTeamsofUserorMngr(t *testing.T) {
	t.Run("It should Get the tenant teams data based on the query provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "USER-alok"},
							"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "TEAM--ID-001"},
							"TeamName":        &dynamodb_types.AttributeValueMemberS{Value: "test"},
							"TeamDesc":        &dynamodb_types.AttributeValueMemberS{Value: "desc"},
							"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: "Active"},
						},
					},
				},
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "USER-alok"},
							"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "TEAM-ID-002"},
							"TeamName":        &dynamodb_types.AttributeValueMemberS{Value: "test2"},
							"TeamDesc":        &dynamodb_types.AttributeValueMemberS{Value: "desc2"},
							"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: "Inactive"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
				nil,
			},
		}
		svc := TenantTeamsService{
			dynamodbClient:        &ddbClient,
			logger:                log.New(logBuffer, "TEST:", 0),
			TenantTeamsTable:      "TenantTeamsTable-test",
			TenantTeams_TeamIndex: "RelatedEntityId-EntityId_Index",
		}

		expectedOutput := TenantTeams{
			Active: []TenantTeamsTable{
				{
					EntityId:        "USER-alok",
					RelatedEntityId: "TEAM--ID-001",
					TeamName:        "test",
					TeamDesc:        "desc",
					IsActive:        "Active",
				},
			},
			Draft: []TenantTeamsTable{
				{
					EntityId:        "USER-alok",
					RelatedEntityId: "TEAM-ID-002",
					TeamName:        "test2",
					TeamDesc:        "desc2",
					IsActive:        "Inactive",
				},
			},
		}

		output, err := svc.GetTeams("USER-alok")

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)

	})
}

func Test_GetTeamUsersorMngr(t *testing.T) {
	t.Run("It should Get the users of a tenant team based on the query provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"EntityId": &dynamodb_types.AttributeValueMemberS{Value: "USER-alok"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
			},
		}
		svc := TenantTeamsService{
			dynamodbClient:        &ddbClient,
			logger:                log.New(logBuffer, "TEST:", 0),
			TenantTeamsTable:      "TenantTeamsTable-test",
			TenantTeams_TeamIndex: "RelatedEntityId-EntityId_Index",
		}

		expectedOutput := TeamUserorMngrList{
			Members: []TeamUserorMngr{
				{
					EntityId: "USER-alok",
				},
			},
		}

		output, err := svc.GetTeamUsers("TEAM--ID-001")

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)

	})
}
