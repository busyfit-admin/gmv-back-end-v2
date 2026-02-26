package common

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
	"github.com/google/uuid"
)

// ==================== Route: Posts ====================
//
// GET  /v2/teams/{teamId}/feed
// POST /v2/teams/{teamId}/posts
// GET  /v2/teams/{teamId}/posts/{postId}
// PUT  /v2/teams/{teamId}/posts/{postId}
// DELETE /v2/teams/{teamId}/posts/{postId}

func (svc *Service) handlePosts(request events.APIGatewayProxyRequest, parts []string, userName, cognitoID string) (events.APIGatewayProxyResponse, error) {
	// /v2/teams/{teamId}/feed  (5 parts: v2, teams, {teamId}, feed)
	if len(parts) == 4 && parts[1] == "teams" && parts[3] == "feed" && request.HTTPMethod == "GET" {
		teamID := parts[2]
		if err := svc.ensureTeamMember(teamID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		return svc.listTeamFeed(teamID, userName, request.QueryStringParameters)
	}

	// /v2/teams/{teamId}/posts  (4 parts: v2, teams, {teamId}, posts)
	if len(parts) == 4 && parts[1] == "teams" && parts[3] == "posts" {
		teamID := parts[2]
		if err := svc.ensureTeamMember(teamID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		if request.HTTPMethod == "POST" {
			return svc.createPost(teamID, userName, cognitoID, request.Body)
		}
	}

	// /v2/teams/{teamId}/posts/{postId}  (5 parts: v2, teams, {teamId}, posts, {postId})
	if len(parts) == 5 && parts[1] == "teams" && parts[3] == "posts" {
		teamID := parts[2]
		postID := parts[4]
		if err := svc.ensureTeamMember(teamID, userName); err != nil {
			return svc.errResp(http.StatusForbidden, "FORBIDDEN", "You are not a member of this team")
		}
		switch request.HTTPMethod {
		case "GET":
			return svc.getPost(postID, userName)
		case "PUT":
			return svc.updatePost(teamID, postID, userName, request.Body)
		case "DELETE":
			return svc.deletePost(teamID, postID, userName)
		}
	}

	return svc.errResp(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
}

// ==================== List Team Feed ====================

func (svc *Service) listTeamFeed(teamID, userName string, queryParams map[string]string) (events.APIGatewayProxyResponse, error) {
	page := queryInt(queryParams, "page", 1)
	limit := queryInt(queryParams, "limit", 20)
	typeFilter := queryString(queryParams, "type")

	gsi1pk := PrefixTeam + teamID

	input := &dynamodb.QueryInput{
		TableName:              aws.String(svc.feedTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :gsi1pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk": &types.AttributeValueMemberS{Value: gsi1pk},
		},
		ScanIndexForward: aws.Bool(false), // newest first
	}

	if typeFilter != "" {
		input.FilterExpression = aws.String("#postType = :ptype")
		input.ExpressionAttributeNames = map[string]string{"#postType": "type"}
		input.ExpressionAttributeValues[":ptype"] = &types.AttributeValueMemberS{Value: typeFilter}
	}

	result, err := svc.ddb.Query(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Error querying feed: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list feed")
	}

	var records []PostRecord
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &records); err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to parse feed items")
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
	page_records := records[start:end]

	posts := make([]map[string]interface{}, 0, len(page_records))
	for _, r := range page_records {
		posts = append(posts, svc.buildPostResponse(r, userName, nil, nil))
	}

	return svc.okResp(posts, &MetaResponse{Total: total, Page: page, Limit: limit})
}

// ==================== Create Post ====================

func (svc *Service) createPost(teamID, userName, cognitoID, body string) (events.APIGatewayProxyResponse, error) {
	req, err := parseBody[CreatePostRequest](body)
	if err != nil {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
	}
	if req.Type == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Post type is required")
	}

	employee, err := svc.empSVC.GetEmployeeDataByCognitoId(cognitoID)
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch author data")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	postID := uuid.New().String()

	record := PostRecord{
		PK:           PrefixPost + postID,
		SK:           SKMetadata,
		GSI1PK:       PrefixTeam + teamID,
		GSI1SK:       now + "#" + postID,
		PostID:       postID,
		TeamID:       teamID,
		Type:         string(req.Type),
		AuthorUserID: userName,
		AuthorName:   employee.FirstName + " " + employee.LastName,
		Content:      req.Content,
		Tags:         req.Tags,
		LikeCount:    0,
		CommentCount: 0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// Populate type-specific fields
	switch req.Type {
	case PostTypeKudos:
		if req.RecipientUserID == "" {
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "recipientUserId is required for kudos posts")
		}
		record.KudosRecipientUserID = req.RecipientUserID
		if recip, err2 := svc.empSVC.GetEmployeeDataByUserName(req.RecipientUserID); err2 == nil {
			record.KudosRecipientName = recip.FirstName + " " + recip.LastName
		}

	case PostTypeTask:
		if req.TaskSummary == "" {
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "taskSummary is required for task posts")
		}
		record.TaskNumber = fmt.Sprintf("TASK-%s", strings.ToUpper(postID[:6]))
		record.TaskSummary = req.TaskSummary
		record.TaskDesc = req.TaskDescription
		record.AssigneeUserID = req.AssigneeUserID
		record.DueDate = req.DueDate
		record.Urgency = req.Urgency
		record.TaskStatus = "todo"
		if req.AssigneeUserID != "" {
			if assignee, err2 := svc.empSVC.GetEmployeeDataByUserName(req.AssigneeUserID); err2 == nil {
				record.AssigneeName = assignee.FirstName + " " + assignee.LastName
			}
		}

	case PostTypePoll:
		if req.Question == "" || len(req.Options) < 2 {
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Poll requires a question and at least 2 options")
		}
		opts := make([]PollOption, len(req.Options))
		for i, o := range req.Options {
			opts[i] = PollOption{OptionID: uuid.New().String(), Text: o.Text}
		}
		record.PollQuestion = req.Question
		record.PollOptions = opts

	case PostTypeChecklist:
		if req.Title == "" {
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "title is required for checklist posts")
		}
		record.ChecklistTitle = req.Title
		record.IsRecurring = req.IsRecurring
		record.RecurringFrequency = req.RecurringFrequency

	case PostTypeEvent:
		if req.Title == "" || req.EventDate == "" {
			return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "title and eventDate are required for event posts")
		}
		record.EventTitle = req.Title
		record.EventDate = req.EventDate
		record.EventTime = req.EventTime
		record.Location = req.Location
	}

	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to marshal post")
	}

	_, err = svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.feedTable),
		Item:      item,
	})
	if err != nil {
		svc.logger.Printf("Error creating post: %v", err)
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create post")
	}

	// If checklist post, store items as separate records
	if req.Type == PostTypeChecklist && len(req.Items) > 0 {
		for _, item := range req.Items {
			itemID := uuid.New().String()
			ci := ChecklistItemRecord{
				PK:        PrefixPost + postID,
				SK:        SKItemPrefix + itemID,
				ItemID:    itemID,
				PostID:    postID,
				Text:      item.Text,
				Completed: false,
				CreatedAt: now,
			}
			if av, err2 := attributevalue.MarshalMap(ci); err2 == nil {
				svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{TableName: aws.String(svc.feedTable), Item: av})
			}
		}
	}

	return svc.createdResp(svc.buildPostResponse(record, userName, nil, nil))
}

