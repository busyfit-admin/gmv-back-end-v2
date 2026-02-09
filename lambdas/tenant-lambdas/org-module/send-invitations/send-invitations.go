package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/golang-jwt/jwt/v5"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx      context.Context
	logger   *log.Logger
	emailSVC *companylib.EmailService
	orgSVC   *companylib.OrgServiceV2
	empSVC   *companylib.EmployeeService

	ddbClient *dynamodb.Client
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
	empSvc.TenantTeamsTable = os.Getenv("TENANT_TEAMS_TABLE")

	// Email service
	emailSvc := companylib.CreateEmailService(ctx, sesClient, logger)

	// Initialize organization service
	orgSvc := companylib.CreateOrgServiceV2(ctx, ddbclient, logger, empSvc, emailSvc)
	orgSvc.OrganizationTable = os.Getenv("ORGANIZATION_TABLE")
	orgSvc.PromoCodesTable = os.Getenv("PROMO_CODES_TABLE")

	svc := &Service{
		ctx:       ctx,
		logger:    logger,
		emailSVC:  emailSvc,
		orgSVC:    orgSvc,
		empSVC:    empSvc,
		ddbClient: ddbclient,
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
		return svc.sendInvitations(employee, request)
	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// InviteeInfo represents information about a single invitee
type InviteeInfo struct {
	Email  string `json:"email" validate:"required,email"`
	Role   string `json:"role" validate:"required"`
	TeamId string `json:"teamId,omitempty"`
}

// SendInvitationsRequest represents the request body for sending invitations
type SendInvitationsRequest struct {
	Invitees         []InviteeInfo `json:"invitees" validate:"required,min=1"`
	OrganizationName string        `json:"organizationName,omitempty"`
	InvitationLink   string        `json:"invitationLink,omitempty"`
	CustomMessage    string        `json:"customMessage,omitempty"`
}

// InvitationTokenClaims represents the JWT claims for invitation tokens
type InvitationTokenClaims struct {
	Email            string `json:"email"`
	OrganizationId   string `json:"organizationId"`
	OrganizationName string `json:"organizationName"`
	TeamId           string `json:"teamId,omitempty"`
	Role             string `json:"role"`
	InvitedBy        string `json:"invitedBy"`
	jwt.RegisteredClaims
}

// sendInvitations sends invitation emails to the provided email addresses
func (svc *Service) sendInvitations(employee companylib.EmployeeDynamodbData, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Sending invitations from user: %s", employee.DisplayName)

	// Parse request body
	var req SendInvitationsRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate input
	if len(req.Invitees) == 0 {
		return svc.errorResponse(http.StatusBadRequest, "At least one invitee is required", nil)
	}

	inviterName := employee.EmailID
	if employee.DisplayName != "" {
		inviterName = employee.DisplayName
	}

	// Get organization details if user is part of one
	var organizationId string
	organizationName := req.OrganizationName
	org, err := svc.orgSVC.GetAdminOrganization(employee.EmailID)
	if err != nil {
		svc.logger.Printf("Failed to get organization details: %v", err)
	} else {
		svc.logger.Printf("User is part of organization: %s (%s)", org.OrgName, org.OrganizationId)
		organizationId = org.OrganizationId
	}

	// Get base URL for invitation links
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "https://app.gomovo.com"
	}

	// Process each invitee: generate token, send email, create DDB record
	successCount := 0
	failedCount := 0
	results := make([]companylib.InvitationEmailResult, 0, len(req.Invitees))

	for _, invitee := range req.Invitees {
		result := companylib.InvitationEmailResult{
			Email: invitee.Email,
		}

		// Fetch team name if teamId is provided
		var teamName string
		if invitee.TeamId != "" {
			fetchedTeamName, err := svc.getTeamName(invitee.TeamId)
			if err != nil {
				svc.logger.Printf("Warning: Failed to fetch team name for %s: %v", invitee.TeamId, err)
			} else {
				teamName = fetchedTeamName
			}
		}

		// Generate invitation link with JWT token for this specific invitee
		var invitationLink string
		if req.InvitationLink != "" {
			// Use custom invitation link if provided
			invitationLink = req.InvitationLink
		} else {
			// Generate JWT token with invitation data
			token, err := svc.GenerateInvitationToken(invitee.Email, organizationId, organizationName, invitee.TeamId, invitee.Role, employee.EmailID)
			if err != nil {
				svc.logger.Printf("Warning: Failed to generate invitation token for %s: %v", invitee.Email, err)
				invitationLink = fmt.Sprintf("%s/accept-invitation", baseURL)
			} else {
				invitationLink = fmt.Sprintf("%s/accept-invitation?token=%s", baseURL, token)
			}
		}

		// Prepare invitation input for this specific invitee
		invitationInput := companylib.InvitationEmailInput{
			EmailAddresses:   []string{invitee.Email},
			OrganizationName: organizationName,
			TeamName:         teamName,
			InviterName:      inviterName,
			InvitationLink:   invitationLink,
			CustomMessage:    req.CustomMessage,
		}

		// Send invitation email
		emailResults, err := svc.emailSVC.SendInvitationEmails(invitationInput)
		if err != nil || len(emailResults) == 0 || !emailResults[0].Success {
			svc.logger.Printf("Failed to send invitation to %s: %v", invitee.Email, err)
			result.Success = false
			if err != nil {
				result.Error = err.Error()
			} else if len(emailResults) > 0 {
				result.Error = emailResults[0].Error
			}
			failedCount++
		} else {
			result.Success = true
			successCount++

			// Create employee record with INVITED status
			if err := svc.createInvitedEmployee(invitee.Email, invitee.Role, invitee.TeamId, organizationId, employee.UserName); err != nil {
				svc.logger.Printf("Warning: Failed to create employee record for %s: %v", invitee.Email, err)
				// Don't fail the invitation if employee record creation fails
			}
		}

		results = append(results, result)
	}

	svc.logger.Printf("Invitation results - Success: %d, Failed: %d", successCount, failedCount)

	// Return results
	body, err := json.Marshal(map[string]interface{}{
		"message":      fmt.Sprintf("Sent %d invitations successfully, %d failed", successCount, failedCount),
		"totalSent":    len(req.Invitees),
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

// getTeamName fetches the team name by teamId from DynamoDB
func (svc *Service) getTeamName(teamId string) (string, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(svc.empSVC.TenantTeamsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: teamId},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	}

	result, err := svc.ddbClient.GetItem(svc.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get team: %w", err)
	}

	if result.Item == nil {
		return "", fmt.Errorf("team not found")
	}

	// Extract team name from the result
	if teamNameAttr, ok := result.Item["TeamName"]; ok {
		if teamNameVal, ok := teamNameAttr.(*types.AttributeValueMemberS); ok {
			return teamNameVal.Value, nil
		}
	}

	return "", fmt.Errorf("team name not found in team metadata")
}

// createInvitedEmployee creates an employee record with INVITED status and optionally adds to team
func (svc *Service) createInvitedEmployee(email, role, teamId, organizationId, invitedBy string) error {
	// Check if employee already exists
	emp, err := svc.empSVC.GetEmployeeDataByUserName(email)
	if err == nil && emp.CognitoId != "" {
		svc.logger.Printf("Employee %s already exists, skipping creation", email)
		return nil
	}

	// Create employee record with INVITED status

	// Build transaction items
	transactItems := []types.TransactWriteItem{}

	// 1. Add to Employee table - disabling this as the new employee is record is created only when user is accepting the invitiation.
	//
	// employeeData := companylib.EmployeeDynamodbData{
	// 	UserName:  email,
	// 	EmailID:   email,
	// 	Status:    "INVITED",
	// 	Source:    fmt.Sprintf("Invitation-By-%s", invitedBy),
	// 	CreatedAt: time.Now().UTC().Format(time.RFC3339),
	// 	UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	// }
	// putItemEmployeeTable := types.TransactWriteItem{
	// 	Put: &types.Put{
	// 		TableName: aws.String(svc.empSVC.EmployeeTable),
	// 		Item: map[string]types.AttributeValue{
	// 			"UserName":  &types.AttributeValueMemberS{Value: employeeData.UserName},
	// 			"EmailId":   &types.AttributeValueMemberS{Value: employeeData.EmailID},
	// 			"Status":    &types.AttributeValueMemberS{Value: employeeData.Status},
	// 			"Source":    &types.AttributeValueMemberS{Value: employeeData.Source},
	// 			"CreatedAt": &types.AttributeValueMemberS{Value: employeeData.CreatedAt},
	// 			"UpdatedAt": &types.AttributeValueMemberS{Value: employeeData.UpdatedAt},
	// 		},
	// 		ConditionExpression: aws.String("attribute_not_exists(UserName)"),
	// 	},
	// }
	// transactItems = append(transactItems, putItemEmployeeTable)

	// 2. Add to Organization table (ORG#orgId -> USER#email mapping)
	if organizationId != "" {
		putItemOrgTable := types.TransactWriteItem{
			Put: &types.Put{
				TableName: aws.String(svc.orgSVC.OrganizationTable),
				Item: map[string]types.AttributeValue{
					"PK":             &types.AttributeValueMemberS{Value: fmt.Sprintf("%s", organizationId)},
					"SK":             &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", email)},
					"GSI1PK":         &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", email)},
					"GSI1SK":         &types.AttributeValueMemberS{Value: fmt.Sprintf("%s", organizationId)},
					"OrganizationId": &types.AttributeValueMemberS{Value: organizationId},
					"UserName":       &types.AttributeValueMemberS{Value: email},
					"Role":           &types.AttributeValueMemberS{Value: role},
					"Status":         &types.AttributeValueMemberS{Value: "INVITED"},
					"IsActive":       &types.AttributeValueMemberBOOL{Value: true}, // Active in organization by default unless they are removed later.
					"AddedAt":        &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
					"UpdatedAt":      &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
				},
				ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
			},
		}
		transactItems = append(transactItems, putItemOrgTable)
	}

	// 3. Add to Team table if teamId is provided
	if teamId != "" {
		putItemTeamTable := types.TransactWriteItem{
			Put: &types.Put{
				TableName: aws.String(svc.empSVC.TenantTeamsTable), // Teams use same table
				Item: map[string]types.AttributeValue{
					"PK":       &types.AttributeValueMemberS{Value: fmt.Sprintf("%s", teamId)},
					"SK":       &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", email)},
					"GSI1PK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", email)},
					"GSI1SK":   &types.AttributeValueMemberS{Value: fmt.Sprintf("%s", teamId)},
					"TeamId":   &types.AttributeValueMemberS{Value: teamId},
					"UserName": &types.AttributeValueMemberS{Value: email},
					"Role":     &types.AttributeValueMemberS{Value: role},
					"JoinedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
					"IsActive": &types.AttributeValueMemberBOOL{Value: true}, // Active in teams by default unless they are removed later.
				},
				ConditionExpression: aws.String("attribute_not_exists(PK) AND attribute_not_exists(SK)"),
			},
		}
		transactItems = append(transactItems, putItemTeamTable)
	}

	// Execute transaction
	_, err = svc.ddbClient.TransactWriteItems(svc.ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})
	if err != nil {
		// Ignore conditional check failures (user already exists)
		if strings.Contains(err.Error(), "ConditionalCheckFailed") {
			svc.logger.Printf("Employee %s already exists in one or more tables", email)
			return nil
		}
		return fmt.Errorf("failed to create employee record: %w", err)
	}

	logMsg := fmt.Sprintf("Created INVITED employee record for %s with role %s", email, role)
	if teamId != "" {
		logMsg += fmt.Sprintf(" in team %s", teamId)
	}
	svc.logger.Printf(logMsg)
	return nil
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
