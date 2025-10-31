package Companylib

import (
	"bytes"
	"fmt"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	lib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_GetEmployeeDataByUserName(t *testing.T) {

	employeeTestData1 := EmployeeDynamodbData{
		CognitoId:  "cogn-123",
		UserName:   "test-123",
		ProfilePic: "users/profilepics/test-123.png",
	}

	logBuffer := &bytes.Buffer{}
	ddbAttData1, _ := attributevalue.MarshalMap(employeeTestData1)

	t.Run("It should handle the request when all parameters are set correctly", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: ddbAttData1,
				},
			},
			GetItemErrors: []error{
				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),
			EmployeeTable:  "test-employeeTable",
		}

		expectedDDBInput := []dynamodb.GetItemInput{
			{
				TableName: aws.String("test-employeeTable"),
				Key: map[string]types.AttributeValue{
					"UserName": &dynamodb_types.AttributeValueMemberS{Value: "test-123"},
				},
				ConsistentRead: aws.Bool(true),
			},
		}

		output, err := svc.GetEmployeeDataByUserName("test-123")

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.GetItemInputs)
		assert.Equal(t, employeeTestData1, output)
	})

	t.Run("It should return the error when DDB has an error", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{},
			},
			GetItemErrors: []error{
				fmt.Errorf("Get Item not found"),
			},
		}

		svc := EmployeeService{
			dynamodbClient: &ddbClient,
			logger:         log.New(logBuffer, "TEST:", 0),
			EmployeeTable:  "test-employeeTable",
		}

		_, err := svc.GetEmployeeDataByUserName("test-123")

		assert.Equal(t, fmt.Errorf("Get Item not found"), err)

	})
}

func Test_GetEmployeeDataByEmail(t *testing.T) {
	employeeTestData1 := EmployeeDynamodbData{

		CognitoId:  "cogn-123",
		UserName:   "test-123",
		EmailID:    "test-employee-emailId-123",
		ProfilePic: "default.jpeg",
	}
	logBuffer := &bytes.Buffer{}

	t.Run("It should handle the request when all parameters are set correctly", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 1,
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"UserName":  &dynamodb_types.AttributeValueMemberS{Value: "test-123"},
							"CognitoId": &dynamodb_types.AttributeValueMemberS{Value: "cogn-123"},
							"EmailId":   &dynamodb_types.AttributeValueMemberS{Value: "test-employee-emailId-123"},
						},
					},
				},
			},
			QueryErrors: []error{
				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient:              &ddbClient,
			logger:                      log.New(logBuffer, "TEST:", 0),
			EmployeeTable:               "test-employee-table",
			EmployeeTable_EmailId_Index: "test-employee-usernameIndex",
		}

		expectedQueryInput := []dynamodb.QueryInput{
			{
				TableName:              aws.String("test-employee-table"),
				IndexName:              aws.String("test-employee-usernameIndex"),
				KeyConditionExpression: aws.String("EmailId = :EmailId"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":EmailId": &dynamodb_types.AttributeValueMemberS{Value: "test-employee-emailId-123"},
				},
			},
		}

		output, err := svc.GetEmployeeDataByEmail("test-employee-emailId-123")

		assert.NoError(t, err)
		assert.Equal(t, employeeTestData1, output)
		assert.Equal(t, expectedQueryInput, ddbClient.QueryInputs)
	})

	t.Run("It should handle the request correctly when there is zero items from query", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
					Items: []map[string]dynamodb_types.AttributeValue{},
				},
			},
			QueryErrors: []error{
				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient:              &ddbClient,
			logger:                      log.New(logBuffer, "TEST:", 0),
			EmployeeTable:               "test-employee-table",
			EmployeeTable_EmailId_Index: "test-employee-usernameIndex",
		}

		expectedQueryInput := []dynamodb.QueryInput{
			{
				TableName:              aws.String("test-employee-table"),
				IndexName:              aws.String("test-employee-usernameIndex"),
				KeyConditionExpression: aws.String("EmailId = :EmailId"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":EmailId": &dynamodb_types.AttributeValueMemberS{Value: "test-employee-emailId-123"},
				},
			},
		}

		Expectedemployee := EmployeeDynamodbData{}

		output, err := svc.GetEmployeeDataByEmail("test-employee-emailId-123")

		assert.NoError(t, err)
		assert.Equal(t, Expectedemployee, output)
		assert.Equal(t, expectedQueryInput, ddbClient.QueryInputs)
	})

	t.Run("It should handle the error correctly when the exception is ConditionalCheckFail", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
					Items: []map[string]dynamodb_types.AttributeValue{},
				},
			},
			QueryErrors: []error{
				&dynamodb_types.ConditionalCheckFailedException{},
			},
		}

		svc := EmployeeService{
			dynamodbClient:              &ddbClient,
			logger:                      log.New(logBuffer, "TEST:", 0),
			EmployeeTable:               "test-employee-table",
			EmployeeTable_EmailId_Index: "test-employee-usernameIndex",
		}

		expectedQueryInput := []dynamodb.QueryInput{
			{
				TableName:              aws.String("test-employee-table"),
				IndexName:              aws.String("test-employee-usernameIndex"),
				KeyConditionExpression: aws.String("EmailId = :EmailId"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":EmailId": &dynamodb_types.AttributeValueMemberS{Value: "test-employee-emailId-123"},
				},
			},
		}

		Expectedemployee := EmployeeDynamodbData{}

		output, err := svc.GetEmployeeDataByEmail("test-employee-emailId-123")

		assert.NoError(t, err)
		assert.Equal(t, Expectedemployee, output)
		assert.Equal(t, expectedQueryInput, ddbClient.QueryInputs)
	})

	t.Run("It should handle the all other errors correctly", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
					Items: []map[string]dynamodb_types.AttributeValue{},
				},
			},
			QueryErrors: []error{
				fmt.Errorf("ddb query failed"),
			},
		}

		svc := EmployeeService{
			dynamodbClient:              &ddbClient,
			logger:                      log.New(logBuffer, "TEST:", 0),
			EmployeeTable:               "test-employee-table",
			EmployeeTable_EmailId_Index: "test-employee-usernameIndex",
		}

		_, err := svc.GetEmployeeDataByEmail("test-employee-emailId-123")

		assert.Equal(t, fmt.Errorf("ddb query failed"), err)

	})

}

