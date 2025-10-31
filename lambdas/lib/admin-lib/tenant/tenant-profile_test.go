package adminlib

import (
	"bytes"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	ddb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
	"github.com/stretchr/testify/assert"
)

func Test_GetAllTenantDetails(t *testing.T) {

	t.Run("It should return the expected output when this function is called", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-1"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 1"},
							"Industry":           &types.AttributeValueMemberS{Value: "Automotive"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-19"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG01"},
						},
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-2"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 2"},
							"Industry":           &types.AttributeValueMemberS{Value: "Retail"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-18"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG03"},
						},
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-3"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 3"},
							"Industry":           &types.AttributeValueMemberS{Value: "Health"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-20"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG06"},
						},
					},
				},
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-6"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 6"},
							"Industry":           &types.AttributeValueMemberS{Value: "Automotive"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-19"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG08"},
						},
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-7"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 7"},
							"Industry":           &types.AttributeValueMemberS{Value: "Retail"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-18"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG08"},
						},
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-8"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 8"},
							"Industry":           &types.AttributeValueMemberS{Value: "Health"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-20"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG08"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
				nil,
			},
		}

		logBuffer := &bytes.Buffer{}

		svc := TenantDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			TenantDetailsTable:              "DetailsTable",
			TenantDetails_TenantStatusIndex: "DetailsTable_Index",
		}

		expectedDDBInput1 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT TenantId, TenantName, Industry, TenantCreationDate, TenantStageId FROM \"DetailsTable\".\"DetailsTable_Index\" WHERE TenantStageId IN ('STG01', 'STG02','STG03','STG04','STG05','STG06','STG07') ORDER BY TenantStageId ASC"),
			ConsistentRead: aws.Bool(false),
		}
		expectedDDBInput2 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT TenantId, TenantName, Industry, TenantCreationDate, TenantStageId FROM \"DetailsTable\".\"DetailsTable_Index\" WHERE TenantStageId = 'STG08'"),
			ConsistentRead: aws.Bool(false),
		}

		expectedOutput := ListInProgActiveTenants{
			OnboardingInProg: []TenantDetails{
				{
					TenantId:           "TenantId-1",
					TenantName:         "Name of Tenant 1",
					Industry:           "Automotive",
					TenantCreationDate: "2024-04-19",
					TenantStageId:      "STG01",
					TenantStageName:    "InitialOnboarding",
				},
				{
					TenantId:           "TenantId-2",
					TenantName:         "Name of Tenant 2",
					Industry:           "Retail",
					TenantCreationDate: "2024-04-18",
					TenantStageId:      "STG03",
					TenantStageName:    "TrialSetup",
				},
				{
					TenantId:           "TenantId-3",
					TenantName:         "Name of Tenant 3",
					Industry:           "Health",
					TenantCreationDate: "2024-04-20",
					TenantStageId:      "STG06",
					TenantStageName:    "PreProvisioningChecks",
				},
			},
			Active: []TenantDetails{
				{
					TenantId:           "TenantId-6",
					TenantName:         "Name of Tenant 6",
					Industry:           "Automotive",
					TenantCreationDate: "2024-04-19",
					TenantStageId:      "STG08",
					TenantStageName:    "Active",
				},
				{
					TenantId:           "TenantId-7",
					TenantName:         "Name of Tenant 7",
					Industry:           "Retail",
					TenantCreationDate: "2024-04-18",
					TenantStageId:      "STG08",
					TenantStageName:    "Active",
				},
				{
					TenantId:           "TenantId-8",
					TenantName:         "Name of Tenant 8",
					Industry:           "Health",
					TenantCreationDate: "2024-04-20",
					TenantStageId:      "STG08",
					TenantStageName:    "Active",
				},
			},
		}

		output, err := svc.GetAllTenantDetails()

		assert.NoError(t, err)

		assert.Equal(t, expectedOutput, output)

		assert.Equal(t, expectedDDBInput1, ddbClient.ExecuteStatementInputs[0])
		assert.Equal(t, expectedDDBInput2, ddbClient.ExecuteStatementInputs[1])

	})

	t.Run("It should return the expected output there are no items in one of the query output", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-1"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 1"},
							"Industry":           &types.AttributeValueMemberS{Value: "Automotive"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-19"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG01"},
						},
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-2"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 2"},
							"Industry":           &types.AttributeValueMemberS{Value: "Retail"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-18"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG03"},
						},
						{
							"TenantId":           &types.AttributeValueMemberS{Value: "TenantId-3"},
							"TenantName":         &types.AttributeValueMemberS{Value: "Name of Tenant 3"},
							"Industry":           &types.AttributeValueMemberS{Value: "Health"},
							"TenantCreationDate": &types.AttributeValueMemberS{Value: "2024-04-20"},
							"TenantStageId":      &types.AttributeValueMemberS{Value: "STG06"},
						},
					},
				},
				{
					Items: []map[string]ddb_types.AttributeValue{},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
				nil,
			},
		}

		logBuffer := &bytes.Buffer{}

		svc := TenantDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			TenantDetailsTable:              "DetailsTable",
			TenantDetails_TenantStatusIndex: "DetailsTable_Index",
		}

		expectedDDBInput1 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT TenantId, TenantName, Industry, TenantCreationDate, TenantStageId FROM \"DetailsTable\".\"DetailsTable_Index\" WHERE TenantStageId IN ('STG01', 'STG02','STG03','STG04','STG05','STG06','STG07') ORDER BY TenantStageId ASC"),
			ConsistentRead: aws.Bool(false),
		}
		expectedDDBInput2 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT TenantId, TenantName, Industry, TenantCreationDate, TenantStageId FROM \"DetailsTable\".\"DetailsTable_Index\" WHERE TenantStageId = 'STG08'"),
			ConsistentRead: aws.Bool(false),
		}

		expectedOutput := ListInProgActiveTenants{
			OnboardingInProg: []TenantDetails{
				{
					TenantId:           "TenantId-1",
					TenantName:         "Name of Tenant 1",
					Industry:           "Automotive",
					TenantCreationDate: "2024-04-19",
					TenantStageId:      "STG01",
					TenantStageName:    "InitialOnboarding",
				},
				{
					TenantId:           "TenantId-2",
					TenantName:         "Name of Tenant 2",
					Industry:           "Retail",
					TenantCreationDate: "2024-04-18",
					TenantStageId:      "STG03",
					TenantStageName:    "TrialSetup",
				},
				{
					TenantId:           "TenantId-3",
					TenantName:         "Name of Tenant 3",
					Industry:           "Health",
					TenantCreationDate: "2024-04-20",
					TenantStageId:      "STG06",
					TenantStageName:    "PreProvisioningChecks",
				},
			},
			Active: []TenantDetails{},
		}

		output, err := svc.GetAllTenantDetails()

		assert.NoError(t, err)

		assert.Equal(t, expectedOutput, output)

		assert.Equal(t, expectedDDBInput1, ddbClient.ExecuteStatementInputs[0])
		assert.Equal(t, expectedDDBInput2, ddbClient.ExecuteStatementInputs[1])

	})

}

