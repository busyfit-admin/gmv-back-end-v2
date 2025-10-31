package Companylib

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type RewardsTransferLogsService struct {
	ctx context.Context

	logger *log.Logger

	dynamodbClient awsclients.DynamodbClient

	RewardsTransferLogsTable string
	EmployeeTable            string
}

func CreateRewardsTransferLogsService(ctx context.Context, logger *log.Logger, ddbClient awsclients.DynamodbClient) *RewardsTransferLogsService {
	return &RewardsTransferLogsService{
		ctx:            ctx,
		logger:         logger,
		dynamodbClient: ddbClient,
	}
}

const (
	REWARDS_SENT     = "SENT"
	REWARDS_REDEEMED = "REDEEMED"

	REWARDS_RECIEVED     = "RECIEVED"
	REWARDS_NEW_GENERATE = "CREATED"
)

type EntityDataBasic struct {
	EntityName string `json:"EntityName" dynamodbav:"EntityName"` // User's Display Name, Redeemed Card Name
	EntityPic  string `json:"EntityPic" dynamodbav:"EntityPic"`   // Pic of Users Profile, Redeemed Card Template
}

type RewardsTransferLogsTable struct {
	PK string `dynamodbav:"PK"` // ENTITY#<username>
	SK string `dynamodbav:"SK"` // TIMESTAMP#<timestamp>

	RewardsTransferId      string `json:"RewardsTransferId" dynamodbav:"RewardsTransferId"`
	RewardsTransferBatchId string `json:"RewardsTransferBatchId" dynamodbav:"RewardsTransferBatchId"` // Applicable for batch reward transfers in Reward Rules

	Counterparty string `json:"Counterparty" dynamodbav:"Counterparty"`
	TxnType      string `json:"TxnType" dynamodbav:"TxnType"` // Spend OR Receive
	Points       string `json:"Points" dynamodbav:"Points"`
	RewardTypeId string `json:"RewardTypeId" dynamodbav:"RewardTypeId"`

	RewardsTransferStatus  string `json:"RewardsTransferStatus" dynamodbav:"RewardsTransferStatus"`
	RewardsTransferLogTime string `json:"RewardsTransferLogTime" dynamodbav:"RewardsTransferLogTime"`
	Error                  string
}

// Functions to Put Reward Transfer Logs

type UpdateRewardTransferLogsInput struct {
	TxId      string
	TxBatchId string

	Source      string
	Destination string
	Points      int32
	RewardId    string

	TxStatus    string
	TxTimeStamp string
	Error       string
}

type UpdateRewardTransferLogsOutput struct {
	TxId string `json:"TxId"`
}

