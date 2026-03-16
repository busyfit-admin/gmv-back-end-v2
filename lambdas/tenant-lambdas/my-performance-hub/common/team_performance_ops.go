package common

// ==================== Routes ====================
//
// GET  /v2/teams/{teamId}/performance/members                          — list team members with review status
// GET  /v2/teams/{teamId}/members/{memberId}/goals                     — member OKRs & KPIs (manager view)
// GET  /v2/teams/{teamId}/members/{memberId}/meetings                  — member 1-on-1 meetings (manager view)
// GET  /v2/teams/{teamId}/members/{memberId}/appreciations             — member appreciations (manager view)
// GET  /v2/teams/{teamId}/members/{memberId}/comments                  — manager comments & feedback
// POST /v2/teams/{teamId}/members/{memberId}/comments                  — add manager comment
// GET  /v2/teams/{teamId}/members/{memberId}/performance-summary       — all-in-one member detail

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// handleTeamPerformance dispatches all /v2/teams/{teamId}/... performance routes.
//
// Accepted path shapes (parts[0] == "v2", parts[1] == "teams"):
//
//	[v2, teams, {teamId}, performance, members]                              — GET list members
//	[v2, teams, {teamId}, members, {memberId}, goals]                        — GET member goals (manager view)
//	[v2, teams, {teamId}, members, {memberId}, goals, {goalId}, comments]    — POST manager comment on a goal
//	[v2, teams, {teamId}, members, {memberId}, meetings]
//	[v2, teams, {teamId}, members, {memberId}, appreciations]
//	[v2, teams, {teamId}, members, {memberId}, comments]
//	[v2, teams, {teamId}, members, {memberId}, performance-summary]
func (svc *Service) handleTeamPerformance(request events.APIGatewayProxyRequest, parts []string, managerUserName, managerDisplayName string) (events.APIGatewayProxyResponse, error) {
	teamID := parts[2]

	// /v2/teams/{teamId}/performance/members  (5 parts)
	if len(parts) == 5 && parts[3] == "performance" && parts[4] == "members" {
		if request.HTTPMethod == "GET" {
			return svc.getTeamPerformanceMembers(teamID, managerUserName)
		}
		return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// /v2/teams/{teamId}/members/{memberId}/goals/{goalId}/comments  (8 parts)
	if len(parts) == 8 && parts[3] == "members" && parts[5] == "goals" && parts[7] == "comments" {
		memberID := parts[4]
		goalID := parts[6]
		if request.HTTPMethod == "POST" {
			return svc.addManagerGoalComment(teamID, memberID, goalID, managerUserName, managerDisplayName, request.Body)
		}
		return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	// /v2/teams/{teamId}/members/{memberId}/{resource}  (6 parts)
	if len(parts) == 6 && parts[3] == "members" {
		memberID := parts[4]
		resource := parts[5]

		switch resource {
		case "goals":
			if request.HTTPMethod == "GET" {
				return svc.getMemberGoalsForManager(teamID, memberID, managerUserName)
			}
		case "meetings":
			if request.HTTPMethod == "GET" {
				return svc.getMemberMeetingsForManager(teamID, memberID, managerUserName)
			}
		case "appreciations":
			if request.HTTPMethod == "GET" {
				return svc.getMemberAppreciationsForManager(teamID, memberID, managerUserName)
			}
		case "comments":
			switch request.HTTPMethod {
			case "GET":
				return svc.getMemberComments(teamID, memberID, managerUserName)
			case "POST":
				return svc.addManagerComment(teamID, memberID, managerUserName, managerDisplayName, request.Body)
			}
		case "performance-summary":
			if request.HTTPMethod == "GET" {
				return svc.getMemberPerformanceSummary(teamID, memberID, managerUserName)
			}
		}
		return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
	}

	return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Route not found")
}

// ==================== 1. Team Members List ====================
//
// GET /v2/teams/{teamId}/performance/members
// Returns all team members enriched with their review lifecycle state.

func (svc *Service) getTeamPerformanceMembers(teamID, managerUserName string) (events.APIGatewayProxyResponse, error) {
	// Verify caller is a member (or manager) of the team
	_, err := svc.teamsSVC.GetTeamMemberDetails(teamID, managerUserName)
	if err != nil {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
	}

	members, err := svc.teamsSVC.GetTeamMembers(teamID)
	if err != nil {
		svc.logger.Printf("getTeamPerformanceMembers GetTeamMembers error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch team members")
	}

	// Load all review status records for this team in a single scan
	reviewMap, err := svc.fetchTeamReviewMap(teamID)
	if err != nil {
		svc.logger.Printf("getTeamPerformanceMembers fetchTeamReviewMap error: %v", err)
		// Non-fatal — continue with empty review data
		reviewMap = map[string]TeamMemberReviewRecord{}
	}

	out := make([]map[string]interface{}, 0, len(members))
	for _, m := range members {
		review, hasReview := reviewMap[m.UserName]
		memberMap := map[string]interface{}{
			"id":          m.UserName,
			"name":        m.DisplayName,
			"initials":    initials(m.DisplayName),
			"role":        string(m.Role),
			"department":  "",
			"avatarColor": "",
		}
		if hasReview {
			memberMap["overallRating"] = review.OverallRating
			memberMap["lastReviewDate"] = review.LastReviewDate
			memberMap["isPendingReview"] = review.IsPendingReview
			memberMap["hasUserUpdatedReviews"] = review.HasUserUpdatedReviews
		} else {
			memberMap["overallRating"] = 0.0
			memberMap["lastReviewDate"] = nil
			memberMap["isPendingReview"] = false
			memberMap["hasUserUpdatedReviews"] = false
		}
		out = append(out, memberMap)
	}

	return svc.okResp(map[string]interface{}{"members": out})
}

// ==================== 2.1 Member Goals (OKRs + KPIs) ====================
//
// GET /v2/teams/{teamId}/members/{memberId}/goals
// Returns all goals for a team member split into okrs and kpis.

func (svc *Service) getMemberGoalsForManager(teamID, memberID, managerUserName string) (events.APIGatewayProxyResponse, error) {
	if err := svc.assertTeamMember(teamID, managerUserName); err != nil {
		return *err, nil
	}

	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKGoalPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("getMemberGoalsForManager query error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch member goals")
	}

	var okrs []map[string]interface{}
	var kpis []map[string]interface{}

	for _, item := range result.Items {
		skAttr, ok := item["SK"].(*types.AttributeValueMemberS)
		if !ok {
			continue
		}
		if strings.Contains(skAttr.Value, SKCommentInfix) {
			continue
		}
		var rec GoalRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
			continue
		}

		switch GoalType(rec.Type) {
		case GoalTypeOKR:
			okrs = append(okrs, buildOKRResponse(rec))
		case GoalTypeKPI:
			kpis = append(kpis, buildKPIResponse(rec))
		}
	}

	if okrs == nil {
		okrs = []map[string]interface{}{}
	}
	if kpis == nil {
		kpis = []map[string]interface{}{}
	}

	return svc.okResp(map[string]interface{}{
		"okrs": okrs,
		"kpis": kpis,
	})
}

