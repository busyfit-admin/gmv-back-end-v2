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
	ctx, root := xray.BeginSegment(context.TODO(), "list-user-organizations")
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

	switch request.HTTPMethod {
	case "GET":
		return svc.listUserOrganizations(employee.EmailID, request)
	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// listUserOrganizations retrieves all organizations where user is an admin
func (svc *Service) listUserOrganizations(userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Listing organizations for user: %s", userName)

	// Get an organizations where user is an admin
	organization, err := svc.orgSVC.GetAdminOrganization(userName)
	if err != nil {
		svc.logger.Printf("Failed to get user organization: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to retrieve organization", err)
	}

	// Get available subscription plans for reference
	availablePlans := svc.orgSVC.GetAvailableSubscriptionPlans()
	planMap := make(map[string]companylib.SubscriptionPlan)
	for _, plan := range availablePlans {
		planMap[plan.PlanID] = plan
	}

	// marshall organization summary from organization details from struct:
	/*  Create new org summary struct from :
		type Organization struct {
	    // Composite key structure
	    PK     string `dynamodbav:"PK" json:"-"`     // ORG#{organizationId}
	    SK     string `dynamodbav:"SK" json:"-"`     // METADATA
	    GSI1PK string `dynamodbav:"GSI1PK" json:"-"` // For GSI queries
	    GSI1SK string `dynamodbav:"GSI1SK" json:"-"` // For GSI queries

	    // Primary key - matches CloudFormation template
	    OrganizationId string `dynamodbav:"OrganizationId" json:"organizationId"`

	    // Basic organization info
	    OrgName string `dynamodbav:"OrgName" json:"orgName"`
	    OrgDesc string `dynamodbav:"OrgDesc" json:"orgDesc"`
	}
	*/

	OrgSummary := struct {
		OrganizationId string `json:"organizationId"`
		OrgName        string `json:"orgName"`
		OrgDesc        string `json:"orgDesc"`
	}{}

	if organization != nil {
		OrgSummary.OrganizationId = organization.OrganizationId
		OrgSummary.OrgName = organization.OrgName
		OrgSummary.OrgDesc = organization.OrgDesc
	}

	// Return the organizations list
	body, err := json.Marshal(map[string]interface{}{
		"organization":   OrgSummary,
		"availablePlans": availablePlans,
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
