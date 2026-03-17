package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	bedrock "github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/google/uuid"
)

// maxToolIterations caps the Bedrock tool-use loop to prevent runaway recursion.
const maxToolIterations = 5

// historyLimit is how many past messages (user+assistant pairs) to load per request.
const historyLimit = 20

// Handle is the Lambda entry point. It parses the HTTP request, runs the
// Bedrock converse loop, persists the conversation turn, and returns a response.
func (svc *Service) Handle(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx, seg := xray.BeginSegment(ctx, "ai-chat")
	defer seg.Close(nil)

	// --- 1. Parse request ---
	var req ChatRequest
	if err := json.Unmarshal([]byte(request.Body), &req); err != nil {
		return errResponse(http.StatusBadRequest, "invalid request body")
	}
	if strings.TrimSpace(req.Message) == "" {
		return errResponse(http.StatusBadRequest, "message is required")
	}
	if req.ChatID == "" {
		req.ChatID = uuid.NewString()
	}

	// --- 2. Resolve caller identity ---
	cognitoID, err := getCognitoIDFromRequest(request)
	if err != nil {
		return errResponse(http.StatusUnauthorized, "missing authentication")
	}

	emp, err := svc.ctrlSVC.FindEmployeeByCognitoId(ctx, cognitoID)
	if err != nil {
		svc.logger.Printf("warn: could not resolve employee for cognitoId=%q: %v", cognitoID, err)
	}

	chatCtx := ChatContext{
		CallerCognitoID:   cognitoID,
		CallerUserName:    emp.UserName,
		CallerDisplayName: emp.DisplayName,
		CallerTeamID:      req.Context.TeamID,
		CallerOrgID:       req.Context.OrgID,
		TargetUserID:      req.Context.TargetUserID,
	}

	// --- 3. Load conversation history ---
	history, err := loadChatHistory(ctx, svc.ddb, svc.chatHistoryTable, req.ChatID, historyLimit)
	if err != nil {
		svc.logger.Printf("warn: could not load chat history chatId=%q: %v", req.ChatID, err)
		history = nil
	}

	// --- 4. Append new user message ---
	messages := make([]bedrocktypes.Message, len(history))
	copy(messages, history)
	messages = append(messages, bedrocktypes.Message{
		Role: bedrocktypes.ConversationRoleUser,
		Content: []bedrocktypes.ContentBlock{
			&bedrocktypes.ContentBlockMemberText{Value: req.Message},
		},
	})

	// --- 5. Run Bedrock converse loop ---
	finalText, toolsUsed, err := svc.converseWithTools(ctx, messages, chatCtx)
	if err != nil {
		svc.logger.Printf("error: bedrock converse failed chatId=%q: %v", req.ChatID, err)
		return errResponse(http.StatusInternalServerError, "AI service error")
	}

	// --- 6. Persist conversation turn ---
	if err := saveChatTurn(ctx, svc.ddb, svc.chatHistoryTable, req.ChatID, cognitoID, req.Message, finalText); err != nil {
		svc.logger.Printf("warn: could not save chat turn chatId=%q: %v", req.ChatID, err)
		// non-fatal — response is still returned
	}

	// --- 7. Return response ---
	resp := ChatResponse{
		ChatID:    req.ChatID,
		Response:  finalText,
		ToolsUsed: toolsUsed,
	}
	body, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}

