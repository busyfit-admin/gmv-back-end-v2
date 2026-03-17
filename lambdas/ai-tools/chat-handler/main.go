package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	svc, err := NewService()
	if err != nil {
		panic("ai-chat: failed to initialise service: " + err.Error())
	}

	lambda.Start(func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return svc.Handle(request)
	})
}
