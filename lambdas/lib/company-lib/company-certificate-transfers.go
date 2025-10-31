/*
Certificate Assignment Logic Flow:
	CertificateAssignInput:
		A data structure containing:
			From:
			DestID: Referring to the destination ID or user/entity for certificate assignment.
			Criteria and Thresholds:
				These are the conditions that need to be checked before assigning a certificate, based on various thresholds.


	Steps for Certificate Assignment:
		Check Validity:
				The system needs to validate the input (e.g., check if the criteria and thresholds are appropriate or valid for the given DestID).
		Is Criteria Active:
				Check whether the given criteria (such as anniversary, skills, or appreciations) are active or applicable at the time of execution.
		Perform Certificate Assignment:
				If everything is valid and the criteria are active, the system assigns the certificate.
		Put Logs:
				Log the entire process or the results of the certificate assignment for tracking and auditing purposes.



Different Types of Criteria

Criteria	Threshold
1) Anniversary	<1, 3, 5, 10, 15, 20, 30>
2) Skills	<1, 3, 5, 7>
3) Appreciations Received	<1, 3, 5, 7>
4) Values	<ValueId>

*/

package Companylib

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"

	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type TenantCertificateTransferService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	TenantCertificatesTable                        string
	TenantCertificatesTable_CriteriaThresholdIndex string

	EmployeesTable                string
	CertificatesTransferLogsTable string
}

func CreateTenantCertificateTransferService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *TenantCertificateTransferService {
	return &TenantCertificateTransferService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

//----------- DDB Tables -------

type CertificatesTransferLogsTable struct {
	CertificateTransferId string `json:"CertificateTransferId" dynamodbav:"CertificateTransferId"` // PK
	SourceUserName        string `json:"SourceUserName" dynamodbav:"SourceUserName"`               // Index - PK
	DestinationUserName   string `json:"DestinationUserName" dynamodbav:"DestinationUserName"`     // Index - PK

	CertificateId   string `json:"CertificateId" dynamodbav:"CertificateId"`
	CertificateName string `json:"CertificateName" dynamodbav:"CertificateName"`
	Criteria        string `json:"Criteria" dynamodbav:"Criteria"`
	Message         string `json:"Message" dynamodbav:"Message"`

	CertificateTransferStatus  string `json:"CertificateTransferStatus" dynamodbav:"CertificateTransferStatus"`
	CertificateTransferLogTime string `json:"CertificateTransferLogTime" dynamodbav:"CertificateTransferLogTime"` // Index - SK
	CertificateTransferError   string `json:"Error" dynamodbav:"CertificateTransferError"`
}

/*
Handle Certificate Assign Input takes in the following Input :

	{
		"From" : "Admin",
		CertificateId: id: "<CertificateId ID>",
		"DestID": "<ENTITY ID>",
		"Criteria": "",
		"Threshold": int value,
		 Message: message,
	}
*/
type CertificateAssignInput struct {
	CertificateName        string
	MessageDeduplicationId string
	From                   string
	DestID                 string
	CertificatesId         string
	Criteria               string
	Threshold              int
	Message                string
}

func (t *TenantCertificateTransferService) GetCertificatePicPath(certificateId string) (string, error) {
	output, err := t.dynamodbClient.GetItem(t.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(t.TenantCertificatesTable),
		Key: map[string]types.AttributeValue{
			"CertificateId": &types.AttributeValueMemberS{Value: certificateId},
		},
	})
	if err != nil {
		t.logger.Printf("Failed to get the certificate with certificateID: %v, error :%v", certificateId, err)
		return "Couldn't get the img", err
	}

	certificateData := TenantCertificates{}
	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &certificateData)
	if err != nil {
		t.logger.Printf("Failed to Unmarshal the ddb output: %v, error :%v", certificateId, err)
		return "Couldn't get the img", err
	}

	return certificateData.CertificateImage, nil
}

// PerformCertificateAssignment carries out the certificate assignment once the criteria are valid and active
func (t *TenantCertificateTransferService) PerformCertificateAssignment(input CertificateAssignInput, destPath string) error {

	t.logger.Println("The cert ID:", input.CertificatesId)

	certificateId := input.CertificatesId

	// Add Entry to EmployeeCertificates
	err := t.PutCertificateToEmployee(EmployeeCertificates{
		CertificatesId: certificateId,
		DateAwarded:    utils.GenerateTimestamp(),
	}, input.DestID, input.Message, destPath)
	if err != nil {
		return err
	}

	return nil
}

func (t *TenantCertificateTransferService) PutCertificateToEmployee(certificateData EmployeeCertificates, destId string, message string, destPath string) error {
	// Fetch the existing data for the employee
	output, err := t.dynamodbClient.GetItem(t.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(t.EmployeesTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: destId},
		},
	})
	if err != nil {
		return err
	}

	// Check if CertificatesData exists
	_, found := output.Item["CertificatesData"]
	if !found {
		// Create an empty CertificatesData map if it doesn't exist
		_, err = t.dynamodbClient.UpdateItem(t.ctx, &dynamodb.UpdateItemInput{
			TableName:           aws.String(t.EmployeesTable),
			ConditionExpression: aws.String("attribute_not_exists(CertificatesData)"),
			Key: map[string]dynamodb_types.AttributeValue{
				"UserName": &dynamodb_types.AttributeValueMemberS{Value: destId},
			},
			UpdateExpression: aws.String("SET CertificatesData = :emptyMap"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":emptyMap": &dynamodb_types.AttributeValueMemberM{Value: map[string]dynamodb_types.AttributeValue{}},
			},
			ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
		})
		if err != nil {
			var conditionFailed *dynamodb_types.ConditionalCheckFailedException
			if !errors.As(err, &conditionFailed) {
				return err
			}
		}
	}

	t.logger.Println("The dist", destPath)

	// Convert certificate data to DynamoDB format
	certDataDDB := ObjectToDynamoDBAttribute(certificateData, message, destPath)

	// Update the CertificatesData map
	_, err = t.dynamodbClient.UpdateItem(t.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(t.EmployeesTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: destId},
		},
		UpdateExpression: aws.String("SET CertificatesData.#CertId = :CertData"),
		ExpressionAttributeNames: map[string]string{
			"#CertId": certificateData.CertificatesId,
		},
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":CertData": certDataDDB,
		},
		ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return err
	}

	t.logger.Printf("Certificate updated for user: %s, CertificateId: %s", destId, certificateData.CertificatesId)

	return nil
}

func ObjectToDynamoDBAttribute(certificateData EmployeeCertificates, message string, destPath string) *dynamodb_types.AttributeValueMemberM {
	return &dynamodb_types.AttributeValueMemberM{
		Value: map[string]dynamodb_types.AttributeValue{
			"CertificateId":   &dynamodb_types.AttributeValueMemberS{Value: certificateData.CertificatesId},
			"CertificateName": &dynamodb_types.AttributeValueMemberS{Value: certificateData.CertificateName},
			"CertificatesImg": &dynamodb_types.AttributeValueMemberS{Value: destPath},
			"Message":         &dynamodb_types.AttributeValueMemberS{Value: message},
			"DateAwarded":     &dynamodb_types.AttributeValueMemberS{Value: certificateData.DateAwarded},
		},
	}
}
