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

func Test_GetAllSupplierDetails(t *testing.T) {

	t.Run("It should return the expected output when this function is called", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-1"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 1"},
							"Industry":           &types.AttributeValueMemberS{Value: "Automotive"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-19"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG01"},
						},
						{
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-2"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 2"},
							"Industry":           &types.AttributeValueMemberS{Value: "Retail"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-18"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG03"},
						},
						{
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-3"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 3"},
							"Industry":           &types.AttributeValueMemberS{Value: "Health"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-20"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG06"},
						},
					},
				},
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-6"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 6"},
							"Industry":           &types.AttributeValueMemberS{Value: "Automotive"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-19"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG08"},
						},
						{
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-7"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 7"},
							"Industry":           &types.AttributeValueMemberS{Value: "Retail"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-18"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG08"},
						},
						{
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-8"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 8"},
							"Industry":           &types.AttributeValueMemberS{Value: "Health"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-20"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG08"},
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

		svc := SupplierDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			SupplierDetailsTable:              "DetailsTable",
			SupplierDetails_SupplierStatusIndex: "DetailsTable_Index",
		}

		expectedDDBInput1 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT SupplierId, SupplierName, Industry, SupplierCreationDate, SupplierStageId FROM \"DetailsTable\".\"DetailsTable_Index\" WHERE SupplierStageId IN ('STG01', 'STG02','STG03','STG04','STG05','STG06','STG07') ORDER BY SupplierStageId ASC"),
			ConsistentRead: aws.Bool(false),
		}
		expectedDDBInput2 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT SupplierId, SupplierName, Industry, SupplierCreationDate, SupplierStageId FROM \"DetailsTable\".\"DetailsTable_Index\" WHERE SupplierStageId = 'STG08'"),
			ConsistentRead: aws.Bool(false),
		}

		expectedOutput := ListInProgActiveSuppliers{
			OnboardingInProg: []SupplierDetails{
				{
					SupplierId:           "SupplierId-1",
					SupplierName:         "Name of Supplier 1",
					Industry:           "Automotive",
					SupplierCreationDate: "2024-04-19",
					SupplierStageId:      "STG01",
					SupplierStageName:    "InitialOnboarding",
				},
				{
					SupplierId:           "SupplierId-2",
					SupplierName:         "Name of Supplier 2",
					Industry:           "Retail",
					SupplierCreationDate: "2024-04-18",
					SupplierStageId:      "STG03",
					SupplierStageName:    "TrialSetup",
				},
				{
					SupplierId:           "SupplierId-3",
					SupplierName:         "Name of Supplier 3",
					Industry:           "Health",
					SupplierCreationDate: "2024-04-20",
					SupplierStageId:      "STG06",
					SupplierStageName:    "PreProvisioningChecks",
				},
			},
			Active: []SupplierDetails{
				{
					SupplierId:           "SupplierId-6",
					SupplierName:         "Name of Supplier 6",
					Industry:           "Automotive",
					SupplierCreationDate: "2024-04-19",
					SupplierStageId:      "STG08",
					SupplierStageName:    "Active",
				},
				{
					SupplierId:           "SupplierId-7",
					SupplierName:         "Name of Supplier 7",
					Industry:           "Retail",
					SupplierCreationDate: "2024-04-18",
					SupplierStageId:      "STG08",
					SupplierStageName:    "Active",
				},
				{
					SupplierId:           "SupplierId-8",
					SupplierName:         "Name of Supplier 8",
					Industry:           "Health",
					SupplierCreationDate: "2024-04-20",
					SupplierStageId:      "STG08",
					SupplierStageName:    "Active",
				},
			},
		}

		output, err := svc.GetAllSupplierDetails()

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
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-1"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 1"},
							"Industry":           &types.AttributeValueMemberS{Value: "Automotive"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-19"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG01"},
						},
						{
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-2"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 2"},
							"Industry":           &types.AttributeValueMemberS{Value: "Retail"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-18"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG03"},
						},
						{
							"SupplierId":           &types.AttributeValueMemberS{Value: "SupplierId-3"},
							"SupplierName":         &types.AttributeValueMemberS{Value: "Name of Supplier 3"},
							"Industry":           &types.AttributeValueMemberS{Value: "Health"},
							"SupplierCreationDate": &types.AttributeValueMemberS{Value: "2024-04-20"},
							"SupplierStageId":      &types.AttributeValueMemberS{Value: "STG06"},
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

		svc := SupplierDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			SupplierDetailsTable:              "DetailsTable",
			SupplierDetails_SupplierStatusIndex: "DetailsTable_Index",
		}

		expectedDDBInput1 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT SupplierId, SupplierName, Industry, SupplierCreationDate, SupplierStageId FROM \"DetailsTable\".\"DetailsTable_Index\" WHERE SupplierStageId IN ('STG01', 'STG02','STG03','STG04','STG05','STG06','STG07') ORDER BY SupplierStageId ASC"),
			ConsistentRead: aws.Bool(false),
		}
		expectedDDBInput2 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT SupplierId, SupplierName, Industry, SupplierCreationDate, SupplierStageId FROM \"DetailsTable\".\"DetailsTable_Index\" WHERE SupplierStageId = 'STG08'"),
			ConsistentRead: aws.Bool(false),
		}

		expectedOutput := ListInProgActiveSuppliers{
			OnboardingInProg: []SupplierDetails{
				{
					SupplierId:           "SupplierId-1",
					SupplierName:         "Name of Supplier 1",
					Industry:           "Automotive",
					SupplierCreationDate: "2024-04-19",
					SupplierStageId:      "STG01",
					SupplierStageName:    "InitialOnboarding",
				},
				{
					SupplierId:           "SupplierId-2",
					SupplierName:         "Name of Supplier 2",
					Industry:           "Retail",
					SupplierCreationDate: "2024-04-18",
					SupplierStageId:      "STG03",
					SupplierStageName:    "TrialSetup",
				},
				{
					SupplierId:           "SupplierId-3",
					SupplierName:         "Name of Supplier 3",
					Industry:           "Health",
					SupplierCreationDate: "2024-04-20",
					SupplierStageId:      "STG06",
					SupplierStageName:    "PreProvisioningChecks",
				},
			},
			Active: []SupplierDetails{},
		}

		output, err := svc.GetAllSupplierDetails()

		assert.NoError(t, err)

		assert.Equal(t, expectedOutput, output)

		assert.Equal(t, expectedDDBInput1, ddbClient.ExecuteStatementInputs[0])
		assert.Equal(t, expectedDDBInput2, ddbClient.ExecuteStatementInputs[1])

	})

}

