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
	emailSVC *companylib.EmailService
	orgSVC   *companylib.OrgServiceV2
	empSVC   *companylib.EmployeeService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("OrganizationAPI")

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "send-invitations")
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
		ctx:      ctx,
		logger:   logger,
		emailSVC: emailSvc,
		orgSVC:   orgSvc,
		empSVC:   empSvc,
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
	case "POST":
		return svc.sendInvitations(employee.EmailID, request)
	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// SendInvitationsRequest represents the request body for sending invitations
type SendInvitationsRequest struct {
	EmailAddresses   []string `json:"emailAddresses" validate:"required,min=1"`
	OrganizationName string   `json:"organizationName,omitempty"`
	InvitationLink   string   `json:"invitationLink,omitempty"`
	CustomMessage    string   `json:"customMessage,omitempty"`
}

// sendInvitations sends invitation emails to the provided email addresses
func (svc *Service) sendInvitations(userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Sending invitations from user: %s", userName)

	// Parse request body
	var req SendInvitationsRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate input
	if len(req.EmailAddresses) == 0 {
		return svc.errorResponse(http.StatusBadRequest, "At least one email address is required", nil)
	}

	// Get employee details to use display name
	employee, err := svc.empSVC.GetEmployeeDataByUserName(userName)
	if err != nil {
		svc.logger.Printf("Warning: Failed to get employee details: %v", err)
	}

	inviterName := userName
	if employee.DisplayName != "" {
		inviterName = employee.DisplayName
	}

	// Get organization details if user is part of one
	organizationName := req.OrganizationName
	if organizationName == "" {
		// Try to get organization from user's membership
		org, err := svc.orgSVC.GetAdminOrganization(userName)
		if err == nil && org != nil {
			organizationName = org.OrgName
		}
	}

	// Set default invitation link if not provided
	invitationLink := req.InvitationLink
	if invitationLink == "" {
		baseURL := os.Getenv("APP_BASE_URL")
		if baseURL == "" {
			baseURL = "https://app.gomovo.com"
		}
		invitationLink = fmt.Sprintf("%s/accept-invitation", baseURL)
	}

	// Prepare invitation input
	invitationInput := companylib.InvitationEmailInput{
		EmailAddresses:   req.EmailAddresses,
		OrganizationName: organizationName,
		InviterName:      inviterName,
		InvitationLink:   invitationLink,
		CustomMessage:    req.CustomMessage,
	}

	// Send invitation emails
	results, err := svc.emailSVC.SendInvitationEmails(invitationInput)
	if err != nil {
		svc.logger.Printf("Failed to send invitations: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to send invitations", err)
	}

	// Count successful and failed sends
	successCount := 0
	failedCount := 0
	for _, result := range results {
		if result.Success {
			successCount++
		} else {
			failedCount++
		}
	}

	svc.logger.Printf("Invitation results - Success: %d, Failed: %d", successCount, failedCount)

	// Return results
	body, err := json.Marshal(map[string]interface{}{
		"message":      fmt.Sprintf("Sent %d invitations successfully, %d failed", successCount, failedCount),
		"totalSent":    len(req.EmailAddresses),
		"successCount": successCount,
		"failedCount":  failedCount,
		"results":      results,
	})
	if err != nil {
		svc.logger.Printf("Failed to marshal response: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to create response", err)
	}

	statusCode := http.StatusOK
	if failedCount > 0 && successCount == 0 {
		// All failed
		statusCode = http.StatusInternalServerError
	} else if failedCount > 0 {
		// Partial success
		statusCode = http.StatusMultiStatus
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}

// getCognitoIdFromRequest extracts Cognito ID from Cognito authorizer context
func (svc *Service) getCognitoIdFromRequest(request events.APIGatewayProxyRequest) (string, error) {
	// Try to get from authorizer context first
	if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		if sub, ok := claims["sub"].(string); ok {
			return sub, nil
		}
	}

	// Fallback to headers (for testing/development)
	if cognitoId := request.Headers["cognito-id"]; cognitoId != "" {
		return cognitoId, nil
	}

	return "", fmt.Errorf("cognito ID not found in request")
}

// errorResponse creates an error response
func (svc *Service) errorResponse(statusCode int, message string, err error) (events.APIGatewayProxyResponse, error) {
	errorMessage := message
	if err != nil {
		errorMessage = fmt.Sprintf("%s: %v", message, err)
	}

	body, _ := json.Marshal(map[string]string{
		"error":   message,
		"details": errorMessage,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}
