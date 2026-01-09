package supplierlib

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_GetAllBranches(t *testing.T) {

	t.Run("It should All Branches output when called", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-1"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "MumbaiBranch"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Andheri"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Mumbai"},
						},
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-2"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "DelhiBranch"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Delhi East"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Delhi"},
						},
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-3"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "Bengalore"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Bengalore North"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Koramangala"},
						},
					},
				},
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-4"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "INACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "MumbaiBranch"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Andheri West"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Mumbai"},
						},
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-5"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "INACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "DelhiBranch"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Delhi East"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Delhi"},
						},
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-6"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "INACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "Bengalore"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Bengalore South"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Domlur"},
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

		svc := SupplierService{
			dynamodbClient:                    &ddbClient,
			logger:                            log.New(logBuffer, "TEST:", 0),
			SupplierBranchTable:               "SupplierBranchTable",
			SupplierBranchTable_IsActiveIndex: "IsActive_Index",
		}

		expectedDDBInput1 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT BranchId, IsActive, BranchType, BranchName, BranchAddressField1, BranchCity FROM \"SupplierBranchTable\".\"IsActive_Index\" WHERE IsActive = 'ACTIVE' ORDER BY BranchType ASC"),
			ConsistentRead: aws.Bool(false),
		}
		expectedDDBInput2 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT BranchId, IsActive, BranchType, BranchName, BranchAddressField1, BranchCity FROM \"SupplierBranchTable\".\"IsActive_Index\" WHERE IsActive = 'INACTIVE' ORDER BY BranchType ASC"),
			ConsistentRead: aws.Bool(false),
		}

		expectedItems := AllBranches{
			ActiveBranches: []SupplierBranchShort{
				{
					BranchId:            "BranchId-1",
					IsActive:            "ACTIVE",
					BranchType:          "Store",
					BranchName:          "MumbaiBranch",
					BranchAddressField1: "Andheri",
					BranchCity:          "Mumbai",
				},
				{
					BranchId:            "BranchId-2",
					IsActive:            "ACTIVE",
					BranchType:          "Store",
					BranchName:          "DelhiBranch",
					BranchAddressField1: "Delhi East",
					BranchCity:          "Delhi",
				},
				{
					BranchId:            "BranchId-3",
					IsActive:            "ACTIVE",
					BranchType:          "Store",
					BranchName:          "Bengalore",
					BranchAddressField1: "Bengalore North",
					BranchCity:          "Koramangala",
				},
			},
			InactiveBranches: []SupplierBranchShort{
				{
					BranchId:            "BranchId-4",
					IsActive:            "INACTIVE",
					BranchType:          "Store",
					BranchName:          "MumbaiBranch",
					BranchAddressField1: "Andheri West",
					BranchCity:          "Mumbai",
				},
				{
					BranchId:            "BranchId-5",
					IsActive:            "INACTIVE",
					BranchType:          "Store",
					BranchName:          "DelhiBranch",
					BranchAddressField1: "Delhi East",
					BranchCity:          "Delhi",
				},
				{
					BranchId:            "BranchId-6",
					IsActive:            "INACTIVE",
					BranchType:          "Store",
					BranchName:          "Bengalore",
					BranchAddressField1: "Bengalore South",
					BranchCity:          "Domlur",
				},
			},
		}

		output, err := svc.GetAllBranches()

		assert.Equal(t, expectedItems, output)
		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput1, ddbClient.ExecuteStatementInputs[0])
		assert.Equal(t, expectedDDBInput2, ddbClient.ExecuteStatementInputs[1])

	})
}

