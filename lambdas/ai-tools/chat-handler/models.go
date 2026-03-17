// Package main implements the AI chat handler Lambda.
// It receives chat messages from the frontend, loads conversation history
// from DynamoDB, invokes Claude via Bedrock Converse API (with tool use),
// and returns the assistant's response.
package main

// ChatRequest is the JSON body expected at POST /v2/ai/chat.
type ChatRequest struct {
	// ChatID identifies an ongoing conversation. If empty, a new UUID is generated.
	ChatID string `json:"chatId"`
	// Message is the user's natural-language input.
	Message string `json:"message"`
	// Context provides optional frontend-supplied hints about the calling user's scope.
	Context ChatContextInput `json:"context"`
}

// ChatContextInput is the "context" object inside ChatRequest.
type ChatContextInput struct {
	TeamID string `json:"teamId"`
	OrgID  string `json:"orgId"`
	// TargetUserID lets managers/admins scope queries to a specific member.
	TargetUserID string `json:"targetUserId"`
}

// ChatResponse is returned on success.
type ChatResponse struct {
	ChatID    string   `json:"chatId"`
	Response  string   `json:"response"`
	ToolsUsed []string `json:"toolsUsed"`
}

// ChatContext holds resolved runtime context about the caller. It is derived
// from Cognito claims + company DB and is passed to every tool executor.
type ChatContext struct {
	CallerCognitoID   string
	CallerUserName    string
	CallerDisplayName string
	CallerTeamID      string
	CallerOrgID       string
	TargetUserID      string
}

// chatHistoryRecord is the DynamoDB item shape for chat history.
// PK = chatId, SK = {epoch_millis_padded}#{uuid}.
type chatHistoryRecord struct {
	ChatID      string `dynamodbav:"chatId"`
	MsgKey      string `dynamodbav:"msgKey"`
	Role        string `dynamodbav:"role"` // "user" | "assistant"
	MessageText string `dynamodbav:"messageText"`
	UserID      string `dynamodbav:"userId"`
	CreatedAt   string `dynamodbav:"createdAt"`
	ExpiresAt   int64  `dynamodbav:"expiresAt"` // Unix seconds TTL
}

// chatTTLSeconds is the TTL duration applied to every chat history record (6 months).
const chatTTLSeconds = 6 * 30 * 24 * 60 * 60 // 15 552 000
