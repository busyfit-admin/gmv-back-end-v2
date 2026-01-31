/*
This lambda gets the information from the TransferLogs Table to the User
*/
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type RewardTransferLogService struct {
	ctx    context.Context
	logger *log.Logger

	employeeSvc   companylib.EmployeeService
	rewardLogsSvc companylib.RewardsTransferLogsService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("RewardsAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "reward-transfer-logs")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	employeeSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	rewardLogSvc := companylib.CreateRewardsTransferLogsService(ctx, logger, ddbclient)
	rewardLogSvc.RewardsTransferLogsTable = os.Getenv("REWARDS_TRANSFER_LOGS_TABLE")

	svc := RewardTransferLogService{
		ctx:           ctx,
		logger:        logger,
		employeeSvc:   *employeeSvc,
		rewardLogsSvc: *rewardLogSvc,
	}

	lambda.Start(svc.GetRewardTransferLogs)

}

func (svc *RewardTransferLogService) GetRewardTransferLogs(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	switch request.HTTPMethod {
	case "GET":
		return svc.handleGetMethod(request)
	default:
		svc.logger.Printf("Request type not defined for GetRewardTransferLogs: %s", request.HTTPMethod)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}
}

func (svc *RewardTransferLogService) handleGetMethod(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	switch request.Headers["get_type"] {
	case "get_users_reward_logs":
		return svc.GetUserRewardLogs(request)
	case "get_reward_admin_logs":
		return svc.GetAdminRewardLogs(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil

	}

}

func (svc *RewardTransferLogService) GetUserRewardLogs(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level for rewards management
	data, isAuth, err := svc.employeeSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	resp, err := svc.rewardLogsSvc.GetAllLogsForEntity(data.Username, 20) // Send data only for the user who is authorized.
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	respBytes, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		Body:       string(respBytes),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil

}

func (svc *RewardTransferLogService) GetAdminRewardLogs(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level for rewards management
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	resp, err := svc.rewardLogsSvc.GetAllLogsForEntity(companylib.REWARDS_DEFAULT_ADMIN, 20) // Send data only for the user who is authorized.
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	respBytes, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		Body:       string(respBytes),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil

}