// Adding new Log data for User Level Rewards Send
// Case 1: When User sends Kudo's to another User
// Case 2: When Admin sends rewards to user Either Via Automation/ Manual
func (svc *RewardsTransferLogsService) UpdateRewardsTransferLogs_REWARDS_SEND(txInput UpdateRewardTransferLogsInput) (UpdateRewardTransferLogsOutput, error) {

	if txInput.TxId == "" {
		return UpdateRewardTransferLogsOutput{}, nil
	}
	updateLogTimeStamp := utils.GenerateTimestamp()

	// Update the Logs as Deduction at the Source Entity
	putItemInput_source := dynamodb.PutItemInput{
		TableName: aws.String(svc.RewardsTransferLogsTable),
		Item: map[string]dynamodb_types.AttributeValue{
			"PK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("ENTITY#%s", txInput.Source)},
			"SK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("TIMESTAMP#%s", updateLogTimeStamp)},

			"RewardsTransferId":      &dynamodb_types.AttributeValueMemberS{Value: txInput.TxId},
			"RewardsTransferBatchId": &dynamodb_types.AttributeValueMemberS{Value: txInput.TxBatchId},

			"Counterparty": &dynamodb_types.AttributeValueMemberS{Value: txInput.Destination},
			"TxnType":      &dynamodb_types.AttributeValueMemberS{Value: REWARDS_SENT},
			"Points":       &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("-%v", txInput.Points)},
			"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: txInput.RewardId},

			"RewardsTransferStatus":  &dynamodb_types.AttributeValueMemberS{Value: txInput.TxStatus},
			"RewardsTransferLogTime": &dynamodb_types.AttributeValueMemberS{Value: updateLogTimeStamp},
			"Error":                  &dynamodb_types.AttributeValueMemberS{Value: txInput.Error},
		},
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput_source)
	if err != nil {
		svc.logger.Printf("Put Item Failed to enter the Rewards Transfer Logs for Input : %v \n error: %v", txInput, err)
		return UpdateRewardTransferLogsOutput{}, nil
	}

	putItemInput_dest := dynamodb.PutItemInput{
		TableName: aws.String(svc.RewardsTransferLogsTable),
		Item: map[string]dynamodb_types.AttributeValue{
			"PK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("ENTITY#%s", txInput.Destination)},
			"SK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("TIMESTAMP#%s", updateLogTimeStamp)},

			"RewardsTransferId":      &dynamodb_types.AttributeValueMemberS{Value: txInput.TxId},
			"RewardsTransferBatchId": &dynamodb_types.AttributeValueMemberS{Value: txInput.TxBatchId},

			"Counterparty": &dynamodb_types.AttributeValueMemberS{Value: txInput.Source},
			"TxnType":      &dynamodb_types.AttributeValueMemberS{Value: REWARDS_RECIEVED},
			"Points":       &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("+%v", txInput.Points)},
			"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: txInput.RewardId},

			"RewardsTransferStatus":  &dynamodb_types.AttributeValueMemberS{Value: txInput.TxStatus},
			"RewardsTransferLogTime": &dynamodb_types.AttributeValueMemberS{Value: updateLogTimeStamp},
			"Error":                  &dynamodb_types.AttributeValueMemberS{Value: txInput.Error},
		},
	}
	_, err = svc.dynamodbClient.PutItem(svc.ctx, &putItemInput_dest)
	if err != nil {
		svc.logger.Printf("Put Item Failed to enter the Rewards Transfer Logs for Input : %v \n error: %v", txInput, err)
		return UpdateRewardTransferLogsOutput{}, nil
	}

	// Update the Logs as Addition at the Destination Entity

	return UpdateRewardTransferLogsOutput{
		TxId: txInput.TxId,
	}, nil
}

// Adding new Log data for Admin Level - Automated Rule Initial Log
// When Auto Rule is triggered, this log would be the inital entry point with reference BatchId used in the Transfer Rewards
// Case 2: When Admin sends rewards to user Either Via Automation/ Manual - Inital Step
func (svc *RewardsTransferLogsService) UpdateRewardsTransferLogs_REWARDS_AUTO_RULE(txInput UpdateRewardTransferLogsInput) (UpdateRewardTransferLogsOutput, error) {

	if txInput.TxId == "" {
		return UpdateRewardTransferLogsOutput{}, nil
	}

	updateLogTimeStamp := utils.GenerateTimestamp()

	// Update the Logs as Deduction at the Source Entity
	putItemInput_source := dynamodb.PutItemInput{
		TableName: aws.String(svc.RewardsTransferLogsTable),
		Item: map[string]dynamodb_types.AttributeValue{
			"PK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("ENTITY#%s", txInput.Source)},
			"SK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("TIMESTAMP#%s", updateLogTimeStamp)},

			"RewardsTransferId":      &dynamodb_types.AttributeValueMemberS{Value: txInput.TxId},
			"RewardsTransferBatchId": &dynamodb_types.AttributeValueMemberS{Value: txInput.TxBatchId},

			"Counterparty": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("LIST#%s", txInput.TxBatchId)},
			"TxnType":      &dynamodb_types.AttributeValueMemberS{Value: REWARDS_SENT},
			"Points":       &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("-%v", txInput.Points)},
			"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: txInput.RewardId},

			"RewardsTransferStatus":  &dynamodb_types.AttributeValueMemberS{Value: txInput.TxStatus},
			"RewardsTransferLogTime": &dynamodb_types.AttributeValueMemberS{Value: updateLogTimeStamp},
			"Error":                  &dynamodb_types.AttributeValueMemberS{Value: txInput.Error},
		},
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput_source)
	if err != nil {
		svc.logger.Printf("Put Item Failed to enter the Rewards Transfer Logs for Input : %v \n error: %v", txInput, err)
		return UpdateRewardTransferLogsOutput{}, nil
	}

	return UpdateRewardTransferLogsOutput{
		TxId: txInput.TxId,
	}, nil
}

