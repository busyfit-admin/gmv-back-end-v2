package common

// ==================== Routes ====================
//
// GET   /v2/users/me/goals                               — list goals
// POST  /v2/users/me/goals                               — create goal
// PATCH /v2/users/me/goals/{goalId}                      — update progress / status
// POST  /v2/users/me/goals/{goalId}/comments             — add comment
// POST  /v2/users/me/goals/{goalId}/tasks                — add linked task (sets goalId attribute)

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

func (svc *Service) handleGoals(request events.APIGatewayProxyRequest, parts []string, userName, displayName, teamID string) (events.APIGatewayProxyResponse, error) {
	// /v2/users/me/goals  (4 parts)
	if len(parts) == 4 {
		switch request.HTTPMethod {
		case "GET":
			return svc.listGoals(userName, teamID, request.QueryStringParameters)
		case "POST":
			return svc.createGoal(userName, teamID, request.Body)
		}
	}

	// /v2/users/me/goals/{goalId}  (5 parts)
	if len(parts) == 5 {
		goalID := parts[4]
		if request.HTTPMethod == "PATCH" {
			return svc.updateGoal(userName, teamID, goalID, request.Body)
		}
	}

	// /v2/users/me/goals/{goalId}/comments  (6 parts)
	if len(parts) == 6 && parts[5] == "comments" {
		goalID := parts[4]
		if request.HTTPMethod == "POST" {
			return svc.addGoalComment(userName, teamID, displayName, goalID, request.Body)
		}
	}

	// /v2/users/me/goals/{goalId}/tasks  (6 parts)
	if len(parts) == 6 && parts[5] == "tasks" {
		goalID := parts[4]
		if request.HTTPMethod == "POST" {
			return svc.addLinkedTask(userName, teamID, goalID, request.Body)
		}
	}

	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== List Goals ====================

func (svc *Service) listGoals(userName, teamID string, queryParams map[string]string) (events.APIGatewayProxyResponse, error) {
	typeFilter := queryString(queryParams, "type")
	statusFilter := queryString(queryParams, "status")

	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKGoalPrefix},
		},
	})
	if err != nil {
		svc.logger.Printf("listGoals query error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list goals")
	}

	// Only goal metadata rows — skip sub-item rows by inspecting SK directly.
	// Comment SKs contain "#CMMNT#"; tasks are now top-level "TASK#" and won't appear in GOAL# prefix queries.
	var goals []GoalRecord
	for _, item := range result.Items {
		skAttr, ok := item["SK"].(*types.AttributeValueMemberS)
		if !ok {
			continue
		}
		sk := skAttr.Value
		if strings.Contains(sk, SKCommentInfix) {
			continue
		}
		var rec GoalRecord
		if err := attributevalue.UnmarshalMap(item, &rec); err != nil {
			continue
		}
		if typeFilter != "" && rec.Type != typeFilter {
			continue
		}
		if statusFilter != "" && rec.Status != statusFilter {
			continue
		}
		goals = append(goals, rec)
	}

	// Build response enriched with tasks and comments
	goalResp := make([]map[string]interface{}, 0, len(goals))
	for _, g := range goals {
		tasks, _ := svc.fetchLinkedTasks(userName, teamID, g.GoalID)
		comments, _ := svc.fetchGoalComments(userName, teamID, g.GoalID)
		goalResp = append(goalResp, buildGoalResponse(g, tasks, comments))
	}

	return svc.okResp(map[string]interface{}{"goals": goalResp})
}

// ==================== Create Goal ====================

func (svc *Service) createGoal(userName, teamID, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[CreateGoalRequest](body)
	if err != nil || req.Title == "" || req.Type == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "title and type are required")
	}
	if req.DueDate == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "dueDate is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	today := time.Now().UTC().Format("2006-01-02")
	goalID := uuid.New().String()
	status := req.Status
	if status == "" {
		status = string(GoalStatusOnTrack)
	}

	rec := GoalRecord{
		PK:          buildPK(userName, teamID),
		SK:          SKGoalPrefix + goalID,
		GoalID:      goalID,
		UserName:    userName,
		Title:       req.Title,
		Type:        req.Type,
		Progress:    0,
		DueDate:     req.DueDate,
		Status:      status,
		Description: req.Description,
		CreatedAt:   today,
		UpdatedAt:   now,
	}

	item, err := attributevalue.MarshalMap(rec)
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to marshal goal")
	}
	if _, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.perfHubTable),
		Item:      item,
	}); err != nil {
		svc.logger.Printf("createGoal PutItem error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create goal")
	}

	return svc.createdResp(map[string]interface{}{"goal": buildGoalResponse(rec, nil, nil)})
}

// ==================== Update Goal ====================

func (svc *Service) updateGoal(userName, teamID, goalID, body string) (events.APIGatewayProxyResponse, error) {
	rec, err := svc.fetchGoal(userName, teamID, goalID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Goal not found")
	}

	req, err := parseBody[UpdateGoalRequest](body)
	if err != nil {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
	}

	now := time.Now().UTC().Format(time.RFC3339)

	if req.Progress != nil {
		rec.Progress = *req.Progress
	}
	if req.Status != "" {
		rec.Status = req.Status
	}
	rec.UpdatedAt = now

	item, _ := attributevalue.MarshalMap(rec)
	if _, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.perfHubTable),
		Item:      item,
	}); err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update goal")
	}

	// Upsert linked tasks if provided
	if len(req.LinkedTasks) > 0 {
		for _, t := range req.LinkedTasks {
			taskID := t.ID
			if taskID == "" {
				taskID = uuid.New().String()
			}
			taskRec := LinkedTaskRecord{
				PK:        buildPK(userName, teamID),
				SK:        SKTaskPrefix + taskID,
				TaskID:    taskID,
				GoalID:    goalID,
				UserName:  userName,
				Title:     t.Title,
				Done:      t.Done,
				CreatedAt: now,
			}
			if av, err := attributevalue.MarshalMap(taskRec); err == nil {
				svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
					TableName: aws.String(svc.perfHubTable),
					Item:      av,
				})
			}
		}
	}

	tasks, _ := svc.fetchLinkedTasks(userName, teamID, goalID)
	comments, _ := svc.fetchGoalComments(userName, teamID, goalID)
	return svc.okResp(map[string]interface{}{"goal": buildGoalResponse(*rec, tasks, comments)})
}