func Test_GetEmployeeDataByExternalId(t *testing.T) {
	employeeTestData1 := EmployeeDynamodbData{

		CognitoId:  "cogn-123",
		UserName:   "test-123",
		EmailID:    "test-employee-emailId-123",
		ExternalId: "123456",
		ProfilePic: "default.jpeg",
	}
	logBuffer := &bytes.Buffer{}

	t.Run("It should handle the request when all parameters are set correctly", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 1,
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"UserName":   &dynamodb_types.AttributeValueMemberS{Value: "test-123"},
							"CognitoId":  &dynamodb_types.AttributeValueMemberS{Value: "cogn-123"},
							"EmailId":    &dynamodb_types.AttributeValueMemberS{Value: "test-employee-emailId-123"},
							"ExternalId": &dynamodb_types.AttributeValueMemberS{Value: "123456"},
						},
					},
				},
			},
			QueryErrors: []error{
				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient:                 &ddbClient,
			logger:                         log.New(logBuffer, "TEST:", 0),
			EmployeeTable:                  "test-employee-table",
			EmployeeTable_EmailId_Index:    "test-employee-usernameIndex",
			EmployeeTable_ExternalId_Index: "test-employee-externalIdIndex",
		}

		expectedQueryInput := []dynamodb.QueryInput{
			{
				TableName:              aws.String("test-employee-table"),
				IndexName:              aws.String("test-employee-externalIdIndex"),
				KeyConditionExpression: aws.String("ExternalId = :ExternalId"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":ExternalId": &dynamodb_types.AttributeValueMemberS{Value: "123456"},
				},
			},
		}

		output, err := svc.GetEmployeeDataByExternalId("123456")

		assert.NoError(t, err)
		assert.Equal(t, employeeTestData1, output)
		assert.Equal(t, expectedQueryInput, ddbClient.QueryInputs)
	})

	t.Run("It should handle the request correctly when there is zero items from query", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
					Items: []map[string]dynamodb_types.AttributeValue{},
				},
			},
			QueryErrors: []error{
				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient:                 &ddbClient,
			logger:                         log.New(logBuffer, "TEST:", 0),
			EmployeeTable:                  "test-employee-table",
			EmployeeTable_EmailId_Index:    "test-employee-usernameIndex",
			EmployeeTable_ExternalId_Index: "test-employee-externalIdIndex",
		}

		expectedQueryInput := []dynamodb.QueryInput{
			{
				TableName:              aws.String("test-employee-table"),
				IndexName:              aws.String("test-employee-externalIdIndex"),
				KeyConditionExpression: aws.String("ExternalId = :ExternalId"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":ExternalId": &dynamodb_types.AttributeValueMemberS{Value: "123456"},
				},
			},
		}

		Expectedemployee := EmployeeDynamodbData{}

		output, err := svc.GetEmployeeDataByExternalId("123456")

		assert.NoError(t, err)
		assert.Equal(t, Expectedemployee, output)
		assert.Equal(t, expectedQueryInput, ddbClient.QueryInputs)
	})

	t.Run("It should handle the error correctly when the exception is ConditionalCheckFail", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
					Items: []map[string]dynamodb_types.AttributeValue{},
				},
			},
			QueryErrors: []error{
				&dynamodb_types.ConditionalCheckFailedException{},
			},
		}

		svc := EmployeeService{
			dynamodbClient:                 &ddbClient,
			logger:                         log.New(logBuffer, "TEST:", 0),
			EmployeeTable:                  "test-employee-table",
			EmployeeTable_EmailId_Index:    "test-employee-usernameIndex",
			EmployeeTable_ExternalId_Index: "test-employee-externalIdIndex",
		}

		expectedQueryInput := []dynamodb.QueryInput{
			{
				TableName:              aws.String("test-employee-table"),
				IndexName:              aws.String("test-employee-externalIdIndex"),
				KeyConditionExpression: aws.String("ExternalId = :ExternalId"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":ExternalId": &dynamodb_types.AttributeValueMemberS{Value: "123456"},
				},
			},
		}

		Expectedemployee := EmployeeDynamodbData{}

		output, err := svc.GetEmployeeDataByExternalId("123456")

		assert.NoError(t, err)
		assert.Equal(t, Expectedemployee, output)
		assert.Equal(t, expectedQueryInput, ddbClient.QueryInputs)
	})

	t.Run("It should handle the all other errors correctly", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
					Items: []map[string]dynamodb_types.AttributeValue{},
				},
			},
			QueryErrors: []error{
				fmt.Errorf("ddb query failed"),
			},
		}

		svc := EmployeeService{
			dynamodbClient:                 &ddbClient,
			logger:                         log.New(logBuffer, "TEST:", 0),
			EmployeeTable:                  "test-employee-table",
			EmployeeTable_EmailId_Index:    "test-employee-usernameIndex",
			EmployeeTable_ExternalId_Index: "test-employee-externalIdIndex",
		}

		_, err := svc.GetEmployeeDataByExternalId("123456")

		assert.Equal(t, fmt.Errorf("ddb query failed"), err)

	})

}

