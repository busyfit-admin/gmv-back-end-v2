package adminlib

import (
	"bytes"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_CheckSubDomainsAvailability(t *testing.T) {

	t.Run("It should return true when there are no subdomains found", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 0,
				},
			},
			QueryErrors: []error{
				nil,
			},
		}

		logBuffer := &bytes.Buffer{}

		svc := SubDomainService{
			logger:         log.New(logBuffer, "TEST:", 0),
			dynamodbClient: &ddbClient,

			SupplierSubDomainsTable:           "Subdomain-table",
			SupplierSubDomains_SubDomainIndex: "SubDomain-Index-1",
		}

		expectedDDBInput := dynamodb.QueryInput{
			TableName:              aws.String("Subdomain-table"),
			IndexName:              aws.String("SubDomain-Index-1"),
			KeyConditionExpression: aws.String("SubDomain = :SubDomain"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":SubDomain": &dynamodb_types.AttributeValueMemberS{Value: "testdomain"},
			},
		}

		status, err := svc.CheckSubDomainsAvailability("testdomain")

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.QueryInputs[0])
		assert.Equal(t, true, status)
	})
	t.Run("It should return false when there are there subdomains found", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Count: 1,
				},
			},
			QueryErrors: []error{
				nil,
			},
		}

		logBuffer := &bytes.Buffer{}

		svc := SubDomainService{
			logger:         log.New(logBuffer, "TEST:", 0),
			dynamodbClient: &ddbClient,

			SupplierSubDomainsTable:           "Subdomain-table",
			SupplierSubDomains_SubDomainIndex: "SubDomain-Index-1",
		}

		status, err := svc.CheckSubDomainsAvailability("test")

		assert.NoError(t, err)
		assert.Equal(t, false, status)
	})
}

func Test_CreateSupplierSubDomain(t *testing.T) {

	t.Run("It should Create the SubDomain when all the inputs are provided correctly", func(t *testing.T) {

		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{
					ConsumedCapacity: nil,
				},
			},
			PutItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SubDomainService{
			logger:         log.New(logBuffer, "TEST:", 0),
			dynamodbClient: &ddbClient,

			SupplierSubDomainsTable:           "Subdomain-table",
			SupplierSubDomains_SubDomainIndex: "SubDomain-Index-1",
		}

		err := svc.CreateSupplierSubDomain(CreateSubDomainInput{
			SupplierId:  "test-SupplierId",
			SubDomain: "test-domain",
			EnvName:   "PROD",
		})

		expectedDDBInput := dynamodb.PutItemInput{
			TableName: aws.String("Subdomain-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"SupplierId":  &dynamodb_types.AttributeValueMemberS{Value: "test-SupplierId"},
				"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: "test-domain"},
				"Status":    &dynamodb_types.AttributeValueMemberS{Value: "STACK_INPROG"},
				"EnvName":   &dynamodb_types.AttributeValueMemberS{Value: "PROD"},
			},
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.PutItemInputs[0])

	})
}

func Test_UpdateSubDomainStackInfo(t *testing.T) {
	t.Run("It should Update the SubDomain Table with Supplier StackName and CognitoUserPool Id", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SubDomainService{
			logger:         log.New(logBuffer, "TEST:", 0),
			dynamodbClient: &ddbClient,

			SupplierSubDomainsTable:           "Subdomain-table",
			SupplierSubDomains_SubDomainIndex: "SubDomain-Index-1",
		}

		err := svc.UpdateSubDomainStackInfo(UpdateSubDomainStackInfo{
			SubDomain: "test-subdomain",
			SupplierId:  "testId-1",

			SupplierStackName:  "stack-test",
			SupplierUserPoolId: "userpool-123",
		})

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("Subdomain-table"),
			Key: map[string]dynamodb_types.AttributeValue{
				"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: "test-subdomain"},
				"SupplierId":  &dynamodb_types.AttributeValueMemberS{Value: "testId-1"},
			},
			UpdateExpression: aws.String("SET SupplierStackName = :SupplierStackName, SupplierUserPoolId = :SupplierUserPoolId, Status = :Status"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":SupplierStackName":  &dynamodb_types.AttributeValueMemberS{Value: "stack-test"},
				":SupplierUserPoolId": &dynamodb_types.AttributeValueMemberS{Value: "userpool-123"},
				":Status":           &dynamodb_types.AttributeValueMemberS{Value: "STACK_DEPLOYED"},
			},
			ReturnValues: dynamodb_types.ReturnValueNone,
		}

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs[0])
	})
}

