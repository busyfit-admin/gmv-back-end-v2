package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	employeeSvc companylib.EmployeeService
	cdnSvc      companylib.CDNService

	GroupsData map[string]companylib.EmployeeGroups // Key: Group IDs as Names of Groups
}

var RESP_HEADERS = companylib.GetHeadersForAPI("ProfileAPI")

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "manage-employee-groups")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	dynamodbClient := dynamodb.NewFromConfig(cfg)
	secretsClient := secretsmanager.NewFromConfig(cfg)
	logger := log.New(os.Stdout, "", log.LstdFlags)

	employeeSvc := companylib.CreateEmployeeService(ctx, dynamodbClient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")
	employeeSvc.EmployeeGroupsTable = os.Getenv("EMPLOYEE_GROUPS_TABLE")

	// Here we are creating a CDN Service
	cdnSvc := companylib.CDNService{}
	err = cdnSvc.CreateCDNService(ctx, logger, secretsClient, os.Getenv("SECRETS_CND_PK_ARN"), os.Getenv("PUBLIC_KEY_ID"))
	if err != nil {
		log.Fatalf("Error creating CDN Service: %v\n", err)
	}
	cdnSvc.CDNDomain = os.Getenv("CDN_DOMAIN")

	// Load the groups data
	grpsData, err := employeeSvc.GetAllEmployeeGroupsInMap()
	if err != nil {
		log.Fatalf("Error loading groups data: %v\n", err)
	}

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		employeeSvc: *employeeSvc,
		cdnSvc:      cdnSvc,
		GroupsData:  grpsData,
	}

	lambda.Start(svc.handleAPIRequests)
}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	svc.ctx = ctx

	switch request.HTTPMethod {
	case "GET":
		return svc.GetAllGroupsForEmployee(ctx, request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

type EmployeeGroupsOutput struct {
	Groups map[string]companylib.EmployeeGroups `json:"groups"` // Key: Group IDs as Names of Groups and Value: Active Status
}

func (svc *Service) GetAllGroupsForEmployee(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Get AuthData
	authData, _, err := svc.employeeSvc.Authorizer(request, companylib.QUERY_NULL)
	if err != nil {
		svc.logger.Printf("Error getting AuthData: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	outputRes := GetGroupStatus(authData.Groups, svc.GroupsData)

	// json marshall the outputRes
	jsonOutput, err := json.Marshal(outputRes)
	if err != nil {
		svc.logger.Printf("Error marshalling the output: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonOutput),
	}, nil
}
func GetGroupStatus(groups []string, groupsData map[string]companylib.EmployeeGroups) EmployeeGroupsOutput {
	empGroups := make(map[string]companylib.EmployeeGroups)

	for _, group := range groups {
		if _, ok := groupsData[group]; ok {
			empGroups[group] = groupsData[group]
		}
	}

	return EmployeeGroupsOutput{
		Groups: empGroups,
	}
}