// ==================== 2.2 Member Meetings ====================
//
// GET /v2/teams/{teamId}/members/{memberId}/meetings

func (svc *Service) getMemberMeetingsForManager(teamID, memberID, managerUserName string) (events.APIGatewayProxyResponse, error) {
	if err := svc.assertTeamMember(teamID, managerUserName); err != nil {
		return *err, nil
	}

	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKMeetingPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("getMemberMeetingsForManager query error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch member meetings")
	}

	var meetings []MeetingRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &meetings)

	sort.Slice(meetings, func(i, j int) bool {
		return meetings[i].Date > meetings[j].Date // newest first for manager view
	})

	meetingList := make([]map[string]interface{}, 0, len(meetings))
	for _, m := range meetings {
		meetingList = append(meetingList, map[string]interface{}{
			"id":          m.MeetingID,
			"date":        m.Date,
			"title":       m.Summary,
			"notes":       m.Summary,
			"actionItems": m.ActionItems,
		})
	}

	return svc.okResp(map[string]interface{}{"meetings": meetingList})
}

// ==================== 2.3 Member Appreciations ====================
//
// GET /v2/teams/{teamId}/members/{memberId}/appreciations

func (svc *Service) getMemberAppreciationsForManager(teamID, memberID, managerUserName string) (events.APIGatewayProxyResponse, error) {
	if err := svc.assertTeamMember(teamID, managerUserName); err != nil {
		return *err, nil
	}

	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKAppreciationPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("getMemberAppreciationsForManager query error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch member appreciations")
	}

	var records []AppreciationRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)

	sort.Slice(records, func(i, j int) bool {
		return records[i].Date > records[j].Date
	})

	appreciationList := make([]map[string]interface{}, 0, len(records))
	for _, a := range records {
		appreciationList = append(appreciationList, map[string]interface{}{
			"id":           a.AppreciationID,
			"from":         a.From,
			"fromInitials": a.FromInitials,
			"date":         a.Date,
			"message":      a.Message,
			"category":     a.Category,
		})
	}

	return svc.okResp(map[string]interface{}{"appreciations": appreciationList})
}

