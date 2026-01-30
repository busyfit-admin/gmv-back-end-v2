package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"

	"github.com/aws/aws-lambda-go/lambda"

	clients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	sfnClient clients.StepFunctionClient
	sfnArn    string

	employeeSvc   companylib.EmployeeService
	handleCardSVC companylib.HandleCardService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("RewardsAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-sub-domains")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	sfnclient := sfn.NewFromConfig(cfg)

	employeeSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	handleCardSVC := companylib.CreateHandleCardService(ctx, ddbclient, sfnclient, logger, os.Getenv("CARDS_TRACKER_TABLE"), os.Getenv("CARDS_TABLE"))

	svc := Service{
		ctx:       ctx,
		logger:    logger,
		sfnClient: sfnclient,

		sfnArn: os.Getenv("SFN_ARN"),

		handleCardSVC: *handleCardSVC,
		employeeSvc: *employeeSvc,
	}

	lambda.Start(svc.handleAPIRequests)
}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	svc.ctx = ctx
	reqType := request.HTTPMethod

	// 1) Authorization at User Level and Check if user to create the card is Admin or Rewards Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORRewardsManagerRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// 2) perform the cards creation requests

	switch reqType {
	case "POST":
		responseOutput, err := svc.POSTRequestHandler(request)
		if err != nil {
			return responseOutput, err
		}
		return responseOutput, nil
	case "GET":
		responseOutput, err := svc.GETRequestHandler(request)
		if err != nil {
			return responseOutput, err
		}
		return responseOutput, nil

	default:
		svc.logger.Printf("entered Default section of the switch, Erroring by returning 500")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

}

type CreateCardsPostREQ struct {
	CardsQuantity int    `json:"CardsQuantity"`
	CardId        string `json:"CardId"`
	CardType      string `json:"CardType"`
}

type CreateCardsPostRES struct {
	TrackingId string `json:"TrackingId"`
}

func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	requestId := "REQ-" + utils.GenerateRandomString(12)

	ReqBodyData := CreateCardsPostREQ{}
	err := json.Unmarshal([]byte(request.Body), &ReqBodyData)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal the Input Request")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	sfnInput := companylib.CardsMetaDataInput{
		TrackingId: string(requestId),

		CardsOrderQuantity: ReqBodyData.CardsQuantity,
		CardId:             ReqBodyData.CardId,
		CardType: ReqBodyData.CardType,

		CompanyId: "main", // NOTE : Need to change in future iterations
	}

	// 1c. Start the step function
	sfnInputByte, err := json.Marshal(sfnInput)
	if err != nil {
		svc.logger.Printf("Failed to marshal the SFN Request")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	StartExecutionOutput, err := svc.sfnClient.StartExecution(svc.ctx, &sfn.StartExecutionInput{
		StateMachineArn: aws.String(svc.sfnArn),
		Name:            aws.String("CardsCreation-" + requestId),
		Input:           aws.String(string(sfnInputByte)),
	})
	svc.logger.Print("Execution Output Arn is: ", StartExecutionOutput)
	if err != nil {
		svc.logger.Print("Error starting Step Functions execution:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error starting Step Functions execution",
		}, nil
	}

	// 2b. Respond back to the API providing the TrackingId

	postReqResponse := CreateCardsPostRES{
		TrackingId: string(requestId),
	}
	responseByte, err := json.Marshal(postReqResponse)
	if err != nil {
		svc.logger.Print("Error Marshalling the postReqResponse struct:", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseByte),
	}, nil
}

type CreateCardsGetREQ struct {
	TrackingId string `json:"TrackingId"`
}

func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1. Get the Tracker ID (aka JobId) from the headers
	JobId := request.Headers["TrackingId"]

	// 2. Get the status of the TrackerId from the DDB table
	CardsTrackingData, err := svc.handleCardSVC.GetCardTrackingDetails(JobId)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Convert it into JSON Type
	responseBody, err := json.Marshal(CardsTrackingData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	// 4. Return the Data to the API Gateway
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}
