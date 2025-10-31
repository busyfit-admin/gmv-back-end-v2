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

type CardsTransferService struct {
	ctx context.Context

	logger *log.Logger

	dynamodbClient awsclients.DynamodbClient

	RewardsTransferLogsTable string
	EmployeeTable            string
	CompanyCardsTable        string
}

func CreateCardsTransferService(ctx context.Context, logger *log.Logger, ddbClient awsclients.DynamodbClient) *CardsTransferService {
	return &CardsTransferService{
		ctx:            ctx,
		logger:         logger,
		dynamodbClient: ddbClient,
	}
}

//----------- DDB Tables -------

// type RewardsTransferLogsTable struct {
// 	RewardsTransferId   string `json:"RewardsTransferId" dynamodbav:"RewardsTransferId"`     // PK
// 	SourceUserName      string `json:"SourceUserName" dynamodbav:"SourceUserName"`           // Index - PK
// 	DestinationUserName string `json:"DestinationUserName" dynamodbav:"DestinationUserName"` // Index - PK
// 	Points              int32  `json:"Points" dynamodbav:"Points"`
// 	RewardTypeId        string `json:"RewardTypeId" dynamodbav:"RewardTypeId"`

// 	RewardsTransferStatus  string `json:"RewardsTransferStatus" dynamodbav:"RewardsTransferStatus"`
// 	RewardsTransferLogTime string `json:"RewardsTransferLogTime" dynamodbav:"RewardsTransferLogTime"` // Index - SK
// 	Error                  string
// }

type CardsTransferInput struct {
	TxId string

	SourceUserName      string
	DestinationUserName string

	TransferPoints int

	CardData RewardCards

	RewardType string
}

func (svc *CardsTransferService) HandleCardsTransfer(txInput CardsTransferInput) error {

	// 1. if UUID Is Empty , then Create a UUID for the RewardsTransfer Ref ID
	if txInput.TxId == "" {
		uuId := uuid.New()
		txInput.TxId = uuId.String()
	}

	TXStatus := TX_SUCCESS
	var err error

	err = ValidateCardsInput(&txInput)
	if err != nil {
		return err
	}

	// 2. Initiate RewardsTransfer
	err = svc.PerformCardsTransfer(txInput)
	if err != nil {
		TXStatus = TX_FAIL
	}

	// 3. Update the RewardsTransfer log table
	err = svc.UpdateCardsTransferLogs(txInput, TXStatus, fmt.Sprintf("%s", err))
	if err != nil {
		return err
	}

	return nil
}

/*

	RewardsData under the Employee's table:

	"RewardsData": {
		"RD00": {
			"RewardExpiryDate": "10-10-2025",
			"RewardId": "RD00",
			"RewardPoints": 3400,
			"TransferablePoints": 10000
			},
		"RD01": {
			"RewardExpiryDate": "10-10-2025",
			"RewardId": "RD01",
			"RewardPoints": 500,
			"TransferablePoints": 7000
			},
		"RD02": {
			"RewardExpiryDate": "28-02-2025",
			"RewardId": "RD02",
			"RewardPoints": 210,
			"TransferablePoints": 3000
			}
 },

*/

