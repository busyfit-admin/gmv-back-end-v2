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

// ==================== User — Goals ====================

// GetMyGoals returns all goals belonging to a user in a specific team.
// Use GoalFilters to narrow results by type or status; zero-value means no filter.
func (s *Service) GetMyGoals(ctx context.Context, userName, teamID string, filters GoalFilters) ([]GoalRecord, error) {
	s.logger.Printf("ctrl: GetMyGoals input: userName=%q teamID=%q type=%q status=%q", userName, teamID, filters.Type, filters.Status)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skGoalPrefix},
		},
	})
	if err != nil {
		return nil, err
	}

	var goals []GoalRecord
	for _, item := range result.Items {
		skAttr, ok := item["SK"].(*types.AttributeValueMemberS)
		if !ok {
			continue
		}
		// Skip comment sub-rows stored under GOAL# prefix
		if strings.Contains(skAttr.Value, skCommentInfix) {
			continue
		}
		var rec GoalRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
			continue
		}
		if filters.Type != "" && !strings.EqualFold(rec.Type, filters.Type) {
			continue
		}
		if filters.Status != "" && !strings.EqualFold(rec.Status, filters.Status) {
			continue
		}
		goals = append(goals, rec)
	}

	sort.Slice(goals, func(i, j int) bool { return goals[i].CreatedAt < goals[j].CreatedAt })
	s.logger.Printf("ctrl: GetMyGoals output: count=%d", len(goals))
	return goals, nil
}

// GetMyGoal fetches a single goal by goalID for (userName, teamID).
// Returns nil, nil when the goal does not exist.
func (s *Service) GetMyGoal(ctx context.Context, userName, teamID, goalID string) (*GoalRecord, error) {
	s.logger.Printf("ctrl: GetMyGoal input: userName=%q teamID=%q goalID=%q", userName, teamID, goalID)
	result, err := s.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			"SK": &types.AttributeValueMemberS{Value: skGoalPrefix + goalID},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMyGoal output: err=%v", err)
		return nil, err
	}
	if result.Item == nil {
		s.logger.Printf("ctrl: GetMyGoal output: not found")
		return nil, nil
	}
	var rec GoalRecord
	if err := attributevalue.UnmarshalMap(result.Item, &rec); err != nil {
		s.logger.Printf("ctrl: GetMyGoal output: unmarshal err=%v", err)
		return nil, err
	}
	s.logger.Printf("ctrl: GetMyGoal output: found goalId=%q", rec.GoalID)
	return &rec, nil
}

// GetGoalLinkedTasks returns all tasks that are linked to a specific goal.
func (s *Service) GetGoalLinkedTasks(ctx context.Context, userName, teamID, goalID string) ([]LinkedTaskRecord, error) {
	s.logger.Printf("ctrl: GetGoalLinkedTasks input: userName=%q teamID=%q goalID=%q", userName, teamID, goalID)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		FilterExpression:       aws.String("goalId = :goalId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skTaskPrefix},
			":goalId": &types.AttributeValueMemberS{Value: goalID},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetGoalLinkedTasks output: err=%v", err)
		return nil, err
	}
	var tasks []LinkedTaskRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &tasks)
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt < tasks[j].CreatedAt })
	s.logger.Printf("ctrl: GetGoalLinkedTasks output: count=%d", len(tasks))
	return tasks, nil
}

// GetGoalComments returns all comments on a specific goal, oldest first.
func (s *Service) GetGoalComments(ctx context.Context, userName, teamID, goalID string) ([]GoalCommentRecord, error) {
	s.logger.Printf("ctrl: GetGoalComments input: userName=%q teamID=%q goalID=%q", userName, teamID, goalID)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skGoalPrefix + goalID + skCommentInfix},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetGoalComments output: err=%v", err)
		return nil, err
	}
	var comments []GoalCommentRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &comments)
	sort.Slice(comments, func(i, j int) bool { return comments[i].CreatedAt < comments[j].CreatedAt })
	s.logger.Printf("ctrl: GetGoalComments output: count=%d", len(comments))
	return comments, nil
}
