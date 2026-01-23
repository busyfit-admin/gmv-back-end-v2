package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	ctx          context.Context
	logger       *log.Logger
	attributeSVC *companylib.TeamAttributeServiceV2
	teamsSVC     *companylib.TeamsServiceV2
	empSVC       *companylib.EmployeeService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("EngagementsAPI")

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "manage-team-attributes")
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
	teamsSvc := companylib.CreateTeamsServiceV2(ctx, ddbclient, logger, empSvc, nil)
	teamsSvc.TeamsTable = os.Getenv("TEAMS_TABLE")

	// Initialize attribute service
	attributeSvc := companylib.CreateTeamAttributeServiceV2(ctx, ddbclient, logger)
	attributeSvc.TeamAttributesTable = os.Getenv("TEAM_ATTRIBUTES_TABLE")
	attributeSvc.TeamAttributesTeamIdIndex = os.Getenv("TEAM_ATTRIBUTES_TEAMID_INDEX")

	svc := &Service{
		ctx:          ctx,
		logger:       logger,
		attributeSVC: attributeSvc,
		teamsSVC:     teamsSvc,
		empSVC:       empSvc,
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

	userName := employee.EmailID

	// Route based on path and method
	// Expected paths:
	// POST /v2/teams/{teamId}/attributes - Create custom attribute (admin only)
	// GET /v2/teams/{teamId}/attributes - List all attributes for team (all members)
	// GET /v2/teams/{teamId}/attributes?type=SKILL - List attributes filtered by type
	// PATCH /v2/teams/{teamId}/attributes/{attributeId} - Update custom attribute (admin only)
	// DELETE /v2/teams/{teamId}/attributes/{attributeId} - Delete custom attribute (admin only)

	pathParts := strings.Split(strings.Trim(request.Path, "/"), "/")

	// Validate path structure: /v2/teams/{teamId}/attributes[/{attributeId}]
	// pathParts[0] = "v2", pathParts[1] = "teams", pathParts[2] = teamId, pathParts[3] = "attributes", pathParts[4] = attributeId (optional)
	if len(pathParts) < 4 || pathParts[0] != "v2" || pathParts[1] != "teams" || pathParts[3] != "attributes" {
		svc.logger.Printf("Invalid path structure. pathParts: %v", pathParts)
		return svc.errorResponse(http.StatusBadRequest, "Invalid path format. Expected /v2/teams/{teamId}/attributes[/{attributeId}]", nil)
	}

	// URL decode the team ID in case it contains encoded characters like %23 (#)
	teamId, err := url.QueryUnescape(pathParts[2])
	if err != nil {
		svc.logger.Printf("Failed to decode team ID: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid team ID format", err)
	}

	// Verify team exists and user is a member
	isMember, isAdmin, err := svc.verifyTeamMembership(teamId, userName)
	if err != nil {
		svc.logger.Printf("Failed to verify team membership: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to verify team membership", err)
	}

	if !isMember {
		return svc.errorResponse(http.StatusForbidden, "User is not a member of this team", nil)
	}

	switch request.HTTPMethod {
	case "POST":
		// Only team admins can create custom attributes
		if !isAdmin {
			return svc.errorResponse(http.StatusForbidden, "Only team admins can create custom attributes", nil)
		}
		return svc.createCustomAttribute(teamId, userName, request)

	case "GET":
		// All team members can list attributes
		return svc.listTeamAttributes(teamId, request)

	case "PATCH":
		// Only team admins can update custom attributes
		if !isAdmin {
			return svc.errorResponse(http.StatusForbidden, "Only team admins can update custom attributes", nil)
		}
		// Validate that attributeId is provided in path
		if len(pathParts) < 5 {
			return svc.errorResponse(http.StatusBadRequest, "Attribute ID is required for update operation", nil)
		}
		attributeId, err := url.QueryUnescape(pathParts[4])
		if err != nil {
			return svc.errorResponse(http.StatusBadRequest, "Invalid attribute ID format", err)
		}
		return svc.updateCustomAttribute(teamId, attributeId, userName, request)

	case "DELETE":
		// Only team admins can delete custom attributes
		if !isAdmin {
			return svc.errorResponse(http.StatusForbidden, "Only team admins can delete custom attributes", nil)
		}
		// Validate that attributeId is provided in path
		if len(pathParts) < 5 {
			return svc.errorResponse(http.StatusBadRequest, "Attribute ID is required for delete operation", nil)
		}
		attributeId, err := url.QueryUnescape(pathParts[4])
		if err != nil {
			return svc.errorResponse(http.StatusBadRequest, "Invalid attribute ID format", err)
		}
		return svc.deleteCustomAttribute(teamId, attributeId, userName)

	default:
		return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
	}
}