func Test_GetAllEmployeeData(t *testing.T) {

	logBuffer := &bytes.Buffer{}
	t.Run("It should return all the employee data when scan input is done", func(t *testing.T) {
		ddbClient := lib.MockDynamodbClient{
			ScanOutputs: []dynamodb.ScanOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"UserName":        &dynamodb_types.AttributeValueMemberS{Value: "test-123"},
							"CognitoId":       &dynamodb_types.AttributeValueMemberS{Value: "cogn-123"},
							"EmailId":         &dynamodb_types.AttributeValueMemberS{Value: "test-employee-emailId-123"},
							"ExternalId":      &dynamodb_types.AttributeValueMemberS{Value: "123456"},
							"DisplayName":     &dynamodb_types.AttributeValueMemberS{Value: "Person A"},
							"TopLevelGroupId": &dynamodb_types.AttributeValueMemberS{Value: "1"},
						},
					},
				},
			},
			ScanErrors: []error{
				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient:              &ddbClient,
			logger:                      log.New(logBuffer, "TEST:", 0),
			EmployeeTable:               "test-employee-table",
			EmployeeTable_EmailId_Index: "test-employee-usernameIndex",
		}

		output, err := svc.GetAllEmployeeData()

		expectedScanInput := []dynamodb.ScanInput{
			{
				TableName:            aws.String("test-employee-table"),
				Limit:                aws.Int32(1000),
				ProjectionExpression: aws.String("UserName, EmailId, ExternalId, DisplayName, Designation, IsManager, MgrUserName, StartDate, EndDate, IsActive, RolesData, ProfilePic"),
			},
		}

		expectedOutput := []EmployeeDynamodbData{
			{
				UserName:    "test-123",
				CognitoId:   "cogn-123",
				EmailID:     "test-employee-emailId-123",
				ExternalId:  "123456",
				DisplayName: "Person A",
				ProfilePic:  "default.jpeg",
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)
		assert.Equal(t, expectedScanInput, ddbClient.ScanInputs)

	})
}

