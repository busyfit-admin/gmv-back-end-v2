package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

// ==================== Extended Org-Goal Read Functions ====================
//
// These functions expose PerformanceService methods not yet covered by
// admin_performance.go, plus two direct-DDB reads that have no service wrapper.

// GetGoalLadderUp returns the ladder-up request history for an org goal.
// statusFilter may be "", "pending", "approved", or "rejected".
func (s *Service) GetGoalLadderUp(goalID, statusFilter string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetGoalLadderUp input: goalID=%q statusFilter=%q", goalID, statusFilter)
	result, err := s.perfSVC.GetGoalLadderUp(goalID, statusFilter)
	s.logger.Printf("ctrl: GetGoalLadderUp output: err=%v", err)
	return result, err
}

// GetGoalValueHistory returns value-update history for an org goal.
// startDate / endDate are ISO-8601 UTC strings; both are optional ("" = no bound).
func (s *Service) GetGoalValueHistory(goalID, startDate, endDate string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetGoalValueHistory input: goalID=%q startDate=%q endDate=%q", goalID, startDate, endDate)
	filters := map[string]string{}
	if startDate != "" {
		filters["startDate"] = startDate
	}
	if endDate != "" {
		filters["endDate"] = endDate
	}
	result, err := s.perfSVC.GetGoalValueHistory(goalID, filters, companylib.ListQueryOptions{})
	s.logger.Printf("ctrl: GetGoalValueHistory output: err=%v", err)
	return result, err
}

// GetOrgGoalTasks returns tasks linked to an org goal.
// userName limits results to a single user's tasks; leave empty for all.
// statusFilter may be "", "todo", "in-progress", or "done".
func (s *Service) GetOrgGoalTasks(goalID, userName, statusFilter string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetOrgGoalTasks input: goalID=%q userName=%q statusFilter=%q", goalID, userName, statusFilter)
	filters := map[string]string{}
	if statusFilter != "" {
		filters["status"] = statusFilter
	}
	result, err := s.perfSVC.GetGoalTasks(goalID, userName, filters, companylib.ListQueryOptions{})
	s.logger.Printf("ctrl: GetOrgGoalTasks output: err=%v", err)
	return result, err
}

// GetGoalTaggedTeams returns the list of teams tagged to an org goal.
func (s *Service) GetGoalTaggedTeams(goalID string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetGoalTaggedTeams input: goalID=%q", goalID)
	result, err := s.perfSVC.GetGoalTeams(goalID)
	s.logger.Printf("ctrl: GetGoalTaggedTeams output: err=%v", err)
	return result, err
}

// ==================== Direct DDB — UserPerformanceHubTable ====================

// userGoalProjection is a minimal unmarshal target for GSI results.
type userGoalProjection struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	GoalID    string `dynamodbav:"goalId"`
	OrgGoalID string `dynamodbav:"orgGoalId"`
	UserName  string `dynamodbav:"userName"`
	Title     string `dynamodbav:"title"`
	Type      string `dynamodbav:"type"`
	Progress  int    `dynamodbav:"progress"`
	Status    string `dynamodbav:"status"`
	DueDate   string `dynamodbav:"dueDate"`
	UpdatedAt string `dynamodbav:"updatedAt"`
}

// linkedTaskProjection is a minimal unmarshal target for task items.
type linkedTaskProjection struct {
	SK          string  `dynamodbav:"SK"`
	TaskID      string  `dynamodbav:"taskId"`
	TaskNumber  int     `dynamodbav:"taskNumber"`
	GoalID      string  `dynamodbav:"goalId"`
	Title       string  `dynamodbav:"title"`
	Description string  `dynamodbav:"description"`
	Priority    string  `dynamodbav:"priority"`
	Status      string  `dynamodbav:"status"`
	Done        bool    `dynamodbav:"done"`
	DueDate     string  `dynamodbav:"dueDate"`
	TimeHours   float64 `dynamodbav:"timeHours"`
	TimeDays    float64 `dynamodbav:"timeDays"`
	UpdatedAt   string  `dynamodbav:"updatedAt"`
}

// fetchLinkedTasks queries TASK# items on PK whose goalId matches the given goalID.
func (s *Service) fetchLinkedTasks(ctx context.Context, pk, goalID string) ([]map[string]interface{}, error) {
	if s.perfHubTable == "" {
		return nil, fmt.Errorf("PERF_HUB_TABLE is not configured")
	}
	out, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		FilterExpression:       aws.String("goalId = :goalId"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":pk":     &ddbTypes.AttributeValueMemberS{Value: pk},
			":prefix": &ddbTypes.AttributeValueMemberS{Value: "TASK#"},
			":goalId": &ddbTypes.AttributeValueMemberS{Value: goalID},
		},
	})
	if err != nil {
		return nil, err
	}
	tasks := make([]map[string]interface{}, 0, len(out.Items))
	for _, raw := range out.Items {
		var t linkedTaskProjection
		if err := attributevalue.UnmarshalMap(raw, &t); err != nil {
			continue
		}
		entry := map[string]interface{}{
			"taskId":   t.TaskID,
			"title":    t.Title,
			"status":   t.Status,
			"done":     t.Done,
			"priority": t.Priority,
			"dueDate":  t.DueDate,
		}
		if t.TaskNumber > 0 {
			entry["taskNumber"] = t.TaskNumber
		}
		if t.Description != "" {
			entry["description"] = t.Description
		}
		if t.UpdatedAt != "" {
			entry["updatedAt"] = t.UpdatedAt
		}
		tasks = append(tasks, entry)
	}
	return tasks, nil
}