func Test_GetAllSupplierSubDomains(t *testing.T) {
	t.Run("It should Get all Supplier SubDomains when correct Inputs are sent", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{
			QueryOutputs: []dynamodb.QueryOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: "domain-a"},
							"SupplierId":  &dynamodb_types.AttributeValueMemberS{Value: "Supplier-a"},

							"Status":           &dynamodb_types.AttributeValueMemberS{Value: "STACK_DEPLOYED"},
							"EnvName":          &dynamodb_types.AttributeValueMemberS{Value: "PROD"},
							"SupplierStackName":  &dynamodb_types.AttributeValueMemberS{Value: "stack-a"},
							"SupplierUserPoolId": &dynamodb_types.AttributeValueMemberS{Value: "userpool-123"},
							"AdminUsers": &dynamodb_types.AttributeValueMemberL{
								Value: []dynamodb_types.AttributeValue{
									&dynamodb_types.AttributeValueMemberS{Value: "test@gmail.com"},
									&dynamodb_types.AttributeValueMemberS{Value: "test2@gmail.com"},
								},
							},
						},
						{
							"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: "domain-b"},
							"SupplierId":  &dynamodb_types.AttributeValueMemberS{Value: "Supplier-a"},

							"Status":           &dynamodb_types.AttributeValueMemberS{Value: "STACK_DEPLOYED"},
							"EnvName":          &dynamodb_types.AttributeValueMemberS{Value: "UAT"},
							"SupplierStackName":  &dynamodb_types.AttributeValueMemberS{Value: "stack-b"},
							"SupplierUserPoolId": &dynamodb_types.AttributeValueMemberS{Value: "userpool-345"},
							"AdminUsers": &dynamodb_types.AttributeValueMemberL{
								Value: []dynamodb_types.AttributeValue{
									&dynamodb_types.AttributeValueMemberS{Value: "test@gmail.com"},
									&dynamodb_types.AttributeValueMemberS{Value: "test2@gmail.com"},
								},
							},
						},
					},
					Count: 2,
				},
			},
			QueryErrors: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SubDomainService{
			logger:         log.New(logBuffer, "TEST:", 0),
			dynamodbClient: &ddbClient,

			SupplierSubDomainsTable:           "Subdomain-table",
			SupplierSubDomains_SubDomainIndex: "SubDomain-Index-1",
		}

		output, err := svc.GetAllSupplierSubDomains("Supplier-a")

		expectedOutput := []SupplierSubDomainsTable{
			{
				SubDomain: "domain-a",
				SupplierId:  "Supplier-a",

				Status:  "STACK_DEPLOYED",
				EnvName: "PROD",

				SupplierStack:      "stack-a",
				SupplierUserPoolId: "userpool-123",

				AdminUsers: []string{
					"test@gmail.com",
					"test2@gmail.com",
				},
			},
			{
				SubDomain: "domain-b",
				SupplierId:  "Supplier-a",

				Status:  "STACK_DEPLOYED",
				EnvName: "UAT",

				SupplierStack:      "stack-b",
				SupplierUserPoolId: "userpool-345",

				AdminUsers: []string{
					"test@gmail.com",
					"test2@gmail.com",
				},
			},
		}

		assert.Equal(t, expectedOutput, output)
		assert.NoError(t, err)

	})
}

