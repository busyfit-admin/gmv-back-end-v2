package common

// ==================== Routes ====================
//
// GET  /v2/users/me/meetings  — list meetings
// POST /v2/users/me/meetings  — create meeting

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

func (svc *Service) handleMeetings(request events.APIGatewayProxyRequest, parts []string, userName, _ string) (events.APIGatewayProxyResponse, error) {
	// /v2/users/me/meetings  (4 parts)
	if len(parts) == 4 {
		switch request.HTTPMethod {
		case "GET":
			return svc.listMeetings(userName, request.QueryStringParameters)
		case "POST":
			return svc.createMeeting(userName, request.Body)
		}
	}
	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== List Meetings ====================

func (svc *Service) listMeetings(userName string, queryParams map[string]string) (events.APIGatewayProxyResponse, error) {
	statusFilter := queryString(queryParams, "status")

	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: PrefixUser + userName},
			":prefix": &types.AttributeValueMemberS{Value: SKMeetingPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("listMeetings query error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list meetings")
	}

	var meetings []MeetingRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &meetings)

	if statusFilter != "" {
		filtered := make([]MeetingRecord, 0)
		for _, m := range meetings {
			if m.Status == statusFilter {
				filtered = append(filtered, m)
			}
		}
		meetings = filtered
	}

	sort.Slice(meetings, func(i, j int) bool {
		return meetings[i].Date < meetings[j].Date
	})

	meetingList := make([]map[string]interface{}, 0, len(meetings))
	for _, m := range meetings {
		meetingList = append(meetingList, buildMeetingResponse(m))
	}

	return svc.okResp(map[string]interface{}{"meetings": meetingList})
}

// ==================== Create Meeting ====================

func (svc *Service) createMeeting(userName, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[CreateMeetingRequest](body)
	if err != nil || req.Date == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "date is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	meetingID := uuid.New().String()

	rec := MeetingRecord{
		PK:          PrefixUser + userName,
		SK:          SKMeetingPrefix + meetingID,
		MeetingID:   meetingID,
		UserName:    userName,
		Date:        req.Date,
		Summary:     req.Summary,
		ManagerName: req.ManagerName,
		ManagerRole: req.ManagerRole,
		Tags:        req.Tags,
		ActionItems: req.ActionItems,
		Status:      string(MeetingStatusScheduled),
		CreatedAt:   now,
	}

	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to marshal meeting")
	}
	if _, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.perfHubTable),
		Item:      item,
	}); err != nil {
		svc.logger.Printf("createMeeting PutItem error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create meeting")
	}

	return svc.createdResp(map[string]interface{}{"meeting": buildMeetingResponse(rec)})
}

// ==================== Response Builder ====================

func buildMeetingResponse(m MeetingRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":          m.MeetingID,
		"date":        m.Date,
		"summary":     m.Summary,
		"managerName": m.ManagerName,
		"managerRole": m.ManagerRole,
		"tags":        m.Tags,
		"actionItems": m.ActionItems,
		"status":      m.Status,
		"createdAt":   m.CreatedAt,
	}
}
