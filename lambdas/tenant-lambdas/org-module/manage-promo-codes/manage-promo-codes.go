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
	ctx, root := xray.BeginSegment(context.TODO(), "manage-promo-codes")
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
	case "POST":
		return svc.applyPromoCode(orgId, employee.EmailID, request)
	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// applyPromoCode applies a promo code to an organization
func (svc *Service) applyPromoCode(orgId string, userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Applying promo code for organization %s, user: %s", orgId, userName)

	// Parse request body
	var input companylib.ApplyPromoCodeInput
	if err := json.Unmarshal([]byte(request.Body), &input); err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate input
	if input.PromoCode == "" {
		return svc.errorResponse(http.StatusBadRequest, "Promo code is required", nil)
	}

	// Set the organization ID
	input.OrganizationId = orgId

	// Apply the promo code
	err := svc.orgSVC.ApplyPromoCode(input, userName)
	if err != nil {
		svc.logger.Printf("Failed to apply promo code: %v", err)

		// Handle specific error cases
		switch err.Error() {
		case fmt.Sprintf("user %s is not an admin of organization %s", userName, orgId):
			return svc.errorResponse(http.StatusForbidden, "Access denied: Not an organization admin", err)
		case "promo code is not active":
			return svc.errorResponse(http.StatusBadRequest, "Promo code is not active", err)
		case "promo code is not yet valid":
			return svc.errorResponse(http.StatusBadRequest, "Promo code is not yet valid", err)
		case "promo code has expired":
			return svc.errorResponse(http.StatusBadRequest, "Promo code has expired", err)
		case "promo code usage limit exceeded":
			return svc.errorResponse(http.StatusBadRequest, "Promo code usage limit exceeded", err)
		case "promo code is not applicable to current plan":
			return svc.errorResponse(http.StatusBadRequest, "Promo code is not applicable to your current plan", err)
		default:
			if fmt.Sprintf("promo code not found: %s", input.PromoCode) == err.Error() {
				return svc.errorResponse(http.StatusNotFound, "Promo code not found", err)
			}
			return svc.errorResponse(http.StatusInternalServerError, "Failed to apply promo code", err)
		}
	}

	// Get updated organization details
	organization, err := svc.orgSVC.GetOrganization(orgId)
	if err != nil {
		svc.logger.Printf("Failed to get updated organization: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Promo code applied but failed to retrieve updated details", err)
	}

	// Get promo code details for response
	promoCode, err := svc.orgSVC.GetPromoCode(input.PromoCode)
	if err != nil {
		svc.logger.Printf("Failed to get promo code details: %v", err)
		promoCode = &companylib.PromoCode{PromoCode: input.PromoCode}
	}

	// Return the result
	body, err := json.Marshal(map[string]interface{}{
		"message": "Promo code applied successfully",
		"appliedPromo": map[string]interface{}{
			"promoCode":       promoCode.PromoCode,
			"discountPercent": promoCode.DiscountPercent,
			"discountAmount":  promoCode.DiscountAmount,
			"freeTrialDays":   promoCode.FreeTrialDays,
			"validUntil":      organization.PromoValidUntil,
		},
		"updatedSubscription": map[string]interface{}{
			"billingStatus":    organization.OrgBillingStatus,
			"promoDiscount":    organization.PromoDiscountPercent,
			"trialEndDate":     organization.TrialEndDate,
			"appliedPromoCode": organization.AppliedPromoCode,
		},
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