func Test_GetSupplierProfileById(t *testing.T) {
	t.Run("It should get the Supplier Details when SupplierID is passed", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: map[string]dynamodb_types.AttributeValue{

						"SupplierId":   &dynamodb_types.AttributeValueMemberS{Value: "Supplier-id"},
						"SupplierName": &dynamodb_types.AttributeValueMemberS{Value: "Name of Supplier 1"},
						"SupplierDesc": &dynamodb_types.AttributeValueMemberS{Value: "Desc of Supplier"},
						"Industry":   &dynamodb_types.AttributeValueMemberS{Value: "Automotive"},
						"SupplierContacts": &dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"contact1@example.com": &dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"SupplierEmail":       &dynamodb_types.AttributeValueMemberS{Value: "contact1@example.com"},
										"SupplierContactName": &dynamodb_types.AttributeValueMemberS{Value: "Person1"},
										"SupplierPh":          &dynamodb_types.AttributeValueMemberS{Value: "999999999999"},
										"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: true},
									},
								},
								"contact2@example.com": &dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"SupplierEmail":       &dynamodb_types.AttributeValueMemberS{Value: "contact2@example.com"},
										"SupplierContactName": &dynamodb_types.AttributeValueMemberS{Value: "Person2"},
										"SupplierPh":          &dynamodb_types.AttributeValueMemberS{Value: "999999999998"},
										"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: false},
									},
								},
							},
						},
						"SupplierCreationDate": &dynamodb_types.AttributeValueMemberS{Value: "2024-04-19"},
						"SupplierStageId":      &dynamodb_types.AttributeValueMemberS{Value: "STG01"},
					},
				},
			},
			GetItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			SupplierDetailsTable:              "DetailsTable",
			SupplierDetails_SupplierStatusIndex: "DetailsTable_Index",
		}

		expectedDDBInput := dynamodb.GetItemInput{
			Key: map[string]dynamodb_types.AttributeValue{
				"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: "Supplier-id"},
			},
			TableName:      aws.String("DetailsTable"),
			ConsistentRead: aws.Bool(true),
		}

		expectedSupplierDetailsData := SupplierDetailsTable{
			SupplierId:   "Supplier-id",
			SupplierName: "Name of Supplier 1",
			SupplierDesc: "Desc of Supplier",
			Industry:   "Automotive",
			SupplierContacts: map[string]SupplierContacts{
				"contact1@example.com": {
					SupplierEmail:       "contact1@example.com",
					SupplierContactName: "Person1",
					SupplierPh:          "999999999999",
					IsPrimary:         true,
				},
				"contact2@example.com": {
					SupplierEmail:       "contact2@example.com",
					SupplierContactName: "Person2",
					SupplierPh:          "999999999998",
					IsPrimary:         false,
				},
			},
			SupplierCreationDate: "2024-04-19",
			SupplierStageId:      "STG01",
			SupplierStageName:    "InitialOnboarding",
		}
		output, err := svc.GetSupplierProfileById("Supplier-id")

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.GetItemInputs[0])
		assert.Equal(t, expectedSupplierDetailsData, output)

	})
}

