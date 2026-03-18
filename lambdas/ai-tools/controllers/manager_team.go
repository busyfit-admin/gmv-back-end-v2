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

// ==================== Manager — Team Performance Functions ====================

// GetTeamPerformanceMembers returns all members of a team enriched with their
// review lifecycle state (overall rating, pending review flag, etc.).
func (s *Service) GetTeamPerformanceMembers(ctx context.Context, teamID string) ([]TeamMemberWithReview, error) {
	s.logger.Printf("ctrl: GetTeamPerformanceMembers input: teamID=%q", teamID)
	members, err := s.teamsSVC.GetTeamMembers(teamID)
	if err != nil {
		s.logger.Printf("ctrl: GetTeamPerformanceMembers output: err=%v", err)
		return nil, err
	}

	reviewMap, _ := s.fetchTeamReviewMap(ctx, teamID) // non-fatal if missing

	out := make([]TeamMemberWithReview, 0, len(members))
	for _, m := range members {
		row := TeamMemberWithReview{
			UserName:    m.UserName,
			DisplayName: m.DisplayName,
			Role:        string(m.Role),
		}
		if rev, ok := reviewMap[m.UserName]; ok {
			row.OverallRating = rev.OverallRating
			row.LastReviewDate = rev.LastReviewDate
			row.IsPendingReview = rev.IsPendingReview
			row.HasUserUpdatedReviews = rev.HasUserUpdatedReviews
		}
		out = append(out, row)
	}
	s.logger.Printf("ctrl: GetTeamPerformanceMembers output: count=%d", len(out))
	return out, nil
}

// GetMemberGoals returns all goals for a team member split into OKRs and KPIs.
func (s *Service) GetMemberGoals(ctx context.Context, teamID, memberID string) (MemberGoalsResult, error) {
	s.logger.Printf("ctrl: GetMemberGoals input: teamID=%q memberID=%q", teamID, memberID)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skGoalPrefix},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMemberGoals output: err=%v", err)
		return MemberGoalsResult{OKRs: []GoalRecord{}, KPIs: []GoalRecord{}}, err
	}

	var okrs, kpis []GoalRecord
	for _, item := range result.Items {
		skAttr, ok := item["SK"].(*types.AttributeValueMemberS)
		if !ok || strings.Contains(skAttr.Value, skCommentInfix) {
			continue
		}
		var rec GoalRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
			continue
		}
		switch rec.Type {
		case GoalTypeOKR:
			okrs = append(okrs, rec)
		case GoalTypeKPI:
			kpis = append(kpis, rec)
		}
	}

	if okrs == nil {
		okrs = []GoalRecord{}
	}
	if kpis == nil {
		kpis = []GoalRecord{}
	}
	s.logger.Printf("ctrl: GetMemberGoals output: okrs=%d kpis=%d", len(okrs), len(kpis))
	return MemberGoalsResult{OKRs: okrs, KPIs: kpis}, nil
}

// GetMemberTasks returns all tasks for a team member, optionally filtered.
func (s *Service) GetMemberTasks(ctx context.Context, teamID, memberID string, filters TaskFilters) ([]LinkedTaskRecord, error) {
	s.logger.Printf("ctrl: GetMemberTasks input: teamID=%q memberID=%q status=%q", teamID, memberID, filters.Status)
	result, err := s.GetAllTasks(ctx, memberID, teamID, filters)
	s.logger.Printf("ctrl: GetMemberTasks output: count=%d err=%v", len(result), err)
	return result, err
}

// GetMemberMeetings returns all 1-on-1 meetings for a team member, newest first.
func (s *Service) GetMemberMeetings(ctx context.Context, teamID, memberID string) ([]MeetingRecord, error) {
	s.logger.Printf("ctrl: GetMemberMeetings input: teamID=%q memberID=%q", teamID, memberID)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skMeetingPrefix},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMemberMeetings output: err=%v", err)
		return nil, err
	}
	var meetings []MeetingRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &meetings)
	sort.Slice(meetings, func(i, j int) bool { return meetings[i].Date > meetings[j].Date })
	s.logger.Printf("ctrl: GetMemberMeetings output: count=%d", len(meetings))
	return meetings, nil
}