// ==================== Add Comment ====================

func (svc *Service) addGoalComment(userName, teamID, displayName, goalID, body string) (events.APIGatewayProxyResponse, error) {
	if _, err := svc.fetchGoal(userName, teamID, goalID); err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Goal not found")
	}

	req, err := parseBody[AddGoalCommentRequest](body)
	if err != nil || req.Text == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "text is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	today := time.Now().UTC().Format("2006-01-02")
	commentID := uuid.New().String()

	rec := GoalCommentRecord{
		PK:        buildPK(userName, teamID),
		SK:        SKGoalPrefix + goalID + SKCommentInfix + commentID,
		CommentID: commentID,
		GoalID:    goalID,
		UserName:  userName,
		Author:    displayName,
		Initials:  initials(displayName),
		Text:      req.Text,
		Date:      today,
		CreatedAt: now,
	}
	item, _ := attributevalue.MarshalMap(rec)
	if _, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.perfHubTable),
		Item:      item,
	}); err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add comment")
	}

	return svc.createdResp(map[string]interface{}{
		"comment": map[string]interface{}{
			"id":       commentID,
			"author":   displayName,
			"initials": initials(displayName),
			"text":     req.Text,
			"date":     today,
		},
	})
}

// ==================== Add Linked Task ====================

func (svc *Service) addLinkedTask(userName, teamID, goalID, body string) (events.APIGatewayProxyResponse, error) {
	if _, err := svc.fetchGoal(userName, teamID, goalID); err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Goal not found")
	}

	req, err := parseBody[AddLinkedTaskRequest](body)
	if err != nil || req.Title == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "title is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	taskID := uuid.New().String()

	rec := LinkedTaskRecord{
		PK:        buildPK(userName, teamID),
		SK:        SKTaskPrefix + taskID,
		TaskID:    taskID,
		GoalID:    goalID,
		UserName:  userName,
		Title:     req.Title,
		Done:      false,
		CreatedAt: now,
	}
	item, _ := attributevalue.MarshalMap(rec)
	if _, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.perfHubTable),
		Item:      item,
	}); err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add task")
	}

	return svc.createdResp(map[string]interface{}{
		"task": map[string]interface{}{
			"id":     taskID,
			"goalId": goalID,
			"title":  req.Title,
			"done":   false,
		},
	})
}

// ==================== DDB Helpers ====================

func (svc *Service) fetchGoal(userName, teamID, goalID string) (*GoalRecord, error) {
	result, err := svc.ddb.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			"SK": &types.AttributeValueMemberS{Value: SKGoalPrefix + goalID},
		},
	})
	if err != nil || result.Item == nil {
		return nil, err
	}
	var rec GoalRecord
	attributevalue.UnmarshalMap(result.Item, &rec)
	return &rec, nil
}

func (svc *Service) fetchLinkedTasks(userName, teamID, goalID string) ([]LinkedTaskRecord, error) {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		FilterExpression:       aws.String("goalId = :goalId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKTaskPrefix},
			":goalId": &types.AttributeValueMemberS{Value: goalID},
		},
	})
	if err != nil {
		return nil, err
	}
	var tasks []LinkedTaskRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &tasks)
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt < tasks[j].CreatedAt })
	return tasks, nil
}

func (svc *Service) fetchGoalComments(userName, teamID, goalID string) ([]GoalCommentRecord, error) {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKGoalPrefix + goalID + SKCommentInfix},
		},
	})
	if err != nil {
		return nil, err
	}
	var comments []GoalCommentRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &comments)
	sort.Slice(comments, func(i, j int) bool { return comments[i].CreatedAt < comments[j].CreatedAt })
	return comments, nil
}

// ==================== Response Builder ====================

func buildGoalResponse(g GoalRecord, tasks []LinkedTaskRecord, comments []GoalCommentRecord) map[string]interface{} {
	taskList := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		taskList = append(taskList, map[string]interface{}{
			"id":        t.TaskID,
			"title":     t.Title,
			"done":      t.Done,
			"goalId":    t.GoalID,
			"createdAt": t.CreatedAt,
		})
	}
	commentList := make([]map[string]interface{}, 0, len(comments))
	for _, c := range comments {
		commentList = append(commentList, map[string]interface{}{
			"id":       c.CommentID,
			"author":   c.Author,
			"initials": c.Initials,
			"text":     c.Text,
			"date":     c.Date,
		})
	}
	return map[string]interface{}{
		"id":          g.GoalID,
		"title":       g.Title,
		"type":        g.Type,
		"progress":    g.Progress,
		"dueDate":     g.DueDate,
		"status":      g.Status,
		"createdDate": g.CreatedAt,
		"description": g.Description,
		"linkedTasks": taskList,
		"comments":    commentList,
	}
}
