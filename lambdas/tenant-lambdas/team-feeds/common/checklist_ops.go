package common

import (
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// ==================== Route: Checklist ====================
//
// PATCH  /v2/posts/{postId}/checklist/items/{itemId}
// POST   /v2/posts/{postId}/checklist/items
// DELETE /v2/posts/{postId}/checklist/items/{itemId}

func (svc *Service) handleChecklist(request events.APIGatewayProxyRequest, parts []string, userName, cognitoID string) (events.APIGatewayProxyResponse, error) {
	// /v2/posts/{postId}/checklist/items/{itemId}  (6 parts: v2, posts, {postId}, checklist, items, {itemId})
	if len(parts) == 6 && parts[1] == "posts" && parts[3] == "checklist" && parts[4] == "items" {
		postID := parts[2]
		itemID := parts[5]
		switch request.HTTPMethod {
		case "PATCH":
			return svc.toggleChecklistItem(postID, itemID, userName, request.Body)
		case "DELETE":
			return svc.removeChecklistItem(postID, itemID, userName)
		}
	}

	// /v2/posts/{postId}/checklist/items  (5 parts)
	if len(parts) == 5 && parts[1] == "posts" && parts[3] == "checklist" && parts[4] == "items" {
		postID := parts[2]
		if request.HTTPMethod == "POST" {
			return svc.addChecklistItem(postID, userName, request.Body)
		}
	}

	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== Toggle Checklist Item ====================

func (svc *Service) toggleChecklistItem(postID, itemID, userName, body string) (events.APIGatewayProxyResponse, error) {
	if err := svc.ensurePostTeamMember(postID, userName); err != nil {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
	}

	req, err := parseBody[ToggleChecklistItemRequest](body)
	if err != nil {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
	}

	completedVal := "false"
	if req.Completed {
		completedVal = "true"
	}

	_, err = svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKItemPrefix + itemID},
		},
		UpdateExpression: aws.String("SET completed = :completed"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":completed": &types.AttributeValueMemberBOOL{Value: completedVal == "true"},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	})
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Checklist item not found")
	}

	return svc.okResp(map[string]interface{}{
		"itemId":    itemID,
		"postId":    postID,
		"completed": req.Completed,
	}, nil)
}

// ==================== Add Checklist Item ====================

func (svc *Service) addChecklistItem(postID, userName, body string) (events.APIGatewayProxyResponse, error) {
	post, err := svc.fetchPostRecord(postID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
	}
	if post.Type != string(PostTypeChecklist) {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "This post is not a checklist")
	}

	if err := svc.ensureTeamMember(post.TeamID, userName); err != nil {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
	}

	req, err := parseBody[AddChecklistItemRequest](body)
	if err != nil || req.Text == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "text is required")
	}

	itemID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	record := ChecklistItemRecord{
		PK:        PrefixPost + postID,
		SK:        SKItemPrefix + itemID,
		ItemID:    itemID,
		PostID:    postID,
		Text:      req.Text,
		Completed: false,
		CreatedAt: now,
	}
	item, _ := attributevalue.MarshalMap(record)
	_, err = svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.feedTable),
		Item:      item,
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add checklist item")
	}

	return svc.createdResp(map[string]interface{}{
		"itemId":    itemID,
		"postId":    postID,
		"text":      req.Text,
		"completed": false,
		"createdAt": now,
	})
}

// ==================== Remove Checklist Item ====================

func (svc *Service) removeChecklistItem(postID, itemID, userName string) (events.APIGatewayProxyResponse, error) {
	post, err := svc.fetchPostRecord(postID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
	}

	if err := svc.ensureTeamAdminOrAuthor(post.TeamID, userName, post.AuthorUserID); err != nil {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", err.Error())
	}

	_, err = svc.ddb.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKItemPrefix + itemID},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to remove checklist item")
	}

	return svc.noContentResp()
}

// ==================== Helper ====================

func (svc *Service) ensurePostTeamMember(postID, userName string) error {
	post, err := svc.fetchPostRecord(postID)
	if err != nil {
		return err
	}
	return svc.ensureTeamMember(post.TeamID, userName)
}
