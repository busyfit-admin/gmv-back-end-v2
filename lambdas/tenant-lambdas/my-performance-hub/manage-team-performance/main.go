package main

import (
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	common "github.com/busyfit-admin/saas-integrated-apis/lambdas/tenant-lambdas/my-performance-hub/common"
)

func main() {
	svc, err := common.NewService()
	if err != nil {
		log.Fatalf("failed to initialize manage-team-performance service: %v", err)
	}

	lambda.Start(func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return svc.Handle(request)
	})
}
