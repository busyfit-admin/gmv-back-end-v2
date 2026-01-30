package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/aws/aws-lambda-go/lambda"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	surveySvc companylib.SurveyService
	empSVC    companylib.EmployeeService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("SurveysAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-surveys")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	empSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	empSvc.EmployeeTable = os.Getenv("EMPLOYEES_TABLE")
	empSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEES_TABLE_COGNITO_ID_INDEX")

	surveySvc := companylib.CreateSurveyService(ctx, ddbclient, logger)
	surveySvc.SurveyQuestionsTable = os.Getenv("SURVEY_QUESTIONS_TABLE")
	surveySvc.SurveyResponsesTable = os.Getenv("SURVEY_RESPONSE_TABLE")

	svc := Service{
		ctx:       ctx,
		logger:    logger,
		surveySvc: *surveySvc,
		empSVC:    *empSvc,
	}

	lambda.Start(svc.handleAPIRequests)
}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.ctx = ctx

	switch request.HTTPMethod {
	case "GET":
		return svc.GETRequestHandler(request)
	case "POST":
		return svc.POSTRequestHandler(request)
	default:
		svc.logger.Printf("entered Default section of the switch, Erroring by returning 500")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

const (
	GET_SURVEY_QUESTIONS = "get-survey-questions"
	GET_SURVEY_RESPONSES = "get-survey-responses"
)

func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	getType := request.Headers["get_type"]
	switch getType {
	case GET_SURVEY_QUESTIONS:
		return svc.getSurveyQuestions(request)
	// case GET_SURVEY_RESPONSES:
	// 	return svc.getSurveyResponses(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) getSurveyQuestions(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	surveyId := request.Headers["survey_id"]

	surveyQuestions, err := svc.surveySvc.GetSurveyQuestions(surveyId)
	if err != nil {
		svc.logger.Printf("Error getting Survey Questions: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	responseBody, err := json.Marshal(surveyQuestions)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

const (
	POST_SURVEY_RESPONSE = "submit-survey-response"
)

func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	post_type := request.Headers["post_type"]
	switch post_type {
	case POST_SURVEY_RESPONSE:
		return svc.SubmitSurveyResponse(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) SubmitSurveyResponse(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	authData, status, err := svc.empSVC.Authorizer(request, companylib.QUERY_NULL)
	if !status || err != nil {
		svc.logger.Printf("Error getting Employee Authorization Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	var surveyResponse companylib.SurveyResponsesInput
	err = json.Unmarshal([]byte(request.Body), &surveyResponse)
	if err != nil {
		svc.logger.Printf("Error unmarshalling Survey Response: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	err = svc.surveySvc.SubmitSurveyResponse(surveyResponse, authData.Username)
	if err != nil {
		svc.logger.Printf("Error submitting Survey Response: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}