// converseWithTools executes the Bedrock Converse API in a tool-use loop.
// It appends tool results back into the conversation and recurses until
// stop_reason is "end_turn" or the iteration cap is reached.
func (svc *Service) converseWithTools(
	ctx context.Context,
	messages []bedrocktypes.Message,
	chatCtx ChatContext,
) (finalText string, toolsUsed []string, err error) {
	tools := buildToolList()
	systemPrompt := buildSystemPrompt(chatCtx)

	for i := 0; i < maxToolIterations; i++ {
		output, err := svc.bedrockClient.Converse(ctx, &bedrock.ConverseInput{
			ModelId: aws.String(svc.modelID),
			System: []bedrocktypes.SystemContentBlock{
				&bedrocktypes.SystemContentBlockMemberText{Value: systemPrompt},
			},
			Messages: messages,
			ToolConfig: &bedrocktypes.ToolConfiguration{
				Tools: tools,
			},
		})
		if err != nil {
			return "", toolsUsed, fmt.Errorf("Converse call %d failed: %w", i+1, err)
		}

		msgOutput, ok := output.Output.(*bedrocktypes.ConverseOutputMemberMessage)
		if !ok {
			return "", toolsUsed, fmt.Errorf("unexpected Converse output type")
		}
		assistantMsg := msgOutput.Value

		switch output.StopReason {
		case bedrocktypes.StopReasonEndTurn:
			// Extract the text from the final assistant message.
			for _, block := range assistantMsg.Content {
				if txt, ok := block.(*bedrocktypes.ContentBlockMemberText); ok {
					return txt.Value, toolsUsed, nil
				}
			}
			return "", toolsUsed, nil

		case bedrocktypes.StopReasonToolUse:
			// Append the assistant message (with tool_use blocks) to the conversation.
			messages = append(messages, bedrocktypes.Message{
				Role:    bedrocktypes.ConversationRoleAssistant,
				Content: assistantMsg.Content,
			})

			// Execute each tool and collect results.
			toolResults := make([]bedrocktypes.ContentBlock, 0)
			for _, block := range assistantMsg.Content {
				toolUse, ok := block.(*bedrocktypes.ContentBlockMemberToolUse)
				if !ok {
					continue
				}
				toolName := aws.ToString(toolUse.Value.Name)
				toolsUsed = append(toolsUsed, toolName)
				svc.logger.Printf("tool_use: %s", toolName)

				resultText, execErr := executeToolCall(ctx, toolName, toolUse.Value.Input, svc.ctrlSVC, chatCtx)
				if execErr != nil {
					svc.logger.Printf("tool %s error: %v", toolName, execErr)
					resultText = fmt.Sprintf(`{"error":"%v"}`, execErr)
				}

				toolResults = append(toolResults, &bedrocktypes.ContentBlockMemberToolResult{
					Value: bedrocktypes.ToolResultBlock{
						ToolUseId: toolUse.Value.ToolUseId,
						Content: []bedrocktypes.ToolResultContentBlock{
							&bedrocktypes.ToolResultContentBlockMemberText{Value: resultText},
						},
					},
				})
			}

			// Append tool results as a user message and re-enter the loop.
			messages = append(messages, bedrocktypes.Message{
				Role:    bedrocktypes.ConversationRoleUser,
				Content: toolResults,
			})

		default:
			// max_tokens, stop_sequence, etc. — try to extract whatever text we have.
			for _, block := range assistantMsg.Content {
				if txt, ok := block.(*bedrocktypes.ContentBlockMemberText); ok {
					return txt.Value, toolsUsed, nil
				}
			}
			return "", toolsUsed, nil
		}
	}

	return "", toolsUsed, fmt.Errorf("reached maximum tool iterations (%d)", maxToolIterations)
}

// buildSystemPrompt constructs the system prompt injecting the caller's identity
// and current date for temporal context.
func buildSystemPrompt(ctx ChatContext) string {
	var sb strings.Builder
	sb.WriteString("You are an AI performance management assistant for a SaaS platform. ")
	sb.WriteString("You help employees, managers and admins understand performance data, track goals, and gain insights.\n\n")
	sb.WriteString("Use the provided tools to retrieve accurate, real-time data before answering. ")
	sb.WriteString("Never invent IDs, names or statistics — always fetch them via tools.\n\n")

	sb.WriteString(fmt.Sprintf("Current user: %s", ctx.CallerDisplayName))
	if ctx.CallerUserName != "" {
		sb.WriteString(fmt.Sprintf(" (username: %s)", ctx.CallerUserName))
	}
	sb.WriteRune('\n')
	if ctx.CallerTeamID != "" {
		sb.WriteString(fmt.Sprintf("Current team: %s\n", ctx.CallerTeamID))
	}
	if ctx.CallerOrgID != "" {
		sb.WriteString(fmt.Sprintf("Current organisation: %s\n", ctx.CallerOrgID))
	}
	if ctx.TargetUserID != "" {
		sb.WriteString(fmt.Sprintf("Focus member (manager/admin context): %s\n", ctx.TargetUserID))
	}
	sb.WriteString(fmt.Sprintf("Current date (UTC): %s\n", time.Now().UTC().Format("2006-01-02")))
	return sb.String()
}

// getCognitoIDFromRequest extracts the Cognito sub from Authorizer claims
// or falls back to the X-Cognito-Id header (used in local / testing environments).
func getCognitoIDFromRequest(request events.APIGatewayProxyRequest) (string, error) {
	if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		if sub, ok := claims["sub"].(string); ok && sub != "" {
			return sub, nil
		}
	}
	if id := request.Headers["X-Cognito-Id"]; id != "" {
		return id, nil
	}
	return "", fmt.Errorf("cognito ID not found in request")
}

// errResponse is a convenience helper that returns a JSON error body.
func errResponse(statusCode int, message string) (events.APIGatewayProxyResponse, error) {
	body, _ := json.Marshal(map[string]string{"error": message})
	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(body),
	}, nil
}
