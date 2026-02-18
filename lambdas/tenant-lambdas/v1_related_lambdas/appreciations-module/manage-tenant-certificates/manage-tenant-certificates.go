package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type SQSClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type Service struct {
	ctx            context.Context
	logger         *log.Logger
	certificateSvc companylib.TenantCertificatesService

	sqsClient                     SQSClient
	CERTIFICATE_TRANSFER_SQS_NAME string

	cdnSvc     companylib.CDNService
	contentSvc companylib.TenantUploadContentService

	employeeSvc companylib.EmployeeService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("CertificatesAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-tenant-certificates")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	secretsClient := secretsmanager.NewFromConfig(cfg)

	certificateSvc := companylib.CreateTenantCertificatesService(ctx, ddbclient, logger)
	certificateSvc.TenantCertificatesTable = os.Getenv("TENANT_CERTIFICATES_TABLE")
	certificateSvc.CertificatesTransferLogsTable = os.Getenv("BADGES_TRANSFER_LOGS_TABLE")
	certificateSvc.CertificateLogsIndex =  os.Getenv("CERTIFICATELOGS_INDEX")

	// Content Service
	contentSvc := companylib.CreateTenantUploadContentService(ctx, s3Client, logger)
	contentSvc.S3Bucket = os.Getenv("S3_BUCKET")

	// SQS queue service
	sqsClient := sqs.NewFromConfig(cfg)

	// CDN Service
	cdnSvc := companylib.CDNService{}
	err = cdnSvc.CreateCDNService(ctx, logger, secretsClient, os.Getenv("SECRETS_CND_PK_ARN"), os.Getenv("PUBLIC_KEY_ID"))
	if err != nil {
		log.Fatalf("Error creating CDN Service: %v\n", err)
	}
	cdnSvc.CDNDomain = os.Getenv("CDN_DOMAIN")

	// Employee Service
	employeeSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	svc := Service{
		ctx:                           ctx,
		logger:                        logger,
		certificateSvc:                *certificateSvc,
		contentSvc:                    *contentSvc,
		cdnSvc:                        cdnSvc,
		sqsClient:                     sqsClient,
		employeeSvc:                   *employeeSvc,
		CERTIFICATE_TRANSFER_SQS_NAME: os.Getenv("CERTIFICATE_TRANSFER_SQS_QUEUE"),
	}
	lambda.Start(svc.handleAPIRequests)

}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.ctx = ctx

	// 1) Authorization at User Level and Check if user to create the card is Admin or Rewards Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		svc.logger.Printf("error authorizing the request: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	switch request.HTTPMethod {
	case "GET":
		return svc.GETRequestHandler(request)
	case "PATCH":
		return svc.PATCHRequestHandler(request)
	case "POST":
		return svc.POSTRequestHandler(request)
	case "DELETE":
		return svc.DELETERequestHandler(request)
	default:
		svc.logger.Printf("entered Default section of the switch, Erroring by returning 500")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

}

const (
	GET_ALL_CERTIFICATES_DATA        = "get-all-certificates"
	GET_CERTIFICATE_DATA             = "get-certificate"
	GET_CERTIFICATE_TRANSFER_DETAILS = "get-certificate-transfer-details"

	CREATE_CERTIFICATES = "create-certificates"
	UPDATE_CERTIFICATES = "update-certificates"
	DELETE_CERTIFICATES = "delete-certificates"

	TRANSFER_CERTIFICATE = "transfer-certificate"
)