func Test_GetAllEmployeeGroupsInMap(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	t.Run("It should load all the employee groups in the map correctly", func(t *testing.T) {

		ddbClient := lib.MockDynamodbClient{

			ScanOutputs: []dynamodb.ScanOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-1"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-1"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "1.this is a test group"},
						},
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-2"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-2"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "2.this is a test group"},
						},
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-3"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-3"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "3.this is a test group"},
						},
					},
					LastEvaluatedKey: nil,
				},
			},
			ScanErrors: []error{
				nil,
			},
		}

		svc := EmployeeService{
			EmployeeGroupsTable: "test-group-table",
			dynamodbClient:      &ddbClient,
			logger:              log.New(logBuffer, "TEST:", 0),
		}

		expectedOutput := map[string]EmployeeGroups{
			"GroupId-1": {
				GroupId:   "GroupId-1",
				GroupName: "test-group-1",
				GroupDesc: "1.this is a test group",
			},
			"GroupId-2": {
				GroupId:   "GroupId-2",
				GroupName: "test-group-2",
				GroupDesc: "2.this is a test group",
			},
			"GroupId-3": {
				GroupId:   "GroupId-3",
				GroupName: "test-group-3",
				GroupDesc: "3.this is a test group",
			},
		}

		expectedScanInput := []dynamodb.ScanInput{
			{
				TableName: aws.String("test-group-table"),
				Limit:     aws.Int32(1000),
			},
		}

		output, err := svc.GetAllEmployeeGroupsInMap()

		assert.NoError(t, err)
		assert.Equal(t, expectedScanInput, ddbClient.ScanInputs)
		assert.Equal(t, expectedOutput, output)

	})

	t.Run("It should load all the employee groups in the map correctly when there is pagination in scan", func(t *testing.T) {

		ddbClient := lib.MockDynamodbClient{

			ScanOutputs: []dynamodb.ScanOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-1"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-1"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "1.this is a test group"},
						},
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-2"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-2"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "2.this is a test group"},
						},
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-3"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-3"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "3.this is a test group"},
						},
					},
					LastEvaluatedKey: map[string]dynamodb_types.AttributeValue{
						"GroupId": &dynamodb_types.AttributeValueMemberS{Value: "GroupId-3"},
					},
				},
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-4"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-4"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "4.this is a test group"},
						},
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-5"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-5"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "5.this is a test group"},
						},
						{
							"GroupId":   &dynamodb_types.AttributeValueMemberS{Value: "GroupId-6"},
							"GroupName": &dynamodb_types.AttributeValueMemberS{Value: "test-group-6"},
							"GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: "6.this is a test group"},
						},
					},
					LastEvaluatedKey: nil,
				},
			},
			ScanErrors: []error{
				nil,
				nil,
			},
		}

		svc := EmployeeService{
			EmployeeGroupsTable: "test-group-table",
			dynamodbClient:      &ddbClient,
			logger:              log.New(logBuffer, "TEST:", 0),
		}

		expectedOutput := map[string]EmployeeGroups{
			"GroupId-1": {
				GroupId:   "GroupId-1",
				GroupName: "test-group-1",
				GroupDesc: "1.this is a test group",
			},
			"GroupId-2": {
				GroupId:   "GroupId-2",
				GroupName: "test-group-2",
				GroupDesc: "2.this is a test group",
			},
			"GroupId-3": {
				GroupId:   "GroupId-3",
				GroupName: "test-group-3",
				GroupDesc: "3.this is a test group",
			},
			"GroupId-4": {
				GroupId:   "GroupId-4",
				GroupName: "test-group-4",
				GroupDesc: "4.this is a test group",
			},
			"GroupId-5": {
				GroupId:   "GroupId-5",
				GroupName: "test-group-5",
				GroupDesc: "5.this is a test group",
			},
			"GroupId-6": {
				GroupId:   "GroupId-6",
				GroupName: "test-group-6",
				GroupDesc: "6.this is a test group",
			},
		}

		expectedScanInput := []dynamodb.ScanInput{
			{
				TableName: aws.String("test-group-table"),
				Limit:     aws.Int32(1000),
			},
			{
				TableName: aws.String("test-group-table"),
				Limit:     aws.Int32(1000),
				ExclusiveStartKey: map[string]dynamodb_types.AttributeValue{
					"GroupId": &dynamodb_types.AttributeValueMemberS{Value: "GroupId-3"},
				},
			},
		}

		output, err := svc.GetAllEmployeeGroupsInMap()

		assert.NoError(t, err)
		assert.Equal(t, expectedScanInput, ddbClient.ScanInputs)
		assert.Equal(t, expectedOutput, output)

	})

	t.Run("It should return error correctly when there is an error in scan", func(t *testing.T) {

		ddbClient := lib.MockDynamodbClient{

			ScanOutputs: []dynamodb.ScanOutput{
				{},
			},
			ScanErrors: []error{
				fmt.Errorf("error while scanning the ddb table"),
			},
		}

		svc := EmployeeService{
			EmployeeGroupsTable: "test-group-table",
			dynamodbClient:      &ddbClient,
			logger:              log.New(logBuffer, "TEST:", 0),
		}

		_, err := svc.GetAllEmployeeGroupsInMap()

		assert.Equal(t, fmt.Errorf("error while scanning the ddb table"), err)

	})

}

