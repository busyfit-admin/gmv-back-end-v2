package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx      context.Context
	logger   *log.Logger
	teamsSVC companylib.TenantTeamsService
}

var RESP_HEADERS = map[string]string{
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Methods": "*",
	"Access-Control-Allow-Headers": "X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
}

// Struct for Slack message payload
type SlackMessage struct {
	Channel  string `json:"channel"`
	Username string `json:"username"`
	Text     string `json:"text"`
}

// Struct for Microsoft Teams Adaptive Card message format
type AdaptiveCard struct {
	Type        string       `json:"type"`
	Attachments []Attachment `json:"attachments"`
}

// Struct for Microsoft Teams card attachments
type Attachment struct {
	ContentType string      `json:"contentType"`
	Content     CardContent `json:"content"`
}

// Struct for card content in Microsoft Teams messages
type CardContent struct {
	Schema  string          `json:"$schema"`
	Type    string          `json:"type"`
	Version string          `json:"version"`
	Body    []CardTextBlock `json:"body"`
}

// Struct for text block inside card content
type CardTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "tenant-outbound-integrations")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)

	ddbclient := dynamodb.NewFromConfig(cfg)

	teamssvc := companylib.CreateTenantTeamsService(ctx, ddbclient, logger)
	teamssvc.OutBound_Integration = os.Getenv("TENANT_INTEGRATION_TABLE")

	svc := Service{
		ctx:      ctx,
		logger:   logger,
		teamsSVC: *teamssvc,
	}

	lambda.Start(svc.handleRequests)
}

func (svc *Service) handleRequests(ctx context.Context, snsEvent events.SNSEvent) error {
	// Fetch tenant webhooks from DynamoDB or a similar data source

	for _, record := range snsEvent.Records {
		svc.logger.Println("SNS Record:", record)
		svc.logger.Println("SNS Message:", record.SNS.Message)
	}
	// we can get the team id from the sns message
	url, err := svc.teamsSVC.GetAllTenantWebHooks("team1")
	if err != nil {
		svc.logger.Printf("Error fetching tenant webhooks: %v\n", err)
		return err
	}

	// Extract Slack and Microsoft Teams webhook URLs
	slackWebhookURL := url.SlackWebHookUrl
	teamsWebhookURL := url.TeamsWebHookUrl

	// Check if either of the webhooks exists
	if slackWebhookURL == "" && teamsWebhookURL == "" {
		svc.logger.Println("No valid Slack or Teams Webhook URL set")
		return fmt.Errorf("no webhook URL set for either Slack or Teams")
	}

	// Log the fetched webhook URLs for debugging
	svc.logger.Println("Fetched Slack Webhook URL:", slackWebhookURL)
	svc.logger.Println("Fetched Teams Webhook URL:", teamsWebhookURL)

	// Marshal the SNS event for logging purposes
	snsEventJSON, err := json.MarshalIndent(snsEvent, "", "  ")
	if err != nil {
		svc.logger.Printf("Failed to marshal snsEvent: %v", err)
		return err
	}
	svc.logger.Println("snsEvent:", string(snsEventJSON))

	// Process the SNS message records
	var combinedText string
	for _, record := range snsEvent.Records {
		var snsMessage map[string]interface{}
		// Unmarshal the SNS message into a map
		if err := json.Unmarshal([]byte(record.SNS.Message), &snsMessage); err != nil {
			svc.logger.Printf("Failed to unmarshal SNS message: %v", err)
			return err
		}
		combinedText += fmt.Sprintf("Received message: %v\n", snsMessage)
	}

	// Create a final message to be sent to Slack/Teams
	finalMessage := "Test Message." // This would normally contain actual information from the SNS message

	// Prepare the Slack message payload
	slackMsg := SlackMessage{
		Channel:  "#Test",
		Username: "WEBHOOK_USERNAME",
		Text:     finalMessage,
	}

	// Send the message to Slack if the Slack webhook URL is available
	if slackWebhookURL != "" {
		// Marshal the Slack message payload
		slackMsgBytes, err := json.Marshal(slackMsg)
		if err != nil {
			svc.logger.Printf("Failed to marshal Slack message: %v", err)
			return err
		}

		// Send the HTTP POST request to the Slack webhook URL
		resp1, err := http.Post(slackWebhookURL, "application/json", bytes.NewBuffer(slackMsgBytes))
		if err != nil {
			svc.logger.Printf("Failed to send message to Slack: %v", err)
			return err
		}
		defer resp1.Body.Close()

		// Log the response from Slack
		svc.logger.Printf("Message sent to Slack: %s, Status Code: %d", finalMessage, resp1.StatusCode)
	}

	// Send the message to Microsoft Teams if the Teams webhook URL is available
	if teamsWebhookURL != "" {
		// Prepare the Adaptive Card payload for Microsoft Teams
		cardPayload := AdaptiveCard{
			Type: "message",
			Attachments: []Attachment{
				{
					ContentType: "application/vnd.microsoft.card.adaptive",
					Content: CardContent{
						Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
						Type:    "AdaptiveCard",
						Version: "1.2",
						Body: []CardTextBlock{
							{
								Type: "TextBlock",
								Text: finalMessage,
							},
						},
					},
				},
			},
		}

		// Marshal the card payload for Teams
		cardPayloadBytes, err := json.Marshal(cardPayload)
		if err != nil {
			svc.logger.Printf("Failed to marshal Microsoft Teams card payload: %v", err)
			return err
		}

		// Send the HTTP POST request to the Microsoft Teams webhook URL
		resp2, err := http.Post(teamsWebhookURL, "application/json", bytes.NewBuffer(cardPayloadBytes))
		if err != nil {
			svc.logger.Printf("Failed to send message to Microsoft Teams: %v", err)
			return err
		}
		defer resp2.Body.Close()

		// Log the response from Teams
		svc.logger.Printf("Message sent to Microsoft Teams, Status Code: %d", resp2.StatusCode)
	}

	return nil
}
