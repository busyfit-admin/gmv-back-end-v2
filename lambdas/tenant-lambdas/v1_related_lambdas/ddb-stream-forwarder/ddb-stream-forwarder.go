// we make two different clients for the event bridge w.r.t the employee table streams and
// the appreciation table streams and send evb events to respective clients

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridge_types "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	lib "github.com/busyfit-admin/saas-ddb-data-stack/lambdas/lib/clients"
)

type DDBService struct {
	ctx                   context.Context
	logger                *log.Logger
	EvbClientEmployee     *lib.EVBClientSvc // EventBridge client for employee data
	EvbClientAppreciation *lib.EVBClientSvc // EventBridge client for appreciation data
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "ddb-stream-forwarder")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Initialize the EventBridge client for Employee data
	evbClientEmployee := lib.EVBClientSvc{
		Ctx:             ctx,
		Logger:          logger,
		EVBClient:       eventbridge.NewFromConfig(cfg),
		EventSourceName: os.Getenv("EVB_SOURCE_NAME_EMPLOYEE"),
		EventDetailType: os.Getenv("EVB_DETAIL_TYPE_EMPLOYEE"),
		DestEVBName:     os.Getenv("EVB_NAME"),
		Environment:     os.Getenv("Environment"),
	}

	// Initialize the EventBridge client for Appreciation data
	evbClientAppreciation := lib.EVBClientSvc{
		Ctx:             ctx,
		Logger:          logger,
		EVBClient:       eventbridge.NewFromConfig(cfg),
		EventSourceName: os.Getenv("EVB_SOURCE_NAME_APPRECIATION"),
		EventDetailType: os.Getenv("EVB_DETAIL_TYPE_APPRECIATION"),
		DestEVBName:     os.Getenv("EVB_NAME"),
		Environment:     os.Getenv("Environment"),
	}

	svc := DDBService{
		ctx:                   ctx,
		logger:                logger,
		EvbClientEmployee:     &evbClientEmployee,
		EvbClientAppreciation: &evbClientAppreciation,
	}

	lambda.Start(svc.DDBEventsHandler)
}

func DynamodbEventsToJsonString(ddbData map[string]events.DynamoDBAttributeValue) (string, error) {
	keyValuePair := make(map[string]string)

	for key, value := range ddbData {
		keyValuePair[key] = value.String()
	}

	ddbDataJSON, err := json.Marshal(keyValuePair)
	if err != nil {
		return "", err
	}
	return string(ddbDataJSON), nil
}

func (svc *DDBService) CreateEVBEvent(client *lib.EVBClientSvc, detailString string) eventbridge_types.PutEventsRequestEntry {
	return eventbridge_types.PutEventsRequestEntry{
		EventBusName: &client.DestEVBName,
		Source:       &client.EventSourceName,
		DetailType:   &client.EventDetailType,
		Detail:       aws.String(detailString),
	}
}

func (svc *DDBService) DDBEventsHandler(ctx context.Context, ddbEvents events.DynamoDBEvent) error {

	events := []eventbridge_types.PutEventsRequestEntry{}

	var client *lib.EVBClientSvc // Variable to hold the appropriate EventBridge client

	for _, record := range ddbEvents.Records {
		// Check if the event comes from DynamoDB
		if record.EventSource == "aws:dynamodb" {

			// Extract table name from the ARN
			arnParts := strings.Split(record.EventSourceArn, "/")
			if len(arnParts) < 2 {
				svc.logger.Printf("Invalid ARN format: %v", record.EventSourceArn)
				continue
			}
			tableName := arnParts[1]
			svc.logger.Println("Extracted Table Name:", tableName)

			svc.logger.Println("Env:", svc.EvbClientEmployee.Environment)
			// Check if the table is EmployeeDataTable or TenantAppreciationsTable
			switch tableName {

			case "EmployeeDataTable-" + svc.EvbClientEmployee.Environment: // Handle employee data events
				svc.logger.Println("Got EmployeeDataTable-" + svc.EvbClientEmployee.Environment)
				client = svc.EvbClientEmployee

			case "TenantAppreciationsTable-" + svc.EvbClientAppreciation.Environment: // Handle appreciation data events
				svc.logger.Println("Got TenantAppreciationsTable-" + svc.EvbClientAppreciation.Environment)
				client = svc.EvbClientAppreciation

			default: // If it's neither table, log the event and skip
				svc.logger.Printf("Unknown source for event: %v", record)
				continue
			}

			// Process only INSERT events
			if record.EventName == "INSERT" {

				ddbData, err := DynamodbEventsToJsonString(record.Change.NewImage)
				if err != nil {
					svc.logger.Printf("Failed to convert DDB data to JSON, error: %v", err)
					return err
				}

				putEvent := svc.CreateEVBEvent(client, ddbData)
				events = append(events, putEvent)
			}
		}
	}

	// Send all events to EventBridge using the appropriate client
	err := client.SendEventsToEVB(events)
	if err != nil {
		svc.logger.Printf("Failed to send events to EVB: %v", err)
		return err
	}

	return nil
}
