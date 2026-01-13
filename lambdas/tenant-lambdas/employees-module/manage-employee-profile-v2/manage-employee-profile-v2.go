package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	cognitotypes "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

var RESP_HEADERS = map[string]string{
	"Content-Type":                 "application/json",
	"Access-Control-Allow-Headers": "*",
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Methods": "OPTIONS,POST,GET,PATCH",
}

type Service struct {
	ctx    context.Context
	logger *log.Logger

	employeeSvc companylib.EmployeeService
	cdnSvc      companylib.CDNService
	contentSvc  companylib.TenantUploadContentService

	// Cards MetaData Service
	cardsMetaSvc  companylib.CompanyCardsMetadataService
	CardsMetaData map[string]companylib.CompanyCardsMetaDataTable
}

var RESP_HEADERS = companylib.GetHeadersForAPI("ProfileAPI")

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "manage-employee-profile")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	dynamodbClient := dynamodb.NewFromConfig(cfg)
	secretsClient := secretsmanager.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	logger := log.New(os.Stdout, "", log.LstdFlags)

	employeeSvc := companylib.CreateEmployeeService(ctx, dynamodbClient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	// Tenant Upload Content Service
	contentSvc := companylib.CreateTenantUploadContentService(ctx, s3Client, logger)
	contentSvc.S3Bucket = os.Getenv("BUCKET_NAME")

	// Here we are creating a CDN Service
	cdnSvc := companylib.CDNService{}
	err = cdnSvc.CreateCDNService(ctx, logger, secretsClient, os.Getenv("SECRETS_CND_PK_ARN"), os.Getenv("PUBLIC_KEY_ID"))
	if err != nil {
		log.Fatalf("Error creating CDN Service: %v\n", err)
	}
	cdnSvc.CDNDomain = os.Getenv("CDN_DOMAIN")

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		employeeSvc: *employeeSvc,
		cdnSvc:      cdnSvc,
		contentSvc:  *contentSvc,
	}

	lambda.Start(svc.handleAPIRequests)
}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.ctx = ctx

	switch request.HTTPMethod {
	case "GET":
		return svc.GetUserProfile(request)
	case "PATCH":
		return svc.UpdateUserProfile(request)
	default:
		svc.logger.Printf("Unsupported HTTP method: %s", request.HTTPMethod)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 405,
			Body:       `{"error":"Method not allowed"}`,
		}, nil
	}
}

// GetUserProfile returns the basic profile information for the authenticated user
type ProfileResponse struct {
	UserName    string `json:"UserName"`
	DisplayName string `json:"DisplayName"`
	EmailID     string `json:"EmailId"`
	PhoneNumber string `json:"PhoneNumber"`
	Designation string `json:"Designation"`
	ProfilePic  string `json:"ProfilePic"`
	Location    string `json:"Location"`
}

func (svc *Service) GetUserProfile(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract Cognito ID from JWT sub claim
	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["sub"].(string)

	employeeData, err := svc.employeeSvc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting employee data: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       `{"error":"Failed to retrieve employee data"}`,
		}, nil
	}

	// Sign the profile picture URL
	profilePic := svc.cdnSvc.GetPreSignedCDN_URL_noError(employeeData.ProfilePic)

	profile := ProfileResponse{
		UserName:    employeeData.UserName,
		DisplayName: employeeData.DisplayName,
		EmailID:     employeeData.EmailID,
		PhoneNumber: employeeData.PhoneNumber,
		Designation: employeeData.Designation,
		ProfilePic:  profilePic,
		Location:    employeeData.Location,
	}

	respBody, err := json.Marshal(profile)
	if err != nil {
		svc.logger.Printf("Error marshalling profile response: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       `{"error":"Failed to marshal response"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(respBody),
	}, nil
}

// UpdateUserProfile allows updating phone number, designation, display name, and profile picture
type ProfileUpdateRequest struct {
	PhoneNumber string `json:"PhoneNumber,omitempty"`
	Designation string `json:"Designation,omitempty"`
	DisplayName string `json:"DisplayName,omitempty"`
	ProfilePic  string `json:"ProfilePic,omitempty"` // base64 encoded image
}