// GetUserGoalsForOrgGoal queries OrgGoalIdIndex on UserPerformanceHubTable to
// return all user-level goals linked to the given org goal, together with each
// goal's linked tasks and a rolled-up status summary.
// statusFilter is optional; leave empty to return all statuses.
func (s *Service) GetUserGoalsForOrgGoal(ctx context.Context, orgGoalID, statusFilter string) (map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetUserGoalsForOrgGoal input: orgGoalID=%q statusFilter=%q", orgGoalID, statusFilter)
	if s.perfHubTable == "" {
		s.logger.Printf("ctrl: GetUserGoalsForOrgGoal output: err=PERF_HUB_TABLE not configured")
		return nil, fmt.Errorf("PERF_HUB_TABLE is not configured")
	}
	out, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		IndexName:              aws.String("OrgGoalIdIndex"),
		KeyConditionExpression: aws.String("orgGoalId = :orgGoalId"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":orgGoalId": &ddbTypes.AttributeValueMemberS{Value: orgGoalID},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetUserGoalsForOrgGoal output: err=%v", err)
		return nil, fmt.Errorf("OrgGoalIdIndex query failed: %w", err)
	}

	summary := map[string]int{
		"total": 0, "onTrack": 0, "ahead": 0,
		"atRisk": 0, "behind": 0, "completed": 0,
	}
	goals := make([]map[string]interface{}, 0)

	for _, raw := range out.Items {
		var g userGoalProjection
		if err := attributevalue.UnmarshalMap(raw, &g); err != nil {
			continue
		}
		// Only process goal records (SK starts with "GOAL#" and is not a comment).
		if !strings.HasPrefix(g.SK, "GOAL#") || strings.Contains(g.SK, "#CMMNT#") {
			continue
		}
		if statusFilter != "" && !strings.EqualFold(g.Status, statusFilter) {
			continue
		}
		// Parse teamId from PK: USER#{userName}#TEAM#{teamId}
		teamID := ""
		if idx := strings.LastIndex(g.PK, "#TEAM#"); idx != -1 {
			teamID = g.PK[idx+6:]
		}
		tasks, err := s.fetchLinkedTasks(ctx, g.PK, g.GoalID)
		if err != nil {
			s.logger.Printf("[GetUserGoalsForOrgGoal] warn: tasks for goal %q: %v", g.GoalID, err)
			tasks = []map[string]interface{}{}
		}
		summary["total"]++
		switch strings.ToLower(g.Status) {
		case "on-track":
			summary["onTrack"]++
		case "ahead":
			summary["ahead"]++
		case "at-risk":
			summary["atRisk"]++
		case "behind":
			summary["behind"]++
		case "completed":
			summary["completed"]++
		}
		goals = append(goals, map[string]interface{}{
			"goalId":      g.GoalID,
			"userName":    g.UserName,
			"teamId":      teamID,
			"title":       g.Title,
			"type":        g.Type,
			"progress":    g.Progress,
			"status":      g.Status,
			"dueDate":     g.DueDate,
			"updatedAt":   g.UpdatedAt,
			"linkedTasks": tasks,
		})
	}

	s.logger.Printf("ctrl: GetUserGoalsForOrgGoal output: goalCount=%d", len(goals))
	return map[string]interface{}{
		"orgGoalId": orgGoalID,
		"userGoals": goals,
		"summary":   summary,
	}, nil
}

// GetTeamMemberDirectory returns a simplified member list for a team, useful
// for building quick-reference directories. Each entry contains userName,
// displayName, initials (first letter of each word in displayName), and role.
func (s *Service) GetTeamMemberDirectory(teamID string) ([]map[string]interface{}, error) {
	s.logger.Printf("ctrl: GetTeamMemberDirectory input: teamID=%q", teamID)
	members, err := s.teamsSVC.GetTeamMembers(teamID)
	if err != nil {
		s.logger.Printf("ctrl: GetTeamMemberDirectory output: err=%v", err)
		return nil, err
	}
	dir := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		initials := initials(m.DisplayName)
		dir = append(dir, map[string]interface{}{
			"userName":    m.UserName,
			"displayName": m.DisplayName,
			"initials":    initials,
			"role":        string(m.Role),
			"joinedAt":    m.JoinedAt,
			"isActive":    m.IsActive,
		})
	}
	s.logger.Printf("ctrl: GetTeamMemberDirectory output: count=%d", len(dir))
	return dir, nil
}

// initials derives up-to-two-letter initials from a display name.
func initials(name string) string {
	words := strings.Fields(name)
	result := ""
	for i, w := range words {
		if i >= 2 {
			break
		}
		if len(w) > 0 {
			result += strings.ToUpper(string([]rune(w)[0]))
		}
	}
	return result
}
