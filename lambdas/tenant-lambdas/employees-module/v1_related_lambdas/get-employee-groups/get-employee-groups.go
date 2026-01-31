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
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	employeeSVC companylib.EmployeeService
}

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "get-employee-groupsData")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	employeesvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	employeesvc.EmployeeGroupsTable = os.Getenv("EMPLOYEE_GROUPS_TABLE")

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		employeeSVC: *employeesvc,
	}

	lambda.Start(svc.handleAPIRequest)
}

func (svc *Service) handleAPIRequest(ctx context.Context, request events.APIGatewayProxyResponse) (events.APIGatewayProxyResponse, error) {

	headers := request.Headers
	filter := ""
	// Filter using headers
	for key, value := range headers {
		if key == "Groupid" {
			filter = value
			break
		}
	}
	svc.logger.Println("The filter sent :", filter)

	// Getting all the data in table
	employeeGroupsMap, err := svc.employeeSVC.GetAllEmployeeGroupsInMap()
	if err != nil {
		svc.logger.Println("There was an error :", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error retrieving employee groups",
		}, nil
	}

	if filter == "" {
		allEmployeeGroups, err := json.Marshal(employeeGroupsMap)
		if err != nil {
			svc.logger.Println("Error converting all groups to JSON:", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "Error converting all groups to JSON",
			}, nil
		}
		svc.logger.Println("All groups JSON:", string(allEmployeeGroups))
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Body:       string(allEmployeeGroups),
		}, nil
	}

	// Extract the filtered group from employeeGroupsMap
	filteredGroup, found := employeeGroupsMap[filter]
	if !found {
		svc.logger.Println("Filtered group not found")
		return events.APIGatewayProxyResponse{
			StatusCode: 404,
			Body:       "Filtered group not found",
		}, nil
	}

	// Marshal the filtered group back to JSON
	filteredGroupJSON, err := json.Marshal(filteredGroup)
	if err != nil {
		svc.logger.Println("Error converting filtered group to JSON:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error converting filtered group to JSON",
		}, nil
	}
	svc.logger.Println("Filtered group JSON:", string(filteredGroupJSON))

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "Groupid,GroupId,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
		Body:       string(filteredGroupJSON),
	}, nil
}
