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

// ==================== User — Tasks ====================

// GetAllTasks returns all tasks for (userName, teamID).
// Use TaskFilters to narrow results. All filter fields are optional.
//
// TaskFilters.GoalID:
//   - "" — return all tasks regardless of goal linkage
//   - "none" — return only tasks with no linked goal
//   - "<uuid>" — return only tasks linked to that goal
//
// TaskFilters.Done:
//   - nil — no filter
//   - pointer to true/false — filter by completion flag
func (s *Service) GetAllTasks(ctx context.Context, userName, teamID string, filters TaskFilters) ([]LinkedTaskRecord, error) {
	s.logger.Printf("ctrl: GetAllTasks input: userName=%q teamID=%q goalID=%q status=%q", userName, teamID, filters.GoalID, filters.Status)
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skTaskPrefix},
		},
	}

	switch filters.GoalID {
	case "":
		// no goal filter
	case "none":
		input.FilterExpression = aws.String("attribute_not_exists(goalId) OR goalId = :empty")
		input.ExpressionAttributeValues[":empty"] = &types.AttributeValueMemberS{Value: ""}
	default:
		input.FilterExpression = aws.String("goalId = :goalId")
		input.ExpressionAttributeValues[":goalId"] = &types.AttributeValueMemberS{Value: filters.GoalID}
	}

	result, err := s.ddb.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	var tasks []LinkedTaskRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &tasks)

	// Done filter (applied in Go to avoid reserved-word issues with bool in FilterExpression)
	if filters.Done != nil {
		want := *filters.Done
		filtered := tasks[:0]
		for _, t := range tasks {
			if t.Done == want {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	// Status filter
	if filters.Status != "" {
		filtered := tasks[:0]
		for _, t := range tasks {
			if strings.EqualFold(t.Status, filters.Status) {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt < tasks[j].CreatedAt })
	s.logger.Printf("ctrl: GetAllTasks output: count=%d", len(tasks))
	return tasks, nil
}

// GetTask fetches a single task by its taskId UUID for (userName, teamID).
// Returns nil, nil when the task does not exist.
func (s *Service) GetTask(ctx context.Context, userName, teamID, taskID string) (*LinkedTaskRecord, error) {
	s.logger.Printf("ctrl: GetTask input: userName=%q teamID=%q taskID=%q", userName, teamID, taskID)
	result, err := s.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			"SK": &types.AttributeValueMemberS{Value: skTaskPrefix + taskID},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetTask output: err=%v", err)
		return nil, err
	}
	if result.Item == nil {
		s.logger.Printf("ctrl: GetTask output: not found")
		return nil, nil
	}
	var rec LinkedTaskRecord
	if err := attributevalue.UnmarshalMap(result.Item, &rec); err != nil {
		s.logger.Printf("ctrl: GetTask output: unmarshal err=%v", err)
		return nil, err
	}
	s.logger.Printf("ctrl: GetTask output: found taskId=%q", rec.TaskID)
	return &rec, nil
}
