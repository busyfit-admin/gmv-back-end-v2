/*
The "manage-cards-template" program is an AWS Lambda function designed to handle events related to managing card templates.
The program performs operations like creating, retrieving card templates in a DynamoDB table.
High-Level Steps:


1) Handling HTTP POST Requests (`handlePostMethod`):
   - Generates a new UUID for the object key (image filename).
   - Uploads the image data to the specified S3 bucket.
   - Updates the metadata table in DynamoDB with details of the new card template.

2) Handling HTTP GET Requests (`handleGetMethod`)
   - Retrieves headers from the incoming request.
   - Retrieves card metadata from the DynamoDB table based on the provided CardId.
   - Retrieves a pre-signed URL for the card template image from the S3 bucket.


*/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
	libutils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	employeeSvc companylib.EmployeeService
	cdnSvc      companylib.CDNService
	contentSvc  companylib.TenantUploadContentService
	transferSvc companylib.CardsTransferService

	// Cards MetaData Service
	cardsMetaSvc    companylib.CompanyCardsMetadataService
	companyCardsSvc companylib.CompanyCardsService
	// Cards Creation Service
	cardCreationSvc companylib.HandleCardService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("RewardsAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-cards-template")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	dynamodbClient := dynamodb.NewFromConfig(cfg)
	secretsClient := secretsmanager.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	logger := log.New(os.Stdout, "", log.LstdFlags)

	employeeSvc := companylib.CreateEmployeeService(ctx, dynamodbClient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	// Create Cards Meta Data Service
	cardsMetaSvc := companylib.CreateCompanyCardsMetadataService(ctx, logger, dynamodbClient, s3Client)
	cardsMetaSvc.CompanyCardsMetaDataTable = os.Getenv("COMPANY_CARDS_META_DATA_TABLE")
	cardsMetaSvc.CardType_Index = os.Getenv("CARD_TYPE_INDEX")

	// company cards service creation
	companyCardsSvc := companylib.CreateCompanyCardsService(ctx, logger, dynamodbClient)
	companyCardsSvc.CompanyCardsTable = os.Getenv("REWARDS_CARDS_TABLE")
	companyCardsSvc.CardId_Status_Index = os.Getenv("CARD_ID_STATUS_INDEX") // RewardsCardsTable Index
	companyCardsSvc.CompanyCardsMetadataTable = os.Getenv("COMPANY_CARDS_META_DATA_TABLE")
	companyCardsSvc.CardType_Index = os.Getenv("CARD_TYPE_INDEX") // CardsMetaData Index

	transferSvc := companylib.CreateCardsTransferService(ctx, logger, dynamodbClient)
	transferSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	transferSvc.RewardsTransferLogsTable = os.Getenv("REWARDS_TRANSFER_LOGS_TABLE")
	transferSvc.CompanyCardsTable = os.Getenv("REWARDS_CARDS_TABLE")

	// Cards Creation Tracker Svc
	//CardId-StartTimestamp_Index
	HandleCardService := companylib.CreateHandleCardService(ctx, dynamodbClient, nil, logger, os.Getenv("CARDS_TRACKER_TABLE"), "")
	HandleCardService.CardsCreation_JobIdStartTimestampIndex = os.Getenv("CARD_TRACKER_CARDID_TIMESTAMP_INDEX")

	// Here we are creating a CDN Service
	cdnSvc := companylib.CDNService{}
	err = cdnSvc.CreateCDNService(ctx, logger, secretsClient, os.Getenv("SECRETS_CND_PK_ARN"), os.Getenv("PUBLIC_KEY_ID"))
	if err != nil {
		log.Fatalf("Error creating CDN Service: %v\n", err)
	}
	cdnSvc.CDNDomain = os.Getenv("CDN_DOMAIN")

	// Content Service
	contentSvc := companylib.CreateTenantUploadContentService(ctx, s3Client, logger)
	contentSvc.S3Bucket = os.Getenv("S3_BUCKET")

	svc := Service{
		ctx:             ctx,
		logger:          logger,
		employeeSvc:     *employeeSvc,
		cdnSvc:          cdnSvc,
		cardsMetaSvc:    *cardsMetaSvc,
		companyCardsSvc: *companyCardsSvc,
		transferSvc:     *transferSvc,
		cardCreationSvc: *HandleCardService,
	}

	lambda.Start(svc.handleCompanyEvents)
}

