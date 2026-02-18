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
	ctx, root := xray.BeginSegment(context.TODO(), "manage-org-users")
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

	// Extract orgId from headers
	orgId := request.Headers["organization-id"]
	if orgId == "" {
		orgId = request.Headers["Organization-Id"]
	}
	if orgId == "" {
		return svc.errorResponse(http.StatusBadRequest, "Organization ID header is required", nil)
	}

	switch request.HTTPMethod {
	case "GET":
		return svc.listOrgUsers(orgId, employee.EmailID, request)
	case "POST":
		return svc.addOrgUser(orgId, employee.EmailID, request)
	case "PUT":
		return svc.updateOrgUser(orgId, employee.EmailID, request)
	case "DELETE":
		return svc.removeOrgUser(orgId, employee.EmailID, request)
	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// listOrgUsers retrieves all users (admins and regular users) in an organization
func (svc *Service) listOrgUsers(orgId string, requestingUser string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Listing users for organization %s requested by: %s", orgId, requestingUser)

	// Verify user is admin of the organization
	isAdmin, err := svc.orgSVC.IsOrgAdmin(orgId, requestingUser)
	if err != nil {
		svc.logger.Printf("Failed to check admin status: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to verify permissions", err)
	}
	if !isAdmin {
		return svc.errorResponse(http.StatusForbidden, "Access denied: Not an organization admin", nil)
	}

	// Get all admins
	admins, err := svc.orgSVC.GetOrgAdmins(orgId, requestingUser)
	if err != nil {
		svc.logger.Printf("Failed to get org admins: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to retrieve admins", err)
	}

	// Get all regular users
	users, err := svc.orgSVC.GetOrgUsers(orgId)
	if err != nil {
		svc.logger.Printf("Failed to get org users: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to retrieve users", err)
	}

	// Combine admins and users into a unified response
	type UserInfo struct {
		UserName    string `json:"userName"`
		DisplayName string `json:"displayName"`
		Role        string `json:"role"`
		JoinedAt    string `json:"joinedAt,omitempty"`
		AddedAt     string `json:"addedAt,omitempty"`
		IsActive    bool   `json:"isActive"`
		UserType    string `json:"userType"` // "admin" or "user"
	}

	allUsers := make([]UserInfo, 0)

	// Add admins
	for _, admin := range admins {
		allUsers = append(allUsers, UserInfo{
			UserName:    admin.UserName,
			DisplayName: admin.DisplayName,
			Role:        string(admin.Role),
			AddedAt:     admin.AddedAt,
			IsActive:    admin.IsActive,
			UserType:    "admin",
		})
	}

	// Add regular users
	for _, user := range users {
		allUsers = append(allUsers, UserInfo{
			UserName:    user.UserName,
			DisplayName: user.DisplayName,
			Role:        string(user.Role),
			JoinedAt:    user.JoinedAt,
			IsActive:    user.IsActive,
			UserType:    "user",
		})
	}

	// Return the combined list
	body, err := json.Marshal(map[string]interface{}{
		"users":      allUsers,
		"totalCount": len(allUsers),
		"adminCount": len(admins),
		"userCount":  len(users),
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

// AddOrgUserRequest represents the request to add a user
type AddOrgUserRequest struct {
	UserName string `json:"userName"`
	Role     string `json:"role"`               // "owner", "admin", "manager", "member"
	UserType string `json:"userType,omitempty"` // "admin" or "user" - defaults to "user"
}

// addOrgUser adds a new user (admin or regular user) to the organization
func (svc *Service) addOrgUser(orgId string, requestingUser string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Adding user to organization %s requested by: %s", orgId, requestingUser)

	// Parse request body
	var input AddOrgUserRequest
	if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate input
	if input.UserName == "" {
		return svc.errorResponse(http.StatusBadRequest, "userName is required", nil)
	}
	if input.Role == "" {
		return svc.errorResponse(http.StatusBadRequest, "role is required", nil)
	}

	// Default to "user" type if not specified
	if input.UserType == "" {
		input.UserType = "user"
	}

	// Verify requesting user is admin of the organization
	isAdmin, err := svc.orgSVC.IsOrgAdmin(orgId, requestingUser)
	if err != nil {
		svc.logger.Printf("Failed to check admin status: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to verify permissions", err)
	}
	if !isAdmin {
		return svc.errorResponse(http.StatusForbidden, "Access denied: Not an organization admin", nil)
	}

	// Determine if we're adding an admin or regular user
	if input.UserType == "admin" {
		// Validate role is valid for admin
		validAdminRoles := map[string]bool{
			string(companylib.OrgAdminRoleOwner): true,
			string(companylib.OrgAdminRoleAdmin): true,
		}
		if !validAdminRoles[input.Role] {
			return svc.errorResponse(http.StatusBadRequest, "Invalid admin role. Must be 'owner', 'admin', or 'manager'", nil)
		}

		// Add as admin
		role := companylib.OrgAdminRole(input.Role)
		err = svc.orgSVC.AddOrgAdmin(orgId, input.UserName, role, requestingUser)
		if err != nil {
			svc.logger.Printf("Failed to add org admin: %v", err)
			return svc.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to add admin: %v", err), err)
		}
	} else {
		// For regular users, we don't have an AddOrgUser method yet
		// This would be added to the company-lib
		return svc.errorResponse(http.StatusNotImplemented, "Adding regular users is not yet implemented. Please add users as admins with appropriate roles.", nil)
	}

	// Return success response
	body, err := json.Marshal(map[string]interface{}{
		"message":  "User added successfully",
		"userName": input.UserName,
		"role":     input.Role,
		"userType": input.UserType,
	})
	if err != nil {
		svc.logger.Printf("Failed to marshal response: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to create response", err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}

// UpdateOrgUserRequest represents the request to update a user's role
type UpdateOrgUserRequest struct {
	UserName string `json:"userName"`
	Role     string `json:"role"`
}

// updateOrgUser updates a user's role in the organization
func (svc *Service) updateOrgUser(orgId string, requestingUser string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Updating user in organization %s requested by: %s", orgId, requestingUser)

	// Parse request body
	var input UpdateOrgUserRequest
	if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate input
	if input.UserName == "" {
		return svc.errorResponse(http.StatusBadRequest, "userName is required", nil)
	}
	if input.Role == "" {
		return svc.errorResponse(http.StatusBadRequest, "role is required", nil)
	}

	// Verify requesting user is admin of the organization
	isAdmin, err := svc.orgSVC.IsOrgAdmin(orgId, requestingUser)
	if err != nil {
		svc.logger.Printf("Failed to check admin status: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to verify permissions", err)
	}
	if !isAdmin {
		return svc.errorResponse(http.StatusForbidden, "Access denied: Not an organization admin", nil)
	}

	// Note: Updating user roles would require additional methods in company-lib
	// For now, we'll return not implemented
	return svc.errorResponse(http.StatusNotImplemented, "Updating user roles is not yet implemented", nil)
}

// RemoveOrgUserRequest can be passed in body or path parameter
type RemoveOrgUserRequest struct {
	UserName string `json:"userName"`
}

// removeOrgUser removes a user from the organization
func (svc *Service) removeOrgUser(orgId string, requestingUser string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Removing user from organization %s requested by: %s", orgId, requestingUser)

	// Parse request body or get from query parameters
	var userName string
	if request.Body != "" {
		var input RemoveOrgUserRequest
		if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
			svc.logger.Printf("Failed to parse request body: %v", err)
			return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
		}
		userName = input.UserName
	} else {
		// Try to get from query parameters
		userName = request.QueryStringParameters["userName"]
	}

	if userName == "" {
		return svc.errorResponse(http.StatusBadRequest, "userName is required", nil)
	}

	// Verify requesting user is admin of the organization
	isAdmin, err := svc.orgSVC.IsOrgAdmin(orgId, requestingUser)
	if err != nil {
		svc.logger.Printf("Failed to check admin status: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to verify permissions", err)
	}
	if !isAdmin {
		return svc.errorResponse(http.StatusForbidden, "Access denied: Not an organization admin", nil)
	}

	// Try to remove as admin first
	err = svc.orgSVC.RemoveOrgAdmin(orgId, userName, requestingUser)
	if err != nil {
		// If not found as admin, try to remove as regular user
		err = svc.orgSVC.RemoveOrgUser(orgId, userName)
		if err != nil {
			svc.logger.Printf("Failed to remove user: %v", err)
			return svc.errorResponse(http.StatusInternalServerError, fmt.Sprintf("Failed to remove user: %v", err), err)
		}
	}

	// Return success response
	body, err := json.Marshal(map[string]interface{}{
		"message":  "User removed successfully",
		"userName": userName,
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