func Test_AddSubDomainAdmin(t *testing.T) {
	t.Run("it should add the admin user to cognito and dynamodb when correct Input is provided", func(t *testing.T) {
		ddbClient := awsclients.MockDynamodbClient{

			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: map[string]dynamodb_types.AttributeValue{
						"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: "domain-a"},
						"SupplierId":  &dynamodb_types.AttributeValueMemberS{Value: "Supplier-a"},

						"Status":           &dynamodb_types.AttributeValueMemberS{Value: "STACK_DEPLOYED"},
						"EnvName":          &dynamodb_types.AttributeValueMemberS{Value: "PROD"},
						"SupplierStackName":  &dynamodb_types.AttributeValueMemberS{Value: "stack-a"},
						"SupplierUserPoolId": &dynamodb_types.AttributeValueMemberS{Value: "userpool-123"},
						"AdminUsers": &dynamodb_types.AttributeValueMemberL{
							Value: []dynamodb_types.AttributeValue{
								&dynamodb_types.AttributeValueMemberS{Value: "test@gmail.com"},
								&dynamodb_types.AttributeValueMemberS{Value: "test2@gmail.com"},
							},
						},
					},
				},
			},
			GetItemErrors: []error{
				nil,
			},
			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}

		cognitoClient := awsclients.MockCognitoClient{
			AdminCreateUserOutput: []cognitoidentityprovider.AdminCreateUserOutput{
				{},
			},
			AdminCreateUserError: []error{
				nil,
			},
		}
		logBuffer := &bytes.Buffer{}

		svc := SubDomainService{
			logger:         log.New(logBuffer, "TEST:", 0),
			dynamodbClient: &ddbClient,
			cognitoClient:  &cognitoClient,

			SupplierSubDomainsTable:           "Subdomain-table",
			SupplierSubDomains_SubDomainIndex: "SubDomain-Index-1",
		}

		expectedDDBGetItemInput := dynamodb.GetItemInput{
			TableName: aws.String("Subdomain-table"),
			Key: map[string]dynamodb_types.AttributeValue{
				"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: "test"},
				"SupplierId":  &dynamodb_types.AttributeValueMemberS{Value: "Supplierid-123"},
			},
			ConsistentRead: aws.Bool(true),
		}
		expectedDDBQueryInput := dynamodb.UpdateItemInput{
			TableName: aws.String(svc.SupplierSubDomainsTable),
			Key: map[string]dynamodb_types.AttributeValue{
				"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: "test"},
				"SupplierId":  &dynamodb_types.AttributeValueMemberS{Value: "Supplierid-123"},
			},
			UpdateExpression: aws.String("SET AdminUsers = list_append(if_not_exists(AdminUsers, :EmptyList), :AdminUser)"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":AdminUser": &dynamodb_types.AttributeValueMemberL{
					Value: []dynamodb_types.AttributeValue{
						&dynamodb_types.AttributeValueMemberS{Value: "user@gmail.com"},
					},
				},
				":EmptyList": &dynamodb_types.AttributeValueMemberL{},
			},
			ReturnValues: dynamodb_types.ReturnValueNone,
		}
		expectedCognitoInput := cognitoidentityprovider.AdminCreateUserInput{
			UserPoolId: aws.String("userpool-123"),
			Username:   aws.String("user@gmail.com"),
		}

		err := svc.AddSubDomainAdmin(SubDomainAdmin{
			SubDomain:   "test",
			SupplierId:    "Supplierid-123",
			AdminUserId: "user@gmail.com",
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBGetItemInput, ddbClient.GetItemInputs[0])
		assert.Equal(t, expectedDDBQueryInput, ddbClient.UpdateItemInputs[0])
		assert.Equal(t, expectedCognitoInput, cognitoClient.AdminCreateUserInput[0])
	})
}