// ==================== Get Single Post ====================

func (svc *Service) getPost(postID, userName string) (events.APIGatewayProxyResponse, error) {
	record, err := svc.fetchPostRecord(postID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
	}

	// Fetch comments
	comments, _ := svc.fetchComments(postID, userName)

	// Fetch checklist items if applicable
	var checklistItems []ChecklistItemRecord
	if record.Type == string(PostTypeChecklist) {
		checklistItems, _ = svc.fetchChecklistItems(postID)
	}

	// Check user like
	userHasLiked, _ := svc.userHasLikedPost(postID, userName)

	response := svc.buildPostResponse(*record, userName, comments, checklistItems)
	response["userHasLiked"] = userHasLiked
	response["comments"] = comments

	return svc.okResp(response, nil)
}

// ==================== Update Post ====================

func (svc *Service) updatePost(teamID, postID, userName, body string) (events.APIGatewayProxyResponse, error) {
	record, err := svc.fetchPostRecord(postID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
	}
	if record.AuthorUserID != userName {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", "Only the author can edit this post")
	}

	req, err := parseBody[CreatePostRequest](body)
	if err != nil {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	record.Content = req.Content
	record.Tags = req.Tags
	record.UpdatedAt = now

	switch PostType(record.Type) {
	case PostTypeTask:
		if req.TaskSummary != "" {
			record.TaskSummary = req.TaskSummary
		}
		if req.TaskDescription != "" {
			record.TaskDesc = req.TaskDescription
		}
		if req.DueDate != "" {
			record.DueDate = req.DueDate
		}
		if req.Urgency != "" {
			record.Urgency = req.Urgency
		}
	case PostTypeChecklist:
		if req.Title != "" {
			record.ChecklistTitle = req.Title
		}
		record.IsRecurring = req.IsRecurring
		record.RecurringFrequency = req.RecurringFrequency
	case PostTypeEvent:
		if req.Title != "" {
			record.EventTitle = req.Title
		}
		if req.EventDate != "" {
			record.EventDate = req.EventDate
		}
		if req.EventTime != "" {
			record.EventTime = req.EventTime
		}
		if req.Location != "" {
			record.Location = req.Location
		}
	}

	item, _ := attributevalue.MarshalMap(record)
	_, err = svc.ddb.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.feedTable),
		Item:      item,
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to update post")
	}

	return svc.okResp(svc.buildPostResponse(*record, userName, nil, nil), nil)
}

