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
}

var RESP_HEADERS = companylib.GetHeadersForAPI("TeamsAPI")

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "list-user-teams")
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

	emailSvc := companylib.CreateEmailService(ctx, sesClient, logger)
	// Initialize teams service
	teamsSvc := companylib.CreateTeamsServiceV2(ctx, ddbclient, logger, empSvc, emailSvc)
	teamsSvc.TeamsTable = os.Getenv("TEAMS_TABLE")

	svc := &Service{
		ctx:      ctx,
		logger:   logger,
		teamsSVC: teamsSvc,
		empSVC:   empSvc,
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
		return svc.listUserTeams(employee.EmailID, cognitoId, request)
	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// listUserTeams retrieves all teams for the current user
func (svc *Service) listUserTeams(userName string, userCognitoId string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Listing teams for user: %s", userName)

	teams, err := svc.teamsSVC.GetUserTeams(userName, userCognitoId)
	if err != nil {
		svc.logger.Printf("Failed to get user teams: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to retrieve teams", err)
	}

	// Get current team from stored preference (already marked in teams)
	currentTeamId, err := svc.teamsSVC.GetCurrentTeam(userCognitoId)
	if err != nil {
		svc.logger.Printf("Warning: Failed to get current team: %v", err)
	}
	svc.logger.Printf("User %s current team: %s", userName, currentTeamId)

	response := map[string]interface{}{
		"teams":       teams,
		"currentTeam": currentTeamId,
		"count":       len(teams),
	}

	body, err := json.Marshal(response)
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