func (svc *CardsTransferService) PerformCardsTransfer(txInput CardsTransferInput) error {
	cardDataMap, err := dynamodb_attributevalue.MarshalMap(txInput.CardData)
	if err != nil {
		svc.logger.Printf("Failed to marshal CardData: %v\n", err)
		return err
	}

	svc.logger.Println("CardData Map: ", cardDataMap)

	// 1. Deduct the Relevent Reward Points from the Source EntityId
	updatePointsAndRedeemedCards := dynamodb_types.Update{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: txInput.SourceUserName},
		},
		ConditionExpression: aws.String("RewardsData.#REWID.#RP >= :TXPOINTS AND attribute_exists(RedeemedCards)"),
		UpdateExpression:    aws.String("SET RewardsData.#REWID.RewardPoints = RewardsData.#REWID.RewardPoints - :TXPOINTS, RedeemedCards.#CardNumber = :CardData"),
		ExpressionAttributeNames: map[string]string{
			"#REWID": txInput.RewardType,
			"#RP":    "RewardPoints",
			// Card Data Related Attributes
			"#CardNumber": txInput.CardData.CardNumber,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":TXPOINTS": &dynamodb_types.AttributeValueMemberN{Value: strconv.Itoa(int(txInput.TransferPoints))},
			// Card Data Related Attributes
			":CardData": &types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					"CardId":       &dynamodb_types.AttributeValueMemberS{Value: txInput.CardData.CardId},
					"CardName":     &dynamodb_types.AttributeValueMemberS{Value: txInput.CardData.CardName},
					"CardNumber":   &dynamodb_types.AttributeValueMemberS{Value: txInput.CardData.CardNumber},
					"CreatedDate":  &dynamodb_types.AttributeValueMemberS{Value: txInput.CardData.CreatedDate},
					"ExpiryDate":   &dynamodb_types.AttributeValueMemberS{Value: txInput.CardData.ExpiryDate},
					"RedeemStatus": &dynamodb_types.AttributeValueMemberS{Value: txInput.CardData.Status},
					"RewardPoints": &dynamodb_types.AttributeValueMemberN{Value: strconv.Itoa(int(txInput.CardData.RewardPoints))},
				},
			},
		},
	}

	svc.logger.Printf("Update Points and Redeemed Cards Values - TXPOINTS: %v, CARD: %+v\n",
		txInput.TransferPoints, txInput.CardData)

	// 2. Update Card Status to Redeemed in the Cards Table
	updateCardStatus := dynamodb_types.Update{
		TableName: aws.String(svc.CompanyCardsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"CardNumber": &dynamodb_types.AttributeValueMemberS{Value: txInput.CardData.CardNumber},
			"CardId":     &dynamodb_types.AttributeValueMemberS{Value: txInput.CardData.CardId},
		},
		UpdateExpression: aws.String("SET CardStatus = :CardStatus, RedeemedBy = :RedeemedBy, RedeemedOn = :RedeemedOn"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":CardStatus": &dynamodb_types.AttributeValueMemberS{Value: CARD_ISREDEEMED},
			":RedeemedBy": &dynamodb_types.AttributeValueMemberS{Value: txInput.SourceUserName},
			":RedeemedOn": &dynamodb_types.AttributeValueMemberS{Value: utils.GenerateTimestamp()},
		},
	}

	svc.logger.Printf("Update Card Status for %s - Setting RedeemStatus: REDEEMED, RedeemedBy: %s, RedeemedOn: %s\n",
		txInput.CardData.CardNumber, txInput.SourceUserName, utils.GenerateTimestamp())

	// 3. Perform RewardsTransfer
	_, err = svc.dynamodbClient.TransactWriteItems(svc.ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []dynamodb_types.TransactWriteItem{
			{
				Update: &updatePointsAndRedeemedCards,
			},
			{
				Update: &updateCardStatus,
			},
		},
		ClientRequestToken: aws.String(txInput.TxId),
	})

	// NOTE : Enable Cond check fails handling in future to alert user if there are less points in the source account.
	//Currently we'll setup the check in front end only
	if err != nil {
		svc.logger.Printf("Failed to perform transaction due to error : %v", err)
		return err
	}

	return nil
}

func ValidateCardsInput(txInput *CardsTransferInput) error {

	if txInput.DestinationUserName == "" || txInput.SourceUserName == "" || txInput.RewardType == "" || txInput.TransferPoints <= 0 {
		return fmt.Errorf("input data is incomplete or incorrect. data: %v", txInput)
	}
	if !(txInput.RewardType == REWARD_TYPE_General || txInput.RewardType == REWARD_TYPE_Health || txInput.RewardType == REWARD_TYPE_Skills || txInput.RewardType == REWARD_TYPE_EmployeeSupport) {
		return fmt.Errorf(" incorrect reward Type data: %v", txInput)
	}
	return nil
}

func (svc *CardsTransferService) UpdateCardsTransferLogs(txInput CardsTransferInput, txStatus string, errorString string) error {

	// Updated to the new Reward Logging Formats
	RewardLogingService := CreateRewardsTransferLogsService(svc.ctx, svc.logger, svc.dynamodbClient)
	RewardLogingService.RewardsTransferLogsTable = svc.RewardsTransferLogsTable

	_, err := RewardLogingService.UpdateRewardsTransferLogs_REDEEM_CARDS(UpdateRewardTransferLogsInput{
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

	return nil
}
