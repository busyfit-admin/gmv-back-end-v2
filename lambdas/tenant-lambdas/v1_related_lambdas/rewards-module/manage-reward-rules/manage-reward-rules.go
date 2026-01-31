package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/aws/aws-lambda-go/lambda"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type SQSClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}
type Service struct {
	ctx       context.Context
	logger    *log.Logger
	sqsClient SQSClient

	employeeSvc companylib.EmployeeService
	rewardsSVC  companylib.RewardsService

	REWARDS_TRANSFER_SQS_NAME string
}

var RESP_HEADERS = companylib.GetHeadersForAPI("RewardsAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-reward-rules")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)

	// Employee Rewards Services
	rewardssvc := companylib.CreateRewardsService(ctx, ddbclient, logger)
	rewardssvc.EmployeeRewardRulesTable = os.Getenv("REWARDS_RULES_TABLE")
	rewardssvc.EmployeeRewardRulesTable_RuleStatusIndex = os.Getenv("REWARDS_RULES_TABLE_RULE_STATUS_INDEX")
	rewardssvc.EmployeeRewardRulesTables_RewardRuleLogUpdateDateIndex = os.Getenv("REWARDS_RULES_TABLE_LOG_UPDATE_INDEX")

	// Employee Service
	employeeSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")
	employeeSvc.RewardsRuleTable = os.Getenv("REWARDS_RULES_TABLE")

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		rewardsSVC:  *rewardssvc,
		employeeSvc: *employeeSvc,

		sqsClient:                 sqsClient,
		REWARDS_TRANSFER_SQS_NAME: os.Getenv("REWARD_TRANSFER_SQS_QUEUE"),
	}

	lambda.Start(svc.handleAPIRequests)

}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.ctx = ctx

	switch request.HTTPMethod {
	case "GET":
		return svc.GETRequestHandler(request)

	case "PATCH":
		return svc.PATCHRequestHandler(request)

	case "POST":
		return svc.POSTRequestHandler(request)

	case "DELETE":
		return svc.DELETERequestHandler(request)

	default:
		svc.logger.Printf("entered Default section of the switch, Erroring by returning 500")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level for rewards management
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	switch request.Headers["get_type"] {
	case "get_rule_settings":
		return svc.GetRuleSettings(request)
	case "get_reward_rules":
		return svc.GetRewardRules(request)
	case "get_reward_logs":
		return svc.GetRewardLogs(request)
	case "get_reward_admin_points":
		return svc.GetRewardAdminPoints(request)
	default:
		return svc.GetAllSettings(request)
	}

}

