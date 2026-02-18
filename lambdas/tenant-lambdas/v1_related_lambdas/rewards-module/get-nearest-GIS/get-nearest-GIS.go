package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"

	GISlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/GIS-lib"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger
	gisSVC GISlib.GISService
}

func main() {
	// Initialize AWS X-Ray tracing
	ctx, root := xray.BeginSegment(context.TODO(), "get-nearest-GIS")
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
	start := time.Now()

	svc.logger.Println("Called Get nearest GIS lambda")
	svc.logger.Printf("Request: %+v\n", request)

	// Parse latitude, longitude, and radius from query parameters
	latStr := request.Headers["lat"]
	lngStr := request.Headers["lng"]
	radiusStr := request.Headers["radius"]

	if latStr == "" || lngStr == "" || radiusStr == "" {
		svc.logger.Println("Latitude, longitude, or radius not provided")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request: Missing latitude, longitude, or radius",
		}, nil
	}

	latitude, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		svc.logger.Printf("Error parsing latitude: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request: Invalid latitude",
		}, nil
	}

	longitude, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		svc.logger.Printf("Error parsing longitude: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request: Invalid longitude",
		}, nil
	}

	radius, err := strconv.Atoi(radiusStr)
	if err != nil {
		svc.logger.Printf("Error parsing radius: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Bad Request: Invalid radius",
		}, nil
	}

	// Query data
	sbs, err := svc.gisSVC.QueryData(latitude, longitude, radius)
	if err != nil {
		svc.logger.Printf("Error querying data: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	svc.logger.Println("sbs", sbs)

	// Construct response
	responseBody, err := json.Marshal(sbs)
	if err != nil {
		svc.logger.Printf("Error marshaling response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	svc.logger.Printf("Executed in: %v\n", time.Since(start))

	return events.APIGatewayProxyResponse{
		Body: string(responseBody),
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "lat,lng,radius,Groupid,GroupId,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
	}, nil
}