func (svc *Service) handleCompanyEvents(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	switch request.HTTPMethod {

	case "POST":
		return svc.handlePostMethod(request)
	case "GET":
		return svc.handleGetMethod(request)
	case "PATCH":
		return svc.handlePatchMethod(request)
	default:
		svc.logger.Printf("Request type not defined for ManageCardTemplate: %s", request.HTTPMethod)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

}

const (
	POST_CREATE_CARD_TEMPLATE = "create-card-template"
	CARD_CHECKOUT             = "card-checkout"
	REDEEM_CARD               = "redeem-card"
)

func (svc *Service) handlePostMethod(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1. Get Req Headers
	post_type := request.Headers["post_type"]

	switch post_type {
	case POST_CREATE_CARD_TEMPLATE:
		return svc.createCardTemplate(request)
	case REDEEM_CARD:
		return svc.RedeemCard(request)
	default:
		svc.logger.Printf("Request type not defined for ManageCardTemplate: %s", post_type)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

}

type CreateCardTemplateInput struct {

	// Card Name
	CardName string `json:"CardName"`
	CardDesc string `json:"CardDesc"`
	CardType string `json:"CardType"`

	CardPoints int `json:"CardCost"`

	Validity int `json:"Validity"`

	TermsAndConditions string `json:"TermsAndConditions"`
	RedemptionLogic    string `json:"RedemptionLogic"`

	CardImages []string `json:"CardImages"`
}
type CreateCardTemplateOutput struct {
	CardId string `json:"CardId"`
}

func (svc *Service) createCardTemplate(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1. Get Req Body
	var createCardTemplateInput CreateCardTemplateInput
	if err := json.Unmarshal([]byte(request.Body), &createCardTemplateInput); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Headers: RESP_HEADERS}, err
	}

	// 1) Authorization at User Level and Check if user to create the card is Admin or Rewards Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	newUUID := "card-" + libutils.GenerateRandomString(8)

	// 2. Generate Image Map and Upload Images to S3
	imageMap, imageKeys := generateImageMap(createCardTemplateInput.CardImages, newUUID)
	err = svc.contentSvc.UploadMultipleContentsToS3_Base64Content(imageMap)
	if err != nil {
		svc.logger.Printf("Error uploading images to S3: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// Create an Entry in the Cards Meta Data Table
	err = svc.cardsMetaSvc.CreateMetaData(companylib.CompanyCardsMetaDataTable{
		CardId:             newUUID,
		CardName:           createCardTemplateInput.CardName,
		CardDesc:           createCardTemplateInput.CardDesc,
		CardType:           createCardTemplateInput.CardType,
		CardPoints:         createCardTemplateInput.CardPoints,
		Validity:           createCardTemplateInput.Validity,
		TermsAndConditions: createCardTemplateInput.TermsAndConditions,
		RedemptionLogic:    createCardTemplateInput.RedemptionLogic,
		CardImages:         imageKeys,
	})
	if err != nil {
		svc.logger.Printf("Error creating card metadata: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// Return the Card ID
	resp := CreateCardTemplateOutput{
		CardId: newUUID,
	}
	respBytes, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		Body:       string(respBytes),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil

}

// Generate random string for Image Key along with the engagement ID
func generateImageKey(cardId string, countId string) string {
	return fmt.Sprintf("card-templates/%s/images/img-%s-%s", cardId, countId, libutils.GenerateRandomString(8))
}

// Generate map[string]string (ie map[image_key]base64_data from input []string ( ie []base64_data ), image key is generated using the generateImageKey function
func generateImageMap(images []string, cardId string) (map[string]string, []string) {
	imageMap := make(map[string]string)
	var imageKeys []string
	for key, image := range images {
		imageKey := generateImageKey(cardId, fmt.Sprintf("%d", key))
		imageMap[imageKey] = image
		imageKeys = append(imageKeys, imageKey)
	}
	// Return the image map and the keys
	return imageMap, imageKeys
}

const (
	GET_CARD_TEMPLATE       = "get-card-template"
	GET_ALL_CARD_TEMPLATE   = "get-all-card-template"
	GET_ALL_CARDS           = "get-all-cards"
	GET_CARDS_ORDER_HISTORY = "get-cards-order-history"
)

// Get Requests Handler
func (svc *Service) handleGetMethod(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1. Get Req Headers
	get_type := request.Headers["get_type"]

	switch get_type {
	case GET_CARD_TEMPLATE:
		return svc.getCardTemplate(request)
	case GET_ALL_CARD_TEMPLATE:
		return svc.getAllCardTemplates(request)
	case GET_ALL_CARDS:
		return svc.getAllCards(request)
	case GET_CARDS_ORDER_HISTORY:
		return svc.getCardsOrderHistory(request)

	default:
		svc.logger.Printf("Request type not defined for ManageCardTemplate: %s", get_type)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}
}

type GetCardTemplateInput struct {
	CardId   string `json:"CardId"`
	CardType string `json:"CardType"`
}

func (svc *Service) getCardTemplate(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1. Get Req Body
	var getCardTemplateInput GetCardTemplateInput
	if err := json.Unmarshal([]byte(request.Body), &getCardTemplateInput); err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500, Headers: RESP_HEADERS}, err
	}

	// 1) Authorization at User Level and Check if user to create the card is Admin or Rewards Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Get Card Metadata
	cardMeta, err := svc.cardsMetaSvc.GetMetaData(getCardTemplateInput.CardId, getCardTemplateInput.CardType)
	if err != nil {
		svc.logger.Printf("Error getting card metadata: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	respBytes, _ := json.Marshal(cardMeta)
	return events.APIGatewayProxyResponse{
		Body:       string(respBytes),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil

}

func (svc *Service) getAllCardTemplates(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user to create the card is Admin or Rewards Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Get Card Metadata
	cardMeta, err := svc.cardsMetaSvc.GetAllCardsMetaData()
	if err != nil {
		svc.logger.Printf("Error getting card metadata: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	respBytes, _ := json.Marshal(cardMeta)
	return events.APIGatewayProxyResponse{
		Body:       string(respBytes),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil

}

type GetAllCardsOutput struct {
	Health   []companylib.CardMetaDataOutputWithCount `json:"Health"`
	WFH      []companylib.CardMetaDataOutputWithCount `json:"WFH"`
	Vacation []companylib.CardMetaDataOutputWithCount `json:"Vacation"`
}

func (svc *Service) getAllCards(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user to create the card is Admin or Rewards Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	healthCards, err := svc.companyCardsSvc.GetCardsByType("RD01")
	if err != nil {
		svc.logger.Printf("Error getting health cards: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	for i := range healthCards {
		healthCards[i].CardMetaData.CardImages = svc.cdnSvc.SignImages(healthCards[i].CardMetaData.CardImages)
	}

	wfhCards, err := svc.companyCardsSvc.GetCardsByType("RD00")
	if err != nil {
		svc.logger.Printf("Error getting wfh cards: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	for i := range wfhCards {
		wfhCards[i].CardMetaData.CardImages = svc.cdnSvc.SignImages(wfhCards[i].CardMetaData.CardImages)
	}

	vacationCards, err := svc.companyCardsSvc.GetCardsByType("RD04")
	if err != nil {
		svc.logger.Printf("Error getting vacation cards: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	for i := range vacationCards {
		vacationCards[i].CardMetaData.CardImages = svc.cdnSvc.SignImages(vacationCards[i].CardMetaData.CardImages)
	}

	resp := GetAllCardsOutput{
		Health:   healthCards,
		WFH:      wfhCards,
		Vacation: vacationCards,
	}

	respBytes, _ := json.Marshal(resp)
	return events.APIGatewayProxyResponse{
		Body:       string(respBytes),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

func (svc *Service) getCardsOrderHistory(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	cardId := request.Headers["cardid"]

	// 1) Authorization at User Level and Check if user to create the card is Admin or Rewards Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	cardOrderHistory, err := svc.cardCreationSvc.GetCardsCreationOrderHistory(cardId)
	if err != nil {
		svc.logger.Printf("Failed with error: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	output, err := json.Marshal(cardOrderHistory)
	if err != nil {
		svc.logger.Printf("Failed with error: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{

		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(output),
	}, nil
}

type RedeemCardInput struct {
	CardId   string `json:"CardId"`
	CardType string `json:"CardType"`
}

func (svc *Service) RedeemCard(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// NOTE: Cards redeem logic needs updated to latest formats
	// use req body instead of headers
	var redeemCardInput RedeemCardInput

	// 1. Unmarshal the req
	svc.logger.Printf("Request Body: %s\n", request.Body)

	if err := json.Unmarshal([]byte(request.Body), &redeemCardInput); err != nil {
		svc.logger.Printf("Error unmarshalling request body: %v\n", err)
		return events.APIGatewayProxyResponse{
			Body:       "Invalid request body. Please provide a valid JSON.",
			Headers:    RESP_HEADERS,
			StatusCode: 400,
		}, nil
	}

	svc.logger.Printf("Unmarshalled RedeemCardInput: %+v\n", redeemCardInput)

	if redeemCardInput.CardId == "" {
		svc.logger.Println("Card ID is missing in the request body.")
		return events.APIGatewayProxyResponse{
			Body:       "Card ID is required in the request body.",
			Headers:    RESP_HEADERS,
			StatusCode: 400,
		}, nil
	}

	// Authorization check
	authData, isAuth, err := svc.employeeSvc.Authorizer(request, "")
	if err != nil {
		svc.logger.Printf("Authorization error: %v\n", err)
	}
	if !isAuth {
		svc.logger.Println("Unauthorized access attempt.")
	}
	if err != nil || !isAuth {
		return events.APIGatewayProxyResponse{
			Body:       "Unauthorized access. Please check your credentials.",
			Headers:    RESP_HEADERS,
			StatusCode: 403,
		}, nil
	}

	svc.logger.Printf("Authorization successful for user: %s\n", authData.Username)

	// Get card data
	cardData, err := svc.companyCardsSvc.GetCardsDataForCheckout(redeemCardInput.CardId)
	if err != nil {
		svc.logger.Printf("Error getting card data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Body:       "Failed to retrieve card data. Please try again later.",
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	svc.logger.Printf("Retrieved Card Data: %+v\n", cardData)

	// Validate card data
	if cardData.CardNumber == "" || cardData.CardStatus != "ACTIVE" {
		svc.logger.Println("Card is either invalid or inactive.")
		return events.APIGatewayProxyResponse{
			Body:       "The card is either invalid or inactive.",
			Headers:    RESP_HEADERS,
			StatusCode: 400,
		}, nil
	}

	// Get employee data
	employeeUserName := authData.Username
	empData, err := svc.employeeSvc.GetEmployeeDataByUserName(employeeUserName)
	if err != nil {
		svc.logger.Printf("Error getting Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Body:       "Failed to retrieve employee data. Please try again later.",
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Printf("Retrieved Employee Data: %+v\n", empData)

	// Check reward points
	rewardsData := empData.RewardsData
	svc.logger.Printf("Rewards Data: %+v\n", rewardsData)
	svc.logger.Println("Card type", cardData.CardType)
	var reward *companylib.EmployeeRewards

	for _, r := range rewardsData {
		svc.logger.Printf("IN LOOP - RewardId: %v | Comparing with: %v\n", r.RewardId, cardData.CardType)
		if r.RewardId == cardData.CardType {
			svc.logger.Println("MATCH FOUND")
			reward = &r
			break
		}
	}

	svc.logger.Println("Checking reward points...")

	svc.logger.Printf("Reward Data: %+v\n", reward)
	if reward == nil || reward.RewardPoints == 0 {
		svc.logger.Println("Insufficient reward points to redeem the card.")
		return events.APIGatewayProxyResponse{
			Body:       "You do not have enough reward points to redeem a card.",
			Headers:    RESP_HEADERS,
			StatusCode: 400,
		}, nil
	}

	// Get card metadata
	cardMetaData, err := svc.cardsMetaSvc.GetMetaData(redeemCardInput.CardId, redeemCardInput.CardType)
	if err != nil {
		svc.logger.Printf("Error getting card metadata: %v\n", err)
		return events.APIGatewayProxyResponse{
			Body:       "Failed to retrieve card details. Please try again later.",
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Printf("Retrieved Card Metadata: %+v\n", cardMetaData)

	if reward.RewardPoints < cardMetaData.CardPoints {
		svc.logger.Println("Reward points are less than required to redeem the card.")
		return events.APIGatewayProxyResponse{
			Body:       "Insufficient reward points to redeem this card.",
			Headers:    RESP_HEADERS,
			StatusCode: 400,
		}, nil
	}

	// Prepare card data for transfer
	rewardCardData := companylib.RewardCards{
		CardId:       redeemCardInput.CardId,
		CardNumber:   cardData.CardNumber,
		CardName:     cardMetaData.CardName,
		CardDesc:     cardMetaData.CardDesc,
		RewardPoints: cardMetaData.CardPoints,
	}

	svc.logger.Printf("Prepared Reward Card Data: %+v\n", rewardCardData)

	transferInput := companylib.CardsTransferInput{
		SourceUserName:      employeeUserName,
		DestinationUserName: cardData.CardNumber,
		RewardType:          cardData.CardType,
		TransferPoints:      cardMetaData.CardPoints,
		CardData:            rewardCardData,
	}

	// Handle card transfer
	err = svc.transferSvc.HandleCardsTransfer(transferInput)
	if err != nil {
		svc.logger.Printf("Error transferring card: %v\n", err)
		return events.APIGatewayProxyResponse{
			Body:       "Card transfer failed. Please try again later.",
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Println("Card redeemed successfully!")
	return events.APIGatewayProxyResponse{
		Body:       "Card redeemed successfully!",
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

const (
	UPDATE_CARD_TEMPLATE = "edit-card-template"
)

func (svc *Service) handlePatchMethod(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	patch_type := request.Headers["patch_type"]
	switch patch_type {
	case UPDATE_CARD_TEMPLATE:
		return svc.updateCardTemplate(request)
	default:
		svc.logger.Printf("Request type not defined for ManageCardTemplate: %s", patch_type)
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}
}

func (svc *Service) updateCardTemplate(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	var updateCardTemplateInput companylib.UpdateCardTemplateInput

	err = json.Unmarshal([]byte(request.Body), &updateCardTemplateInput)
	if err != nil {
		svc.logger.Printf("Error unmarshalling request body: %v\n", err)
		return events.APIGatewayProxyResponse{
			Body:       "Invalid request body. Please provide a valid JSON.",
			Headers:    RESP_HEADERS,
			StatusCode: 400,
		}, nil
	}

	err = svc.cardsMetaSvc.UpdateMetaData(updateCardTemplateInput)
	if err != nil {
		svc.logger.Printf("Error updating card metadata: %v\n", err)
		return events.APIGatewayProxyResponse{
			Body:       "Failed to update card metadata. Please try again later.",
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	svc.logger.Printf("Card metadata updated successfully: %+v\n", updateCardTemplateInput)

	return events.APIGatewayProxyResponse{
		Body:       "Card Template updated successfully.",
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}