// ==================== 2.4 Member Manager Comments ====================
//
// GET  /v2/teams/{teamId}/members/{memberId}/comments
// POST /v2/teams/{teamId}/members/{memberId}/comments

func (svc *Service) getMemberComments(teamID, memberID, managerUserName string) (events.APIGatewayProxyResponse, error) {
	if err := svc.assertTeamMember(teamID, managerUserName); err != nil {
		return *err, nil
	}

	records, err := svc.fetchManagerComments(teamID, memberID)
	if err != nil {
		svc.logger.Printf("getMemberComments fetchManagerComments error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch comments")
	}

	commentList := make([]map[string]interface{}, 0, len(records))
	for _, c := range records {
		commentList = append(commentList, buildManagerCommentResponse(c))
	}

	return svc.okResp(map[string]interface{}{"comments": commentList})
}

func (svc *Service) addManagerComment(teamID, memberID, managerUserName, managerDisplayName, body string) (events.APIGatewayProxyResponse, error) {
	if err := svc.assertTeamMember(teamID, managerUserName); err != nil {
		return *err, nil
	}

	req, err := parseBody[AddManagerCommentRequest](body)
	if err != nil || strings.TrimSpace(req.Text) == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "text is required")
	}

	commentType := req.Type
	if commentType == "" {
		commentType = "general"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	today := time.Now().UTC().Format("2006-01-02")
	commentID := uuid.New().String()

	rec := ManagerCommentRecord{
		PK:        buildPK(memberID, teamID),
		SK:        SKManagerCommentPrefix + commentID,
		CommentID: commentID,
		TeamID:    teamID,
		MemberID:  memberID,
		Author:    managerDisplayName,
		Initials:  initials(managerDisplayName),
		Text:      req.Text,
		Type:      commentType,
		Date:      today,
		CreatedAt: now,
	}

	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to marshal comment")
	}
	if _, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.perfHubTable),
		Item:      item,
	}); err != nil {
		svc.logger.Printf("addManagerComment PutItem error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save comment")
	}

	return svc.createdResp(map[string]interface{}{"comment": buildManagerCommentResponse(rec)})
}

// ==================== 2.5 Performance Summary (all-in-one) ====================
//
// GET /v2/teams/{teamId}/members/{memberId}/performance-summary

// ==================== 2.4b Manager Goal Comment ====================
//
// POST /v2/teams/{teamId}/members/{memberId}/goals/{goalId}/comments
// Adds a comment with role="manager" directly on a member's individual goal.
// The comment is stored on the member's own partition and appears alongside
// member-authored comments when the goal is fetched.
func (svc *Service) addManagerGoalComment(teamID, memberID, goalID, managerUserName, managerDisplayName, body string) (events.APIGatewayProxyResponse, error) {
	if err := svc.assertTeamMember(teamID, managerUserName); err != nil {
		return *err, nil
	}

	// Verify the goal exists on the member's partition
	memberPK := buildPK(memberID, teamID)
	result, err := svc.ddb.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: memberPK},
			"SK": &types.AttributeValueMemberS{Value: SKGoalPrefix + goalID},
		},
	})
	if err != nil || result.Item == nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Goal not found for this member")
	}

	return svc.writeGoalComment(memberPK, goalID, memberID, managerUserName, managerDisplayName, "manager", body)
}



