package common

import (
	"fmt"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ==================== Route: Task ====================
//
// PATCH /v2/posts/{postId}/task/status
// PATCH /v2/posts/{postId}/task/time

func (svc *Service) handleTask(request events.APIGatewayProxyRequest, parts []string, userName, cognitoID string) (events.APIGatewayProxyResponse, error) {
	if len(parts) == 5 && parts[1] == "posts" && parts[3] == "task" {
		postID := parts[2]
		action := parts[4]

		post, err := svc.fetchPostRecord(postID)
		if err != nil {
			return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
		}
		if post.Type != string(PostTypeTask) {
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "This post is not a task")
		}
		if err := svc.ensureTeamMember(post.TeamID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}

		if request.HTTPMethod == "PATCH" {
			switch action {
			case "status":
				return svc.updateTaskStatus(post, userName, request.Body)
			case "time":
				return svc.logTaskTime(post, userName, request.Body)
			}
		}
	}

	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== Update Task Status ====================

func (svc *Service) updateTaskStatus(post *PostRecord, userName, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[UpdateTaskStatusRequest](body)
	if err != nil || req.Status == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "status is required")
	}

	validStatuses := map[string]bool{"todo": true, "in-progress": true, "done": true}
	if !validStatuses[req.Status] {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "status must be one of: todo, in-progress, done")
	}

	// Only the assignee or author can update status
	if post.AssigneeUserID != userName && post.AuthorUserID != userName {
		isAdmin, err := svc.teamsSVC.IsTeamAdmin(post.TeamID, userName)
		if err != nil || !isAdmin {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "Only the assignee, author, or team admin can update task status")
		}
	}

	_, err = svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + post.PostID},
			"SK": &types.AttributeValueMemberS{Value: SKMetadata},
		},
		UpdateExpression: aws.String("SET taskStatus = :status"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: req.Status},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update task status")
	}

	return svc.okResp(map[string]interface{}{
		"postId": post.PostID,
		"status": req.Status,
	}, nil)
}

// ==================== Log Time ====================

func (svc *Service) logTaskTime(post *PostRecord, userName, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[LogTimeRequest](body)
	if err != nil || req.Hours <= 0 {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "hours must be a positive number")
	}

	newTotal := post.TimeSpentHours + req.Hours

	_, err = svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + post.PostID},
			"SK": &types.AttributeValueMemberS{Value: SKMetadata},
		},
		UpdateExpression: aws.String("SET timeSpentHours = :total"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":total": &types.AttributeValueMemberN{Value: fmt.Sprintf("%g", newTotal)},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to log time")
	}

	return svc.okResp(map[string]interface{}{
		"postId":         post.PostID,
		"timeSpentHours": newTotal,
	}, nil)
}