// Adding new Log data when Admin Generates new Rewards
// Case 3: When Admin Creates new Rewards in Org
func (svc *RewardsTransferLogsService) UpdateRewardsTransferLogs_REWARDS_INCEPTION(txInput UpdateRewardTransferLogsInput) (UpdateRewardTransferLogsOutput, error) {

	if txInput.TxId == "" {
		return UpdateRewardTransferLogsOutput{}, nil
	}

	updateLogTimeStamp := utils.GenerateTimestamp()

	// Update the Logs as Deduction at the Source Entity
	putItemInput_source := dynamodb.PutItemInput{
		TableName: aws.String(svc.RewardsTransferLogsTable),
		Item: map[string]dynamodb_types.AttributeValue{
			"PK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("ENTITY#%s", txInput.Destination)},
			"SK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("TIMESTAMP#%s", updateLogTimeStamp)},

			"RewardsTransferId":      &dynamodb_types.AttributeValueMemberS{Value: txInput.TxId},
			"RewardsTransferBatchId": &dynamodb_types.AttributeValueMemberS{Value: txInput.TxBatchId},

			"Counterparty": &dynamodb_types.AttributeValueMemberS{Value: "ADMIN"},
			"TxnType":      &dynamodb_types.AttributeValueMemberS{Value: REWARDS_NEW_GENERATE},
			"Points":       &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("+%v", txInput.Points)},
			"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: txInput.RewardId},

			"RewardsTransferStatus":  &dynamodb_types.AttributeValueMemberS{Value: txInput.TxStatus},
			"RewardsTransferLogTime": &dynamodb_types.AttributeValueMemberS{Value: updateLogTimeStamp},
			"Error":                  &dynamodb_types.AttributeValueMemberS{Value: txInput.Error},
		},
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput_source)
	if err != nil {
		svc.logger.Printf("Put Item Failed to enter the Rewards Transfer Logs for Input : %v \n error: %v", txInput, err)
		return UpdateRewardTransferLogsOutput{}, nil
	}

	return UpdateRewardTransferLogsOutput{
		TxId: txInput.TxId,
	}, nil
}

// Adding new log data when User Redeems Cards
// Case 4: When User Redeems a Card
func (svc *RewardsTransferLogsService) UpdateRewardsTransferLogs_REDEEM_CARDS(txInput UpdateRewardTransferLogsInput) (UpdateRewardTransferLogsOutput, error) {

	if txInput.TxId == "" {
		return UpdateRewardTransferLogsOutput{}, nil
	}

	updateLogTimeStamp := utils.GenerateTimestamp()

	// Update the Logs as Deduction at the Source Entity
	putItemInput_source := dynamodb.PutItemInput{
		TableName: aws.String(svc.RewardsTransferLogsTable),
		Item: map[string]dynamodb_types.AttributeValue{
			"PK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("ENTITY#%s", txInput.Source)},
			"SK": &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("TIMESTAMP#%s", updateLogTimeStamp)},

			"RewardsTransferId":      &dynamodb_types.AttributeValueMemberS{Value: txInput.TxId},
			"RewardsTransferBatchId": &dynamodb_types.AttributeValueMemberS{Value: txInput.TxBatchId},

			"Counterparty": &dynamodb_types.AttributeValueMemberS{Value: txInput.Destination},
			"TxnType":      &dynamodb_types.AttributeValueMemberS{Value: REWARDS_REDEEMED},
			"Points":       &dynamodb_types.AttributeValueMemberS{Value: fmt.Sprintf("-%v", txInput.Points)},
			"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: txInput.RewardId},

			"RewardsTransferStatus":  &dynamodb_types.AttributeValueMemberS{Value: txInput.TxStatus},
			"RewardsTransferLogTime": &dynamodb_types.AttributeValueMemberS{Value: updateLogTimeStamp},
			"Error":                  &dynamodb_types.AttributeValueMemberS{Value: txInput.Error},
		},
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput_source)
	if err != nil {
		svc.logger.Printf("Put Item Failed to enter the Rewards Transfer Logs for Input : %v \n error: %v", txInput, err)
		return UpdateRewardTransferLogsOutput{}, nil
	}

	return UpdateRewardTransferLogsOutput{
		TxId: txInput.TxId,
	}, nil
}