func Test_GetTenantProfileById(t *testing.T) {
	t.Run("It should get the tenant Details when TenantID is passed", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: map[string]dynamodb_types.AttributeValue{

						"TenantId":   &dynamodb_types.AttributeValueMemberS{Value: "tenant-id"},
						"TenantName": &dynamodb_types.AttributeValueMemberS{Value: "Name of Tenant 1"},
						"TenantDesc": &dynamodb_types.AttributeValueMemberS{Value: "Desc of Tenant"},
						"Industry":   &dynamodb_types.AttributeValueMemberS{Value: "Automotive"},
						"TenantContacts": &dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"contact1@example.com": &dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"TenantEmail":       &dynamodb_types.AttributeValueMemberS{Value: "contact1@example.com"},
										"TenantContactName": &dynamodb_types.AttributeValueMemberS{Value: "Person1"},
										"TenantPh":          &dynamodb_types.AttributeValueMemberS{Value: "999999999999"},
										"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: true},
									},
								},
								"contact2@example.com": &dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"TenantEmail":       &dynamodb_types.AttributeValueMemberS{Value: "contact2@example.com"},
										"TenantContactName": &dynamodb_types.AttributeValueMemberS{Value: "Person2"},
										"TenantPh":          &dynamodb_types.AttributeValueMemberS{Value: "999999999998"},
										"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: false},
									},
								},
							},
						},
						"TenantCreationDate": &dynamodb_types.AttributeValueMemberS{Value: "2024-04-19"},
						"TenantStageId":      &dynamodb_types.AttributeValueMemberS{Value: "STG01"},
					},
				},
			},
			GetItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := TenantDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			TenantDetailsTable:              "DetailsTable",
			TenantDetails_TenantStatusIndex: "DetailsTable_Index",
		}

		expectedDDBInput := dynamodb.GetItemInput{
			Key: map[string]dynamodb_types.AttributeValue{
				"TenantId": &dynamodb_types.AttributeValueMemberS{Value: "tenant-id"},
			},
			TableName:      aws.String("DetailsTable"),
			ConsistentRead: aws.Bool(true),
		}

		expectedTenantDetailsData := TenantDetailsTable{
			TenantId:   "tenant-id",
			TenantName: "Name of Tenant 1",
			TenantDesc: "Desc of Tenant",
			Industry:   "Automotive",
			TenantContacts: map[string]TenantContacts{
				"contact1@example.com": {
					TenantEmail:       "contact1@example.com",
					TenantContactName: "Person1",
					TenantPh:          "999999999999",
					IsPrimary:         true,
				},
				"contact2@example.com": {
					TenantEmail:       "contact2@example.com",
					TenantContactName: "Person2",
					TenantPh:          "999999999998",
					IsPrimary:         false,
				},
			},
			TenantCreationDate: "2024-04-19",
			TenantStageId:      "STG01",
			TenantStageName:    "InitialOnboarding",
		}
		output, err := svc.GetTenantProfileById("tenant-id")

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.GetItemInputs[0])
		assert.Equal(t, expectedTenantDetailsData, output)

	})
}

