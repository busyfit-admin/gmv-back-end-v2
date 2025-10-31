package Companylib

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"

	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	// adminlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/admin-lib"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

// Tracks the Cards Creation Process and provides realtime updates to the Client
type CardsCreationTracker struct {
	JobId                 string `dynamodbav:"JobId" json:"JobId"`
	BatchId               string `dynamodbav:"BatchId" json:"BatchId"`
	NumberOfCards         int    `dynamodbav:"NumberOfCards" json:"NumberOfCards"`
	CardId                string `dynamodbav:"CardId" json:"CardId"`
	JobStatus             string `dynamodbav:"JobStatus" json:"JobStatus"`
	LastModifiedTimestamp string `dynamodbav:"LastModifiedTimestamp" json:"LastModifiedTimestamp"`
}

// -------------------- Cards Tracking Functions -------------

type HandleCardService struct {
	ctx                context.Context
	dynamodbClient     awsclients.DynamodbClient
	stepfunctionClient awsclients.StepFunctionClient
	logger             *log.Logger

	CardsCreationTracker                   string
	CardsCreation_JobIdStartTimestampIndex string
	CardsTable                             string
}

func CreateHandleCardService(ctx context.Context, ddbClient awsclients.DynamodbClient, sfnclient awsclients.StepFunctionClient, logger *log.Logger, CardsCreationTracker string, CardsTable string) *HandleCardService {
	return &HandleCardService{
		ctx:                  ctx,
		dynamodbClient:       ddbClient,
		stepfunctionClient:   sfnclient,
		logger:               logger,
		CardsCreationTracker: CardsCreationTracker,
		CardsTable:           CardsTable,
	}
}

// --------------------------------------------Tracking Related Functions ----------

/*

for below:
type CardsCreationTracker struct {
	JobId               string    `dynamodbav:"JobId"` // PK
	BatchId             string    `dynamodbav:"BatchId"` // HK
	CardId            string    `dynamodbav:"CardId"`             // PK ( INDEX )
	JobStatus              string    `dynamodbav:"JobStatus"`
	LastModifiedTimestamp      time.Time `dynamodbav:"LastModifiedTimestamp"`   // SK (INDEX )
}

we are entering multiple items for each JobID, as there can be multiple Batches for each JobID

for ex:

in the table cards Tracker:
row# 		JobID  	BatchID		CardId				JobStatus
1 			123			1			GEN888			 InProg
2			123			2			GEN888			 Completed
3			123			3			GEN888			Completed


Steps to handle this :

1) When Customer does a GetCardTracking API Call, we need to perform Query on the DDB table

2) check if all the items in the query output are Completed or still in progress or Failed

3) respond accordingly to the API request

*/

// Used by Handle create Cards lambda to track the Job
func (svc *HandleCardService) GetCardTrackingDetails(JobId string) (float64, error) {

	var CompletionPercentage float64
	CompletionPercentage = 0

	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.CardsCreationTracker),
		KeyConditionExpression: aws.String("JobId = :JobId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":JobId": &dynamodb_types.AttributeValueMemberS{Value: JobId},
		},
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryInput)
	if err != nil {
		svc.logger.Printf("Query on tracker table failed with error :%v", err)
		return -1, err
	}
	// Returns nil if Employee Data not found
	if output.Count == 0 {
		return 0, nil
	}

	CompletedBatches := 0

	for i := 0; i < len(output.Items); i++ {
		TrackerData := CardsCreationTracker{}
		err = attributevalue.UnmarshalMap(output.Items[i], &TrackerData)
		if err != nil {
			svc.logger.Printf("Could not Unmarshal Tracker Data for TrackingId:" + JobId + " Failed with Error: " + err.Error())
			return -1, err
		}

		if TrackerData.JobStatus == "COMPLETED" {
			CompletedBatches += 1
		}
	}

	CompletionPercentage = (float64(CompletedBatches) / float64(len(output.Items))) * 100

	return CompletionPercentage, nil
}

// Get OrderHistory of the CardsCreation Job
type CardsCreationOrderHistory struct {
	JobId         string `json:"JobId"`
	OverallStatus string `json:"OverallStatus"`
	TotalCards    int    `json:"TotalCards"`
	JobTimestamp  string `json:"JobTimestamp"`
	//BatchesData   []CardsCreationTracker `json:"BatchesData"`
}

