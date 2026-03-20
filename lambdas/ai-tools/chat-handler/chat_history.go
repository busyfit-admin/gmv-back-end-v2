package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// loadChatHistory fetches the most recent `limit` messages for the given chatId,
// returning them in chronological order (oldest first). Messages are stored in DDB
// ordered by SK (ascending — oldest first). Returns Bedrock-ready Message structs.
func loadChatHistory(ctx context.Context, ddb *dynamodb.Client, table, chatId string, limit int32) ([]bedrocktypes.Message, error) {
	out, err := ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(table),
		KeyConditionExpression: aws.String("chatId = :cid"),
		ExpressionAttributeValues: map[string]ddbTypes.AttributeValue{
			":cid": &ddbTypes.AttributeValueMemberS{Value: chatId},
		},
		ScanIndexForward: aws.Bool(false), // newest first
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("loadChatHistory: query failed: %w", err)
	}

	records := make([]chatHistoryRecord, 0, len(out.Items))
	for _, item := range out.Items {
		var r chatHistoryRecord
		if err := attributevalue.UnmarshalMap(item, &r); err != nil {
			continue
		}
		records = append(records, r)
	}

	// Reverse to chronological order (oldest first) for Bedrock.
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	messages := make([]bedrocktypes.Message, 0, len(records))
	for _, r := range records {
		role := bedrocktypes.ConversationRoleUser
		if r.Role == "assistant" {
			role = bedrocktypes.ConversationRoleAssistant
		}
		messages = append(messages, bedrocktypes.Message{
			Role: role,
			Content: []bedrocktypes.ContentBlock{
				&bedrocktypes.ContentBlockMemberText{Value: r.MessageText},
			},
		})
	}
	return messages, nil
}

// saveChatTurn writes one user message record and one assistant message record
// to the DynamoDB chat history table. Both records receive a 6-month TTL.
func saveChatTurn(ctx context.Context, ddb *dynamodb.Client, table, chatId, userId, userMsg, assistantMsg string) error {
	now := time.Now().UTC()
	expiry := now.Unix() + chatTTLSeconds

	writeItems := []ddbTypes.TransactWriteItem{
		{
			Put: &ddbTypes.Put{
				TableName: aws.String(table),
				Item: mustMarshalRecord(chatHistoryRecord{
					ChatID:      chatId,
					MsgKey:      fmt.Sprintf("%020d#%s", now.UnixMilli(), uuid.NewString()),
					Role:        "user",
					MessageText: userMsg,
					UserID:      userId,
					CreatedAt:   now.Format(time.RFC3339),
					ExpiresAt:   expiry,
				}),
			},
		},
		{
			Put: &ddbTypes.Put{
				TableName: aws.String(table),
				Item: mustMarshalRecord(chatHistoryRecord{
					ChatID:      chatId,
					MsgKey:      fmt.Sprintf("%020d#%s", now.UnixMilli()+1, uuid.NewString()),
					Role:        "assistant",
					MessageText: assistantMsg,
					UserID:      userId,
					CreatedAt:   now.Format(time.RFC3339),
					ExpiresAt:   expiry,
				}),
			},
		},
	}

	_, err := ddb.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: writeItems,
	})
	return err
}

// mustMarshalRecord marshals a chatHistoryRecord to a DDB attribute map.
// Panics only if the hardcoded struct is invalid, which cannot happen at runtime.
func mustMarshalRecord(r chatHistoryRecord) map[string]ddbTypes.AttributeValue {
	item, err := attributevalue.MarshalMap(r)
	if err != nil {
		panic(fmt.Sprintf("mustMarshalRecord: %v", err))
	}
	return item
}
