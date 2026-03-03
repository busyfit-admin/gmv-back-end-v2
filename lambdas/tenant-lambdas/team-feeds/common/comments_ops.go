package common

import (
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// ==================== Route: Comments ====================
//
// GET    /v2/posts/{postId}/comments
// POST   /v2/posts/{postId}/comments
// PUT    /v2/posts/{postId}/comments/{commentId}
// DELETE /v2/posts/{postId}/comments/{commentId}

func (svc *Service) handleComments(request events.APIGatewayProxyRequest, parts []string, userName, cognitoID string) (events.APIGatewayProxyResponse, error) {
	// /v2/posts/{postId}/comments  (4 parts)
	if len(parts) == 4 && parts[1] == "posts" && parts[3] == "comments" {
		postID := parts[2]
		if err := svc.ensurePostTeamMember(postID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		switch request.HTTPMethod {
		case "GET":
			return svc.listComments(postID, request.QueryStringParameters)
		case "POST":
			return svc.addComment(postID, userName, cognitoID, request.Body)
		}
	}

	// /v2/posts/{postId}/comments/{commentId}  (5 parts)
	if len(parts) == 5 && parts[1] == "posts" && parts[3] == "comments" {
		postID := parts[2]
		commentID := parts[4]
		if err := svc.ensurePostTeamMember(postID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		switch request.HTTPMethod {
		case "PUT":
			return svc.editComment(postID, commentID, userName, request.Body)
		case "DELETE":
			return svc.deleteComment(postID, commentID, userName)
		}
	}

	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== List Comments ====================

func (svc *Service) listComments(postID string, queryParams map[string]string) (events.APIGatewayProxyResponse, error) {
	page := queryInt(queryParams, "page", 1)
	limit := queryInt(queryParams, "limit", 50)

	records, err := svc.fetchComments(postID, "")
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch comments")
	}

	total := len(records)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	response := make([]map[string]interface{}, 0, len(records[start:end]))
	for _, c := range records[start:end] {
		response = append(response, buildCommentResponse(c))
	}

	return svc.okResp(response, &MetaResponse{Total: total, Page: page, Limit: limit})
}

// ==================== Add Comment ====================

func (svc *Service) addComment(postID, userName, cognitoID, body string) (events.APIGatewayProxyResponse, error) {
	post, err := svc.fetchPostRecord(postID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
	}

	req, err := parseBody[AddCommentRequest](body)
	if err != nil || req.Content == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "content is required")
	}

	employee, err := svc.empSVC.GetEmployeeDataByCognitoId(cognitoID)
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch user data")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	commentID := uuid.New().String()

	record := CommentRecord{
		PK:              PrefixPost + postID,
		SK:              SKCommentPrefix + now + "#" + commentID,
		GSI1PK:          PrefixComment + commentID,
		GSI1SK:          "META",
		CommentID:       commentID,
		PostID:          postID,
		AuthorUserID:    userName,
		AuthorName:      employee.FirstName + " " + employee.LastName,
		Content:         req.Content,
		ParentCommentID: req.ParentCommentID,
		LikeCount:       0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	item, _ := attributevalue.MarshalMap(record)
	_, err = svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.feedTable),
		Item:      item,
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to add comment")
	}

	// Increment post comment count
	svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + post.PostID},
			"SK": &types.AttributeValueMemberS{Value: SKMetadata},
		},
		UpdateExpression: aws.String("ADD commentCount :inc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{Value: "1"},
		},
	})

	return svc.createdResp(buildCommentResponse(record))
}

// ==================== Edit Comment ====================

func (svc *Service) editComment(postID, commentID, userName, body string) (events.APIGatewayProxyResponse, error) {
	record, err := svc.fetchCommentByID(postID, commentID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Comment not found")
	}
	if record.AuthorUserID != userName {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", "Only the author can edit this comment")
	}

	req, err := parseBody[EditCommentRequest](body)
	if err != nil || req.Content == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "content is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: record.SK},
		},
		UpdateExpression: aws.String("SET #content = :content, updatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#content": "content",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":content":   &types.AttributeValueMemberS{Value: req.Content},
			":updatedAt": &types.AttributeValueMemberS{Value: now},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to edit comment")
	}

	record.Content = req.Content
	record.UpdatedAt = now
	return svc.okResp(buildCommentResponse(*record), nil)
}

// ==================== Delete Comment ====================

func (svc *Service) deleteComment(postID, commentID, userName string) (events.APIGatewayProxyResponse, error) {
	record, err := svc.fetchCommentByID(postID, commentID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Comment not found")
	}

	post, _ := svc.fetchPostRecord(postID)
	teamID := ""
	if post != nil {
		teamID = post.TeamID
	}

	if err := svc.ensureTeamAdminOrAuthor(teamID, userName, record.AuthorUserID); err != nil {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", err.Error())
	}

	_, err = svc.ddb.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: record.SK},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete comment")
	}

	// Decrement post comment count
	if post != nil {
		svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(svc.feedTable),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
				"SK": &types.AttributeValueMemberS{Value: SKMetadata},
			},
			UpdateExpression:    aws.String("ADD commentCount :dec"),
			ConditionExpression: aws.String("commentCount > :zero"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":dec":  &types.AttributeValueMemberN{Value: "-1"},
				":zero": &types.AttributeValueMemberN{Value: "0"},
			},
		})
	}

	return svc.noContentResp()
}

// ==================== DDB Helpers ====================

func (svc *Service) fetchComments(postID, _ string) ([]CommentRecord, error) {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.feedTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: PrefixPost + postID},
			":prefix": &types.AttributeValueMemberS{Value: SKCommentPrefix},
		},
		ScanIndexForward: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	var records []CommentRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)
	return records, nil
}

func (svc *Service) fetchCommentByID(postID, commentID string) (*CommentRecord, error) {
	// Query GSI1 to locate the comment's SK
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.feedTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :gsi1pk AND GSI1SK = :gsi1sk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk": &types.AttributeValueMemberS{Value: PrefixComment + commentID},
			":gsi1sk": &types.AttributeValueMemberS{Value: "META"},
		},
	})
	if err != nil || len(result.Items) == 0 {
		return nil, fmt.Errorf("comment not found: %s", commentID)
	}
	var record CommentRecord
	if err := attributevalue.UnmarshalMap(result.Items[0], &record); err != nil {
		return nil, err
	}
	return &record, nil
}

// ==================== Response Builder ====================

func buildCommentResponse(c CommentRecord) map[string]interface{} {
	resp := map[string]interface{}{
		"commentId": c.CommentID,
		"postId":    c.PostID,
		"author": map[string]interface{}{
			"userId": c.AuthorUserID,
			"name":   c.AuthorName,
		},
		"content":         c.Content,
		"parentCommentId": nilIfEmptyStr(c.ParentCommentID),
		"likeCount":       c.LikeCount,
		"createdAt":       c.CreatedAt,
		"updatedAt":       c.UpdatedAt,
	}
	return resp
}

func nilIfEmptyStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