func Test_CreateTenantProfile(t *testing.T) {
	t.Run("It should create a new tenant profile when all the request params are correct", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}

		logBuffer := &bytes.Buffer{}

		svc := TenantDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			TenantDetailsTable:              "DetailsTable",
			TenantDetails_TenantStatusIndex: "DetailsTable_Index",
		}

		createTenantInput := CreateTenantProfile{
			TenantName:          "Test Tenant",
			TenantDesc:          "Test Desc",
			Industry:            "Auto",
			EnvType:             "PROD",
			PrimaryContactName:  "Person1",
			PrimaryContactEmail: "person1@gmail.com",
			PrimaryContactPh:    "99999999999",
		}

		date := utils.GenerateTimestamp()

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("DetailsTable"),
			Key: map[string]dynamodb_types.AttributeValue{
				"TenantId": &dynamodb_types.AttributeValueMemberS{Value: ""}, // Not testing this as its unique value for every run
			},
			ConditionExpression: aws.String("attribute_not_exists(TenantContacts)"),
			UpdateExpression:    aws.String("SET TenantName = :TenantName, TenantDesc = :TenantDesc, Industry = :Industry, EnvType = :EnvType, TenantContacts = :TenantContacts, TenantCreationDate = :TenantCreationDate, TenantStageId = :TenantStageId"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":TenantName": &dynamodb_types.AttributeValueMemberS{Value: "Test Tenant"},
				":TenantDesc": &dynamodb_types.AttributeValueMemberS{Value: "Test Desc"},
				":Industry":   &dynamodb_types.AttributeValueMemberS{Value: "Auto"},
				":TenantContacts": &dynamodb_types.AttributeValueMemberM{
					Value: map[string]dynamodb_types.AttributeValue{
						"person1@gmail.com": &dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"TenantEmail":       &dynamodb_types.AttributeValueMemberS{Value: "person1@gmail.com"},
								"TenantContactName": &dynamodb_types.AttributeValueMemberS{Value: "Person1"},
								"TenantPh":          &dynamodb_types.AttributeValueMemberS{Value: "99999999999"},
								"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: true},
							},
						},
					}},
				":EnvType":            &dynamodb_types.AttributeValueMemberS{Value: "PROD"},
				":TenantCreationDate": &dynamodb_types.AttributeValueMemberS{Value: date},
				":TenantStageId":      &dynamodb_types.AttributeValueMemberS{Value: "STG01"},
			},
			ReturnValues: dynamodb_types.ReturnValueAllNew,
		}

		err := svc.CreateTenantProfile(createTenantInput)

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput.ExpressionAttributeValues, ddbClient.UpdateItemInputs[0].ExpressionAttributeValues)
		assert.Equal(t, expectedDDBInput.TableName, ddbClient.UpdateItemInputs[0].TableName)
		assert.Equal(t, expectedDDBInput.ConditionExpression, ddbClient.UpdateItemInputs[0].ConditionExpression)

		// check for uuid gen
		//assert.Equal(t, expectedDDBInput.Key, ddbClient.UpdateItemInputs[0].Key)

	})
}

