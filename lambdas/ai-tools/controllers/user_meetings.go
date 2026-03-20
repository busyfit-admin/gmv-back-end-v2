package controllers

import (
	"context"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ==================== User — Meetings ====================

// GetMyMeetings returns all 1-on-1 meetings for (userName, teamID).
// statusFilter: "scheduled" | "completed" | "" (all).
// Results are sorted by Date ascending (chronological order).
func (s *Service) GetMyMeetings(ctx context.Context, userName, teamID, statusFilter string) ([]MeetingRecord, error) {
	s.logger.Printf("ctrl: GetMyMeetings input: userName=%q teamID=%q statusFilter=%q", userName, teamID, statusFilter)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skMeetingPrefix},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMyMeetings output: err=%v", err)
		return nil, err
	}

	var meetings []MeetingRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &meetings)

	if statusFilter != "" {
		filtered := meetings[:0]
		for _, m := range meetings {
			if strings.EqualFold(m.Status, statusFilter) {
				filtered = append(filtered, m)
			}
		}
		meetings = filtered
	}

	sort.Slice(meetings, func(i, j int) bool { return meetings[i].Date < meetings[j].Date })
	s.logger.Printf("ctrl: GetMyMeetings output: count=%d", len(meetings))
	return meetings, nil
}

// GetMeeting fetches a single meeting by meetingID for (userName, teamID).
// Returns nil, nil when the meeting does not exist.
func (s *Service) GetMeeting(ctx context.Context, userName, teamID, meetingID string) (*MeetingRecord, error) {
	s.logger.Printf("ctrl: GetMeeting input: userName=%q teamID=%q meetingID=%q", userName, teamID, meetingID)
	result, err := s.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			"SK": &types.AttributeValueMemberS{Value: skMeetingPrefix + meetingID},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMeeting output: err=%v", err)
		return nil, err
	}
	if result.Item == nil {
		s.logger.Printf("ctrl: GetMeeting output: not found")
		return nil, nil
	}
	var rec MeetingRecord
	if err := attributevalue.UnmarshalMap(result.Item, &rec); err != nil {
		s.logger.Printf("ctrl: GetMeeting output: unmarshal err=%v", err)
		return nil, err
	}
	s.logger.Printf("ctrl: GetMeeting output: found meetingId=%q", rec.MeetingID)
	return &rec, nil
}
