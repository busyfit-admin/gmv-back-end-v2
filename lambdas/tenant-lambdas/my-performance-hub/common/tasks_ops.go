package common

// ==================== Routes ====================
//
// GET   /v2/users/me/tasks              — list all tasks (filter: ?goalId=, ?done=true|false, ?status=)
// POST  /v2/users/me/tasks              — create a standalone task (optional goalId in body)
// PATCH /v2/users/me/tasks/{taskId}     — update task fields (status, title, priority, tags, time, dueDate, goalId)

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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
//   - done=true|false  (backward-compat; true = status is done|closed)
//   - status=todo|in-progress|done|closed
func (svc *Service) listAllTasks(userName, teamID string, queryParams map[string]string) (events.APIGatewayProxyResponse, error) {
	goalIDFilter := queryString(queryParams, "goalId")
	doneFilter := queryString(queryParams, "done")
	statusFilter := strings.ToLower(queryString(queryParams, "status"))

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

	// Apply done filter in Go (backward-compat; avoids reserved-word issues with bool in FilterExpression)
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

	// Apply status filter (applied on top of any done filter)
	if statusFilter != "" {
		filtered := tasks[:0]
		for _, t := range tasks {
			if strings.EqualFold(t.Status, statusFilter) {
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

	// Validate priority
	if req.Priority != "" {
		switch TaskPriority(req.Priority) {
		case TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh, TaskPriorityUrgent:
		default:
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "priority must be one of: low, medium, high, urgent")
		}
	}

	// Validate and default status
	status := req.Status
	if status == "" {
		status = string(TaskStatusTodo)
	}
	switch TaskStatus(status) {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone, TaskStatusClosed:
	default:
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "status must be one of: todo, in-progress, done, closed")
	}

	// If goalId provided, verify the goal exists
	if req.GoalID != "" {
		if _, err := svc.fetchGoal(userName, teamID, req.GoalID); err != nil {
			return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Goal not found")
		}
	}

	// Allocate team-scoped TASK-N identifier
	taskNum, err := svc.nextTaskNumber(teamID)
	if err != nil {
		svc.logger.Printf("createTask nextTaskNumber error: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to allocate task number")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	taskID := fmt.Sprintf("TASK-%d", taskNum)
	done := status == string(TaskStatusDone) || status == string(TaskStatusClosed)

	rec := LinkedTaskRecord{
		PK:          buildPK(userName, teamID),
		SK:          SKTaskPrefix + taskID,
		TaskID:      taskID,
		TaskNumber:  taskNum,
		GoalID:      req.GoalID,
		UserName:    userName,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      status,
		Done:        done,
		Tags:        req.Tags,
		TimeHours:   req.TimeHours,
		TimeDays:    req.TimeDays,
		DueDate:     req.DueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
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

// updateTask supports updating all task fields in one call.
// GoalID behaviour:
//   - field absent (nil pointer): existing goalId unchanged
//   - field present with value "": unlink from goal (removes goalId attribute)
//   - field present with UUID: link to / relink to that goal
//
// Tags behaviour:
//   - field absent (nil): existing tags unchanged
//   - field present as []: clear tags
//   - field present as ["a","b"]: replace tags entirely
func (svc *Service) updateTask(userName, teamID, taskID, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[UpdateTaskRequest](body)
	if err != nil {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
	}

	// Validate priority if provided
	if req.Priority != "" {
		switch TaskPriority(req.Priority) {
		case TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh, TaskPriorityUrgent:
		default:
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "priority must be one of: low, medium, high, urgent")
		}
	}

	// Validate status if provided
	if req.Status != "" {
		switch TaskStatus(req.Status) {
		case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone, TaskStatusClosed:
		default:
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "status must be one of: todo, in-progress, done, closed")
		}
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
	exprNames := map[string]string{}

	// Always stamp updatedAt
	setExprs = append(setExprs, "updatedAt = :updatedAt")
	exprValues[":updatedAt"] = &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)}

	// status and done are synced bidirectionally
	if req.Status != "" {
		isDone := req.Status == string(TaskStatusDone) || req.Status == string(TaskStatusClosed)
		setExprs = append(setExprs, "#status = :status")
		exprNames["#status"] = "status"
		exprValues[":status"] = &types.AttributeValueMemberS{Value: req.Status}
		setExprs = append(setExprs, "#done = :done")
		exprNames["#done"] = "done"
		exprValues[":done"] = &types.AttributeValueMemberBOOL{Value: isDone}
	} else if req.Done != nil {
		// Backward-compat: derive status from done bool
		derivedStatus := string(TaskStatusTodo)
		if *req.Done {
			derivedStatus = string(TaskStatusDone)
		}
		setExprs = append(setExprs, "#done = :done")
		exprNames["#done"] = "done"
		exprValues[":done"] = &types.AttributeValueMemberBOOL{Value: *req.Done}
		setExprs = append(setExprs, "#status = :status")
		exprNames["#status"] = "status"
		exprValues[":status"] = &types.AttributeValueMemberS{Value: derivedStatus}
	}

	if req.Title != "" {
		setExprs = append(setExprs, "title = :title")
		exprValues[":title"] = &types.AttributeValueMemberS{Value: req.Title}
	}
	if req.Description != "" {
		setExprs = append(setExprs, "description = :description")
		exprValues[":description"] = &types.AttributeValueMemberS{Value: req.Description}
	}
	if req.Priority != "" {
		setExprs = append(setExprs, "priority = :priority")
		exprValues[":priority"] = &types.AttributeValueMemberS{Value: req.Priority}
	}
	if req.DueDate != "" {
		setExprs = append(setExprs, "dueDate = :dueDate")
		exprValues[":dueDate"] = &types.AttributeValueMemberS{Value: req.DueDate}
	}
	if req.TimeHours != nil {
		setExprs = append(setExprs, "timeHours = :timeHours")
		exprValues[":timeHours"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%g", *req.TimeHours)}
	}
	if req.TimeDays != nil {
		setExprs = append(setExprs, "timeDays = :timeDays")
		exprValues[":timeDays"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%g", *req.TimeDays)}
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
	if req.Tags != nil {
		if len(*req.Tags) == 0 {
			removeExprs = append(removeExprs, "tags")
		} else {
			tagList := make([]types.AttributeValue, len(*req.Tags))
			for i, tag := range *req.Tags {
				tagList[i] = &types.AttributeValueMemberS{Value: tag}
			}
			setExprs = append(setExprs, "tags = :tags")
			exprValues[":tags"] = &types.AttributeValueMemberL{Value: tagList}
		}
	}

	// setExprs always has at least updatedAt, so no empty check needed
	updateExpr := "SET " + strings.Join(setExprs, ", ")
	if len(removeExprs) > 0 {
		updateExpr += " REMOVE " + strings.Join(removeExprs, ", ")
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
	if len(exprNames) > 0 {
		updateInput.ExpressionAttributeNames = exprNames
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
		"id":          t.TaskID,
		"taskNumber":  t.TaskNumber,
		"title":       t.Title,
		"description": t.Description,
		"status":      t.Status,
		"done":        t.Done,
		"priority":    t.Priority,
		"tags":        t.Tags,
		"timeHours":   t.TimeHours,
		"timeDays":    t.TimeDays,
		"dueDate":     t.DueDate,
		"goalId":      t.GoalID,
		"createdAt":   t.CreatedAt,
		"updatedAt":   t.UpdatedAt,
	}
}

// ==================== Task Number Counter ====================

// nextTaskNumber atomically increments the team-scoped task counter and returns
// the next task number. The first task in a team gets number 101.
func (svc *Service) nextTaskNumber(teamID string) (int, error) {
	result, err := svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.perfHubTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: buildTeamPK(teamID)},
			"SK": &types.AttributeValueMemberS{Value: "COUNTER#TASK_NUM"},
		},
		UpdateExpression: aws.String("ADD taskCounter :incr"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":incr": &types.AttributeValueMemberN{Value: "1"},
		},
		ReturnValues: types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return 0, err
	}

	var counter struct {
		TaskCounter int `dynamodbav:"taskCounter"`
	}
	if err := attributevalue.UnmarshalMap(result.Attributes, &counter); err != nil {
		return 0, err
	}
	// DDB ADD starts from 0, so offset by 100 to make the first task TASK-101
	return counter.TaskCounter + 100, nil
}