func Test_CreateSupplierProfile(t *testing.T) {
	t.Run("It should create a new Supplier profile when all the request params are correct", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}

		logBuffer := &bytes.Buffer{}

		svc := SupplierDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			SupplierDetailsTable:              "DetailsTable",
			SupplierDetails_SupplierStatusIndex: "DetailsTable_Index",
		}

		createSupplierInput := CreateSupplierProfile{
			SupplierName:          "Test Supplier",
			SupplierDesc:          "Test Desc",
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
				"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: ""}, // Not testing this as its unique value for every run
			},
			ConditionExpression: aws.String("attribute_not_exists(SupplierContacts)"),
			UpdateExpression:    aws.String("SET SupplierName = :SupplierName, SupplierDesc = :SupplierDesc, Industry = :Industry, EnvType = :EnvType, SupplierContacts = :SupplierContacts, SupplierCreationDate = :SupplierCreationDate, SupplierStageId = :SupplierStageId"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":SupplierName": &dynamodb_types.AttributeValueMemberS{Value: "Test Supplier"},
				":SupplierDesc": &dynamodb_types.AttributeValueMemberS{Value: "Test Desc"},
				":Industry":   &dynamodb_types.AttributeValueMemberS{Value: "Auto"},
				":SupplierContacts": &dynamodb_types.AttributeValueMemberM{
					Value: map[string]dynamodb_types.AttributeValue{
						"person1@gmail.com": &dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"SupplierEmail":       &dynamodb_types.AttributeValueMemberS{Value: "person1@gmail.com"},
								"SupplierContactName": &dynamodb_types.AttributeValueMemberS{Value: "Person1"},
								"SupplierPh":          &dynamodb_types.AttributeValueMemberS{Value: "99999999999"},
								"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: true},
							},
						},
					}},
				":EnvType":            &dynamodb_types.AttributeValueMemberS{Value: "PROD"},
				":SupplierCreationDate": &dynamodb_types.AttributeValueMemberS{Value: date},
				":SupplierStageId":      &dynamodb_types.AttributeValueMemberS{Value: "STG01"},
			},
			ReturnValues: dynamodb_types.ReturnValueAllNew,
		}

		err := svc.CreateSupplierProfile(createSupplierInput)

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput.ExpressionAttributeValues, ddbClient.UpdateItemInputs[0].ExpressionAttributeValues)
		assert.Equal(t, expectedDDBInput.TableName, ddbClient.UpdateItemInputs[0].TableName)
		assert.Equal(t, expectedDDBInput.ConditionExpression, ddbClient.UpdateItemInputs[0].ConditionExpression)

		// check for uuid gen
		//assert.Equal(t, expectedDDBInput.Key, ddbClient.UpdateItemInputs[0].Key)

	})
}

