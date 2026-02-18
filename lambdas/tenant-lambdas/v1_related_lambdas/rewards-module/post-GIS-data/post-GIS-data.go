package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	GISlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/GIS-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger
	gisSVC GISlib.GISService
}

func main() {
	// Initialize AWS X-Ray tracing
	ctx, root := xray.BeginSegment(context.TODO(), "post-GIS-data")
	defer root.Close(nil)

	// Load AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	// Instrument AWS SDK for X-Ray
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)
	logger := log.New(os.Stdout, "", log.LstdFlags)
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-south-1"),
	}))
	ddbclient := dynamodb.New(sess)

	// Create GIS service instance with proper configuration
	GISsvc := GISlib.CreateGISService(ctx, ddbclient, logger, os.Getenv("GIS_TABLE"))

	if GISsvc == nil {
		log.Fatal("Failed to create GISService")
	}

	// Create service instance with the GIS service
	svc := Service{
		ctx:    ctx,
		logger: logger,
		gisSVC: *GISsvc,
	}
	// Start the Lambda function handler
	lambda.Start(svc.handleAPIRequest)
}

func (svc *Service) handleAPIRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Read file from the Lambda environment
	filePath := "/starbucks_us_locations.json" // Update path if necessary
	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		svc.logger.Printf("Failed to read file: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Failed to read file",
		}, nil
	}

	// Unmarshal JSON data
	coffeeShops := []GISlib.Starbucks{}
	err = json.Unmarshal(f, &coffeeShops)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal JSON: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Failed to unmarshal JSON",
		}, nil
	}

	// Load data into DynamoDB using GISService
	svc.gisSVC.LoadData(coffeeShops)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Data loaded successfully",
	}, nil
}