type GetRewardTransferLogsOutput struct {
	TxId string `json:"TxId"`

	RecipientListRefId string `json:"RecipientListRefId"` //TxBatchId

	Desc       string `json:"Desc"`
	Points     string `json:"Points"`
	RewardType string `json:"RewardType"`

	TxStatus    string `json:"TxStatus"`
	TxTimeStamp string `json:"Timestamp"`
	Error       string
}

// Get All Logs at Entity Level
func (svc *RewardsTransferLogsService) GetAllLogsForEntity(entityId string, limit int32) ([]GetRewardTransferLogsOutput, error) {
	pk := fmt.Sprintf("ENTITY#%s", entityId)

	out, err := svc.dynamodbClient.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.RewardsTransferLogsTable),
		KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":pk": &dynamodb_types.AttributeValueMemberS{Value: pk},
		},
		ScanIndexForward: aws.Bool(false), // Descending by SK (time)
		Limit:            aws.Int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}

	var result []GetRewardTransferLogsOutput

	for _, item := range out.Items {

		// Unmarshal to RewardsTransferLogsTable struct
		var logData RewardsTransferLogsTable
		err = attributevalue.UnmarshalMap(item, &logData)
		if err != nil {
			svc.logger.Printf("Failed to Unmarshal the output to DDB Struct")
			return []GetRewardTransferLogsOutput{}, err
		}

		var entry GetRewardTransferLogsOutput

		rewardTypeId := logData.RewardTypeId
		rewardTypeName := ConvertEmpRewardTypeToRewardName(rewardTypeId)
		points := logData.Points
		txnType := logData.TxnType

		// Format: -$ for Spend, $ for Receive
		formattedPoints := fmt.Sprintf("%v", points)
		if txnType == REWARDS_SENT {
			entry.Desc = fmt.Sprintf("REWARD SENT TO %s (%s)", logData.Counterparty, rewardTypeName)
		} else if txnType == REWARDS_REDEEMED {
			entry.Desc = fmt.Sprintf("REWARD REDEEMED FOR %s (%s)", logData.Counterparty, rewardTypeName)
		} else if txnType == REWARDS_NEW_GENERATE {
			entry.Desc = fmt.Sprintf("REWARD GENERATED FOR %s (%s)", logData.Counterparty, rewardTypeName)
		} else {
			entry.Desc = fmt.Sprintf("REWARD RECEIVED FROM %s (%s)", logData.Counterparty, rewardTypeName)
		}

		entry.TxId = logData.RewardsTransferId
		entry.RecipientListRefId = logData.RewardsTransferBatchId
		entry.Points = formattedPoints
		entry.RewardType = rewardTypeName
		entry.TxStatus = logData.RewardsTransferStatus
		entry.TxTimeStamp = logData.RewardsTransferLogTime
		entry.Error = logData.Error

		result = append(result, entry)
	}

	return result, nil
}