func (svc *Service) getMemberPerformanceSummary(teamID, memberID, managerUserName string) (events.APIGatewayProxyResponse, error) {
	if err := svc.assertTeamMember(teamID, managerUserName); err != nil {
		return *err, nil
	}

	// Fetch review record for profile enrichment
	reviewRec, _ := svc.fetchMemberReviewRecord(teamID, memberID)

	// Fetch team member info for name / role
	memberInfo, _ := svc.teamsSVC.GetTeamMemberDetails(teamID, memberID)

	profile := map[string]interface{}{
		"id":          memberID,
		"name":        "",
		"initials":    "",
		"role":        "",
		"department":  "",
		"avatarColor": "",
	}
	if memberInfo != nil {
		profile["name"] = memberInfo.DisplayName
		profile["initials"] = initials(memberInfo.DisplayName)
		profile["role"] = string(memberInfo.Role)
	}
	if reviewRec != nil {
		profile["overallRating"] = reviewRec.OverallRating
		profile["lastReviewDate"] = reviewRec.LastReviewDate
		profile["isPendingReview"] = reviewRec.IsPendingReview
		profile["hasUserUpdatedReviews"] = reviewRec.HasUserUpdatedReviews
	} else {
		profile["overallRating"] = 0.0
		profile["lastReviewDate"] = nil
		profile["isPendingReview"] = false
		profile["hasUserUpdatedReviews"] = false
	}

	// Goals
	okrs, kpis := svc.fetchMemberGoalsSplit(teamID, memberID)

	// Meetings
	meetings := svc.fetchMemberMeetingsList(teamID, memberID)

	// Appreciations
	appreciations := svc.fetchMemberAppreciationsList(teamID, memberID)

	// Manager comments
	comments, _ := svc.fetchManagerComments(teamID, memberID)
	commentList := make([]map[string]interface{}, 0, len(comments))
	for _, c := range comments {
		commentList = append(commentList, buildManagerCommentResponse(c))
	}

	return svc.okResp(map[string]interface{}{
		"profile":       profile,
		"okrs":          okrs,
		"kpis":          kpis,
		"meetings":      meetings,
		"appreciations": appreciations,
		"comments":      commentList,
	})
}

// ==================== DDB Helpers ====================

// assertTeamMember verifies that userName is a member of teamID.
// Returns a pointer to an APIGatewayProxyResponse if the check fails, nil otherwise.
func (svc *Service) assertTeamMember(teamID, userName string) *events.APIGatewayProxyResponse {
	_, err := svc.teamsSVC.GetTeamMemberDetails(teamID, userName)
	if err != nil {
		resp, _ := svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		return &resp
	}
	return nil
}

// fetchTeamReviewMap loads all TeamMemberReviewRecord rows for a team keyed by member userName.
func (svc *Service) fetchTeamReviewMap(teamID string) (map[string]TeamMemberReviewRecord, error) {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildTeamPK(teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKMemberReviewPrefix},
		},
	})
	if err != nil {
		return nil, err
	}

	reviewMap := make(map[string]TeamMemberReviewRecord, len(result.Items))
	for _, item := range result.Items {
		var rec TeamMemberReviewRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err == nil {
			reviewMap[rec.MemberUserName] = rec
		}
	}
	return reviewMap, nil
}

// fetchMemberReviewRecord fetches a single TeamMemberReviewRecord for a member.
func (svc *Service) fetchMemberReviewRecord(teamID, memberUserName string) (*TeamMemberReviewRecord, error) {
	result, err := svc.ddb.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: buildTeamPK(teamID)},
			"SK": &types.AttributeValueMemberS{Value: SKMemberReviewPrefix + memberUserName},
		},
	})
	if err != nil || result.Item == nil {
		return nil, err
	}
	var rec TeamMemberReviewRecord
	attributevalue.UnmarshalMap(result.Item, &rec)
	return &rec, nil
}

