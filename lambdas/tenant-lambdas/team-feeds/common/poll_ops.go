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

// ==================== Route: Poll ====================
//
// POST   /v2/posts/{postId}/poll/vote
// DELETE /v2/posts/{postId}/poll/vote
// GET    /v2/posts/{postId}/poll/results

func (svc *Service) handlePoll(request events.APIGatewayProxyRequest, parts []string, userName, cognitoID string) (events.APIGatewayProxyResponse, error) {
	// /v2/posts/{postId}/poll/vote  (5 parts: v2, posts, {postId}, poll, vote)
	if len(parts) == 5 && parts[1] == "posts" && parts[3] == "poll" && parts[4] == "vote" {
		postID := parts[2]
		if err := svc.ensurePostTeamMember(postID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		switch request.HTTPMethod {
		case "POST":
			return svc.castVote(postID, userName, request.Body)
		case "DELETE":
			return svc.retractVote(postID, userName)
		}
	}

	// /v2/posts/{postId}/poll/results  (5 parts)
	if len(parts) == 5 && parts[1] == "posts" && parts[3] == "poll" && parts[4] == "results" {
		postID := parts[2]
		if err := svc.ensurePostTeamMember(postID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		if request.HTTPMethod == "GET" {
			return svc.getPollResults(postID, userName)
		}
	}

	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== Cast / Change Vote ====================

func (svc *Service) castVote(postID, userName, body string) (events.APIGatewayProxyResponse, error) {
	post, err := svc.fetchPostRecord(postID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
	}
	if post.Type != string(PostTypePoll) {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "This post is not a poll")
	}

	req, err := parseBody[CastVoteRequest](body)
	if err != nil || req.OptionID == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "optionId is required")
	}

	// Validate option exists
	optionValid := false
	for _, o := range post.Data.PollOptions {
		if o.OptionID == req.OptionID {
			optionValid = true
			break
		}
	}
	if !optionValid {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid optionId")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	record := VoteRecord{
		PK:       PrefixPost + postID,
		SK:       SKVotePrefix + userName,
		UserID:   userName,
		OptionID: req.OptionID,
		VotedAt:  now,
	}
	item, _ := attributevalue.MarshalMap(record)

	// Upsert — replaces existing vote
	_, err = svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.feedTable),
		Item:      item,
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to cast vote")
	}

	return svc.okResp(map[string]interface{}{
		"postId":        postID,
		"votedOptionId": req.OptionID,
		"votedAt":       now,
	}, nil)
}

// ==================== Retract Vote ====================

func (svc *Service) retractVote(postID, userName string) (events.APIGatewayProxyResponse, error) {
	_, err := svc.ddb.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKVotePrefix + userName},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to retract vote")
	}
	return svc.okResp(map[string]interface{}{"postId": postID, "retracted": true}, nil)
}

// ==================== Get Poll Results ====================

func (svc *Service) getPollResults(postID, userName string) (events.APIGatewayProxyResponse, error) {
	post, err := svc.fetchPostRecord(postID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
	}
	if post.Type != string(PostTypePoll) {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "This post is not a poll")
	}

	// Fetch all votes
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.feedTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: PrefixPost + postID},
			":prefix": &types.AttributeValueMemberS{Value: SKVotePrefix},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch votes")
	}

	var votes []VoteRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &votes)

	// Build per-option counts
	voteCounts := make(map[string]int)
	var userVotedOptionID *string
	for _, v := range votes {
		voteCounts[v.OptionID]++
		if v.UserID == userName {
			oid := v.OptionID
			userVotedOptionID = &oid
		}
	}

	opts := make([]map[string]interface{}, 0, len(post.Data.PollOptions))
	for _, o := range post.Data.PollOptions {
		cnt := voteCounts[o.OptionID]
		opts = append(opts, map[string]interface{}{
			"optionId": o.OptionID,
			"text":     o.Text,
			"votes":    cnt,
		})
	}

	totalVotes := len(votes)
	var userVotedStr interface{} = nil
	if userVotedOptionID != nil {
		userVotedStr = *userVotedOptionID
	}

	return svc.okResp(map[string]interface{}{
		"postId":            postID,
		"question":          post.Data.PollQuestion,
		"options":           opts,
		"totalVotes":        totalVotes,
		"userVotedOptionId": userVotedStr,
	}, &MetaResponse{Total: totalVotes, Page: 1, Limit: totalVotes + 1})
}
