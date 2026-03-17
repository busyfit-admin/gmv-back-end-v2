// Package controllers provides a typed Go function library for reading and writing
// performance management data. Intended to be called directly by an AI agent
// rather than through HTTP APIs.
package controllers

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

// Service is the entry point for all AI-tool controllers.
// It holds every dependency needed to service queries across employees,
// teams, organisations, org-level performance, and the per-user performance hub.
type Service struct {
	ctx    context.Context
	logger *log.Logger

	// company-lib service wrappers
	empSVC   *companylib.EmployeeService
	teamsSVC *companylib.TeamsServiceV2
	orgSVC   *companylib.OrgServiceV2
	perfSVC  *companylib.PerformanceService

	// raw DynamoDB client for direct performance-hub table queries
	ddb          *dynamodb.Client
	perfHubTable string
}

// NewService initialises all AWS clients and companylib services from environment variables.
//
// Required environment variables:
//
//	PERF_HUB_TABLE                    — DynamoDB table for user performance hub data
//	ORG_PERFORMANCE_TABLE             — DynamoDB table for org-level performance data
//	ORGANIZATION_TABLE                — DynamoDB table for organisation records
//	EMPLOYEE_TABLE                    — DynamoDB table for employee records
//	EMPLOYEE_TABLE_COGNITO_ID_INDEX   — GSI name for Cognito ID lookups
//	EMPLOYEE_TABLE_EMAIL_ID_INDEX     — GSI name for email lookups
//	TEAMS_TABLE                       — DynamoDB table for team records
func NewService() (*Service, error) {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("ai-tools: cannot load AWS config: %w", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "[ai-tools] ", log.LstdFlags)
	ddbClient := dynamodb.NewFromConfig(cfg)

	// Employee service
	empSVC := companylib.CreateEmployeeService(context.Background(), ddbClient, nil, logger)
	empSVC.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	empSVC.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")
	empSVC.EmployeeTable_EmailId_Index = os.Getenv("EMPLOYEE_TABLE_EMAIL_ID_INDEX")

	// Teams service
	teamsSVC := companylib.CreateTeamsServiceV2(context.Background(), ddbClient, logger, empSVC, nil)
	teamsSVC.TeamsTable = os.Getenv("TEAMS_TABLE")

	// Org service
	orgSVC := companylib.CreateOrgServiceV2(context.Background(), ddbClient, logger, empSVC, nil)
	orgSVC.OrganizationTable = os.Getenv("ORGANIZATION_TABLE")

	// Org-performance service
	perfSVC := companylib.CreatePerformanceService(context.Background(), ddbClient, logger)
	perfSVC.OrgPerformanceTable = os.Getenv("ORG_PERFORMANCE_TABLE")
	perfSVC.OrganizationTable = os.Getenv("ORGANIZATION_TABLE")

	return &Service{
		ctx:          context.Background(),
		logger:       logger,
		empSVC:       empSVC,
		teamsSVC:     teamsSVC,
		orgSVC:       orgSVC,
		perfSVC:      perfSVC,
		ddb:          ddbClient,
		perfHubTable: os.Getenv("PERF_HUB_TABLE"),
	}, nil
}

// PropagateContext updates the internal context on every sub-service so that
// AWS SDK calls (including X-Ray subsegments) use the request-scoped context.
// Call this once at the top of each Lambda invocation.
func (s *Service) PropagateContext(ctx context.Context) {
	s.ctx = ctx
	s.empSVC.SetContext(ctx)
	s.teamsSVC.SetContext(ctx)
	s.orgSVC.SetContext(ctx)
	s.perfSVC.SetContext(ctx)
}
