package common

import (
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ==================== Route: Likes ====================
//
// POST   /v2/posts/{postId}/likes
// DELETE /v2/posts/{postId}/likes
// GET    /v2/posts/{postId}/likes

func (svc *Service) handleLikes(request events.APIGatewayProxyRequest, parts []string, userName, cognitoID string) (events.APIGatewayProxyResponse, error) {
	// /v2/posts/{postId}/likes  (4 parts: v2, posts, {postId}, likes)
	if len(parts) == 4 && parts[1] == "posts" && parts[3] == "likes" {
		postID := parts[2]
		if err := svc.ensurePostTeamMember(postID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		switch request.HTTPMethod {
		case "POST":
			return svc.likePost(postID, userName)
		case "DELETE":
			return svc.unlikePost(postID, userName)
		case "GET":
			return svc.getPostLikes(postID)
		}
	}

	// /v2/posts/{postId}/comments/{commentId}/likes  (6 parts)
	if len(parts) == 6 && parts[1] == "posts" && parts[3] == "comments" && parts[5] == "likes" {
		postID := parts[2]
		commentID := parts[4]
		if err := svc.ensurePostTeamMember(postID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		switch request.HTTPMethod {
		case "POST":
			return svc.likeComment(commentID, userName)
		case "DELETE":
			return svc.unlikeComment(commentID, userName)
		}
	}

	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== Like Post ====================

func (svc *Service) likePost(postID, userName string) (events.APIGatewayProxyResponse, error) {
	// Idempotent — check if already liked
	existing, _ := svc.userHasLikedPost(postID, userName)
	if existing {
		return svc.okResp(map[string]interface{}{"liked": true}, nil)
	}

	record := LikeRecord{
		PK:      PrefixPost + postID,
		SK:      SKLikePrefix + userName,
		UserID:  userName,
		LikedAt: time.Now().UTC().Format(time.RFC3339),
	}
	item, _ := attributevalue.MarshalMap(record)
	_, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.feedTable),
		Item:      item,
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to like post")
	}

	// Increment like count atomically
	svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKMetadata},
		},
		UpdateExpression: aws.String("ADD likeCount :inc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{Value: "1"},
		},
	})

	return svc.okResp(map[string]interface{}{"liked": true}, nil)
}

// ==================== Unlike Post ====================

func (svc *Service) unlikePost(postID, userName string) (events.APIGatewayProxyResponse, error) {
	_, err := svc.ddb.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKLikePrefix + userName},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unlike post")
	}

	// Decrement like count atomically (min 0)
	svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKMetadata},
		},
		UpdateExpression:    aws.String("ADD likeCount :dec"),
		ConditionExpression: aws.String("likeCount > :zero"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":dec":  &types.AttributeValueMemberN{Value: "-1"},
			":zero": &types.AttributeValueMemberN{Value: "0"},
		},
	})

	return svc.noContentResp()
}

// ==================== Get Post Likes ====================

func (svc *Service) getPostLikes(postID string) (events.APIGatewayProxyResponse, error) {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.feedTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: PrefixPost + postID},
			":prefix": &types.AttributeValueMemberS{Value: SKLikePrefix},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch likes")
	}

	var records []LikeRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &records)

	users := make([]map[string]interface{}, 0, len(records))
	for _, r := range records {
		users = append(users, map[string]interface{}{
			"userId":  r.UserID,
			"likedAt": r.LikedAt,
		})
	}
	return svc.okResp(users, &MetaResponse{Total: len(users), Page: 1, Limit: len(users)})
}

// ==================== Like Comment ====================

func (svc *Service) likeComment(commentID, userName string) (events.APIGatewayProxyResponse, error) {
	record := LikeRecord{
		PK:      PrefixComment + commentID,
		SK:      SKLikePrefix + userName,
		UserID:  userName,
		LikedAt: time.Now().UTC().Format(time.RFC3339),
	}
	item, _ := attributevalue.MarshalMap(record)
	_, err := svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(svc.feedTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PK)"), // idempotent
	})
	if err != nil {
		// Ignore ConditionalCheckFailedException (already liked)
		return svc.okResp(map[string]interface{}{"liked": true}, nil)
	}

	// Increment comment like count
	svc.ddb.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"GSI1PK": &types.AttributeValueMemberS{Value: PrefixComment + commentID},
			"GSI1SK": &types.AttributeValueMemberS{Value: "META"},
		},
		UpdateExpression: aws.String("ADD likeCount :inc"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc": &types.AttributeValueMemberN{Value: "1"},
		},
	})

	return svc.okResp(map[string]interface{}{"liked": true}, nil)
}

// ==================== Unlike Comment ====================

func (svc *Service) unlikeComment(commentID, userName string) (events.APIGatewayProxyResponse, error) {
	_, err := svc.ddb.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixComment + commentID},
			"SK": &types.AttributeValueMemberS{Value: SKLikePrefix + userName},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to unlike comment")
	}
	return svc.noContentResp()
}
