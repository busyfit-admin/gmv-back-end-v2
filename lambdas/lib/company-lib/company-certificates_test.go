package Companylib

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/stretchr/testify/assert"
)

func Test_GetAllTenantCertificates(t *testing.T) {
	t.Run("It should return all active and inactive tenant certificates", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"CertificateId":   &dynamodb_types.AttributeValueMemberS{Value: "certificate-1"},
							"CertificateName": &dynamodb_types.AttributeValueMemberS{Value: "First Certificate"},
							"Criteria":        &dynamodb_types.AttributeValueMemberS{Value: "Skills"},
							"Threshold":       &dynamodb_types.AttributeValueMemberN{Value: "100"},
							"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: "Active"},
						},
					},
				},
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"CertificateId":   &dynamodb_types.AttributeValueMemberS{Value: "certificate-2"},
							"CertificateName": &dynamodb_types.AttributeValueMemberS{Value: "Second Certificate"},
							"Criteria":        &dynamodb_types.AttributeValueMemberS{Value: "Anniversary"},
							"Threshold":       &dynamodb_types.AttributeValueMemberN{Value: "5"},
							"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: "Inactive"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil, // No error for active certificates query
				nil, // No error for inactive certificates query
			},
		}

		svc := TenantCertificatesService{
			ctx:                     context.TODO(),
			dynamodbClient:          &ddbClient,
			logger:                  log.New(logBuffer, "TEST:", 0),
			TenantCertificatesTable: "TenantCertificatesTable",
		}

		result, err := svc.GetAllTenantCertificates()

		assert.NoError(t, err)

		assert.Equal(t, "certificate-1", result.Active[0].CertificateId)
		assert.Equal(t, "certificate-2", result.Draft[0].CertificateId)
		assert.Equal(t, "Skills", result.Active[0].Criteria)
		assert.Equal(t, "Anniversary", result.Draft[0].Criteria)
	})
}

func Test_GetCertificatesData(t *testing.T) {
	t.Run("It should return certificates data for a given query", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"CertificateId":   &dynamodb_types.AttributeValueMemberS{Value: "certificate-1"},
							"CertificateName": &dynamodb_types.AttributeValueMemberS{Value: "Test Certificate"},
							"Criteria":        &dynamodb_types.AttributeValueMemberS{Value: "Skills"},
							"Threshold":       &dynamodb_types.AttributeValueMemberN{Value: "100"},
							"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: "Active"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil, // No error
			},
		}

		svc := TenantCertificatesService{
			ctx:                     context.TODO(),
			dynamodbClient:          &ddbClient,
			logger:                  log.New(logBuffer, "TEST:", 0),
			TenantCertificatesTable: "TenantCertificatesTable",
		}

		query := "SELECT * FROM \"TenantCertificatesTable\" WHERE IsActive = 'Active'"

		result, err := svc.GetCertificatesData(query)

		assert.NoError(t, err)

		assert.Equal(t, "certificate-1", result[0].CertificateId)
		assert.Equal(t, "Test Certificate", result[0].CertificateName)
		assert.Equal(t, "Skills", result[0].Criteria)
	})
}

func Test_UpdateCertificateData(t *testing.T) {
	t.Run("It should update the certificate data successfully", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},
		}

		svc := TenantCertificatesService{
			ctx:                     context.TODO(),
			dynamodbClient:          &ddbClient,
			logger:                  log.New(logBuffer, "TEST:", 0),
			TenantCertificatesTable: "TenantCertificatesTable",
		}

		certificate := TenantCertificates{
			CertificateId:    "certificate-1",
			CertificateName:  "Updated Certificate",
			Criteria:         "Skills",
			Threshold:        100,
			IsActive:         "Active",
			LastModifiedDate: "2021-09-01T00:00:00Z",
			CertificateMode:  "Manual",
		}

		err := svc.UpdateCertificateData(certificate)

		assert.NoError(t, err)

		expectedPutItemInput := dynamodb.PutItemInput{
			TableName: aws.String("TenantCertificatesTable"),
			Item: map[string]dynamodb_types.AttributeValue{
				"CertificateId":    &dynamodb_types.AttributeValueMemberS{Value: "certificate-1"},
				"CertificateName":  &dynamodb_types.AttributeValueMemberS{Value: "Updated Certificate"},
				"Criteria":         &dynamodb_types.AttributeValueMemberS{Value: "Skills"},
				"Threshold":        &dynamodb_types.AttributeValueMemberN{Value: "100"},
				"IsActive":         &dynamodb_types.AttributeValueMemberS{Value: "Active"},
				"LastModifiedDate": &dynamodb_types.AttributeValueMemberS{Value: "2021-09-01T00:00:00Z"},
				"Mode":             &dynamodb_types.AttributeValueMemberS{Value: "Manual"},
			},
		}

		assert.Equal(t, expectedPutItemInput, ddbClient.PutItemInputs[0])
	})
}
