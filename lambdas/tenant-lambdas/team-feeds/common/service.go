package common

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

// ==================== Route Group Constants ====================

const (
	RouteGroupPosts     = "posts"
	RouteGroupLikes     = "likes"
	RouteGroupComments  = "comments"
	RouteGroupPoll      = "poll"
	RouteGroupChecklist = "checklist"
	RouteGroupTask      = "task"
)

// ==================== Service Struct ====================

type Service struct {
	ctx       context.Context
	logger    *log.Logger
	empSVC    *companylib.EmployeeService
	teamsSVC  *companylib.TeamsServiceV2
	ddb       *dynamodb.Client
	feedTable string
}

var RESP_HEADERS = companylib.GetHeadersForAPI("TeamFeedsAPI")

// ==================== NewService ====================

func NewService() (*Service, error) {
	ctx, root := xray.BeginSegment(context.TODO(), "manage-team-feeds")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load config: %w", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbClient := dynamodb.NewFromConfig(cfg)
	sesClient := ses.NewFromConfig(cfg)

	empSvc := companylib.CreateEmployeeService(ctx, ddbClient, nil, logger)
	empSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	empSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")
	empSvc.EmployeeTable_EmailId_Index = os.Getenv("EMPLOYEE_TABLE_EMAIL_ID_INDEX")

	emailSvc := companylib.CreateEmailService(ctx, sesClient, logger)

	teamsSvc := companylib.CreateTeamsServiceV2(ctx, ddbClient, logger, empSvc, emailSvc)
	teamsSvc.TeamsTable = os.Getenv("TEAMS_TABLE")

	return &Service{
		ctx:       ctx,
		logger:    logger,
		empSVC:    empSvc,
		teamsSVC:  teamsSvc,
		ddb:       ddbClient,
		feedTable: os.Getenv("TEAM_FEED_TABLE"),
	}, nil
}
