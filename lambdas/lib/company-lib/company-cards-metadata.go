// Card Template design : https://busyfit-devops.atlassian.net/wiki/spaces/SD/pages/20676613/Workflow+3+-+Admin+Portal+Company+Registration#Handling-Card-Templates-in-Company-Portal-%3A

package Companylib

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

type CompanyCardsMetaDataTable struct {
	CardId string `dynamodbav:"CardId" json:"CardId"` // PK, Unique Identifier for the Card
	// Card Type that is selected ( card type can be of either : general, health, climate rewards). These can be only 4 types. ( RD00, RD01, RD02, RD03)
	// Ref Employee's Table for the CardType
	CardType string `dynamodbav:"CardType" json:"CardType"` // Sort Key, Card Type of the Card

	CardName string `dynamodbav:"CardName" json:"CardName"` // Custom name of the card that is entered by the Tenant
	CardDesc string `dynamodbav:"CardDesc" json:"CardDesc"` // Custom description of the card that is entered by the Tenant

	CardImages []string `dynamodbav:"CardImages" json:"CardImages"` // Card Images that are uploaded. Key is the image number and value is the image URL
	CardPoints int      `dynamodbav:"CardPoints" json:"CardPoints"` // Cost of the Card in Dollars

	Validity int `dynamodbav:"Validity" json:"Validity"` // Validity of the Card in days

	TermsAndConditions string `dynamodbav:"TermsAndConditions" json:"TermsAndConditions"` // Terms and Conditions of the Card

	RedemptionLogic string `dynamodbav:"RedemptionLogic" json:"RedemptionLogic"` // Redemption Logic of the Card , 2 Values are possible : Auto or Manual

	// Old params:
	// CompanyName string `dynamodbav:"CompanyName" json:"CompanyName"` // Company Name/CompanyId this card belongs to. Also the sort ID in the DDB
	// CardMetaData string `dynamodbav:"CardMetaData" json:"CardMetaData"` // any meta data identifer of the card
	// AdditionalInfo string `dynamodbav:"additionalInfo" json:"additionalInfo"` // Any additional information / comments on this card type

}

type CompanyCardsMetadataService struct {
	ctx    context.Context
	logger *log.Logger

	dynamodbClient awsclients.DynamodbClient
	s3ObjectClient awsclients.S3Client

	CompanyCardsMetaDataTable string
	CardType_Index            string
	BucketName                string
}

func CreateCompanyCardsMetadataService(
	ctx context.Context,
	logger *log.Logger,

	ddbClient awsclients.DynamodbClient,
	s3Client awsclients.S3Client,
) *CompanyCardsMetadataService {

	return &CompanyCardsMetadataService{
		ctx:    ctx,
		logger: logger,

		dynamodbClient: ddbClient,
		s3ObjectClient: s3Client,
	}
}

type AllCardsOutput struct {
	AllCardsTemplates map[string]CompanyCardsMetaDataTable `json:"AllCardsTemplates"`
}

// Gets all the Card Meta Data from the Table
func (svc *CompanyCardsMetadataService) GetAllCardsMetaData() (AllCardsOutput, error) {

	scanInput := dynamodb.ScanInput{
		TableName: aws.String(svc.CompanyCardsMetaDataTable),
	}

	output, err := svc.dynamodbClient.Scan(svc.ctx, &scanInput)
	if err != nil {
		return AllCardsOutput{}, err
	}

	var CardMetaData []CompanyCardsMetaDataTable
	err = dynamodb_attributevalue.UnmarshalListOfMaps(output.Items, &CardMetaData)
	if err != nil {
		svc.logger.Printf("Unmarshal on the CardMetaData Failed :%v", err)
		return AllCardsOutput{}, err
	}

	CardMetaDataMap := make(map[string]CompanyCardsMetaDataTable)
	for _, Card := range CardMetaData {
		CardMetaDataMap[Card.CardId] = Card
	}

	return AllCardsOutput{
		AllCardsTemplates: CardMetaDataMap,
	}, nil
}

