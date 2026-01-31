// First Lambda under the Cards Creation Step function Logic under the Company Related APIs

package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/aws/aws-lambda-go/lambda"

	complib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
	libutils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	cardMetaDataSVC complib.CompanyCardsMetadataService
	handleCardSVC   complib.HandleCardService
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "cards-meta-data")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	// Cards MetaData Svc
	cardsMetaDataSvc := complib.CreateCompanyCardsMetadataService(ctx, logger, ddbclient, nil)
	cardsMetaDataSvc.CompanyCardsMetaDataTable = os.Getenv("CARDS_METADATA_TABLE")

	handleCardSvc := complib.CreateHandleCardService(ctx, ddbclient, nil, logger, os.Getenv("CARDS_TRACKER_TABLE"), "")

	svc := Service{
		ctx:    ctx,
		logger: logger,

		cardMetaDataSVC: *cardsMetaDataSvc,
		handleCardSVC:   *handleCardSvc,
	}

	lambda.Start(svc.handleAPIRequests)
}

func (svc *Service) handleAPIRequests(ctx context.Context, input complib.CardsMetaDataInput) (complib.CardsMetaDataOutput, error) {

	svc.ctx = ctx

	output, err := svc.generateCardBatches(input)
	if err != nil {
		svc.logger.Printf("Error generating card batches: %v", err)
		return complib.CardsMetaDataOutput{}, err
	}

	return output, nil
}

func (svc *Service) generateCardBatches(input complib.CardsMetaDataInput) (complib.CardsMetaDataOutput, error) {

	// 1. Perfom checks on CardsMetaData Table.
	cardMetaData, err := svc.cardMetaDataSVC.GetMetaData(input.CardId, input.CardType)
	if err != nil {
		svc.logger.Printf("Failed to get the Card MetaData info : error: %v", err)
	}

	// 2.
	// Todo for future to get the cardprefix and cardtype from cardsmeta data table
	cardPrefix := "33300"

	// Calculate the number of batches
	numBatches := calculateNumBatches(input.CardsOrderQuantity, complib.DEFAULT_CARDS_PER_BATCH)

	// Generate Cards in batches
	cardCreationBatches, err := svc.generateCardCreationBatches(input, cardPrefix, cardMetaData.CardType, numBatches)
	if err != nil {
		svc.logger.Printf("Failed to generate card creation batches: %v", err)
		return complib.CardsMetaDataOutput{}, err
	}
	// Create CardsMetaDataOutput
	output := complib.CardsMetaDataOutput{
		TrackingId:           input.TrackingId,
		CardsCreationBatches: cardCreationBatches,
	}

	svc.logger.Printf("1. Generated %d batches of cards for TrackingId: %s", numBatches, input.TrackingId)

	return output, nil
}

// Function to calculate numBatches
func calculateNumBatches(cardsOrderQuantity, cardsPerBatch int) int {
	return (cardsOrderQuantity + cardsPerBatch - 1) / cardsPerBatch
}

// generate cards creation in batches
func (svc *Service) generateCardCreationBatches(input complib.CardsMetaDataInput, cardPrefix string, cardType string, numBatches int) ([]complib.CardCreationBatch, error) {
	cardCreationBatches := make([]complib.CardCreationBatch, 0)

	for batchNumber := 1; batchNumber <= numBatches; batchNumber++ {
		batchID := fmt.Sprintf("batch%d", batchNumber)
		numCards := int(math.Min(float64(complib.DEFAULT_CARDS_PER_BATCH), float64(input.CardsOrderQuantity)))

		// Create a batch with common properties
		batch := complib.CardCreationBatch{
			TrackingId:    input.TrackingId,
			BatchId:       batchID,
			NumberOfCards: numCards,
			CardPrefix:    cardPrefix,
			CardId:        input.CardId,
			CardType:      cardType,
		}

		cardCreationBatches = append(cardCreationBatches, batch)

		// Update input.CardsOrderQuantity for the next iteration
		input.CardsOrderQuantity -= numCards

		cardsCreationTrackerData := complib.CardsCreationTracker{
			JobId:                 input.TrackingId,
			BatchId:               batchID,
			NumberOfCards:         numCards,
			CardId:                input.CardId,
			JobStatus:             complib.JOB_STATUS_INPRG,
			LastModifiedTimestamp: libutils.GenerateTimestamp(),
		}
		err := svc.handleCardSVC.UpdateCardsCreationTrackingDDB(cardsCreationTrackerData)
		if err != nil {
			svc.logger.Print("Error Updating Tracking DDB Table:", err)
			return []complib.CardCreationBatch{}, nil
		}

		if input.CardsOrderQuantity <= 0 {
			break
		}
	}

	return cardCreationBatches, nil
}
