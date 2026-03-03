package common

// ==================== Routes ====================
//
// GET  /v2/users/me/appreciations              — list received appreciations
// POST /v2/users/me/feedback-requests          — send feedback request
// GET  /v2/users/me/feedback-requests          — list sent feedback requests
// GET  /v2/teams/{teamId}/members/directory    — team member directory

import (
	"net/http"
	"sort"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// ==================== Appreciations ====================

func (svc *Service) handleAppreciations(request events.APIGatewayProxyRequest, parts []string, userName, teamID string) (events.APIGatewayProxyResponse, error) {
	if len(parts) == 4 && request.HTTPMethod == "GET" {
		return svc.listAppreciations(userName, teamID)
	}
	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

func (svc *Service) listAppreciations(userName, teamID string) (events.APIGatewayProxyResponse, error) {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKAppreciationPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("listAppreciations query error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list appreciations")
	}

	var apprList []AppreciationRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &apprList)

	sort.Slice(apprList, func(i, j int) bool {
		return apprList[i].CreatedAt > apprList[j].CreatedAt // newest first
	})

	out := make([]map[string]interface{}, 0, len(apprList))
	for _, a := range apprList {
		out = append(out, map[string]interface{}{
			"id":       a.AppreciationID,
			"from":     a.From,
			"initials": a.FromInitials,
			"fromRole": a.FromRole,
			"message":  a.Message,
			"skill":    a.Skill,
			"category": a.Category,
			"date":     a.Date,
		})
	}

	return svc.okResp(map[string]interface{}{"appreciations": out})
}

// ==================== Feedback Requests ====================

func (svc *Service) handleFeedbackRequests(request events.APIGatewayProxyRequest, parts []string, userName, displayName, teamID string) (events.APIGatewayProxyResponse, error) {
	if len(parts) == 4 {
		switch request.HTTPMethod {
		case "GET":
			return svc.listFeedbackRequests(userName, teamID, request.QueryStringParameters)
		case "POST":
			return svc.sendFeedbackRequest(userName, teamID, request.Body)
		}
	}
	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

func (svc *Service) listFeedbackRequests(userName, teamID string, queryParams map[string]string) (events.APIGatewayProxyResponse, error) {
	statusFilter := queryString(queryParams, "status")

	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKFeedbackReqPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("listFeedbackRequests query error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list feedback requests")
	}

	var requests []FeedbackRequestRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &requests)

	if statusFilter != "" {
		filtered := make([]FeedbackRequestRecord, 0)
		for _, r := range requests {
			if r.Status == statusFilter {
				filtered = append(filtered, r)
			}
		}
		requests = filtered
	}

	sort.Slice(requests, func(i, j int) bool {
		return requests[i].CreatedAt > requests[j].CreatedAt // newest first
	})

	out := make([]map[string]interface{}, 0, len(requests))
	for _, r := range requests {
		out = append(out, buildFeedbackRequestResponse(r))
	}

	return svc.okResp(map[string]interface{}{"feedbackRequests": out})
}

func (svc *Service) sendFeedbackRequest(userName, teamID, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[SendFeedbackRequestBody](body)
	if err != nil || req.ToUsername == "" || req.Message == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "toUsername and message are required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	today := time.Now().UTC().Format("2006-01-02")
	requestID := uuid.New().String()

	rec := FeedbackRequestRecord{
		PK:        buildPK(userName, teamID),
		SK:        SKFeedbackReqPrefix + requestID,
		RequestID: requestID,
		UserName:  userName,
		To:        req.ToUsername,
		Message:   req.Message,
		Status:    "pending",
		Date:      today,
		CreatedAt: now,
	}

	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to marshal feedback request")
	}
	if _, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.perfHubTable),
		Item:      item,
	}); err != nil {
		svc.logger.Printf("sendFeedbackRequest PutItem error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to send feedback request")
	}

	return svc.createdResp(map[string]interface{}{"feedbackRequest": buildFeedbackRequestResponse(rec)})
}

func buildFeedbackRequestResponse(r FeedbackRequestRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":        r.RequestID,
		"to":        r.To,
		"from":      r.UserName,
		"message":   r.Message,
		"status":    r.Status,
		"date":      r.Date,
		"createdAt": r.CreatedAt,
	}
}

// ==================== Team Member Directory ====================

func (svc *Service) getTeamMemberDirectory(teamID, userName string) (events.APIGatewayProxyResponse, error) {
	// Verify caller is a member of the team
	_, memberErr := svc.teamsSVC.GetTeamMemberDetails(teamID, userName)
	if memberErr != nil {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
	}

	members, err := svc.teamsSVC.GetTeamMembers(teamID)
	if err != nil {
		svc.logger.Printf("getTeamMemberDirectory GetTeamMembers error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch team members")
	}

	out := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		out = append(out, map[string]interface{}{
			"userName":    m.UserName,
			"displayName": m.DisplayName,
			"initials":    initials(m.DisplayName),
			"role":        m.Role,
		})
	}

	return svc.okResp(map[string]interface{}{
		"teamId":  teamID,
		"members": out,
	})
}
