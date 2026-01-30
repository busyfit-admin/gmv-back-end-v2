package main

// This Lambda handles the assignment of certificates to an entity.
// It is triggered by messages from an SQS queue.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"

	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type CertificateTransferService struct {
	ctx context.Context

	dynamodbClient awsclients.DynamodbClient

	logger         *log.Logger
	TransactionSvc companylib.TenantCertificateTransferService
	contentSvc     companylib.TenantUploadContentService
}

func main() {

	ctx, root := xray.BeginSegment(context.Background(), "certificates-transfer")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load AWS config: %v", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "CERT-TRANSFER: ", log.LstdFlags)

	ddbClient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

	// Create content service for S3 operations
	contentSvc := companylib.CreateTenantUploadContentService(ctx, s3Client, logger)
	contentSvc.S3Bucket = os.Getenv("S3_BUCKET")

	// Create certificate transfer service for DynamoDB operations
	transferSvc := companylib.CreateTenantCertificateTransferService(ctx, ddbClient, logger)
	transferSvc.EmployeesTable = os.Getenv("EMPLOYEE_TABLE")
	transferSvc.TenantCertificatesTable = os.Getenv("BADGES_TABLE")
	transferSvc.TenantCertificatesTable_CriteriaThresholdIndex = os.Getenv("BADGES_TABLE_CRITERIA_THRESHOLD_INDEX")
	transferSvc.CertificatesTransferLogsTable = os.Getenv("BADGES_TRANSFER_LOGS_TABLE")

	svc := CertificateTransferService{
		ctx:            ctx,
		logger:         logger,
		TransactionSvc: *transferSvc,
		dynamodbClient: ddbClient,
		contentSvc:     *contentSvc,
	}

	lambda.Start(svc.handleCertificateTransferEvents)
}

// handleCertificateTransferEvents processes SQS events to assign certificates.
func (svc *CertificateTransferService) handleCertificateTransferEvents(sqsEvent events.SQSEvent) error {
	svc.logger.Printf("Processing %d SQS event records", len(sqsEvent.Records))

	for _, record := range sqsEvent.Records {

		svc.logger.Printf("Received SQS message: %s", record.Body)

		var input companylib.CertificateAssignInput
		if err := json.Unmarshal([]byte(record.Body), &input); err != nil {
			svc.logger.Printf("Failed to unmarshal SQS message body: %v", err)
			return err
		}
		// Assign the certificate based on the input
		if err := svc.AssignCertificate(input); err != nil {
			svc.logger.Printf("Failed to assign certificate for input %+v: %v", input, err)
			return err
		}
	}

	svc.logger.Println("Successfully processed all SQS messages")
	return nil
}

