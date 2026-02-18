package main

import (
	"context"

	"log"
	"strconv"

	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/xray"

	"os"

	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"

	userlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/user-lib"
)

// Function to convert a float64 monetary value to cents (integer)
func toCents(amount float64) int {
	return int(amount * 100)
}

type cardInput struct {
	CardId              int     `json:"CardId"`
	Action              string  `json:"Action"` // Can be either Add or Use
	AddRewardAmount     float64 `json:"AddRewardAmount"`
	ConsumeRewardAmount float64 `json:"ConsumeRewardAmount"`
	UpdateExpiry        string  `json:"UpdateExpiry"`
	IsActive            bool    `json:"IsActive"`
	UpdatedBy           string  `json:"UpdatedBy"`
	RewardAmount        int     `json:"RewardAmount"`
	CardName            string  `json:"CardName"`
	CompanyName         string  `json:"CompanyName"`
	CardMetaData        string  `json:"CardMetaData"`
}

type Service struct {
	ctx    context.Context
	logger *log.Logger

	userSvc userlib.UserService
}

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "update-user-cards")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	dynamodbClient := dynamodb.NewFromConfig(cfg)
	logger := log.New(os.Stdout, "", log.LstdFlags)

	userSvc := userlib.CreateUserService(ctx, dynamodbClient, logger, os.Getenv("USER_CARD_TABLE"))

	svc := Service{
		ctx:     ctx,
		logger:  logger,
		userSvc: *userSvc,
	}
	lambda.Start(svc.handleUserEvents)
}

func (svc *Service) handleUserEvents(ctx context.Context, input cardInput) error {
	// CloudWatch logs
	// get the card details

	svc.logger.Printf("Logging to CloudWatch - CardID: %d, Action: %s, UpdatedBy: %s, Timestamp: %s\n", input.CardId, input.Action, input.UpdatedBy, time.Now().Format(time.RFC3339))

	switch input.Action {
	case "Add":
		// Convert float64 to cents
		addRewardCents := toCents(input.AddRewardAmount)
		// for Add reward
		input.RewardAmount += addRewardCents
		svc.logger.Printf("Added reward amount %.2f to card %d\n", input.AddRewardAmount, input.CardId)

	case "Use":
		// Convert float64 to cents
		consumeRewardCents := toCents(input.ConsumeRewardAmount)
		// for Use reward
		if input.RewardAmount >= consumeRewardCents {
			input.RewardAmount -= consumeRewardCents
			svc.logger.Printf("Consumed reward amount %.2f from card %d\n", input.ConsumeRewardAmount, input.CardId)
		} else {
			svc.logger.Println("Insufficient balance. Update cannot be performed.")
		}

	default:
		svc.logger.Println("unknown action")
	}

	if input.UpdateExpiry != "" {
		svc.logger.Printf("Updated expiry date of card %d to %s\n", input.CardId, input.UpdateExpiry)
	}

	svc.logger.Printf("Set activity status of card %d to %v\n", input.CardId, input.IsActive)

	svc.logger.Println("Update Completed")

	// Call the UpdateCardsTable method with input data
	err := svc.userSvc.UpdateCardsTable(strconv.Itoa(input.CardId), userlib.CardInput{
		CardId:              input.CardId,
		Action:              input.Action,
		AddRewardAmount:     input.AddRewardAmount,
		ConsumeRewardAmount: input.ConsumeRewardAmount,
		UpdateExpiry:        input.UpdateExpiry,
		IsActive:            input.IsActive,
		UpdatedBy:           input.UpdatedBy,
		RewardAmount:        input.RewardAmount,
	})
	if err != nil {
		svc.logger.Printf("Failed to update card details: %v", err)
		return err
	}

	return nil
}
