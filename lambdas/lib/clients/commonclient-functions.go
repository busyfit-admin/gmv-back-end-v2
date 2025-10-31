package lib

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

const MAX_BATCH_LIMIT = 10

type EVBClientSvc struct {
	Ctx       context.Context
	EVBClient EventBridgeClient
	Logger    *log.Logger

	DestEVBName     string
	EventSourceName string
	EventDetailType string
	Environment     string
}

func SendEventToEVB(evbClient EVBClientSvc, detail interface{}) error {

	detailBytes, _ := json.Marshal(detail)
	detailString := string(detailBytes)

	event := &eventbridge.PutEventsInput{
		Entries: []types.PutEventsRequestEntry{
			{
				EventBusName: aws.String(evbClient.DestEVBName),
				Source:       aws.String(evbClient.EventSourceName),
				DetailType:   aws.String(evbClient.EventDetailType),
				Detail:       aws.String(detailString),
			},
		},
	}

	result, err := evbClient.EVBClient.PutEvents(evbClient.Ctx, event)
	if err != nil {
		evbClient.Logger.Printf("Error sending events to %v  failed with error %v", evbClient.DestEVBName, err)
		return err
	}

	for _, outputEntry := range result.Entries {
		if outputEntry.ErrorMessage != nil {
			evbClient.Logger.Printf("Error sending events to %v  failed with error \n %v \n %v", evbClient.DestEVBName, *outputEntry.ErrorCode, *outputEntry.ErrorMessage)
			return errors.New("Failed to Send Events to EVB")
		}
	}
	evbClient.Logger.Printf("Successfully sent EventBridge event")

	return nil
}

func (evbClient *EVBClientSvc) SendEventsToEVB(events []types.PutEventsRequestEntry) error {
	if len(events) == 0 {
		evbClient.Logger.Printf("No Events to Send")
		return nil
	}

	for i := 0; i < len(events); i += MAX_BATCH_LIMIT {

		batch := events[i:min(i+MAX_BATCH_LIMIT, len(events))]

		evbClient.Logger.Println("Printing Batch:")
		for _, entry := range batch {
			entryJSON, _ := json.Marshal(entry)
			evbClient.Logger.Printf("%s\n", string(entryJSON))
		}

		result, err := evbClient.EVBClient.PutEvents(
			evbClient.Ctx,
			&eventbridge.PutEventsInput{
				Entries: batch,
			},
		)

		if err != nil {
			evbClient.Logger.Printf("Could not send EventBridge event: %s", err.Error())
			return err
		}

		// PutEvents to EVB may fail, checks to ensure no errors are in the output
		for _, outputEntry := range result.Entries {
			if outputEntry.ErrorMessage != nil {
				evbClient.Logger.Printf("Could not send EventBridge event(inloop): %s", aws.ToString(outputEntry.ErrorMessage))
				continue
			}

			evbClient.Logger.Printf("Sent Events to EVB with event id: %s", aws.ToString(outputEntry.EventId))
		}
	}

	return nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