// AssignCertificate assigns a certificate to a destination user by copying the certificate image
func (svc *CertificateTransferService) AssignCertificate(input companylib.CertificateAssignInput) error {
	svc.logger.Printf("Starting certificate assignment for input: %+v", input)

	// Ensure dynamodbClient is not nil before using it
	if svc.dynamodbClient == nil {
		svc.logger.Fatalf("DynamoDB client is not initialized")
	}

	// Step 0: Initialize the certificate transfer log entry
	certificateTransferLog := companylib.CertificatesTransferLogsTable{
		CertificateTransferId:      input.MessageDeduplicationId,
		CertificateName:            input.CertificateName,
		SourceUserName:             input.From,
		DestinationUserName:        input.DestID,
		CertificateId:              input.CertificatesId,
		Message:                    input.Message,
		Criteria:                   input.Criteria + " " + fmt.Sprintf("%v", input.Threshold),
		CertificateTransferStatus:             "In Progress",
		CertificateTransferLogTime: utils.GenerateTimestamp(),
		CertificateTransferError:   "",
	}

	svc.logger.Println("The certificateTransferLog", certificateTransferLog)

	// Put the initial log entry into the CertificateTransferLogsTable
	item, err := attributevalue.MarshalMap(certificateTransferLog)
	if err != nil {
		return err
	}

	putItemInput := &dynamodb.PutItemInput{
		TableName: aws.String(svc.TransactionSvc.CertificatesTransferLogsTable),
		Item:      item,
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, putItemInput)
	if err != nil {
		return err
	}

	// Step 1: Retrieve the certificate image path from DynamoDB
	path, err := svc.TransactionSvc.GetCertificatePicPath(input.CertificatesId)
	if err != nil {
		svc.logger.Printf("Failed to retrieve image path for CertificateId '%s': %v", input.CertificatesId, err)
		certificateTransferLog.CertificateTransferStatus = "Failed - Image Path Retrieval"
		certificateTransferLog.CertificateTransferError = err.Error()

		if err := svc.UpdateCertificateTransferLog(certificateTransferLog); err != nil {
			svc.logger.Printf("Failed to update certificate transfer log: %v", err)
		}
		return err
	}
	svc.logger.Printf("Retrieved certificate image path: %s", path)

	// Step 2: Construct the destination path for the certificate image
	destPath := "users/" + input.DestID + "/certificates/" + input.CertificatesId
	svc.logger.Printf("Constructed destination path: %s", destPath)

	// Step 3: Copy the certificate image from source to destination
	if err := svc.CopyCertificateImage(path, destPath); err != nil {
		certificateTransferLog.CertificateTransferStatus = "Failed - Image Copy"
		certificateTransferLog.CertificateTransferError = err.Error()

		if err := svc.UpdateCertificateTransferLog(certificateTransferLog); err != nil {
			svc.logger.Printf("Failed to update certificate transfer log: %v", err)
		}
		return err
	}

	// Step 4: Perform the certificate assignment (update DynamoDB, logs)
	if err := svc.TransactionSvc.PerformCertificateAssignment(input, destPath); err != nil {
		svc.logger.Printf("Failed to perform certificate assignment for DestID '%s': %v", input.DestID, err)
		certificateTransferLog.CertificateTransferStatus = "Failed - Certificate Assignment"
		certificateTransferLog.CertificateTransferError = err.Error()

		if err := svc.UpdateCertificateTransferLog(certificateTransferLog); err != nil {
			svc.logger.Printf("Failed to update certificate transfer log: %v", err)
		}
		// Rollback the S3 copy since DynamoDB update failed
		if rollbackErr := svc.contentSvc.DeleteContentFromS3(destPath); rollbackErr != nil {
			svc.logger.Printf("Rollback failed: Could not delete S3 object at '%s': %v", destPath, rollbackErr)
		} else {
			svc.logger.Printf("Successfully rolled back S3 object at '%s'", destPath)
		}
		return err
	}

	// Update the log with the final status
	certificateTransferLog.CertificateTransferStatus = "Completed"
	certificateTransferLog.CertificateTransferError = "No Error" // Clear the error if the operation succeeded
	if err := svc.UpdateCertificateTransferLog(certificateTransferLog); err != nil {
		svc.logger.Printf("Failed to update certificate transfer log: %v", err)
		return err
	}

	svc.logger.Printf("Successfully assigned certificate '%s' to '%s'", input.CertificatesId, input.DestID)
	return nil
}

// CopyCertificateImage copies a certificate image from a source path to a destination path.
func (svc *CertificateTransferService) CopyCertificateImage(from string, to string) error {
	svc.logger.Printf("Copying certificate image from '%s' to '%s'", from, to)

	// Copy operation on the image in S3
	if err := svc.contentSvc.CopyObject(from, to); err != nil {
		svc.logger.Printf("Failed to copy certificate image from '%s' to '%s': %v", from, to, err)
		return err
	}

	svc.logger.Printf("Successfully copied certificate image from '%s' to '%s'", from, to)
	return nil
}

// UpdateCertificateTransferLog updates the certificate transfer log in DynamoDB
func (svc *CertificateTransferService) UpdateCertificateTransferLog(logEntry companylib.CertificatesTransferLogsTable) error {
	updateExpression := "SET CertificateTransferStatus = :status, CertificateTransferError = :error, CertificateTransferLogTime = :time"
	expressionAttributeValues := map[string]dynamodb_types.AttributeValue{
		":status": &dynamodb_types.AttributeValueMemberS{Value: logEntry.CertificateTransferStatus},
		":error":  &dynamodb_types.AttributeValueMemberS{Value: logEntry.CertificateTransferError},
		":time":   &dynamodb_types.AttributeValueMemberS{Value: logEntry.CertificateTransferLogTime},
	}

	updateItemInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TransactionSvc.CertificatesTransferLogsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"CertificateTransferId": &dynamodb_types.AttributeValueMemberS{Value: logEntry.CertificateTransferId},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ReturnValues:              dynamodb_types.ReturnValueUpdatedNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, updateItemInput)
	if err != nil {
		return err
	}

	svc.logger.Println("Updated certificate transfer log:", logEntry)
	return nil
}