func Test_GetActiveBranches(t *testing.T) {
	t.Run("It should return all the active branches", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]ddb_types.AttributeValue{
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-1"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "MumbaiBranch"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Andheri"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Mumbai"},
						},
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-2"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "DelhiBranch"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Delhi East"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Delhi"},
						},
						{
							"BranchId":            &ddb_types.AttributeValueMemberS{Value: "BranchId-3"},
							"IsActive":            &ddb_types.AttributeValueMemberS{Value: "ACTIVE"},
							"BranchName":          &ddb_types.AttributeValueMemberS{Value: "Bengalore"},
							"BranchType":          &ddb_types.AttributeValueMemberS{Value: "Store"},
							"BranchAddressField1": &ddb_types.AttributeValueMemberS{Value: "Bengalore North"},
							"BranchCity":          &ddb_types.AttributeValueMemberS{Value: "Koramangala"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierService{
			dynamodbClient:                    &ddbClient,
			logger:                            log.New(logBuffer, "TEST:", 0),
			SupplierBranchTable:               "SupplierBranchTable",
			SupplierBranchTable_IsActiveIndex: "IsActive_Index",
		}

		expectedDDBInput1 := dynamodb.ExecuteStatementInput{
			Statement:      aws.String("SELECT BranchId, IsActive, BranchType, BranchName, BranchAddressField1, BranchCity FROM \"SupplierBranchTable\".\"IsActive_Index\" WHERE IsActive = 'ACTIVE' ORDER BY BranchType ASC"),
			ConsistentRead: aws.Bool(false),
		}

		expectedItems := []SupplierBranchShort{
			{
				BranchId:            "BranchId-1",
				IsActive:            "ACTIVE",
				BranchType:          "Store",
				BranchName:          "MumbaiBranch",
				BranchAddressField1: "Andheri",
				BranchCity:          "Mumbai",
			},
			{
				BranchId:            "BranchId-2",
				IsActive:            "ACTIVE",
				BranchType:          "Store",
				BranchName:          "DelhiBranch",
				BranchAddressField1: "Delhi East",
				BranchCity:          "Delhi",
			},
			{
				BranchId:            "BranchId-3",
				IsActive:            "ACTIVE",
				BranchType:          "Store",
				BranchName:          "Bengalore",
				BranchAddressField1: "Bengalore North",
				BranchCity:          "Koramangala",
			},
		}

		output, err := svc.GetActiveBranches()

		assert.Equal(t, expectedItems, output)
		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput1, ddbClient.ExecuteStatementInputs[0])

	})
}

func Test_GetBranchDetails(t *testing.T) {
	t.Run("It should get a branch Details when branchId is provided", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: map[string]dynamodb_types.AttributeValue{
						"BranchId":                   &ddb_types.AttributeValueMemberS{Value: "branchId-1"},
						"IsActive":                   &ddb_types.AttributeValueMemberS{Value: "true"},
						"BranchType":                 &ddb_types.AttributeValueMemberS{Value: "Online"},
						"BranchName":                 &ddb_types.AttributeValueMemberS{Value: "Online Card"},
						"BranchAddressField1":        &ddb_types.AttributeValueMemberS{Value: "123 Main St"},
						"BranchAddressField2":        &ddb_types.AttributeValueMemberS{Value: "Suite 4B"},
						"BranchArea":                 &ddb_types.AttributeValueMemberS{Value: "Central"},
						"BranchCity":                 &ddb_types.AttributeValueMemberS{Value: "Delhi"},
						"BranchState":                &ddb_types.AttributeValueMemberS{Value: "Delhi"},
						"BranchPinCode":              &ddb_types.AttributeValueMemberS{Value: "110001"},
						"BranchLocLat":               &ddb_types.AttributeValueMemberS{Value: "28.6139"},
						"BranchLocLng":               &ddb_types.AttributeValueMemberS{Value: "77.2090"},
						"BranchPrimaryContactName":   &ddb_types.AttributeValueMemberS{Value: "John Doe"},
						"BranchPrimaryPh":            &ddb_types.AttributeValueMemberS{Value: "1234567890"},
						"BranchPrimaryEmail":         &ddb_types.AttributeValueMemberS{Value: "john.doe@example.com"},
						"BranchSecondaryContactName": &ddb_types.AttributeValueMemberS{Value: "Jane Smith"},
						"BranchSecondaryPh":          &ddb_types.AttributeValueMemberS{Value: "0987654321"},
						"BranchSecondaryEmail":       &ddb_types.AttributeValueMemberS{Value: "jane.smith@example.com"},
					},
				},
			},
			GetItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierService{
			dynamodbClient:                    &ddbClient,
			logger:                            log.New(logBuffer, "TEST:", 0),
			SupplierBranchTable:               "SupplierBranchTable",
			SupplierBranchTable_IsActiveIndex: "IsActive_Index",
		}

		expectedOutput := SupplierBranch{
			BranchId:                   "branchId-1",
			IsActive:                   "true",
			BranchType:                 "Online",
			BranchName:                 "Online Card",
			BranchAddressField1:        "123 Main St",
			BranchAddressField2:        "Suite 4B",
			BranchArea:                 "Central",
			BranchCity:                 "Delhi",
			BranchState:                "Delhi",
			BranchPinCode:              "110001",
			BranchLocLat:               "28.6139",
			BranchLocLng:               "77.2090",
			BranchPrimaryContactName:   "John Doe",
			BranchPrimaryPh:            "1234567890",
			BranchPrimaryEmail:         "john.doe@example.com",
			BranchSecondaryContactName: "Jane Smith",
			BranchSecondaryPh:          "0987654321",
			BranchSecondaryEmail:       "jane.smith@example.com",
		}

		output, err := svc.GetBranchDetails("branchId-1")

		assert.NoError(t, err)

		assert.Equal(t, expectedOutput, output)

	})
}