// createCustomAttribute creates a new custom attribute for a team
func (svc *Service) createCustomAttribute(teamId string, userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Creating custom attribute for team: %s by user: %s", teamId, userName)

	// Parse request body
	type CreateAttributeRequest struct {
		AttributeType string `json:"attributeType" validate:"required"`
		Name          string `json:"name" validate:"required"`
		Description   string `json:"description"`
	}

	var req CreateAttributeRequest
	err := json.Unmarshal([]byte(request.Body), &req)
	if err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate attribute type
	var attrType companylib.TeamAttributeType
	switch strings.ToUpper(req.AttributeType) {
	case "SKILL":
		attrType = companylib.AttributeTypeSkill
	case "VALUE":
		attrType = companylib.AttributeTypeValue
	case "MILESTONE":
		attrType = companylib.AttributeTypeMilestone
	case "METRIC":
		attrType = companylib.AttributeTypeMetric
	default:
		return svc.errorResponse(http.StatusBadRequest, "Invalid attribute type. Must be SKILL, VALUE, MILESTONE, or METRIC", nil)
	}

	// Create the attribute
	attribute := companylib.TeamAttribute{
		TeamId:        teamId,
		AttributeType: attrType,
		Name:          req.Name,
		Description:   req.Description,
		CreatedBy:     userName,
	}

	err = svc.attributeSVC.CreateCustomAttribute(attribute)
	if err != nil {
		svc.logger.Printf("Failed to create attribute: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to create attribute", err)
	}

	// Return the created attribute
	body, err := json.Marshal(map[string]interface{}{
		"message": "Attribute created successfully",
		"attribute": map[string]string{
			"attributeId":   attribute.AttributeId,
			"teamId":        attribute.TeamId,
			"attributeType": string(attribute.AttributeType),
			"name":          attribute.Name,
			"description":   attribute.Description,
			"createdBy":     attribute.CreatedBy,
		},
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

// listTeamAttributes lists all attributes for a team, optionally filtered by type
func (svc *Service) listTeamAttributes(teamId string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Listing attributes for team: %s", teamId)

	// Check if type filter is provided
	typeFilter := request.QueryStringParameters["type"]

	var attributes []companylib.TeamAttribute
	var err error

	if typeFilter != "" {
		// Validate and filter by type
		var attrType companylib.TeamAttributeType
		switch strings.ToUpper(typeFilter) {
		case "SKILL":
			attrType = companylib.AttributeTypeSkill
		case "VALUE":
			attrType = companylib.AttributeTypeValue
		case "MILESTONE":
			attrType = companylib.AttributeTypeMilestone
		case "METRIC":
			attrType = companylib.AttributeTypeMetric
		default:
			return svc.errorResponse(http.StatusBadRequest, "Invalid type filter. Must be SKILL, VALUE, MILESTONE, or METRIC", nil)
		}

		attributes, err = svc.attributeSVC.ListTeamAttributes(teamId, &attrType)
	} else {
		// Get all attributes
		attributes, err = svc.attributeSVC.ListTeamAttributes(teamId, nil)
	}

	if err != nil {
		svc.logger.Printf("Failed to list attributes: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to list attributes", err)
	}

	// Group by type for better response structure
	grouped, err := svc.attributeSVC.GetAttributesByType(teamId)
	if err != nil {
		svc.logger.Printf("Failed to get attributes by type: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to get attributes by type", err)
	}
	// Prepare response
	body, err := json.Marshal(map[string]interface{}{
		"teamId": teamId,
		"attributes": map[string]interface{}{
			"skills":     grouped.Skills,
			"values":     grouped.Values,
			"milestones": grouped.Milestones,
			"metrics":    grouped.Metrics,
		},
		"total": len(attributes),
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

// verifyTeamMembership checks if a user is a member and/or admin of a team
func (svc *Service) verifyTeamMembership(teamId string, userName string) (isMember bool, isAdmin bool, err error) {
	// Get team membership details
	memberDetails, err := svc.teamsSVC.GetTeamMemberDetails(teamId, userName)
	if err != nil {
		return false, false, fmt.Errorf("failed to get team member details: %v", err)
	}

	if memberDetails == nil {
		return false, false, nil // Not a member
	}

	isMember = true
	isAdmin = memberDetails.Role == companylib.TeamMemberRoleAdmin

	return isMember, isAdmin, nil
}

// getCognitoIdFromRequest extracts Cognito ID from the request
func (svc *Service) getCognitoIdFromRequest(request events.APIGatewayProxyRequest) (string, error) {
	// Extract from Cognito authorizer
	if request.RequestContext.Authorizer != nil {
		if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
			if sub, ok := claims["sub"].(string); ok {
				return sub, nil
			}
		}
	}

	return "", fmt.Errorf("cognito ID not found in request")
}

// errorResponse creates a standardized error response
func (svc *Service) errorResponse(statusCode int, message string, err error) (events.APIGatewayProxyResponse, error) {
	errMsg := message
	if err != nil {
		errMsg = fmt.Sprintf("%s: %v", message, err)
	}

	body, _ := json.Marshal(map[string]string{
		"error": errMsg,
	})

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    RESP_HEADERS,
		Body:       string(body),
	}, nil
}

// updateCustomAttribute updates an existing custom attribute for a team
func (svc *Service) updateCustomAttribute(teamId string, attributeId string, userName string, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Updating custom attribute %s for team: %s by user: %s", attributeId, teamId, userName)

	// Parse request body
	type UpdateAttributeRequest struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}

	var req UpdateAttributeRequest
	err := json.Unmarshal([]byte(request.Body), &req)
	if err != nil {
		svc.logger.Printf("Failed to parse request body: %v", err)
		return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
	}

	// Validate at least one field is provided
	if req.Name == nil && req.Description == nil {
		return svc.errorResponse(http.StatusBadRequest, "At least one field (name or description) must be provided for update", nil)
	}

	// Get existing attribute to verify it exists and belongs to this team
	existingAttr, err := svc.attributeSVC.GetAttributeById(attributeId, teamId)
	if err != nil {
		svc.logger.Printf("Failed to get attribute: %v", err)
		return svc.errorResponse(http.StatusNotFound, "Attribute not found", err)
	}

	// Update only the provided fields
	updatedAttr := *existingAttr
	if req.Name != nil {
		updatedAttr.Name = *req.Name
	}
	if req.Description != nil {
		updatedAttr.Description = *req.Description
	}

	// Update the attribute
	err = svc.attributeSVC.UpdateAttribute(attributeId, teamId, updatedAttr.Name, updatedAttr.Description)
	if err != nil {
		svc.logger.Printf("Failed to update attribute: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to update attribute", err)
	}

	// Return the updated attribute
	body, err := json.Marshal(map[string]interface{}{
		"message": "Attribute updated successfully",
		"attribute": map[string]string{
			"attributeId":   updatedAttr.AttributeId,
			"teamId":        updatedAttr.TeamId,
			"attributeType": string(updatedAttr.AttributeType),
			"name":          updatedAttr.Name,
			"description":   updatedAttr.Description,
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

// deleteCustomAttribute deletes a custom attribute for a team
func (svc *Service) deleteCustomAttribute(teamId string, attributeId string, userName string) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Deleting custom attribute %s for team: %s by user: %s", attributeId, teamId, userName)

	// Verify attribute exists and belongs to this team
	_, err := svc.attributeSVC.GetAttributeById(attributeId, teamId)
	if err != nil {
		svc.logger.Printf("Failed to get attribute: %v", err)
		return svc.errorResponse(http.StatusNotFound, "Attribute not found", err)
	}

	// Delete the attribute
	err = svc.attributeSVC.DeleteAttribute(attributeId, teamId)
	if err != nil {
		svc.logger.Printf("Failed to delete attribute: %v", err)
		return svc.errorResponse(http.StatusInternalServerError, "Failed to delete attribute", err)
	}

	// Return success response
	body, err := json.Marshal(map[string]interface{}{
		"message":     "Attribute deleted successfully",
		"attributeId": attributeId,
		"deletedBy":   userName,
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