func Test_PatchTopLevelInfo(t *testing.T) {
	t.Run("It should patch the top level Supplier info when patch api is called with correct params", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			SupplierDetailsTable:              "DetailsTable",
			SupplierDetails_SupplierStatusIndex: "DetailsTable_Index",
		}

		err := svc.PatchTopLevelInfo(PatchTopLevelInfo{
			SupplierId:   "abc-123",
			SupplierName: "New CompanyName",
			SupplierDesc: "New Desc",
			Industry:   "Updated-Industry",
			EnvType:    "PROD",
		})

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("DetailsTable"),
			Key: map[string]dynamodb_types.AttributeValue{
				"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: "abc-123"},
			},
			UpdateExpression:    aws.String("SET SupplierName = :SupplierName, SupplierDesc = :SupplierDesc, Industry = :Industry, EnvType = :EnvType"),
			ConditionExpression: aws.String("attribute_exists(SupplierId)"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":SupplierName": &dynamodb_types.AttributeValueMemberS{Value: "New CompanyName"},
				":SupplierDesc": &dynamodb_types.AttributeValueMemberS{Value: "New Desc"},
				":Industry":   &dynamodb_types.AttributeValueMemberS{Value: "Updated-Industry"},
				":EnvType":    &dynamodb_types.AttributeValueMemberS{Value: "PROD"},
			},
			ReturnValues: dynamodb_types.ReturnValueAllNew,
		}

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs[0])

	})
}

func Test_PatchSupplierContacts(t *testing.T) {
	t.Run("It should update the contacts in the Supplier Profile table when all details are provided correctly", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			SupplierDetailsTable:              "DetailsTable",
			SupplierDetails_SupplierStatusIndex: "DetailsTable_Index",
		}

		err := svc.PatchSupplierContacts(PatchSupplierContacts{
			SupplierId:     "abc-123",
			ContactName:  "new Contact",
			ContactEmail: "new@gmail.com",
			ContactPh:    "99999933333",
			IsPrimary:    false,
		})

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("DetailsTable"),
			Key: map[string]dynamodb_types.AttributeValue{
				"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: "abc-123"},
			},
			UpdateExpression: aws.String("SET SupplierContacts.#ContactId = :Contact"),
			ExpressionAttributeNames: map[string]string{
				"#ContactId": "new@gmail.com",
			},
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":Contact": &dynamodb_types.AttributeValueMemberM{
					Value: map[string]dynamodb_types.AttributeValue{
						"SupplierEmail":       &dynamodb_types.AttributeValueMemberS{Value: "new@gmail.com"},
						"SupplierContactName": &dynamodb_types.AttributeValueMemberS{Value: "new Contact"},
						"SupplierPh":          &dynamodb_types.AttributeValueMemberS{Value: "99999933333"},
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
	t.Run("It should patch the status of the Supplier when relevant input is provided", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierDetailsSvc{
			dynamodbClient:                  &ddbClient,
			logger:                          log.New(logBuffer, "TEST:", 0),
			SupplierDetailsTable:              "DetailsTable",
			SupplierDetails_SupplierStatusIndex: "DetailsTable_Index",
		}

		err := svc.PatchOverallStageId(PatchSupplierOverallStage{
			SupplierId:        "abc-123",
			SupplierStageName: "Trail_Discontinued",
		})

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("DetailsTable"),
			Key: map[string]dynamodb_types.AttributeValue{
				"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: "abc-123"},
			},
			UpdateExpression:    aws.String("SET SupplierStageId = :SupplierStageId"),
			ConditionExpression: aws.String("attribute_exists(SupplierId)"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":SupplierStageId": &dynamodb_types.AttributeValueMemberS{Value: "STG05"},
			},
			ReturnValues: dynamodb_types.ReturnValueAllNew,
		}

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs[0])

	})
}
