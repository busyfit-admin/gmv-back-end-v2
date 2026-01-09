package UserLib

import (
	"context"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

type UserCardsTable struct {
	CardId              string  `dynamodbav:"CardId"`
	CardName            string  `dynamodbav:"CardName"`
	CompanyName         string  `dynamodbav:"CompanyName"`
	CardMetaData        string  `dynamodbav:"CardMetaData"`
	Action              string  `dynamodbav:"Action"`
	AddRewardAmount     float64 `dynamodbav:"AddRewardAmount"`
	ConsumeRewardAmount float64 `dynamodbav:"ConsumeRewardAmount"`
	UpdateExpiry        string  `dynamodbav:"UpdateExpiry"`
	IsActive            bool    `dynamodbav:"IsActive"`
	UpdatedBy           string  `dynamodbav:"UpdatedBy"`
	RewardAmount        int     `dynamodbav:"RewardAmount"`
}

type UserService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger
	CardsTable     string
}

func CreateUserService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger, CardsTable string) *UserService {
	return &UserService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
		CardsTable:     CardsTable,
	}
}

type CardInput struct {
	CardId              int     `json:"CardId"`
	Action              string  `json:"Action"`
	AddRewardAmount     float64 `json:"AddRewardAmount"`
	ConsumeRewardAmount float64 `json:"ConsumeRewardAmount"`
	UpdateExpiry        string  `json:"UpdateExpiry"`
	IsActive            bool    `json:"IsActive"`
	UpdatedBy           string  `json:"UpdatedBy"`
	RewardAmount        int     `json:"RewardAmount"`
}

func (svc *UserService) UpdateCardsTable(cardId string, data CardInput) error {
	item := map[string]dynamodb_types.AttributeValue{
		"CardId":              &dynamodb_types.AttributeValueMemberS{Value: strconv.Itoa(data.CardId)},
		"Action":              &dynamodb_types.AttributeValueMemberS{Value: data.Action},
		"AddRewardAmount":     &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatFloat(data.AddRewardAmount, 'f', 4, 64)}, // precision = 4
		"ConsumeRewardAmount": &dynamodb_types.AttributeValueMemberN{Value: strconv.FormatFloat(data.ConsumeRewardAmount, 'f', 4, 64)},
		"UpdateExpiry":        &dynamodb_types.AttributeValueMemberS{Value: data.UpdateExpiry},
		"IsActive":            &dynamodb_types.AttributeValueMemberBOOL{Value: data.IsActive},
		"UpdatedBy":           &dynamodb_types.AttributeValueMemberS{Value: data.UpdatedBy},
		"RewardAmount":        &dynamodb_types.AttributeValueMemberN{Value: strconv.Itoa(data.RewardAmount)},
	}

	marshaledItem, err := dynamodb_attributevalue.MarshalMap(item)
	if err != nil {
		svc.logger.Printf("Failed to marshal item: %v", err)
		return err
	}

	putItemInput := &dynamodb.PutItemInput{
		Item:      marshaledItem,
		TableName: aws.String(svc.CardsTable),
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, putItemInput)
	if err != nil {
		svc.logger.Printf("Failed to put item into table: %v", err)
		return err
	}

	return nil
}

func (svc *UserService) GetCardsData(cardId string) (UserCardsTable, error) {
	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(svc.CardsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"CardId": &dynamodb_types.AttributeValueMemberS{Value: cardId},
		},
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, getItemInput)
	if err != nil {
		svc.logger.Printf("Failed to get item from table: %v", err)
		return UserCardsTable{}, err
	}

	item := UserCardsTable{}
	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &item)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal item: %v", err)
		return UserCardsTable{}, err
	}

	return item, nil
}
