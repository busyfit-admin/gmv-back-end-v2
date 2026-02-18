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
	ctx      context.Context
	logger   *log.Logger
	teamsSVC *companylib.TeamsServiceV2
	empSVC   *companylib.EmployeeService
	orgSVC   *companylib.OrgServiceV2
}

var RESP_HEADERS = companylib.GetHeadersForAPI("TeamsAPI")

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "list-org-teams")
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

	// Initialize teams service
	teamsSvc := companylib.CreateTeamsServiceV2(ctx, ddbclient, logger, empSvc, emailSvc)
	teamsSvc.TeamsTable = os.Getenv("TEAMS_TABLE")

	// Initialize organization service
	orgSvc := companylib.CreateOrgServiceV2(ctx, ddbclient, logger, empSvc, emailSvc)
	orgSvc.OrganizationTable = os.Getenv("ORGANIZATION_TABLE")

	svc := &Service{
		ctx:      ctx,
		logger:   logger,
		teamsSVC: teamsSvc,
		empSVC:   empSvc,
		orgSVC:   orgSvc,
	}

	lambda.Start(svc.Handler)
}

// Handler handles the Lambda request
func (svc *Service) Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Received request: %s %s", request.HTTPMethod, request.Path)

	// Handle OPTIONS request for CORS preflight
	if request.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    RESP_HEADERS,
			Body:       "",
		}, nil
	}

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
		return svc.listOrgTeams(employee.EmailID, request)
	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// listOrgTeams retrieves all teams for the organization that the user is an admin of
func (svc *Service) listOrgTeams(userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Listing organization teams for user: %s", userName)

	// Get organizations where user is an admin
	orgAdmins, err := svc.orgSVC.GetAdminsOrganizations(userName)
	if err != nil {
		svc.logger.Printf("Failed to check organization admin status: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to verify permissions", err)
	}

	if len(orgAdmins) == 0 || len(orgAdmins) > 1 {
		svc.logger.Printf("User %s is not an organization admin or is part of more than 1 organization. Needs Review", userName)
		return svc.errorResponse(http.StatusForbidden, "Only organization admins can view all organization teams", nil)
	}

	// Get the organization (assuming user is admin of one org, take the first one)
	orgAdmin := orgAdmins[0]
	orgId := orgAdmin.OrganizationId

	svc.logger.Printf("User %s is admin of organization %s, fetching teams", userName, orgId)

	// Get all teams for the organization
	teams, err := svc.teamsSVC.GetOrganizationTeams(orgId)
	if err != nil {
		svc.logger.Printf("Failed to get organization teams: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to retrieve teams", err)
	}

	// Return the teams list
	body, err := json.Marshal(map[string]interface{}{
		"organizationId": orgId,
		"teams":          teams,
		"totalTeams":     len(teams),
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