func (svc *Service) UpdateUserProfile(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Extract Cognito ID from JWT sub claim
	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["sub"].(string)

	employeeData, err := svc.employeeSvc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting employee data: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       `{"error":"Failed to retrieve employee data"}`,
		}, nil
	}

	var updateReq ProfileUpdateRequest
	err = json.Unmarshal([]byte(request.Body), &updateReq)
	if err != nil {
		svc.logger.Printf("Error unmarshalling request body: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 400,
			Body:       `{"error":"Invalid request body"}`,
		}, nil
	}

	// Handle profile picture upload if provided
	if updateReq.ProfilePic != "" {
		uploadKey := fmt.Sprintf("users/%s/profilepic/profile.png", employeeData.UserName)
		svc.logger.Printf("Uploading profile picture to: %s", uploadKey)

		err = svc.contentSvc.UploadContentToS3_Base64Content(uploadKey, updateReq.ProfilePic)
		if err != nil {
			svc.logger.Printf("Error uploading profile picture: %v", err)
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
				Body:       `{"error":"Failed to upload profile picture"}`,
			}, nil
		}

		// Update the profile pic reference in DynamoDB
		err = svc.employeeSvc.UpdateEmployeeProfilePicByUserName(employeeData.UserName, uploadKey)
		if err != nil {
			svc.logger.Printf("Error updating profile pic in DynamoDB: %v", err)
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
				Body:       `{"error":"Failed to update profile picture"}`,
			}, nil
		}
	}

	// Build update for DynamoDB (phone, designation, display name)
	needsDynamoDBUpdate := false
	updatedData := employeeData

	if updateReq.PhoneNumber != "" && updateReq.PhoneNumber != employeeData.PhoneNumber {
		updatedData.PhoneNumber = updateReq.PhoneNumber
		needsDynamoDBUpdate = true
	}

	if updateReq.Designation != "" && updateReq.Designation != employeeData.Designation {
		updatedData.Designation = updateReq.Designation
		needsDynamoDBUpdate = true
	}

	if updateReq.DisplayName != "" && updateReq.DisplayName != employeeData.DisplayName {
		updatedData.DisplayName = updateReq.DisplayName
		needsDynamoDBUpdate = true

		// Also update Cognito user attributes to keep them in sync
		err = svc.updateCognitoUserName(employeeCognitoId, updateReq.DisplayName)
		if err != nil {
			svc.logger.Printf("Warning: Failed to update Cognito user attributes: %v", err)
			// Don't fail the request, but log the warning
		}
	}

	// Update DynamoDB if there are changes
	if needsDynamoDBUpdate {
		updateDetails := companylib.BasicEmployeeDetails{
			UserName:        employeeData.UserName,
			ExternalId:      updatedData.ExternalId,
			DisplayName:     updatedData.DisplayName,
			PhoneNumber:     updatedData.PhoneNumber,
			LoginType:       updatedData.LoginType,
			IsManager:       updatedData.IsManager,
			ManagerUserName: updatedData.ManagerUserName,
			StartDate:       updatedData.StartDate,
			EndDate:         updatedData.EndDate,
			IsActive:        updatedData.IsActive,
		}

		err = svc.employeeSvc.UpdateEmployeeDetailsByUserName(updateDetails)
		if err != nil {
			svc.logger.Printf("Error updating employee details in DynamoDB: %v", err)
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
				Body:       `{"error":"Failed to update employee details"}`,
			}, nil
		}
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       `{"message":"Profile updated successfully"}`,
	}, nil
}

// updateCognitoUserName updates the user's name attributes in Cognito
func (svc *Service) updateCognitoUserName(cognitoId, displayName string) error {
	// Update Cognito user attributes: name, given_name, family_name
	// Split displayName if it contains space for given_name/family_name
	var givenName, familyName string

	// Simple split: first word is given name, rest is family name
	nameparts := splitName(displayName)
	givenName = nameparts[0]
	if len(nameparts) > 1 {
		familyName = nameparts[1]
	}

	attributes := []cognitotypes.AttributeType{
		{
			Name:  strPtr("name"),
			Value: strPtr(displayName),
		},
		{
			Name:  strPtr("given_name"),
			Value: strPtr(givenName),
		},
	}

	if familyName != "" {
		attributes = append(attributes, cognitotypes.AttributeType{
			Name:  strPtr("family_name"),
			Value: strPtr(familyName),
		})
	}

	input := &cognitoidentityprovider.AdminUpdateUserAttributesInput{
		UserPoolId:     strPtr(svc.userPoolId),
		Username:       strPtr(cognitoId),
		UserAttributes: attributes,
	}

	_, err := svc.cognitoSvc.AdminUpdateUserAttributes(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Error updating Cognito user attributes: %v", err)
		return err
	}

	svc.logger.Printf("Successfully updated Cognito user attributes for user: %s", cognitoId)
	return nil
}

// Helper function to split display name into given and family names
func splitName(displayName string) []string {
	// Simple implementation: split on first space
	for i, ch := range displayName {
		if ch == ' ' {
			return []string{displayName[:i], displayName[i+1:]}
		}
	}
	return []string{displayName, ""}
}

func strPtr(s string) *string {
	return &s
}