func Test_CreateSupplierBranch(t *testing.T) {

	t.Run("It should create a branch when all inputs are provided", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierService{
			dynamodbClient:                    &ddbClient,
			logger:                            log.New(logBuffer, "TEST:", 0),
			SupplierBranchTable:               "SupplierBranchTable",
			SupplierBranchTable_IsActiveIndex: "IsActive_Index",
		}

		supplierBranchInput := SupplierBranch{
			BranchId:                   "branchId-1", // Empty this field to test the auto generated branchId
			IsActive:                   "true",
			BranchType:                 "Online",
			BranchName:                 "Online Card",
			BranchAddressField1:        "123 Main St",
			BranchAddressField2:        "Suite 4B",
			BranchArea:                 "Central",
			BranchCity:                 "Delhi",
			BranchState:                "Delhi",
			BranchPinCode:              "110001",
			BranchLocLat:               "28.6139",
			BranchLocLng:               "77.2090",
			BranchPrimaryContactName:   "John Doe",
			BranchPrimaryPh:            "1234567890",
			BranchPrimaryEmail:         "john.doe@example.com",
			BranchSecondaryContactName: "Jane Smith",
			BranchSecondaryPh:          "0987654321",
			BranchSecondaryEmail:       "jane.smith@example.com",
		}

		expectedDDBInput := dynamodb.PutItemInput{
			TableName:           aws.String("SupplierBranchTable"),
			ConditionExpression: aws.String("attribute_not_exists(BranchId)"),
			Item: map[string]ddb_types.AttributeValue{
				"BranchId":                   &ddb_types.AttributeValueMemberS{Value: "branchId-1"},
				"IsActive":                   &ddb_types.AttributeValueMemberS{Value: "true"},
				"BranchType":                 &ddb_types.AttributeValueMemberS{Value: "Online"},
				"BranchName":                 &ddb_types.AttributeValueMemberS{Value: "Online Card"},
				"BranchAddressField1":        &ddb_types.AttributeValueMemberS{Value: "123 Main St"},
				"BranchAddressField2":        &ddb_types.AttributeValueMemberS{Value: "Suite 4B"},
				"BranchArea":                 &ddb_types.AttributeValueMemberS{Value: "Central"},
				"BranchCity":                 &ddb_types.AttributeValueMemberS{Value: "Delhi"},
				"BranchState":                &ddb_types.AttributeValueMemberS{Value: "Delhi"},
				"BranchPinCode":              &ddb_types.AttributeValueMemberS{Value: "110001"},
				"BranchLocLat":               &ddb_types.AttributeValueMemberS{Value: "28.6139"},
				"BranchLocLng":               &ddb_types.AttributeValueMemberS{Value: "77.2090"},
				"BranchPrimaryContactName":   &ddb_types.AttributeValueMemberS{Value: "John Doe"},
				"BranchPrimaryPh":            &ddb_types.AttributeValueMemberS{Value: "1234567890"},
				"BranchPrimaryEmail":         &ddb_types.AttributeValueMemberS{Value: "john.doe@example.com"},
				"BranchSecondaryContactName": &ddb_types.AttributeValueMemberS{Value: "Jane Smith"},
				"BranchSecondaryPh":          &ddb_types.AttributeValueMemberS{Value: "0987654321"},
				"BranchSecondaryEmail":       &ddb_types.AttributeValueMemberS{Value: "jane.smith@example.com"},
			},
		}

		err := svc.CreateSupplierBranch(supplierBranchInput)

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.PutItemInputs[0])

	})
}