func (svc *HandleCardService) GetCardsCreationOrderHistory(cardId string) ([]CardsCreationOrderHistory, error) {

	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.CardsCreationTracker),
		IndexName:              aws.String(svc.CardsCreation_JobIdStartTimestampIndex),
		KeyConditionExpression: aws.String("CardId = :CardId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":CardId": &dynamodb_types.AttributeValueMemberS{Value: cardId},
		},
		ScanIndexForward: aws.Bool(true),
	}

	// query to DDB execution
	output, err := svc.dynamodbClient.Query(svc.ctx, &queryInput)
	if err != nil {
		svc.logger.Printf("Error Querying the CardsCreationTracker : %v", err)
	}
	if output.Count == 0 {
		return []CardsCreationOrderHistory{}, nil
	}

	return filterJobStatusFromDDBOutput(output.Items), nil
}

func filterJobStatusFromDDBOutput(ddbInput []map[string]dynamodb_types.AttributeValue) []CardsCreationOrderHistory {
	// A map to group batches by JobId
	jobMap := make(map[string][]CardsCreationTracker)

	// Unmarshal and group
	for _, jobItem := range ddbInput {
		var jobItemData CardsCreationTracker
		err := attributevalue.UnmarshalMap(jobItem, &jobItemData)
		if err != nil {
			continue // or handle error
		}
		jobMap[jobItemData.JobId] = append(jobMap[jobItemData.JobId], jobItemData)
	}

	// Create the final output
	var cardsOrderHistoryData []CardsCreationOrderHistory
	for jobId, batches := range jobMap {
		overallStatus, totalCards := deriveOverallStatus(batches)
		historyItem := CardsCreationOrderHistory{
			JobId:         jobId,
			OverallStatus: overallStatus,
			TotalCards:    totalCards,
			JobTimestamp:  batches[0].LastModifiedTimestamp,
		}
		cardsOrderHistoryData = append(cardsOrderHistoryData, historyItem)
	}

	return cardsOrderHistoryData
}

// Utility to derive overall status from batch statuses
func deriveOverallStatus(batches []CardsCreationTracker) (string, int) {
	statusSet := make(map[string]bool)
	var totalCards = 0
	for _, batch := range batches {
		statusSet[batch.JobStatus] = true
		totalCards += batch.NumberOfCards
	}

	if len(statusSet) == 1 && statusSet["COMPLETED"] {
		for status := range statusSet {
			return status, totalCards
		}
	}

	if len(statusSet) >= 1 && statusSet["COMPLETED"] {
		return "Partial Success", totalCards
	}

	return "FAILED", totalCards
}