// fetchManagerComments returns all ManagerCommentRecords for a given member in a team.
func (svc *Service) fetchManagerComments(teamID, memberID string) ([]ManagerCommentRecord, error) {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKManagerCommentPrefix},
		},
	})
	if err != nil {
		return nil, err
	}

	var records []ManagerCommentRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt // newest first
	})
	return records, nil
}

// fetchMemberGoalsSplit loads goals for a member and splits them into OKRs and KPIs.
func (svc *Service) fetchMemberGoalsSplit(teamID, memberID string) (okrs []map[string]interface{}, kpis []map[string]interface{}) {
	okrs = []map[string]interface{}{}
	kpis = []map[string]interface{}{}

	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKGoalPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("fetchMemberGoalsSplit query error: %v", err)
		return
	}

	for _, item := range result.Items {
		skAttr, ok := item["SK"].(*types.AttributeValueMemberS)
		if !ok || strings.Contains(skAttr.Value, SKCommentInfix) {
			continue
		}
		var rec GoalRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
			continue
		}
		switch GoalType(rec.Type) {
		case GoalTypeOKR:
			okrs = append(okrs, buildOKRResponse(rec))
		case GoalTypeKPI:
			kpis = append(kpis, buildKPIResponse(rec))
		}
	}
	return
}

// fetchMemberMeetingsList returns meeting responses for a member sorted newest first.
func (svc *Service) fetchMemberMeetingsList(teamID, memberID string) []map[string]interface{} {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKMeetingPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("fetchMemberMeetingsList query error: %v", err)
		return []map[string]interface{}{}
	}

	var meetings []MeetingRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &meetings)
	sort.Slice(meetings, func(i, j int) bool { return meetings[i].Date > meetings[j].Date })

	out := make([]map[string]interface{}, 0, len(meetings))
	for _, m := range meetings {
		out = append(out, map[string]interface{}{
			"id":          m.MeetingID,
			"date":        m.Date,
			"title":       m.Summary,
			"notes":       m.Summary,
			"actionItems": m.ActionItems,
		})
	}
	return out
}

// fetchMemberAppreciationsList returns appreciation responses for a member sorted newest first.
func (svc *Service) fetchMemberAppreciationsList(teamID, memberID string) []map[string]interface{} {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(memberID, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKAppreciationPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("fetchMemberAppreciationsList query error: %v", err)
		return []map[string]interface{}{}
	}

	var records []AppreciationRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)
	sort.Slice(records, func(i, j int) bool { return records[i].Date > records[j].Date })

	out := make([]map[string]interface{}, 0, len(records))
	for _, a := range records {
		out = append(out, map[string]interface{}{
			"id":           a.AppreciationID,
			"from":         a.From,
			"fromInitials": a.FromInitials,
			"date":         a.Date,
			"message":      a.Message,
			"category":     a.Category,
		})
	}
	return out
}

// ==================== Response Builders ====================

func buildOKRResponse(g GoalRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":       g.GoalID,
		"title":    g.Title,
		"status":   g.Status,
		"progress": g.Progress,
		"dueDate":  g.DueDate,
		// keyResults are not stored in the current GoalRecord schema.
		// The front-end should treat an empty array as "no key results yet".
		"keyResults": []interface{}{},
	}
}

func buildKPIResponse(g GoalRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":   g.GoalID,
		"name": g.Title,
		// current / target / unit / frequency / trend / change are optional extended fields.
		// They default to sensible zero-values until the schema is extended.
		"current":   g.Progress,
		"target":    100,
		"unit":      "",
		"frequency": "Monthly",
		"trend":     "up",
		"change":    "",
	}
}

func buildManagerCommentResponse(c ManagerCommentRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":             c.CommentID,
		"author":         c.Author,
		"authorInitials": c.Initials,
		"date":           c.Date,
		"text":           c.Text,
		"type":           c.Type,
	}
}
