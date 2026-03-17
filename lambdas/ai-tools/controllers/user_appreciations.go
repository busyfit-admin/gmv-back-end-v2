package controllers

import (
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
func (s *Service) GetMyAppreciations(userName, teamID string) ([]AppreciationRecord, error) {
	result, err := s.ddb.Query(s.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skAppreciationPrefix},
		},
	})
	if err != nil {
		return nil, err
	}

	var records []AppreciationRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)
	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt > records[j].CreatedAt })
	return records, nil
}

// GetMyFeedbackRequests returns all feedback requests sent by (userName, teamID).
// statusFilter: "pending" | "completed" | "" (all).
func (s *Service) GetMyFeedbackRequests(userName, teamID, statusFilter string) ([]FeedbackRequestRecord, error) {
	result, err := s.ddb.Query(s.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skFeedbackReqPrefix},
		},
	})
	if err != nil {
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
	return records, nil
}