func (svc *Service) GetRuleSettings(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	output, err := svc.rewardsSVC.GetTopLevelRewardSettings()
	if err != nil {
		svc.logger.Printf("failed to get the top level reward settings, error: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		svc.logger.Printf("failed to marshal to json output")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonBytes),
	}, nil
}

func (svc *Service) GetRewardRules(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	output, err := svc.rewardsSVC.GetRewardRules()
	if err != nil {
		svc.logger.Printf("failed to get all reward rules, error: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		svc.logger.Printf("failed to marshal to json output")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonBytes),
	}, nil
}

func (svc *Service) GetRewardLogs(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	output, err := svc.rewardsSVC.GetRewardUpdateLogsData()
	if err != nil {
		svc.logger.Printf("failed to get all reward rules, error: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		svc.logger.Printf("failed to marshal to json output")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonBytes),
	}, nil
}

func (svc *Service) GetRewardAdminPoints(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	userData, err := svc.employeeSvc.GetEmployeeDataRewardSettingsByUserName(companylib.REWARDS_DEFAULT_ADMIN)
	if err != nil {
		svc.logger.Printf("failed to get RewardsAdmin User Details")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	// Convert to Reward Namings output
	userData.RewardsData = companylib.ConvertEmpRewardTypeToRewardNames(userData.RewardsData)

	jsonBytes, err := json.Marshal(userData)
	if err != nil {
		svc.logger.Printf("failed to marshal to json output")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(jsonBytes),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

func (svc *Service) GetAllSettings(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	RuleId := request.Headers["rule-id"]
	if RuleId != "" {
		svc.logger.Printf("Get Request for a specific rule : RuleId: %v", RuleId)
		companyData, err := svc.rewardsSVC.GetRulesByRuleId(string(RuleId))
		if err != nil {
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
			}, err
		}
		responseBody, err := json.Marshal(companyData)
		if err != nil {
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
			}, err
		}
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 200,
			Body:       string(responseBody),
		}, err
	}

	svc.logger.Printf("Get Request for all Rules initiated")
	allRewardRulesOutput, err := svc.rewardsSVC.GetAllRewardRules()
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	userData, err := svc.employeeSvc.GetEmployeeDataRewardSettingsByUserName(companylib.REWARDS_DEFAULT_ADMIN)
	if err != nil {
		svc.logger.Printf("failed to get RewardsAdmin User Details")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	// Convert to Reward Namings output
	userData.RewardsData = companylib.ConvertEmpRewardTypeToRewardNames(userData.RewardsData)

	allRewardRulesOutput.RewardAdminPoints = userData

	responseBody, err := json.Marshal(allRewardRulesOutput)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user has to the Group
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		svc.logger.Printf("Unable to Authorize the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	switch request.Headers["post_type"] {

	case "create_reward_rule":
		return svc.CreateRewardRule(request)
	case "add_reward_points":
		return svc.AddRewardPoints(request)

	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) CreateRewardRule(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var createRuleData companylib.CreateRewardRuleInput
	err := json.Unmarshal([]byte(request.Body), &createRuleData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	err = svc.rewardsSVC.CreateRewardsRule(createRuleData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

type AddAdminRewardPoints struct {
	RewardType   string `json:"RewardType"`
	RewardPoints int    `json:"RewardPoints"`
}

func (svc *Service) AddRewardPoints(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var inputBody AddAdminRewardPoints
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	txId := "INCEP-" + utils.GenerateRandomString(12)

	rwdTxSQSInput := companylib.RewardsTransferInput{
		TxId:                txId,
		TxType:              companylib.TxType_ADD_TP_ADMIN,
		SourceUserName:      companylib.REWARDPOINTS_INCEPTION,
		DestinationUserName: companylib.REWARDS_DEFAULT_ADMIN,
		TransferPoints:      int32(inputBody.RewardPoints),
		RewardType:          inputBody.RewardType,
	}

	rwdTxInputBytes, _ := json.Marshal(rwdTxSQSInput)

	// Send the message to the SQS queue to transfer the points.
	_, err = svc.sqsClient.SendMessage(svc.ctx, &sqs.SendMessageInput{
		MessageBody: aws.String(string(rwdTxInputBytes)),
		QueueUrl:    aws.String(svc.REWARDS_TRANSFER_SQS_NAME),

		MessageDeduplicationId: aws.String(txId),
		MessageGroupId:         aws.String(companylib.REWARDS_DEFAULT_ADMIN),
	})

	if err != nil {
		svc.logger.Printf("Error sending message to SQS: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

const (
	REWARD_TYPE_PATCH = "reward_type_patch"
	REWARD_UNIT_PATCH = "reward_unit_patch"
	REWARD_RULE_PATCH = "reward_rule_patch"
)

func (svc *Service) PATCHRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	patchType := request.Headers["patch_type"]

	switch patchType {
	case REWARD_TYPE_PATCH:
		return svc.RewardTypePatchHandler(request)

	case REWARD_UNIT_PATCH:
		return svc.RewardUnitPatchHandler(request)

	case REWARD_RULE_PATCH:
		return svc.RewardRulesPatchHandler(request)

	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) RewardTypePatchHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var RewardPatchInput companylib.PatchRewardTypesStatusInput
	err := json.Unmarshal([]byte(request.Body), &RewardPatchInput)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	err = svc.rewardsSVC.PatchRewardTypesStatus(RewardPatchInput)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

func (svc *Service) RewardUnitPatchHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var rewardUnitPatchData companylib.RewardUnits

	err := json.Unmarshal([]byte(request.Body), &rewardUnitPatchData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	err = svc.rewardsSVC.PatchRewardUnits(rewardUnitPatchData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil

}

func (svc *Service) RewardRulesPatchHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var rewardRulesPatchData companylib.RewardRulesPatchInput

	err := json.Unmarshal([]byte(request.Body), &rewardRulesPatchData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	err = svc.rewardsSVC.PatchRewardRules(rewardRulesPatchData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil

}

func (svc *Service) DELETERequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	RuleId := request.Headers["rule-id"]
	if RuleId == "" {
		svc.logger.Printf("RuleID cannot be Empty. RuleID: %v", RuleId)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Printf("RuleID that is requested to be deleted: %v", RuleId)
	err := svc.rewardsSVC.DeleteRuleByRuleId(RuleId)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}
