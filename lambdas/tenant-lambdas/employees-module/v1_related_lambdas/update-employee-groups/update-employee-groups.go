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

	employeeSvc companylib.EmployeeService
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "update-employee-groups")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	dynamodbClient := dynamodb.NewFromConfig(cfg)
	logger := log.New(os.Stdout, "", log.LstdFlags)

	employeeSvc := companylib.CreateEmployeeService(ctx, dynamodbClient, nil, logger)
	employeeSvc.EmployeeGroupsTable = os.Getenv("EMPLOYEE_GROUPS_TABLE")

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		employeeSvc: *employeeSvc,
	}

	lambda.Start(svc.handleEmployeeEvents)

}

func (svc *Service) handleEmployeeEvents(ctx context.Context, event events.CloudWatchEvent) error {

	var detailMap map[string]interface{}
	if err := json.Unmarshal(event.Detail, &detailMap); err != nil {
		svc.logger.Println("Error unmarshaling event.Detail:", err)
		return err
	}

	employeeGroups := companylib.EmployeeGroups{}

	for key, value := range detailMap {
		svc.logger.Printf("%s: %v\n", key, value)
		switch key {
		case "GroupId":
			employeeGroups.GroupId = value.(string)
		case "GroupName":
			employeeGroups.GroupName = value.(string)
		case "GroupDesc":
			employeeGroups.GroupDesc = value.(string)
		}
	}
	err := svc.employeeSvc.UpdateEmployeeGroups(employeeGroups)

	if err != nil {
		svc.logger.Printf("Error updating employee groups: %v\n", err)
		return err
	}
	return nil
}