func Test_UpdateTenantTeams(t *testing.T) {
	t.Run("It should create a Default team when a IsManager Field is Y and has no other manager", func(t *testing.T) {

		employeeTestData := EmployeeDynamodbData{

			CognitoId:   "cogn-123",
			UserName:    "test-123",
			EmailID:     "test-employee-emailId-123",
			DisplayName: "John",
			IsManager:   "Y",
			IsActive:    "Y",
			MgrUserName: "",
		}
		logBuffer := &bytes.Buffer{}

		ddbClient := lib.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
				{},
			},
			PutItemErrors: []error{
				nil,
				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient:   &ddbClient,
			logger:           log.New(logBuffer, "TEST:", 0),
			TenantTeamsTable: "test-teams-table",
		}

		expectedDDBInput1 := dynamodb.PutItemInput{
			TableName: aws.String("test-teams-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &types.AttributeValueMemberS{Value: "TEAM-test-123"},
				"RelatedEntityId": &types.AttributeValueMemberS{Value: "TEAM-DEFAULT"},
				"TeamTypeId":      &types.AttributeValueMemberS{Value: "General"},
				"TeamName":        &types.AttributeValueMemberS{Value: "John's TEAM (GENERAL)"},
				"IsActive":        &types.AttributeValueMemberS{Value: "Y"},
			},
		}
		expectedDDBInput2 := dynamodb.PutItemInput{
			TableName: aws.String("test-teams-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &types.AttributeValueMemberS{Value: "MNGR-test-123"},
				"RelatedEntityId": &types.AttributeValueMemberS{Value: "TEAM-test-123"},
				"IsActive":        &types.AttributeValueMemberS{Value: "Y"},
			},
		}

		err := svc.UpdateTenantTeams(employeeTestData)

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput1, ddbClient.PutItemInputs[0])
		assert.Equal(t, expectedDDBInput2, ddbClient.PutItemInputs[1])

	})

	t.Run("It should create a Default team when a IsManager Field is Y and also create a user when managerId is provided", func(t *testing.T) {

		employeeTestData := EmployeeDynamodbData{

			CognitoId:   "cogn-123",
			UserName:    "test-123",
			EmailID:     "test-employee-emailId-123",
			DisplayName: "John",
			IsManager:   "Y",
			IsActive:    "Y",
			MgrUserName: "admin-123",
		}
		logBuffer := &bytes.Buffer{}

		ddbClient := lib.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
				{},
				{},
			},
			PutItemErrors: []error{
				nil,
				nil,
				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient:   &ddbClient,
			logger:           log.New(logBuffer, "TEST:", 0),
			TenantTeamsTable: "test-teams-table",
		}

		expectedDDBInput1 := dynamodb.PutItemInput{
			TableName: aws.String("test-teams-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &types.AttributeValueMemberS{Value: "TEAM-test-123"},
				"RelatedEntityId": &types.AttributeValueMemberS{Value: "TEAM-DEFAULT"},
				"TeamTypeId":      &types.AttributeValueMemberS{Value: "General"},
				"TeamName":        &types.AttributeValueMemberS{Value: "John's TEAM (GENERAL)"},
				"IsActive":        &types.AttributeValueMemberS{Value: "Y"},
			},
		}
		expectedDDBInput2 := dynamodb.PutItemInput{
			TableName: aws.String("test-teams-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &types.AttributeValueMemberS{Value: "MNGR-test-123"},
				"RelatedEntityId": &types.AttributeValueMemberS{Value: "TEAM-test-123"},
				"IsActive":        &types.AttributeValueMemberS{Value: "Y"},
			},
		}
		expectedDDBInput3 := dynamodb.PutItemInput{
			TableName: aws.String("test-teams-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &types.AttributeValueMemberS{Value: "USER-test-123"},
				"RelatedEntityId": &types.AttributeValueMemberS{Value: "TEAM-admin-123"},
				"IsActive":        &types.AttributeValueMemberS{Value: "Y"},
			},
		}

		err := svc.UpdateTenantTeams(employeeTestData)

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput1, ddbClient.PutItemInputs[0])
		assert.Equal(t, expectedDDBInput2, ddbClient.PutItemInputs[1])
		assert.Equal(t, expectedDDBInput3, ddbClient.PutItemInputs[2])

	})

	t.Run("It should create only user when IsManager is N and managerId is provided", func(t *testing.T) {

		employeeTestData := EmployeeDynamodbData{

			CognitoId:   "cogn-123",
			UserName:    "test-123",
			EmailID:     "test-employee-emailId-123",
			DisplayName: "John",
			IsManager:   "N",
			IsActive:    "Y",
			MgrUserName: "admin-123",
		}
		logBuffer := &bytes.Buffer{}

		ddbClient := lib.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{

				{},
			},
			PutItemErrors: []error{

				nil,
			},
		}

		svc := EmployeeService{
			dynamodbClient:   &ddbClient,
			logger:           log.New(logBuffer, "TEST:", 0),
			TenantTeamsTable: "test-teams-table",
		}

		expectedDDBInput3 := dynamodb.PutItemInput{
			TableName: aws.String("test-teams-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &types.AttributeValueMemberS{Value: "USER-test-123"},
				"RelatedEntityId": &types.AttributeValueMemberS{Value: "TEAM-admin-123"},
				"IsActive":        &types.AttributeValueMemberS{Value: "Y"},
			},
		}

		err := svc.UpdateTenantTeams(employeeTestData)

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput3, ddbClient.PutItemInputs[0])

	})

	t.Run("It should not create any user or manager team when Is Manager is N and ManagerUserName is null", func(t *testing.T) {

		employeeTestData := EmployeeDynamodbData{

			CognitoId:   "cogn-123",
			UserName:    "test-123",
			EmailID:     "test-employee-emailId-123",
			DisplayName: "John",
			IsManager:   "N",
			IsActive:    "Y",
			MgrUserName: "",
		}
		logBuffer := &bytes.Buffer{}

		ddbClient := lib.MockDynamodbClient{}

		svc := EmployeeService{
			dynamodbClient:   &ddbClient,
			logger:           log.New(logBuffer, "TEST:", 0),
			TenantTeamsTable: "test-teams-table",
		}

		err := svc.UpdateTenantTeams(employeeTestData)

		assert.NoError(t, err)

		assert.Equal(t, ddbClient.PutItemInputs, []dynamodb.PutItemInput(nil))

	})

}