// Get a single Card Meta Data
func (svc *CompanyCardsMetadataService) GetMetaData(CardId string, cardType string) (CompanyCardsMetaDataTable, error) {

	if CardId == "" {
		return CompanyCardsMetaDataTable{}, fmt.Errorf("CardId cannot be empty for GetMeta Data for single Card")
	}

	getItemInput := dynamodb.GetItemInput{
		TableName: aws.String(svc.CompanyCardsMetaDataTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"CardId":   &dynamodb_types.AttributeValueMemberS{Value: CardId},
			"CardType": &dynamodb_types.AttributeValueMemberS{Value: cardType},
		},
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)
	if err != nil {
		return CompanyCardsMetaDataTable{}, err
	}

	var CardMetaData CompanyCardsMetaDataTable
	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &CardMetaData)
	if err != nil {
		svc.logger.Printf("Unmarshal on the CardMetaData Failed :%v", err)
		return CompanyCardsMetaDataTable{}, err
	}

	return CardMetaData, nil
}

// Create a new Card Meta Data
func (svc *CompanyCardsMetadataService) CreateMetaData(Card CompanyCardsMetaDataTable) error {

	if Card.CardId == "" {
		return fmt.Errorf("CardId cannot be empty for CreateMeta Data for single Card")
	}

	CardItem, err := dynamodb_attributevalue.MarshalMap(Card)
	if err != nil {
		svc.logger.Printf("MarshalMap on the CardMetaData Failed :%v", err)
		return err
	}

	// PutItem Input
	putItemInput := dynamodb.PutItemInput{
		TableName: aws.String(svc.CompanyCardsMetaDataTable),
		Item:      CardItem,
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, &putItemInput)
	if err != nil {
		svc.logger.Printf("PutItem on the CardMetaData Failed :%v", err)
		return err
	}

	return nil
}

type UpdateCardTemplateInput struct {
	CardId             string `dynamodbav:"CardId" json:"CardId"`
	CardType           string `dynamodbav:"CardType" json:"CardType"`
	CardName           string `dynamodbav:"CardName" json:"CardName"`
	Validity           int    `dynamodbav:"Validity" json:"Validity"`
	CardDesc           string `dynamodbav:"CardDesc" json:"CardDesc"`
	TermsAndConditions string `dynamodbav:"TermsAndConditions" json:"TermsAndConditions"`
}

func (svc *CompanyCardsMetadataService) UpdateMetaData(Card UpdateCardTemplateInput) error {

	if Card.CardId == "" {
		return fmt.Errorf("CardId cannot be empty for Update Meta Data")
	}

	// PutItem Input
	updateItemInput := dynamodb.UpdateItemInput{
		TableName:        aws.String(svc.CompanyCardsMetaDataTable),
		UpdateExpression: aws.String("SET CardName = :CardName, CardDesc = :CardDesc, Validity = :Validity, TermsAndConditions = :TermsAndConditions"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":CardName":           &dynamodb_types.AttributeValueMemberS{Value: Card.CardName},
			":CardDesc":           &dynamodb_types.AttributeValueMemberS{Value: Card.CardDesc},
			":Validity":           &dynamodb_types.AttributeValueMemberN{Value: fmt.Sprintf("%d", Card.Validity)},
			":TermsAndConditions": &dynamodb_types.AttributeValueMemberS{Value: Card.TermsAndConditions},
		},
		Key: map[string]dynamodb_types.AttributeValue{
			"CardId":   &dynamodb_types.AttributeValueMemberS{Value: Card.CardId},
			"CardType": &dynamodb_types.AttributeValueMemberS{Value: Card.CardType},
		},
		ConditionExpression: aws.String("attribute_exists(CardId) AND attribute_exists(CardType)"),
		ReturnValues:        dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("PutItem on the CardMetaData Failed :%v", err)
		return err
	}

	return nil
}