func Test_PatchTopLevelInfo(t *testing.T) {
	t.Run("It should patch the top level tenant info when patch api is called with correct params", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := TenantDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			TenantDetailsTable:              "DetailsTable",
			TenantDetails_TenantStatusIndex: "DetailsTable_Index",
		}

		err := svc.PatchTopLevelInfo(PatchTopLevelInfo{
			TenantId:   "abc-123",
			TenantName: "New CompanyName",
			TenantDesc: "New Desc",
			Industry:   "Updated-Industry",
			EnvType:    "PROD",
		})

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("DetailsTable"),
			Key: map[string]dynamodb_types.AttributeValue{
				"TenantId": &dynamodb_types.AttributeValueMemberS{Value: "abc-123"},
			},
			UpdateExpression:    aws.String("SET TenantName = :TenantName, TenantDesc = :TenantDesc, Industry = :Industry, EnvType = :EnvType"),
			ConditionExpression: aws.String("attribute_exists(TenantId)"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":TenantName": &dynamodb_types.AttributeValueMemberS{Value: "New CompanyName"},
				":TenantDesc": &dynamodb_types.AttributeValueMemberS{Value: "New Desc"},
				":Industry":   &dynamodb_types.AttributeValueMemberS{Value: "Updated-Industry"},
				":EnvType":    &dynamodb_types.AttributeValueMemberS{Value: "PROD"},
			},
			ReturnValues: dynamodb_types.ReturnValueAllNew,
		}

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs[0])

	})
}

func Test_PatchTenantContacts(t *testing.T) {
	t.Run("It should update the contacts in the tenant Profile table when all details are provided correctly", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := TenantDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			TenantDetailsTable:              "DetailsTable",
			TenantDetails_TenantStatusIndex: "DetailsTable_Index",
		}

		err := svc.PatchTenantContacts(PatchTenantContacts{
			TenantId:     "abc-123",
			ContactName:  "new Contact",
			ContactEmail: "new@gmail.com",
			ContactPh:    "99999933333",
			IsPrimary:    false,
		})

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("DetailsTable"),
			Key: map[string]dynamodb_types.AttributeValue{
				"TenantId": &dynamodb_types.AttributeValueMemberS{Value: "abc-123"},
			},
			UpdateExpression: aws.String("SET TenantContacts.#ContactId = :Contact"),
			ExpressionAttributeNames: map[string]string{
				"#ContactId": "new@gmail.com",
			},
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":Contact": &dynamodb_types.AttributeValueMemberM{
					Value: map[string]dynamodb_types.AttributeValue{
						"TenantEmail":       &dynamodb_types.AttributeValueMemberS{Value: "new@gmail.com"},
						"TenantContactName": &dynamodb_types.AttributeValueMemberS{Value: "new Contact"},
						"TenantPh":          &dynamodb_types.AttributeValueMemberS{Value: "99999933333"},
						"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: false},
					},
				},
			},
			ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
		}

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs[0])

	})
}

func Test_PatchOverallStageId(t *testing.T) {
	t.Run("It should patch the status of the Tenant when relevant input is provided", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := TenantDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			TenantDetailsTable:              "DetailsTable",
			TenantDetails_TenantStatusIndex: "DetailsTable_Index",
		}

		err := svc.PatchOverallStageId(PatchTenantOverallStage{
			TenantId:        "abc-123",
			TenantStageName: "Trail_Discontinued",
		})

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("DetailsTable"),
			Key: map[string]dynamodb_types.AttributeValue{
				"TenantId": &dynamodb_types.AttributeValueMemberS{Value: "abc-123"},
			},
			UpdateExpression:    aws.String("SET TenantStageId = :TenantStageId"),
			ConditionExpression: aws.String("attribute_exists(TenantId)"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":TenantStageId": &dynamodb_types.AttributeValueMemberS{Value: "STG05"},
			},
			ReturnValues: dynamodb_types.ReturnValueAllNew,
		}

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs[0])

	})
}