// ==================== Delete Post ====================

func (svc *Service) deletePost(teamID, postID, userName string) (events.APIGatewayProxyResponse, error) {
	record, err := svc.fetchPostRecord(postID)
	if err != nil {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Post not found")
	}

	if err := svc.ensureTeamAdminOrAuthor(teamID, userName, record.AuthorUserID); err != nil {
		return svc.errResp(http.StatusForbidden, "FORBIDDEN", err.Error())
	}

	_, err = svc.ddb.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKMetadata},
		},
	})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to delete post")
	}

	return svc.noContentResp()
}

// ==================== DDB Helpers ====================

func (svc *Service) fetchPostRecord(postID string) (*PostRecord, error) {
	result, err := svc.ddb.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKMetadata},
		},
	})
	if err != nil || result.Item == nil {
		return nil, fmt.Errorf("post not found: %s", postID)
	}
	var record PostRecord
	if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func (svc *Service) fetchChecklistItems(postID string) ([]ChecklistItemRecord, error) {
	result, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.feedTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: PrefixPost + postID},
			":prefix": &types.AttributeValueMemberS{Value: SKItemPrefix},
		},
	})
	if err != nil {
		return nil, err
	}
	var items []ChecklistItemRecord
	attributevalue.UnmarshalListOfMaps(result.Items, &items)
	return items, nil
}

func (svc *Service) userHasLikedPost(postID, userName string) (bool, error) {
	result, err := svc.ddb.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.feedTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: PrefixPost + postID},
			"SK": &types.AttributeValueMemberS{Value: SKLikePrefix + userName},
		},
	})
	if err != nil {
		return false, err
	}
	return result.Item != nil, nil
}

// ==================== Response Builder ====================

func (svc *Service) buildPostResponse(r PostRecord, userName string, comments interface{}, checklistItems []ChecklistItemRecord) map[string]interface{} {
	userHasLiked, _ := svc.userHasLikedPost(r.PostID, userName)

	resp := map[string]interface{}{
		"postId": r.PostID,
		"teamId": r.TeamID,
		"type":   r.Type,
		"author": map[string]interface{}{
			"userId":     r.AuthorUserID,
			"name":       r.AuthorName,
			"role":       r.AuthorRole,
			"profilePic": nilIfEmpty(r.AuthorProfilePic),
		},
		"content":      nilIfEmpty(r.Content),
		"tags":         r.Tags,
		"likeCount":    r.LikeCount,
		"commentCount": r.CommentCount,
		"userHasLiked": userHasLiked,
		"createdAt":    r.CreatedAt,
		"updatedAt":    r.UpdatedAt,
	}

	switch PostType(r.Type) {
	case PostTypeKudos:
		resp["kudos"] = map[string]interface{}{
			"recipient": map[string]interface{}{
				"userId":     r.KudosRecipientUserID,
				"name":       r.KudosRecipientName,
				"profilePic": nil,
			},
		}

	case PostTypeTask:
		resp["task"] = map[string]interface{}{
			"taskNumber":  r.TaskNumber,
			"summary":     r.TaskSummary,
			"description": r.TaskDesc,
			"assignee": map[string]interface{}{
				"userId": r.AssigneeUserID,
				"name":   r.AssigneeName,
			},
			"dueDate":        r.DueDate,
			"urgency":        r.Urgency,
			"status":         r.TaskStatus,
			"timeSpentHours": r.TimeSpentHours,
		}

	case PostTypePoll:
		opts := make([]map[string]interface{}, 0, len(r.PollOptions))
		for _, o := range r.PollOptions {
			opts = append(opts, map[string]interface{}{
				"optionId": o.OptionID,
				"text":     o.Text,
				"votes":    o.Votes,
			})
		}
		resp["poll"] = map[string]interface{}{
			"question":          r.PollQuestion,
			"options":           opts,
			"totalVotes":        0,
			"userVotedOptionId": nil,
		}

	case PostTypeChecklist:
		items := make([]map[string]interface{}, 0)
		completedCount := 0
		if checklistItems != nil {
			sort.Slice(checklistItems, func(i, j int) bool {
				return checklistItems[i].CreatedAt < checklistItems[j].CreatedAt
			})
			for _, ci := range checklistItems {
				items = append(items, map[string]interface{}{
					"itemId":    ci.ItemID,
					"text":      ci.Text,
					"completed": ci.Completed,
				})
				if ci.Completed {
					completedCount++
				}
			}
		}
		resp["checklist"] = map[string]interface{}{
			"title":              r.ChecklistTitle,
			"isRecurring":        r.IsRecurring,
			"recurringFrequency": r.RecurringFrequency,
			"items":              items,
			"completedCount":     completedCount,
			"totalCount":         len(items),
		}

	case PostTypeEvent:
		resp["event"] = map[string]interface{}{
			"title":     r.EventTitle,
			"eventDate": r.EventDate,
			"eventTime": r.EventTime,
			"location":  r.Location,
		}
	}

	return resp
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