func Test_UpdateSupplierBranch(t *testing.T) {
	t.Run("It should Update the supplier Branches when update is provided", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierService{
			dynamodbClient:                    &ddbClient,
			logger:                            log.New(logBuffer, "TEST:", 0),
			SupplierBranchTable:               "SupplierBranchTable",
			SupplierBranchTable_IsActiveIndex: "IsActive_Index",
		}

		supplierBranchInput := SupplierBranch{
			BranchId:                   "branchId-1",
			IsActive:                   "true",
			BranchType:                 "Online",
			BranchName:                 "Online Card",
			BranchAddressField1:        "123 Main St",
			BranchAddressField2:        "Suite 4B",
			BranchArea:                 "Central",
			BranchCity:                 "Delhi",
			BranchState:                "Delhi",
			BranchPinCode:              "110001",
			BranchLocLat:               "28.6139",
			BranchLocLng:               "77.2090",
			BranchPrimaryContactName:   "John Doe",
			BranchPrimaryPh:            "1234567890",
			BranchPrimaryEmail:         "john.doe@example.com",
			BranchSecondaryContactName: "Jane Smith",
			BranchSecondaryPh:          "0987654321",
			BranchSecondaryEmail:       "jane.smith@example.com",
		}

		expectedDDBInput := dynamodb.PutItemInput{
			TableName:           aws.String("SupplierBranchTable"),
			ConditionExpression: aws.String("attribute_exists(BranchId)"),
			Item: map[string]ddb_types.AttributeValue{
				"BranchId":                   &ddb_types.AttributeValueMemberS{Value: "branchId-1"},
				"IsActive":                   &ddb_types.AttributeValueMemberS{Value: "true"},
				"BranchType":                 &ddb_types.AttributeValueMemberS{Value: "Online"},
				"BranchName":                 &ddb_types.AttributeValueMemberS{Value: "Online Card"},
				"BranchAddressField1":        &ddb_types.AttributeValueMemberS{Value: "123 Main St"},
				"BranchAddressField2":        &ddb_types.AttributeValueMemberS{Value: "Suite 4B"},
				"BranchArea":                 &ddb_types.AttributeValueMemberS{Value: "Central"},
				"BranchCity":                 &ddb_types.AttributeValueMemberS{Value: "Delhi"},
				"BranchState":                &ddb_types.AttributeValueMemberS{Value: "Delhi"},
				"BranchPinCode":              &ddb_types.AttributeValueMemberS{Value: "110001"},
				"BranchLocLat":               &ddb_types.AttributeValueMemberS{Value: "28.6139"},
				"BranchLocLng":               &ddb_types.AttributeValueMemberS{Value: "77.2090"},
				"BranchPrimaryContactName":   &ddb_types.AttributeValueMemberS{Value: "John Doe"},
				"BranchPrimaryPh":            &ddb_types.AttributeValueMemberS{Value: "1234567890"},
				"BranchPrimaryEmail":         &ddb_types.AttributeValueMemberS{Value: "john.doe@example.com"},
				"BranchSecondaryContactName": &ddb_types.AttributeValueMemberS{Value: "Jane Smith"},
				"BranchSecondaryPh":          &ddb_types.AttributeValueMemberS{Value: "0987654321"},
				"BranchSecondaryEmail":       &ddb_types.AttributeValueMemberS{Value: "jane.smith@example.com"},
			},
		}

		err := svc.UpdateSupplierBranch(supplierBranchInput)

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.PutItemInputs[0])

	})

	t.Run("It should Fail to Update the supplier Branches when BranchId is empty", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{}
		logBuffer := &bytes.Buffer{}

		svc := SupplierService{
			dynamodbClient:                    &ddbClient,
			logger:                            log.New(logBuffer, "TEST:", 0),
			SupplierBranchTable:               "SupplierBranchTable",
			SupplierBranchTable_IsActiveIndex: "IsActive_Index",
		}

		supplierBranchInput := SupplierBranch{
			BranchId:                   "",
			IsActive:                   "true",
			BranchType:                 "Online",
			BranchName:                 "Online Card",
			BranchAddressField1:        "123 Main St",
			BranchAddressField2:        "Suite 4B",
			BranchArea:                 "Central",
			BranchCity:                 "Delhi",
			BranchState:                "Delhi",
			BranchPinCode:              "110001",
			BranchLocLat:               "28.6139",
			BranchLocLng:               "77.2090",
			BranchPrimaryContactName:   "John Doe",
			BranchPrimaryPh:            "1234567890",
			BranchPrimaryEmail:         "john.doe@example.com",
			BranchSecondaryContactName: "Jane Smith",
			BranchSecondaryPh:          "0987654321",
			BranchSecondaryEmail:       "jane.smith@example.com",
		}

		err := svc.UpdateSupplierBranch(supplierBranchInput)

		assert.Equal(t, fmt.Errorf("branchId is required for update"), err)

	})

}

func Test_DeleteSupplierBranch(t *testing.T) {
	t.Run("It should delete the branch when all criteria is met", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			DeleteItemOutputs: []dynamodb.DeleteItemOutput{
				{},
			},
			DeleteItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SupplierService{
			dynamodbClient:                    &ddbClient,
			logger:                            log.New(logBuffer, "TEST:", 0),
			SupplierBranchTable:               "SupplierBranchTable",
			SupplierBranchTable_IsActiveIndex: "IsActive_Index",
		}

		err := svc.DeleteSupplierBranch("branchId-1")

		expectedDDBInput := dynamodb.DeleteItemInput{
			TableName: aws.String("SupplierBranchTable"),
			Key: map[string]dynamodb_types.AttributeValue{
				"BranchId": &dynamodb_types.AttributeValueMemberS{Value: "branchId-1"},
			},
			ConditionExpression: aws.String("IsActive = :InActive"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				"InActive": &dynamodb_types.AttributeValueMemberS{Value: BRANCH_ISACTIVE_FALSE},
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.DeleteItemInputs[0])

	})
}
