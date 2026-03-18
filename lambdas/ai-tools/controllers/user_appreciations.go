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

// ==================== User — Appreciations & Feedback ====================

// GetMyAppreciations returns all appreciation records received by (userName, teamID),
// sorted newest first.
func (s *Service) GetMyAppreciations(ctx context.Context, userName, teamID string) ([]AppreciationRecord, error) {
	s.logger.Printf("ctrl: GetMyAppreciations input: userName=%q teamID=%q", userName, teamID)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skAppreciationPrefix},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMyAppreciations output: err=%v", err)
		return nil, err
	}

	var records []AppreciationRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)
	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt > records[j].CreatedAt })
	s.logger.Printf("ctrl: GetMyAppreciations output: count=%d", len(records))
	return records, nil
}

// GetMyFeedbackRequests returns all feedback requests sent by (userName, teamID).
// statusFilter: "pending" | "completed" | "" (all).
func (s *Service) GetMyFeedbackRequests(ctx context.Context, userName, teamID, statusFilter string) ([]FeedbackRequestRecord, error) {
	s.logger.Printf("ctrl: GetMyFeedbackRequests input: userName=%q teamID=%q statusFilter=%q", userName, teamID, statusFilter)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skFeedbackReqPrefix},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMyFeedbackRequests output: err=%v", err)
		return nil, err
	}

	var records []FeedbackRequestRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)

	if statusFilter != "" {
		filtered := records[:0]
		for _, r := range records {
			if strings.EqualFold(r.Status, statusFilter) {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt > records[j].CreatedAt })
	s.logger.Printf("ctrl: GetMyFeedbackRequests output: count=%d", len(records))
	return records, nil
}
