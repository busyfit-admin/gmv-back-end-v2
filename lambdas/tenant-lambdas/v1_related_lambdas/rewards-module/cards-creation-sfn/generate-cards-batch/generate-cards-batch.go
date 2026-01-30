package main

import (
	"context"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/aws/aws-lambda-go/lambda"

	clients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	complib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	ddbClient clients.DynamodbClient

	handleCardSVC complib.HandleCardService

	CompanyId string // to be assigned when companyId creation is handled
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "generate-cards-batch")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	sfnclient := sfn.NewFromConfig(cfg)

	handleCardSVC := complib.CreateHandleCardService(ctx, ddbclient, sfnclient, logger,
		os.Getenv("CARDS_TRACKER_TABLE"),
		os.Getenv("CARDS_TABLE"))

	svc := Service{
		ctx:       ctx,
		logger:    logger,
		ddbClient: ddbclient,

		handleCardSVC: *handleCardSVC,
	}

	lambda.Start(svc.handleGenerateCards)
}

func (svc *Service) handleGenerateCards(ctx context.Context, cardCreationBatches complib.GenerateCardsBatchInput) (complib.GenerateCardsBatchOutput, error) {

	svc.ctx = ctx

	svc.logger.Printf("2. Generating Cards for batch: %v, TrackingId: %v", cardCreationBatches.BatchId, cardCreationBatches.TrackingId)
	responseOutput, err := svc.GenerateCards(cardCreationBatches)
	if err != nil {
		svc.logger.Printf("3. Failed to Generate Cards for batch: %v, TrackingId: %v,  error: %v", cardCreationBatches.BatchId, cardCreationBatches.TrackingId, err)
		//  Update the Tracking DDB Table with the BatchId status
		trackerInfo := complib.CardsCreationTracker{
			JobId:                 cardCreationBatches.TrackingId,
			CardId:                cardCreationBatches.CardId,
			BatchId:               cardCreationBatches.BatchId,
			JobStatus:             complib.JOB_STATUS_FAILED,
			LastModifiedTimestamp: utils.GenerateTimestamp(),
		}
		err = svc.handleCardSVC.UpdateCardsCreationTrackingDDB(trackerInfo)

		return responseOutput, err
	}

	return responseOutput, nil
}

func (svc *Service) GenerateCards(BatchInput complib.GenerateCardsBatchInput) (complib.GenerateCardsBatchOutput, error) {

	svc.logger.Printf("5. Generating Cards for batch: %v, TrackingId: %v", BatchInput.BatchId, BatchInput.TrackingId)
	var ddbCardsTableInputData []types.WriteRequest
	for i := 0; i < BatchInput.NumberOfCards; i++ {

		// a. Generate Unique Card Number based on the BatchInput Provided
		id := utils.GenerateRandomNumber(6)
		cardUniqueId := BatchInput.CardPrefix + BatchInput.CardId + id

		svc.logger.Printf("6. Generated Card Number: %v", cardUniqueId)
		ddbItem, err := attributevalue.MarshalMap(complib.CompanyCards{
			CardNumber: cardUniqueId,
			CardId:     BatchInput.CardId,
			CardType:   BatchInput.CardType,
			CardStatus: complib.CARD_ISACTIVE_TRUE,
		})
		if err != nil {
			svc.logger.Printf("7. Failed to Marshal Card Number: %v, error: %v", cardUniqueId, err)
			return complib.GenerateCardsBatchOutput{}, err
		}
		// b. Collect all the Unique IDs in the ddbCardsTableInputData
		ddbCardsTableInputData = append(ddbCardsTableInputData, types.WriteRequest{PutRequest: &types.PutRequest{Item: ddbItem}})
	}

	// 2. Add the cards Data to the DDB table using the Batch Write Op ( https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_BatchWriteItem.html )

	svc.logger.Printf("8. Adding Cards to DDB table: %v", BatchInput)
	err := svc.handleCardSVC.BatchWriteCardsToDDB(ddbCardsTableInputData, BatchInput.TrackingId, BatchInput.CardId, BatchInput.BatchId)
	if err != nil {
		return complib.GenerateCardsBatchOutput{}, err
	}

	return complib.GenerateCardsBatchOutput{
		OverallJobStatus: complib.JOB_STATUS_COMPLETED,
	}, nil
}