// Used by Handle create Cards lambda to track the Job
func (svc *HandleCardService) UpdateCardsCreationTrackingDDB(cardsOrderData CardsCreationTracker) error {

	if cardsOrderData.LastModifiedTimestamp == "" {
		cardsOrderData.LastModifiedTimestamp = utils.GenerateTimestamp()
	}

	switch cardsOrderData.JobStatus {

	case JOB_STATUS_INPRG:
		updateItemInput := dynamodb.UpdateItemInput{
			Key: map[string]dynamodb_types.AttributeValue{
				"JobId":   &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.JobId},
				"BatchId": &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.BatchId},
			},
			TableName:        aws.String(svc.CardsCreationTracker),
			UpdateExpression: aws.String("SET JobStatus = :JobStatus, LastModifiedTimestamp = :LastModifiedTimestamp, CardId = :CardId, NumberOfCards = :NumberOfCards"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":CardId":                &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.CardId},
				":NumberOfCards":         &dynamodb_types.AttributeValueMemberN{Value: fmt.Sprintf("%d", cardsOrderData.NumberOfCards)},
				":JobStatus":             &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.JobStatus},
				":LastModifiedTimestamp": &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.LastModifiedTimestamp},
			},
		}

		svc.logger.Printf("Updating Tracking Table with JobId: %s, BatchId: %s, CardId: %s, JobStatus: %s", cardsOrderData.JobId, cardsOrderData.BatchId, cardsOrderData.CardId, cardsOrderData.JobStatus)

		_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
		if err != nil {
			svc.logger.Printf("100. UpdateItem failed with error :%v", err)
			return err
		}

		return nil

	case JOB_STATUS_COMPLETED:
		updateItemInput := dynamodb.UpdateItemInput{
			Key: map[string]dynamodb_types.AttributeValue{
				"JobId":   &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.JobId},
				"BatchId": &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.BatchId},
			},
			TableName:        aws.String(svc.CardsCreationTracker),
			UpdateExpression: aws.String("SET JobStatus = :JobStatus, LastModifiedTimestamp = :LastModifiedTimestamp, CardId = :CardId, NumberOfCards = :NumberOfCards"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":CardId":                &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.CardId},
				":NumberOfCards":         &dynamodb_types.AttributeValueMemberN{Value: fmt.Sprintf("%d", cardsOrderData.NumberOfCards)},
				":JobStatus":             &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.JobStatus},
				":LastModifiedTimestamp": &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.LastModifiedTimestamp},
			},
		}
		svc.logger.Printf("Updating Tracking Table with JobId: %s, BatchId: %s, CardId: %s, JobStatus: %s, Completion: %s", cardsOrderData.JobId, cardsOrderData.BatchId, cardsOrderData.CardId, cardsOrderData.JobStatus, cardsOrderData.LastModifiedTimestamp)
		_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
		if err != nil {
			svc.logger.Printf("100. UpdateItem failed with error :%v", err)
			return err
		}
		return nil

	case JOB_STATUS_FAILED:
		updateItemInput := dynamodb.UpdateItemInput{
			Key: map[string]dynamodb_types.AttributeValue{
				"JobId":   &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.JobId},
				"BatchId": &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.BatchId},
			},
			TableName:        aws.String(svc.CardsCreationTracker),
			UpdateExpression: aws.String("SET JobStatus = :JobStatus, LastModifiedTimestamp = :LastModifiedTimestamp, CardId = :CardId"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":CardId":                &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.CardId},
				":JobStatus":             &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.JobStatus},
				":LastModifiedTimestamp": &dynamodb_types.AttributeValueMemberS{Value: cardsOrderData.LastModifiedTimestamp},
			},
		}
		svc.logger.Printf("Updating Tracking Table with JobId: %s, BatchId: %s, CardId: %s, JobStatus: %s, Completion: %s", cardsOrderData.JobId, cardsOrderData.BatchId, cardsOrderData.CardId, cardsOrderData.JobStatus, cardsOrderData.LastModifiedTimestamp)
		_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
		if err != nil {
			svc.logger.Printf("100. UpdateItem failed with error :%v", err)
			return err
		}
		return nil
	default:
		svc.logger.Printf("100. Incorrect Job Status Update")
		return fmt.Errorf("incorrect Job Status Update")
	}

}

// --------------------------------------------Batch Write Functions used inside Step Function ----------

func (svc *HandleCardService) BatchWriteCardsToDDB(writeReq []dynamodb_types.WriteRequest, requestId string, cardId string, batchId string) error {

	batches := SplitWriteRequestsIntoBatches(writeReq)

	for i, ddbBatchWrite := range batches {
		_, err := svc.dynamodbClient.BatchWriteItem(svc.ctx,
			&dynamodb.BatchWriteItemInput{
				RequestItems: map[string][]dynamodb_types.WriteRequest{
					svc.CardsTable: ddbBatchWrite,
				},
			})

		if err != nil {
			return fmt.Errorf("error executing BatchWriteItem for batch %d: %v", i+1, err)
		}
	}

	cardsCreationTrackerData := CardsCreationTracker{
		JobId:                 string(requestId),
		CardId:                string(cardId),
		BatchId:               string(batchId),
		NumberOfCards:         len(writeReq),
		JobStatus:             JOB_STATUS_COMPLETED,
		LastModifiedTimestamp: utils.GenerateTimestamp(),
	}
	err := svc.UpdateCardsCreationTrackingDDB(cardsCreationTrackerData)
	if err != nil {
		return nil
	}

	return nil
}

// --------------------------------------------< > ----------

// Helper function

// Split the batch write inputs to max of 25 as the batch write op only takes 25 ops per one API call.
const MAX_DDB_BATCH_SIZE = 25

func SplitWriteRequestsIntoBatches(writeRequests []dynamodb_types.WriteRequest) [][]dynamodb_types.WriteRequest {

	var batches [][]dynamodb_types.WriteRequest
	for i := 0; i < len(writeRequests); i += MAX_DDB_BATCH_SIZE {
		end := i + MAX_DDB_BATCH_SIZE
		if end > len(writeRequests) {
			end = len(writeRequests)
		}
		batches = append(batches, writeRequests[i:end])
	}

	return batches
}
