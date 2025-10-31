package Companylib

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
	"github.com/google/uuid"
)

type RewardsTransferService struct {
	ctx context.Context

	logger *log.Logger

	dynamodbClient awsclients.DynamodbClient

	RewardRulesTable         string
	RewardsTransferLogsTable string
	EmployeeTable            string
}

func CreateRewardsTransferService(ctx context.Context, logger *log.Logger, ddbClient awsclients.DynamodbClient) *RewardsTransferService {
	return &RewardsTransferService{
		ctx:            ctx,
		logger:         logger,
		dynamodbClient: ddbClient,
	}
}

//----------- DDB Tables -------

/*
	Handle Reward RewardsTransfer takes in the following Input :
	{
		"SourceUserName" : "<ENTITY ID>",
		"DestinationUserName": "<ENTITY ID>",
		"POINTS": "<POINTS>",
		"REWARD_TYPE": "REWARD_TYPE_ID"
	}
*/

const (
	TxType_ADD_TP_ADMIN = "TxType_ADD_TP_ADMIN" // Transactions Add New TP to Rewards Admin
	TxType_TX_TP_USERS  = "TxType_TX_TP_USERS"  // Transactions TP --> TP to users
	TxType_TX_RP_USERS  = ""                    // Transactions TP --> RP to users
)

type RewardsTransferInput struct {
	TxId string // Unique TxId

	TxType string /* TxTypes are as follows:
	/*
		TxType_ADD_TP_ADMIN = "TxType_ADD_TP_ADMIN" // Transactions Add New TP to Rewards Admin
		TxType_TX_TP_USERS  = "TxType_TX_TP_USERS"  // Transactions TP --> TP to users
		TxType_TX_RP_USERS  = ""                    // Transactions TP --> RP to users
	*/
	TxBatchId string // Unique Tx BatchId if its sent as a batch. ex: Yearly health Rewards.

	SourceUserName      string // Source UserName or Const from where Rewards are Transferred
	DestinationUserName string // Destination UserName or Const to where the Rewards are sent / Redeemed to

	TransferPoints int32 // Reward Points

	RewardType string /* RewardTypes are described as per below:
	/*
		const (
			REWARD_TYPE_General         = "RD00"
			REWARD_TYPE_Health          = "RD01"
			REWARD_TYPE_Skills          = "RD02"
			REWARD_TYPE_EmployeeSupport = "RD03"
		)
	*/
}

const (
	TX_SUCCESS = "SUCCESS"
	TX_FAIL    = "FAIL"
)

func (svc *RewardsTransferService) HandleRewardTransfer(txInput RewardsTransferInput) error {

	// 1. if UUID Is Empty , then Create a UUID for the RewardsTransfer Ref ID
	if txInput.TxId == "" {
		uuId := uuid.New()
		txInput.TxId = uuId.String()
	}

	TXStatus := TX_SUCCESS
	var err error

	err = ValidateInput(&txInput)
	if err != nil {
		return err
	}

	// 2. Check if Tx RewardType is Valid
	status := svc.ValidateTxRewardType(txInput)
	if !status {
		// Update the RewardsTransfer log table
		err = svc.UpdateRewardsTransferLogs(txInput.TxType, txInput, TXStatus, fmt.Sprintf("Reward type is incorrect or not enabled. RType: %s", txInput.RewardType))
		if err != nil {
			return err
		}
		return nil
	}

	// 3. Initiate RewardsTransfer based on TxType
	switch txInput.TxType {
	case TxType_ADD_TP_ADMIN:
		err = svc.PerformRewardsAdditionToRewardsAdmin(txInput)
		if err != nil {
			TXStatus = TX_FAIL
		}
	case TxType_TX_TP_USERS:
		err = svc.PerformRewardsAdditionToUser(txInput)
		if err != nil {
			TXStatus = TX_FAIL
		}
	case TxType_TX_RP_USERS:
		err = svc.PerformRewardsTransfer(txInput)
		if err != nil {
			TXStatus = TX_FAIL
		}
	default:
		TXStatus = TX_FAIL
	}

	// 4. Update the RewardsTransfer log table
	err = svc.UpdateRewardsTransferLogs(txInput.TxType, txInput, TXStatus, fmt.Sprintf("%s", err))
	if err != nil {
		return err
	}

	return nil
}

