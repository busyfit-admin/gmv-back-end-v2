package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger
	orgSVC *companylib.OrgServiceV2
	empSVC *companylib.EmployeeService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("OrganizationAPI")

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "manage-organization")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	sesClient := ses.NewFromConfig(cfg)

	// Initialize employee service
	empSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	empSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	empSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	// Email service
	emailSvc := companylib.CreateEmailService(ctx, sesClient, logger)

	// Initialize organization service
	orgSvc := companylib.CreateOrgServiceV2(ctx, ddbclient, logger, empSvc, emailSvc)
	orgSvc.OrganizationTable = os.Getenv("ORGANIZATION_TABLE")
	orgSvc.PromoCodesTable = os.Getenv("PROMO_CODES_TABLE")

	svc := &Service{
		ctx:    ctx,
		logger: logger,
		orgSVC: orgSvc,
		empSVC: empSvc,
	}

	lambda.Start(svc.Handler)
}

// Handler handles the Lambda request
func (svc *Service) Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Received request: %s %s", request.HTTPMethod, request.Path)

	// Extract Cognito ID from Cognito authorizer
	cognitoId, err := svc.getCognitoIdFromRequest(request)
	if err != nil {
		svc.logger.Printf("Failed to get Cognito ID: %v", err)
		return svc.errorResponse(http.StatusUnauthorized, "Unauthorized", err)
	}

	// Get employee details by Cognito ID
	employee, err := svc.empSVC.GetEmployeeDataByCognitoId(cognitoId)
	if err != nil {
		svc.logger.Printf("Failed to get employee details: %v", err)
		return svc.errorResponse(http.StatusUnauthorized, "User not found", err)
	}

	// Extract orgId from path parameters
	orgId, ok := request.PathParameters["orgId"]
	if !ok || orgId == "" {
		return svc.errorResponse(http.StatusBadRequest, "Organization ID is required", nil)
	}

	switch request.HTTPMethod {
	case "GET":
		return svc.getOrganization(orgId, employee.EmailID, request)
	case "PUT":
		return svc.updateOrganization(orgId, employee.EmailID, request)
	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// getOrganization retrieves organization details
func (svc *Service) getOrganization(orgId string, userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Getting organization %s for user: %s", orgId, userName)

	// Verify user is admin of the organization
	isAdmin, err := svc.orgSVC.IsOrgAdmin(orgId, userName)
	if err != nil {
		svc.logger.Printf("Failed to check admin status: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to verify permissions", err)
	}
	if !isAdmin {
		return svc.errorResponse(http.StatusForbidden, "Access denied: Not an organization admin", nil)
	}

	// Get the organization
	organization, err := svc.orgSVC.GetOrganization(orgId)
	if err != nil {
		svc.logger.Printf("Failed to get organization: %v", err)
		return svc.errorResponse(http.StatusNotFound, "Organization not found", err)
	}

	// Return the organization
	body, err := json.Marshal(map[string]interface{}{
		"organization": organization,
	})
	if err != nil {
		svc.logger.Printf("Failed to marshal response: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to create response", err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}

// updateOrganization updates organization details
func (svc *Service) updateOrganization(orgId string, userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Updating organization %s for user: %s", orgId, userName)

	// Parse request body
	var input companylib.UpdateOrganizationInput
	if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Set the organization ID
	input.OrganizationId = orgId

	// Update the organization
	err := svc.orgSVC.UpdateOrganization(input, userName)
	if err != nil {
		svc.logger.Printf("Failed to update organization: %v", err)
		if err.Error() == fmt.Sprintf("user %s is not an admin of organization %s", userName, orgId) {
			return svc.errorResponse(http.StatusForbidden, "Access denied: Not an organization admin", err)
		}
		return svc.errorResponse(http.StatusInternalServerError, "Failed to update organization", err)
	}

	// Get updated organization details
	organization, err := svc.orgSVC.GetOrganization(orgId)
	if err != nil {
		svc.logger.Printf("Failed to get updated organization: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Organization updated but failed to retrieve updated details", err)
	}

	// Return the updated organization
	body, err := json.Marshal(map[string]interface{}{
		"message":      "Organization updated successfully",
		"organization": organization,
	})
	if err != nil {
		svc.logger.Printf("Failed to marshal response: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to create response", err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}

// getCognitoIdFromRequest extracts Cognito ID from Cognito authorizer context
func (svc *Service) getCognitoIdFromRequest(request events.APIGatewayProxyRequest) (string, error) {
	// Try to get from authorizer context first
	if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		if sub, ok := claims["sub"].(string); ok && sub != "" {
			return sub, nil
		}
	}

	// Fallback to custom header for testing
	if cognitoId := request.Headers["X-Cognito-Id"]; cognitoId != "" {
		return cognitoId, nil
	}

	return "", fmt.Errorf("cognito ID not found in request")
}

// errorResponse creates an error response
func (svc *Service) errorResponse(statusCode int, message string, err error) (events.APIGatewayProxyResponse, error) {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}

	body, _ := json.Marshal(map[string]string{
		"error":   message,
		"message": errorMsg,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}
