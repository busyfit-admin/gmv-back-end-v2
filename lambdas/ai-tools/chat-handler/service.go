package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"

	ctrl "github.com/busyfit-admin/saas-integrated-apis/lambdas/ai-tools/controllers"
)

// Service holds every dependency needed by the chat-handler Lambda.
type Service struct {
	logger           *log.Logger
	ctrlSVC          *ctrl.Service
	bedrockClient    *bedrockruntime.Client
	ddb              *dynamodb.Client
	chatHistoryTable string
	modelID          string
}

// NewService initialises all AWS clients and controller services from
// environment variables.
//
// Required environment variables:
//
//	AI_CHAT_HISTORY_TABLE  — DynamoDB table for chat history
//	BEDROCK_MODEL_ID       — Bedrock model ID (e.g. anthropic.claude-3-5-sonnet-20241022-v2:0)
//	EMPLOYEE_TABLE, EMPLOYEE_TABLE_COGNITO_ID_INDEX, EMPLOYEE_TABLE_EMAIL_ID_INDEX
//	TEAMS_TABLE, ORGANIZATION_TABLE, ORG_PERFORMANCE_TABLE, PERF_HUB_TABLE
func NewService() (*Service, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("NewService: load AWS config: %w", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "[ai-chat] ", log.LstdFlags)

	ctrlSVC, err := ctrl.NewService()
	if err != nil {
		return nil, fmt.Errorf("NewService: init controllers: %w", err)
	}

	modelID := os.Getenv("BEDROCK_MODEL_ID")
	if modelID == "" {
		modelID = "amazon.nova-pro-v1:0"
	}

	return &Service{
		logger:           logger,
		ctrlSVC:          ctrlSVC,
		bedrockClient:    bedrockruntime.NewFromConfig(cfg),
		ddb:              dynamodb.NewFromConfig(cfg),
		chatHistoryTable: os.Getenv("AI_CHAT_HISTORY_TABLE"),
		modelID:          modelID,
	}, nil
}
