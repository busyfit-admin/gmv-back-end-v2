// This lambda handles the Transferring of Rewards from one Entity to another. The Lambda is triggered from SQS Queue
// Multiple sources can add Transferring of rewards based on reward rules and events.

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

type RewardTransferService struct {
	ctx    context.Context
	logger *log.Logger

	TransactionSvc companylib.RewardsTransferService
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "reward-transfer")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	TransferSvc := companylib.CreateRewardsTransferService(ctx, logger, ddbclient)
	TransferSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	TransferSvc.RewardRulesTable = os.Getenv("REWARD_RULES_TABLE")
	TransferSvc.RewardsTransferLogsTable = os.Getenv("REWARDS_TRANSFER_LOGS_TABLE")

	svc := RewardTransferService{
		ctx:            ctx,
		logger:         logger,
		TransactionSvc: *TransferSvc,
	}

	lambda.Start(svc.handleRewardTransferEvents)

}

func (svc *RewardTransferService) handleRewardTransferEvents(sqsEvent events.SQSEvent) error {

	for _, singleSqsEvent := range sqsEvent.Records {

		var RewardTransferInput companylib.RewardsTransferInput

		err := json.Unmarshal([]byte(singleSqsEvent.Body), &RewardTransferInput)
		if err != nil {
			return err
		}

		svc.logger.Printf("Received Reward Transfer Event : %v\n", RewardTransferInput)

		err = svc.TransactionSvc.HandleRewardTransfer(RewardTransferInput)
		if err != nil {
			return err
		}
	}

	return nil
}
