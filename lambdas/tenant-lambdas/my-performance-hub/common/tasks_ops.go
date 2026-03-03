package common

// ==================== Routes ====================
//
// GET   /v2/users/me/tasks              — list all tasks (filter: ?goalId=, ?done=true|false)
// POST  /v2/users/me/tasks              — create a standalone task (optional goalId in body)
// PATCH /v2/users/me/tasks/{taskId}     — update done, title, or relink/unlink to a goal

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

func (svc *Service) handleTasks(request events.APIGatewayProxyRequest, parts []string, userName, teamID string) (events.APIGatewayProxyResponse, error) {
	// /v2/users/me/tasks  (4 parts: v2, users, me, tasks)
	if len(parts) == 4 {
		switch request.HTTPMethod {
		case "GET":
			return svc.listAllTasks(userName, teamID, request.QueryStringParameters)
		case "POST":
			return svc.createTask(userName, teamID, request.Body)
		}
	}

	// /v2/users/me/tasks/{taskId}  (5 parts)
	if len(parts) == 5 {
		taskID := parts[4]
		if request.HTTPMethod == "PATCH" {
			return svc.updateTask(userName, teamID, taskID, request.Body)
		}
	}

	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== List All Tasks ====================

// listAllTasks returns all tasks for (userName, teamID).
// Optional query params:
//   - goalId=<uuid>  — only tasks linked to that goal
//   - goalId=none    — only tasks with no goal linked
//   - done=true|false
func (svc *Service) listAllTasks(userName, teamID string, queryParams map[string]string) (events.APIGatewayProxyResponse, error) {
	goalIDFilter := queryString(queryParams, "goalId")
	doneFilter := queryString(queryParams, "done")

	input := &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			":prefix": &types.AttributeValueMemberS{Value: SKTaskPrefix},
		},
	}

	// Apply goalId filter via FilterExpression
	switch goalIDFilter {
	case "":
		// no filter
	case "none":
		// tasks with no goal linked
		input.FilterExpression = aws.String("attribute_not_exists(goalId) OR goalId = :empty")
		input.ExpressionAttributeValues[":empty"] = &types.AttributeValueMemberS{Value: ""}
	default:
		input.FilterExpression = aws.String("goalId = :goalId")
		input.ExpressionAttributeValues[":goalId"] = &types.AttributeValueMemberS{Value: goalIDFilter}
	}

	result, err := svc.ddb.Query(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("listAllTasks query error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list tasks")
	}

	var tasks []LinkedTaskRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &tasks)

	// Apply done filter in Go (avoids reserved-word issues with bool in FilterExpression)
	if doneFilter == "true" {
		filtered := tasks[:0]
		for _, t := range tasks {
			if t.Done {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	} else if doneFilter == "false" {
		filtered := tasks[:0]
		for _, t := range tasks {
			if !t.Done {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	sort.Slice(tasks, func(i, j int) bool { return tasks[i].CreatedAt < tasks[j].CreatedAt })

	out := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, buildTaskResponse(t))
	}

	return svc.okResp(map[string]interface{}{"tasks": out})
}

// ==================== Create Task ====================

func (svc *Service) createTask(userName, teamID, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[CreateTaskRequest](body)
	if err != nil || req.Title == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "title is required")
	}

	// If goalId provided, verify the goal exists
	if req.GoalID != "" {
		if _, err := svc.fetchGoal(userName, teamID, req.GoalID); err != nil {
			return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Goal not found")
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	taskID := uuid.New().String()

	rec := LinkedTaskRecord{
		PK:        buildPK(userName, teamID),
		SK:        SKTaskPrefix + taskID,
		TaskID:    taskID,
		GoalID:    req.GoalID,
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
		svc.logger.Printf("createTask PutItem error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create task")
	}

	return svc.createdResp(map[string]interface{}{"task": buildTaskResponse(rec)})
}

// ==================== Update / Relink Task ====================

// updateTask supports updating done, title, and goal linkage in one call.
// GoalID behaviour:
//   - field absent (nil pointer): existing goalId unchanged
//   - field present with value "": unlink from goal (removes goalId attribute)
//   - field present with UUID: link to / relink to that goal
func (svc *Service) updateTask(userName, teamID, taskID, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[UpdateTaskRequest](body)
	if err != nil {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
	}

	// If relinking to a new goal, verify goal exists first
	if req.GoalID != nil && *req.GoalID != "" {
		if _, err := svc.fetchGoal(userName, teamID, *req.GoalID); err != nil {
			return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Goal not found")
		}
	}

	var setExprs []string
	var removeExprs []string
	exprValues := map[string]types.AttributeValue{}

	if req.Done != nil {
		setExprs = append(setExprs, "#done = :done")
		exprValues[":done"] = &types.AttributeValueMemberBOOL{Value: *req.Done}
	}
	if req.Title != "" {
		setExprs = append(setExprs, "title = :title")
		exprValues[":title"] = &types.AttributeValueMemberS{Value: req.Title}
	}
	if req.GoalID != nil {
		if *req.GoalID == "" {
			// Unlink — remove the goalId attribute entirely
			removeExprs = append(removeExprs, "goalId")
		} else {
			setExprs = append(setExprs, "goalId = :goalId")
			exprValues[":goalId"] = &types.AttributeValueMemberS{Value: *req.GoalID}
		}
	}

	if len(setExprs) == 0 && len(removeExprs) == 0 {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "No fields to update")
	}

	var updateExpr string
	if len(setExprs) > 0 {
		updateExpr = "SET " + strings.Join(setExprs, ", ")
	}
	if len(removeExprs) > 0 {
		if updateExpr != "" {
			updateExpr += " "
		}
		updateExpr += "REMOVE " + strings.Join(removeExprs, ", ")
	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: buildPK(userName, teamID)},
			"SK": &types.AttributeValueMemberS{Value: SKTaskPrefix + taskID},
		},
		UpdateExpression:    aws.String(updateExpr),
		ConditionExpression: aws.String("attribute_exists(PK)"),
	}
	// done is a reserved word in DDB — alias it
	if req.Done != nil {
		updateInput.ExpressionAttributeNames = map[string]string{"#done": "done"}
	}
	if len(exprValues) > 0 {
		updateInput.ExpressionAttributeValues = exprValues
	}

	if _, err = svc.ddb.UpdateItem(svc.ctx, updateInput); err != nil {
		svc.logger.Printf("updateTask UpdateItem error: %v", err)
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Task not found or condition failed")
	}

	return svc.okResp(map[string]interface{}{
		"task": map[string]interface{}{
			"id":      taskID,
			"teamId":  teamID,
			"updated": true,
		},
	})
}

// ==================== Response Builder ====================

func buildTaskResponse(t LinkedTaskRecord) map[string]interface{} {
	return map[string]interface{}{
		"id":        t.TaskID,
		"title":     t.Title,
		"done":      t.Done,
		"goalId":    t.GoalID,
		"createdAt": t.CreatedAt,
	}
}