// GetMemberAppreciations returns all appreciations received by a team member, newest first.
func (s *Service) GetMemberAppreciations(ctx context.Context, teamID, memberID string) ([]AppreciationRecord, error) {
	s.logger.Printf("ctrl: GetMemberAppreciations input: teamID=%q memberID=%q", teamID, memberID)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skAppreciationPrefix},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMemberAppreciations output: err=%v", err)
		return nil, err
	}
	var records []AppreciationRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)
	sort.Slice(records, func(i, j int) bool { return records[i].Date > records[j].Date })
	s.logger.Printf("ctrl: GetMemberAppreciations output: count=%d", len(records))
	return records, nil
}

// GetMemberManagerComments returns all manager-authored comments for a team member, newest first.
func (s *Service) GetMemberManagerComments(ctx context.Context, teamID, memberID string) ([]ManagerCommentRecord, error) {
	s.logger.Printf("ctrl: GetMemberManagerComments input: teamID=%q memberID=%q", teamID, memberID)
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skManagerCommentPrefix},
		},
	})
	if err != nil {
		s.logger.Printf("ctrl: GetMemberManagerComments output: err=%v", err)
		return nil, err
	}
	var records []ManagerCommentRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)
	sort.Slice(records, func(i, j int) bool { return records[i].CreatedAt > records[j].CreatedAt })
	s.logger.Printf("ctrl: GetMemberManagerComments output: count=%d", len(records))
	return records, nil
}

// GetMemberPerformanceSummary returns a full performance snapshot for a team member:
// profile info, OKRs, KPIs, meetings, appreciations, and manager comments.
func (s *Service) GetMemberPerformanceSummary(ctx context.Context, teamID, memberID string) (MemberPerformanceSummary, error) {
	s.logger.Printf("ctrl: GetMemberPerformanceSummary input: teamID=%q memberID=%q", teamID, memberID)
	summary := MemberPerformanceSummary{MemberUserName: memberID}

	// Enrich profile from team member record
	if memberInfo, err := s.teamsSVC.GetTeamMemberDetails(teamID, memberID); err == nil && memberInfo != nil {
		summary.DisplayName = memberInfo.DisplayName
		summary.Role = string(memberInfo.Role)
	}

	// Review record
	if rev, err := s.fetchMemberReviewRecord(ctx, teamID, memberID); err == nil && rev != nil {
		summary.OverallRating = rev.OverallRating
		summary.LastReviewDate = rev.LastReviewDate
		summary.IsPendingReview = rev.IsPendingReview
	}

	// Goals
	goals, _ := s.GetMemberGoals(ctx, teamID, memberID)
	summary.Goals = goals

	// Meetings (newest first)
	meetings, _ := s.GetMemberMeetings(ctx, teamID, memberID)
	summary.Meetings = meetings

	// Appreciations
	appreciations, _ := s.GetMemberAppreciations(ctx, teamID, memberID)
	summary.Appreciations = appreciations

	// Manager comments
	comments, _ := s.GetMemberManagerComments(ctx, teamID, memberID)
	summary.Comments = comments

	s.logger.Printf("ctrl: GetMemberPerformanceSummary output: memberID=%q", memberID)
	return summary, nil
}

// ==================== Internal DDB helpers ====================

// fetchTeamReviewMap loads all TeamMemberReviewRecord rows for a team keyed by memberUserName.
func (s *Service) fetchTeamReviewMap(ctx context.Context, teamID string) (map[string]TeamMemberReviewRecord, error) {
	result, err := s.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildTeamPK(teamID)},
			":prefix": &types.AttributeValueMemberS{Value: skMemberReviewPrefix},
		},
	})
	if err != nil {
		return map[string]TeamMemberReviewRecord{}, err
	}

	reviewMap := make(map[string]TeamMemberReviewRecord, len(result.Items))
	for _, item := range result.Items {
		var rec TeamMemberReviewRecord
		if attributevalue.UnmarshalMap(item, &rec) == nil {
			reviewMap[rec.MemberUserName] = rec
		}
	}
	return reviewMap, nil
}

// fetchMemberReviewRecord loads the single review record for one member in a team.
func (s *Service) fetchMemberReviewRecord(ctx context.Context, teamID, memberUserName string) (*TeamMemberReviewRecord, error) {
	result, err := s.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: buildTeamPK(teamID)},
			"SK": &types.AttributeValueMemberS{Value: skMemberReviewPrefix + memberUserName},
		},
	})
	if err != nil || result.Item == nil {
		return nil, err
	}
	var rec TeamMemberReviewRecord
	attributevalue.UnmarshalMap(result.Item, &rec)
	return &rec, nil
}