func (svc *RewardsTransferService) PerformRewardsAdditionToRewardsAdmin(txInput RewardsTransferInput) error {

	writeItemsInput := dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodb_types.TransactWriteItem{
			// 1. Add Points to Destination User
			{
				Update: &dynamodb_types.Update{
					TableName: aws.String(svc.EmployeeTable),
					Key: map[string]dynamodb_types.AttributeValue{
						"UserName": &dynamodb_types.AttributeValueMemberS{Value: txInput.DestinationUserName},
					},
					UpdateExpression: aws.String("SET RewardsData.#REWID.#TP = if_not_exists(RewardsData.#REWID.#TP, :ZERO) + :TXPOINTS"),
					ExpressionAttributeNames: map[string]string{
						"#REWID": txInput.RewardType,
						"#TP":    "TransferablePoints",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":TXPOINTS": &dynamodb_types.AttributeValueMemberN{Value: strconv.Itoa(int(txInput.TransferPoints))},
						":ZERO":     &dynamodb_types.AttributeValueMemberN{Value: "0"},
					},
				},
			},
		},
		ClientRequestToken: aws.String(txInput.TxId),
	}

	_, err := svc.dynamodbClient.TransactWriteItems(svc.ctx, &writeItemsInput)
	//var condCheckFail dynamodb_types.ConditionalCheckFailedException

	// NOTE : Enable Cond check fails handling in future to alert user if there are less points in the source account.
	//Currently we'll setup the check in front end only
	if err != nil {
		svc.logger.Printf("Failed to perform transaction due to error : %v", err)
		return err
	}

	return nil
}
func (svc *RewardsTransferService) PerformRewardsAdditionToUser(txInput RewardsTransferInput) error {

	if txInput.SourceUserName == txInput.DestinationUserName {
		svc.logger.Printf("Cannot transfer within the same user")
		return fmt.Errorf("cannot transfer within the same user")
	}

	writeItemsInput := dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodb_types.TransactWriteItem{
			// 1. Deduct TP Points from Source
			{
				Update: &dynamodb_types.Update{
					TableName: aws.String(svc.EmployeeTable),
					Key: map[string]dynamodb_types.AttributeValue{
						"UserName": &dynamodb_types.AttributeValueMemberS{Value: txInput.SourceUserName},
					},
					ConditionExpression: aws.String("attribute_exists(RewardsData.#REWID) AND RewardsData.#REWID.#TP >= :TXPOINTS"),
					UpdateExpression:    aws.String("SET RewardsData.#REWID.TransferablePoints = RewardsData.#REWID.TransferablePoints - :TXPOINTS"),
					ExpressionAttributeNames: map[string]string{
						"#REWID": txInput.RewardType,
						"#TP":    "TransferablePoints",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":TXPOINTS": &dynamodb_types.AttributeValueMemberN{Value: strconv.Itoa(int(txInput.TransferPoints))},
					},
				},
			},
			// 2. Add TP Points to Destination User
			{
				Update: &dynamodb_types.Update{
					TableName: aws.String(svc.EmployeeTable),
					Key: map[string]dynamodb_types.AttributeValue{
						"UserName": &dynamodb_types.AttributeValueMemberS{Value: txInput.DestinationUserName},
					},
					UpdateExpression: aws.String("SET RewardsData.#REWID.#TP = if_not_exists(RewardsData.#REWID.#TP, :ZERO) + :TXPOINTS"),
					ExpressionAttributeNames: map[string]string{
						"#REWID": txInput.RewardType,
						"#TP":    "TransferablePoints",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":TXPOINTS": &dynamodb_types.AttributeValueMemberN{Value: strconv.Itoa(int(txInput.TransferPoints))},
						":ZERO":     &dynamodb_types.AttributeValueMemberN{Value: "0"},
					},
				},
			},
		},
		ClientRequestToken: aws.String(txInput.TxId),
	}

	_, err := svc.dynamodbClient.TransactWriteItems(svc.ctx, &writeItemsInput)
	//var condCheckFail dynamodb_types.ConditionalCheckFailedException

	// NOTE : Enable Cond check fails handling in future to alert user if there are less points in the source account.
	//Currently we'll setup the check in front end only
	if err != nil {
		svc.logger.Printf("Failed to perform transaction due to error : %v", err)
		return err
	}

	return nil
}
func (svc *RewardsTransferService) PerformRewardsTransfer(txInput RewardsTransferInput) error {

	if txInput.SourceUserName == txInput.DestinationUserName {
		svc.logger.Printf("Cannot transfer within the same user")
		return fmt.Errorf("cannot transfer within the same user")
	}

	// 1. Deduct Points from Source User

	writeItemsInput := dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodb_types.TransactWriteItem{
			{
				Update: &dynamodb_types.Update{
					TableName: aws.String(svc.EmployeeTable),
					Key: map[string]dynamodb_types.AttributeValue{
						"UserName": &dynamodb_types.AttributeValueMemberS{Value: txInput.SourceUserName},
					},
					ConditionExpression: aws.String("attribute_exists(RewardsData.#REWID) AND RewardsData.#REWID.#TP >= :TXPOINTS"),
					UpdateExpression:    aws.String("SET RewardsData.#REWID.TransferablePoints = RewardsData.#REWID.TransferablePoints - :TXPOINTS"),
					ExpressionAttributeNames: map[string]string{
						"#REWID": txInput.RewardType,
						"#TP":    "TransferablePoints",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":TXPOINTS": &dynamodb_types.AttributeValueMemberN{Value: strconv.Itoa(int(txInput.TransferPoints))},
					},
				},
			},
			// 2. Add Points to Destination User
			{
				Update: &dynamodb_types.Update{
					TableName: aws.String(svc.EmployeeTable),
					Key: map[string]dynamodb_types.AttributeValue{
						"UserName": &dynamodb_types.AttributeValueMemberS{Value: txInput.DestinationUserName},
					},
					UpdateExpression: aws.String("SET RewardsData.#REWID.#RP = if_not_exists(RewardsData.#REWID.#RP, :ZERO) + :TXPOINTS"),
					ExpressionAttributeNames: map[string]string{
						"#REWID": txInput.RewardType,
						"#RP":    "RewardPoints",
					},
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":TXPOINTS": &dynamodb_types.AttributeValueMemberN{Value: strconv.Itoa(int(txInput.TransferPoints))},
						":ZERO":     &dynamodb_types.AttributeValueMemberN{Value: "0"},
					},
				},
			},
		},
		ClientRequestToken: aws.String(txInput.TxId),
	}

	_, err := svc.dynamodbClient.TransactWriteItems(svc.ctx, &writeItemsInput)
	//var condCheckFail dynamodb_types.ConditionalCheckFailedException

	// NOTE : Enable Cond check fails handling in future to alert user if there are less points in the source account.
	//Currently we'll setup the check in front end only
	if err != nil {
		svc.logger.Printf("Failed to perform transaction due to error : %v", err)
		return err
	}

	return nil
}
func (svc *RewardsTransferService) UpdateRewardsTransferLogs(txType string, txInput RewardsTransferInput, txStatus string, errorString string) error {

	// Updated to the new Reward Logging Formats
	RewardLogingService := CreateRewardsTransferLogsService(svc.ctx, svc.logger, svc.dynamodbClient)
	RewardLogingService.RewardsTransferLogsTable = svc.RewardsTransferLogsTable

	switch txType {
	case TxType_ADD_TP_ADMIN:
		_, err := RewardLogingService.UpdateRewardsTransferLogs_REWARDS_INCEPTION(UpdateRewardTransferLogsInput{
			TxId:        txInput.TxId,
			Source:      txInput.SourceUserName,
			Destination: txInput.DestinationUserName,
			Points:      int32(txInput.TransferPoints),
			RewardId:    txInput.RewardType,

			TxStatus:    txStatus,
			TxTimeStamp: utils.GenerateTimestamp(),
			Error:       errorString,
		})
		if err != nil {
			return err
		}

	case TxType_TX_TP_USERS:
		svc.logger.Printf("Not setup yet")

	case TxType_TX_RP_USERS:
		_, err := RewardLogingService.UpdateRewardsTransferLogs_REWARDS_SEND(UpdateRewardTransferLogsInput{
			TxId:        txInput.TxId,
			Source:      txInput.SourceUserName,
			Destination: txInput.DestinationUserName,
			Points:      int32(txInput.TransferPoints),
			RewardId:    txInput.RewardType,

			TxStatus:    txStatus,
			TxTimeStamp: utils.GenerateTimestamp(),
			Error:       errorString,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
func (svc *RewardsTransferService) ValidateTxRewardType(txInput RewardsTransferInput) bool {

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.RewardRulesTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus}, // PK for Get Reward Type Settings
			"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus}, // SK
		},
	})
	if err != nil {
		svc.logger.Printf("Unable to perform Get Operation on Reward Rules Table")
		return false
	}

	var ddbData RewardsRuleDynamodbData
	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &ddbData)
	if err != nil {
		svc.logger.Printf("Unable to Unmarshal the output from Get Operation on Reward Rules Table")
		return false
	}
	// If enabled check the type that is enabled and send true, else false

	if ddbData.RewardTypeStatus[txInput.RewardType].Active {
		return true
	}

	svc.logger.Printf("Reward Type Not enabled %v", txInput.RewardType)
	return false
}
func ValidateInput(txInput *RewardsTransferInput) error {

	if !(txInput.TxType == TxType_TX_RP_USERS || txInput.TxType == TxType_ADD_TP_ADMIN || txInput.TxType == TxType_TX_TP_USERS) {
		return fmt.Errorf("incorrect type of tx type: %v", txInput.TxType)
	}

	if txInput.DestinationUserName == "" || txInput.SourceUserName == "" || txInput.RewardType == "" || txInput.TransferPoints <= 0 {
		return fmt.Errorf("input data is incomplete or incorrect. data: %v", txInput)
	}
	if !(txInput.RewardType == REWARD_TYPE_General || txInput.RewardType == REWARD_TYPE_Health || txInput.RewardType == REWARD_TYPE_Skills || txInput.RewardType == REWARD_TYPE_EmployeeSupport) {
		return fmt.Errorf(" incorrect reward Type data: %v", txInput)
	}
	return nil
}
