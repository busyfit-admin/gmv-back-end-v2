package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	svc, err := NewService()
	if err != nil {
		panic("ai-chat: failed to initialise service: " + err.Error())
	}

	lambda.Start(func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return svc.Handle(ctx, request)
	})
}
