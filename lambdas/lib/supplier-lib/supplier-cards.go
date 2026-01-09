package supplierlib

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	utils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

const (
	CARD_ISACTIVE_FALSE = "INACTIVE"
	CARD_ISACTIVE_TRUE  = "ACTIVE"
	CARD_ISACTIVE_EXPIRED = "EXPIRED"
)

type SupplierCards struct {
	CardId   string `dynamodbav:"CardId" json:"CardId"`     // Unique CardId
	CardName string `dynamodbav:"CardName" json:"CardName"` // Custom name of the card that is entered by the Supplier
	CardType string `dynamodbav:"CardType" json:"CardType"` // Card Type that is selected ( card type can be of either : general, health, climate rewards)
	CardDesc string `dynamodbav:"CardDesc" json:"CardDesc"`
	ExpiryDate string `dynamodbav:"ExpiryDate" json:"ExpiryDate"`
	IsActive string `dynamodbav:"IsActive" json:"IsActive"`
}

type SupplierCardsService struct {
	ctx    context.Context
	logger *log.Logger

	dynamodbClient awsclients.DynamodbClient

	SupplierCardsTable               string
	SupplierCardsTable_IsActiveIndex string
}

func CreateSupplierCardsService(ctx context.Context, logger *log.Logger, ddbClient awsclients.DynamodbClient) *SupplierCardsService {
	return &SupplierCardsService{
		ctx:            ctx,
		logger:         logger,
		dynamodbClient: ddbClient,
	}
}

type AllCards struct {
	ActiveCards   []SupplierCards `json:"ActiveCards"`
	InactiveCards []SupplierCards `json:"InactiveCards"`
	ExpiredCards []SupplierCards `json:"ExpiredCards"`
}

func (s *SupplierCardsService) GetAllCards() (AllCards, error) {

	allCards := AllCards{}

	// 1. Get All Active Cards
	queryGetActiveCards := "SELECT CardId, CardDesc, ExpiryDate, IsActive, CardType, CardName FROM \"" + s.SupplierCardsTable + "\".\"" + s.SupplierCardsTable_IsActiveIndex + "\" WHERE IsActive = 'ACTIVE' ORDER BY CardType ASC"
	activeCardsData, err := s.ExecuteCardsQueryDDB(queryGetActiveCards)
	if err != nil {
		return AllCards{}, err
	}
	allCards.ActiveCards = activeCardsData

	// 2. Get All Inactive Cards
	queryGetInActiveCards := "SELECT CardId, CardDesc, ExpiryDate, IsActive, CardType, CardName FROM \"" + s.SupplierCardsTable + "\".\"" + s.SupplierCardsTable_IsActiveIndex + "\" WHERE IsActive = 'INACTIVE' ORDER BY CardType ASC"
	inActiveCardsData, err := s.ExecuteCardsQueryDDB(queryGetInActiveCards)
	if err != nil {
		return AllCards{}, err
	}
	allCards.InactiveCards = inActiveCardsData
	
	// 3. Get All Expired Cards
	queryGetExpiredCards := "SELECT CardId, CardDesc, ExpiryDate, IsActive, CardType, CardName FROM \"" + s.SupplierCardsTable + "\".\"" + s.SupplierCardsTable_IsActiveIndex + "\" WHERE IsActive = 'EXPIRED' ORDER BY CardType ASC"
	expiredCardsData, err := s.ExecuteCardsQueryDDB(queryGetExpiredCards)
	if err != nil {
		return AllCards{}, err
	}
	allCards.ExpiredCards = expiredCardsData

	return allCards, nil
}

func (s *SupplierCardsService) ExecuteCardsQueryDDB(query string) ([]SupplierCards, error) {

	output, err := s.dynamodbClient.ExecuteStatement(s.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(query),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		s.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []SupplierCards{}, err
	}
	if len(output.Items) == 0 {
		s.logger.Printf("No Items found for cards")
		return []SupplierCards{}, nil
	}

	SupplierCardsData := []SupplierCards{}
	for _, stageItem := range output.Items {
		suppCard := SupplierCards{}
		err = attributevalue.UnmarshalMap(stageItem, &suppCard)
		if err != nil {
			s.logger.Printf("Couldn't unmarshal data  Error : %v", err)
			return []SupplierCards{}, err
		}
		// Append data to the overall branch data
		SupplierCardsData = append(SupplierCardsData, suppCard)
	}

	return SupplierCardsData, nil
}

func (s *SupplierCardsService) GetCardDetails(cardId string) (SupplierCards, error) {

	getItemInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"CardId": &dynamodb_types.AttributeValueMemberS{Value: cardId},
		},
		TableName:      aws.String(s.SupplierCardsTable),
		ConsistentRead: aws.Bool(true),
	}

	output, err := s.dynamodbClient.GetItem(s.ctx, &getItemInput)
	if err != nil {
		s.logger.Printf("Get SupplierBranch Failed with error :%v", err)
		return SupplierCards{}, err
	}
	cardData := SupplierCards{}

	err = attributevalue.UnmarshalMap(output.Item, &cardData)
	if err != nil {
		s.logger.Printf("Get SupplierBranch Unmarshal failed with error :%v", err)
		return SupplierCards{}, err
	}

	return cardData, nil
}

// -----_ DML Ops on Supplier Cards -------

func (s *SupplierCardsService) CreateSupplierCards(card SupplierCards) error {
	if card.CardId == "" {
		card.CardId = utils.GenerateRandomString(10)
	}

	av, err := attributevalue.MarshalMap(card)
	if err != nil {
		s.logger.Printf("Failed to marshal supplier branch: %v", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName:           aws.String(s.SupplierCardsTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(CardId)"), // To ensure it does not create a new branch by overriding existing one.
	}

	_, err = s.dynamodbClient.PutItem(s.ctx, input)
	if err != nil {
		s.logger.Printf("Failed to put item in DynamoDB: %v", err)
		return err
	}

	return nil
}

func (s *SupplierCardsService) UpdateSupplierCard(card SupplierCards) error {
	if card.CardId == "" {
		return errors.New("branchId is required for update")
	}

	av, err := attributevalue.MarshalMap(card)
	if err != nil {
		s.logger.Printf("Failed to marshal supplier branch: %v", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName:           aws.String(s.SupplierCardsTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(CardId)"), // to ensure it updates an existing CardId
	}

	_, err = s.dynamodbClient.PutItem(s.ctx, input)
	if err != nil {
		s.logger.Printf("Failed to put item in DynamoDB: %v", err)
		return err
	}

	return nil
}

// Delete a Branch only after making it InActive
func (s *SupplierCardsService) DeleteSupplierCard(cardId string) error {
	if cardId == "" {
		return errors.New("branchId is required for delete")
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(s.SupplierCardsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"CardId": &dynamodb_types.AttributeValueMemberS{Value: cardId},
		},
		ConditionExpression: aws.String("IsActive = :InActive"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			"InActive": &dynamodb_types.AttributeValueMemberS{Value: BRANCH_ISACTIVE_FALSE},
		},
	}

	_, err := s.dynamodbClient.DeleteItem(s.ctx, input)
	if err != nil {
		s.logger.Printf("Failed to delete item from DynamoDB: %v", err)
		return err
	}

	return nil
}
