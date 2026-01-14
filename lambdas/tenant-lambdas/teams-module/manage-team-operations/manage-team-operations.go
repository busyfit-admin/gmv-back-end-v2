package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
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
	ctx, root := xray.BeginSegment(context.TODO(), "manage-team")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	// Initialize employee service
	empSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	empSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	empSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	// Initialize teams service
	teamsSvc := companylib.CreateTeamsServiceV2(ctx, ddbclient, logger, empSvc)
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

	userName := employee.UserName

	// Route based on path and method
	pathParts := strings.Split(strings.Trim(request.Path, "/"), "/")

	switch request.HTTPMethod {
	case "GET":
		if len(pathParts) >= 2 && pathParts[len(pathParts)-1] == "members" {
			// GET /teams/{teamId}/members
			teamId := pathParts[len(pathParts)-2]
			return svc.getTeamMembers(teamId, userName)
		} else if len(pathParts) >= 1 {
			// GET /teams/{teamId}
			teamId := pathParts[len(pathParts)-1]
			return svc.getTeamDetails(teamId, userName)
		}
		return svc.errorResponse(http.StatusBadRequest, "Invalid path", nil)

	case "PATCH":
		if len(pathParts) >= 2 && pathParts[len(pathParts)-1] == "status" {
			// PATCH /teams/{teamId}/status
			teamId := pathParts[len(pathParts)-2]
			return svc.updateTeamStatus(teamId, userName, request)
		}
		return svc.errorResponse(http.StatusBadRequest, "Invalid path", nil)

	case "POST":
		if len(pathParts) >= 2 && pathParts[len(pathParts)-1] == "members" {
			// POST /teams/{teamId}/members
			teamId := pathParts[len(pathParts)-2]
			return svc.addTeamMembers(teamId, userName, request)
		} else if len(pathParts) >= 3 && pathParts[len(pathParts)-1] == "role" {
			// POST /teams/{teamId}/members/{username}/role
			teamId := pathParts[len(pathParts)-4]
			return svc.updateMemberRole(teamId, userName, request)
		}
		return svc.errorResponse(http.StatusBadRequest, "Invalid path", nil)

	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// getTeamDetails retrieves details of a specific team
func (svc *Service) getTeamDetails(teamId string, userName string) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Getting team details for team: %s", teamId)

	// Verify user is a member of the team
	metadata, err := svc.teamsSVC.GetTeamMetadata(teamId)
	if err != nil {
		svc.logger.Printf("Failed to get team metadata: %v", err)
		return svc.errorResponse(http.StatusNotFound, "Team not found", err)
	}

	body, err := json.Marshal(metadata)
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

// getTeamMembers retrieves all members of a team
func (svc *Service) getTeamMembers(teamId string, userName string) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Getting team members for team: %s", teamId)

	members, err := svc.teamsSVC.GetTeamMembers(teamId)
	if err != nil {
		svc.logger.Printf("Failed to get team members: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to retrieve team members", err)
	}

	response := map[string]interface{}{
		"members": members,
		"count":   len(members),
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

// updateTeamStatus updates the status of a team (activate/deactivate)
func (svc *Service) updateTeamStatus(teamId string, userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Updating team status for team: %s by user: %s", teamId, userName)

	// Parse request body
	var input struct {
		Status companylib.TeamStatus `json:"status"`
	}
	if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate status
	if input.Status != companylib.TeamStatusActive && input.Status != companylib.TeamStatusInactive {
		return svc.errorResponse(http.StatusBadRequest, "Invalid status. Must be ACTIVE or INACTIVE", nil)
	}

	// Update team status
	err := svc.teamsSVC.UpdateTeamStatus(teamId, input.Status, userName)
	if err != nil {
		svc.logger.Printf("Failed to update team status: %v", err)
		if strings.Contains(err.Error(), "not an admin") {
			return svc.errorResponse(http.StatusForbidden, "Only admins can update team status", err)
		}
		return svc.errorResponse(http.StatusInternalServerError, "Failed to update team status", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"message": "Team status updated successfully",
		"teamId":  teamId,
		"status":  input.Status,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}

// addTeamMembers adds members to a team
func (svc *Service) addTeamMembers(teamId string, userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Adding members to team: %s by user: %s", teamId, userName)

	// Parse request body
	var input struct {
		UserNames []string `json:"userNames"`
	}
	if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate input
	if len(input.UserNames) == 0 {
		return svc.errorResponse(http.StatusBadRequest, "At least one username is required", nil)
	}

	// Add members
	addInput := companylib.AddTeamMembersInput{
		TeamId:    teamId,
		UserNames: input.UserNames,
	}

	err := svc.teamsSVC.AddTeamMembers(addInput, userName)
	if err != nil {
		svc.logger.Printf("Failed to add team members: %v", err)
		if strings.Contains(err.Error(), "not an admin") {
			return svc.errorResponse(http.StatusForbidden, "Only admins can add members", err)
		}
		if strings.Contains(err.Error(), "inactive team") {
			return svc.errorResponse(http.StatusBadRequest, "Cannot add members to inactive team", err)
		}
		return svc.errorResponse(http.StatusInternalServerError, "Failed to add team members", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"message": fmt.Sprintf("Successfully added %d members to team", len(input.UserNames)),
		"teamId":  teamId,
		"count":   len(input.UserNames),
	})

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}

// updateMemberRole updates a member's role in the team
func (svc *Service) updateMemberRole(teamId string, userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Updating member role in team: %s by user: %s", teamId, userName)

	// Parse request body
	var input struct {
		UserName string                    `json:"userName"`
		Role     companylib.TeamMemberRole `json:"role"`
	}
	if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate input
	if input.UserName == "" {
		return svc.errorResponse(http.StatusBadRequest, "Username is required", nil)
	}
	if input.Role != companylib.TeamMemberRoleAdmin && input.Role != companylib.TeamMemberRoleMember {
		return svc.errorResponse(http.StatusBadRequest, "Invalid role. Must be ADMIN or MEMBER", nil)
	}

	// Update member role
	updateInput := companylib.UpdateMemberRoleInput{
		TeamId:   teamId,
		UserName: input.UserName,
		Role:     input.Role,
	}

	err := svc.teamsSVC.UpdateMemberRole(updateInput, userName)
	if err != nil {
		svc.logger.Printf("Failed to update member role: %v", err)
		if strings.Contains(err.Error(), "not an admin") {
			return svc.errorResponse(http.StatusForbidden, "Only admins can update member roles", err)
		}
		if strings.Contains(err.Error(), "last admin") {
			return svc.errorResponse(http.StatusBadRequest, "Cannot demote the last admin of the team", err)
		}
		return svc.errorResponse(http.StatusInternalServerError, "Failed to update member role", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"message":  "Member role updated successfully",
		"teamId":   teamId,
		"userName": input.UserName,
		"role":     input.Role,
	})

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