// ----- GET Request Handler -----
func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	getType := request.Headers["get_type"]
	switch getType {
	case GET_ALL_CERTIFICATES_DATA:
		return svc.GetAllCertificates(request)
	case GET_CERTIFICATE_DATA:
		return svc.GetCertificateData(request)
	case GET_CERTIFICATE_TRANSFER_DETAILS:
		return svc.GetCertificateTransferDetails(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil

	}
}
func (svc *Service) GetAllCertificates(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allCertificates, err := svc.certificateSvc.GetAllTenantCertificates()
	if err != nil {
		svc.logger.Printf("error getting all certificates data: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	// Sign all the image URLs
	for i := range allCertificates.Active {
		allCertificates.Active[i].CertificateImage = svc.cdnSvc.GetPreSignedCDN_URL_noError(allCertificates.Active[i].CertificateImage)
	}
	for i := range allCertificates.Draft {
		allCertificates.Draft[i].CertificateImage = svc.cdnSvc.GetPreSignedCDN_URL_noError(allCertificates.Draft[i].CertificateImage)
	}

	responseBody, err := json.Marshal(allCertificates)
	if err != nil {
		svc.logger.Printf("error marshalling the response: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}
func (svc *Service) GetCertificateData(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	certificateId := request.Headers["certificate-id"]

	certificateData, err := svc.certificateSvc.GetCertificate(certificateId)

	// Sign the Image URL
	certificateData.CertificateImage = svc.cdnSvc.GetPreSignedCDN_URL_noError(certificateData.CertificateImage)

	if err != nil {
		svc.logger.Printf("error getting certificate data: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(certificateData)
	if err != nil {
		svc.logger.Printf("error marshalling the response: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func (svc *Service) GetCertificateTransferDetails(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	sourceUserName := request.Headers["source-name"]

	svc.logger.Printf("[INFO] Fetching certificate logs for SourceUserName: %s", sourceUserName)

	certificateLogData, err := svc.certificateSvc.GetCertificateLogs(sourceUserName)
	if err != nil {
		svc.logger.Printf("[ERROR] Failed to fetch certificate logs for %s: %v", sourceUserName, err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       `{"error": "Failed to retrieve certificate logs"}`,
		}, nil
	}

	body, err := json.Marshal(certificateLogData)
	if err != nil {
		svc.logger.Printf("[ERROR] Failed to marshal certificate logs response: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       `{"error": "Failed to process response"}`,
		}, nil
	}

	svc.logger.Printf("[INFO] Successfully fetched certificate logs for %s", sourceUserName)

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(body),
	}, nil
}

// ----- POST Request Handler -----
func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	postType := request.Headers["post_type"]
	switch postType {
	case CREATE_CERTIFICATES:
		return svc.CreateCertificates(request)
	case TRANSFER_CERTIFICATE:
		return svc.TransferCertificate(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

}

type CreateCertificateInput struct {
	CertificateName  string `json:"CertificateName"`
	CertificateDesc  string `json:"CertificateDesc"`
	CertificateImage string `json:"CertificateImage"` // Base64 Encoded Image
	Criteria         string `json:"Criteria"`
	Threshold        int    `json:"Threshold"`
	IsActive         string `json:"IsActive"`
	CertificateMode  string `json:"CertificateMode"`
}

func (svc *Service) CreateCertificates(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var createCertificateInputData CreateCertificateInput
	err := json.Unmarshal([]byte(request.Body), &createCertificateInputData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	CertificateId := "cert-" + utils.GenerateRandomString(8)
	// Generate Image Key and Upload Image to S3
	CertImageKey := generateImageKey(CertificateId)
	err = svc.contentSvc.UploadContentToS3_Base64Content(CertImageKey, createCertificateInputData.CertificateImage)
	if err != nil {
		svc.logger.Printf("Error uploading images to S3: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// Create Certificate Data Object
	certificateData := companylib.TenantCertificates{
		CertificateId:    CertificateId, // Generate Unique Certificate ID
		CertificateName:  createCertificateInputData.CertificateName,
		CertificateDesc:  createCertificateInputData.CertificateDesc,
		CertificateImage: CertImageKey,
		Criteria:         createCertificateInputData.Criteria,
		Threshold:        createCertificateInputData.Threshold,
		IsActive:         createCertificateInputData.IsActive,
		CertificateMode:  createCertificateInputData.CertificateMode,
		LastModifiedDate: utils.GenerateTimestamp(),
	}

	err = svc.certificateSvc.CreateCertificateData(certificateData)
	if err != nil {
		svc.logger.Printf("error updating the certificate data: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
	}, nil
}

// Transfer Certificate
type TransferCertificateInput struct {
	CertificatesId  string `json:"CertificatesId"`
	CertificateName string `json:"CertificateName"`

	From   string `json:"From"`
	DestID string `json:"DestID"`

	Criteria  string `json:"Criteria"`
	Threshold int    `json:"Threshold"`
	Message   string `json:"Message"`
}

func (svc *Service) TransferCertificate(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	svc.logger.Println("Received TransferCertificate request:", request.Body)

	var transferCertificateInputData TransferCertificateInput
	err := json.Unmarshal([]byte(request.Body), &transferCertificateInputData)
	if err != nil {
		svc.logger.Printf("ERROR: Failed to unmarshal request body: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Printf("Parsed TransferCertificateInput: %+v", transferCertificateInputData)

	// Ensure a valid MessageDeduplicationId
	messageDeduplicationId := generateDeduplicationId(transferCertificateInputData.CertificatesId)
	svc.logger.Printf("Generated MessageDeduplicationId: %s", messageDeduplicationId)

	// Define the temp structure to include MessageDeduplicationId
	tempAddMessageDeduplicationId := struct {
		TransferCertificateInput
		MessageDeduplicationId string `json:"MessageDeduplicationId"`
	}{
		TransferCertificateInput: transferCertificateInputData,
		MessageDeduplicationId:   messageDeduplicationId,
	}

	// Serialize the certificate transfer data
	certTransferBytes, err := json.Marshal(tempAddMessageDeduplicationId)
	if err != nil {
		svc.logger.Printf("ERROR: Failed to marshal certificate data: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	svc.logger.Printf("Certificate transfer message: %s", string(certTransferBytes))

	// Send the message to the SQS queue
	svc.logger.Println("Sending message to SQS...")
	_, err = svc.sqsClient.SendMessage(svc.ctx, &sqs.SendMessageInput{
		MessageBody:            aws.String(string(certTransferBytes)),
		QueueUrl:               aws.String(svc.CERTIFICATE_TRANSFER_SQS_NAME),
		MessageDeduplicationId: aws.String(messageDeduplicationId),
		MessageGroupId:         aws.String(transferCertificateInputData.From),
	})

	if err != nil {
		svc.logger.Printf("ERROR: Unable to send the certificate transfer event to SQS: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	svc.logger.Println("Successfully sent certificate transfer event to SQS.")

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
		Body: fmt.Sprintf(`{"message": "Certificate transfer request successfully sent to SQS", "certificateId": "%s", "from": "%s", "to": "%s"}`,
			transferCertificateInputData.CertificatesId,
			transferCertificateInputData.From,
			transferCertificateInputData.DestID),
	}, nil
}

// Generate a valid and unique MessageDeduplicationId
func generateDeduplicationId(base string) string {
	randomBytes := make([]byte, 8)
	_, _ = rand.Read(randomBytes)
	randomString := hex.EncodeToString(randomBytes)
	dedupId := fmt.Sprintf("%s-%d-%s", base, time.Now().Unix(), randomString)

	return dedupId
}

// Generate random string for Image Key.
func generateImageKey(certId string) string {
	return fmt.Sprintf("certificate-templates/%s/images/img-%s", certId, utils.GenerateRandomString(8))
}

// ----- PATCH Request Handler -----
func (svc *Service) PATCHRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	postType := request.Headers["patch_type"]
	switch postType {
	case UPDATE_CERTIFICATES:
		return svc.UpdateCertificates(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

}

type UpdateCertificateDetailsInput struct {
	CertificateId   string `json:"CertificateId"` // Required
	CertificateName string `json:"CertificateName"`
	CertificateDesc string `json:"CertificateDesc"`
	Criteria        string `json:"Criteria"`
	Threshold       int    `json:"Threshold"`
	IsActive        string `json:"IsActive"`
	CertificateMode string `json:"CertificateMode"`
}

func (svc *Service) UpdateCertificates(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var UpdateCertificateData UpdateCertificateDetailsInput
	err := json.Unmarshal([]byte(request.Body), &UpdateCertificateData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Enforce patch to have the certificateId to be provided
	if UpdateCertificateData.CertificateId == "" {
		svc.logger.Printf("cannot update certificate data without the certificateId")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	updateCertData := companylib.TenantCertificates{
		CertificateId:    UpdateCertificateData.CertificateId,
		CertificateName:  UpdateCertificateData.CertificateName,
		CertificateDesc:  UpdateCertificateData.CertificateDesc,
		Criteria:         UpdateCertificateData.Criteria,
		Threshold:        UpdateCertificateData.Threshold,
		IsActive:         UpdateCertificateData.IsActive,
		CertificateMode:  UpdateCertificateData.CertificateMode,
		LastModifiedDate: utils.GenerateTimestamp(),
	}

	err = svc.certificateSvc.UpdateCertificateData(updateCertData)
	if err != nil {
		svc.logger.Printf("error updating the certificate data: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
	}, nil
}

// ----- DELETE Request Handler -----
func (svc *Service) DELETERequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	deleteType := request.Headers["delete_type"]
	switch deleteType {
	case DELETE_CERTIFICATES:
		return svc.DeleteCertificates(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}
func (svc *Service) DeleteCertificates(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	certificateId := request.Headers["certificate-id"]

	err := svc.certificateSvc.DeleteCertificateData(certificateId)
	if err != nil {
		svc.logger.Printf("error deleting the certificate data: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
	}, nil
}
