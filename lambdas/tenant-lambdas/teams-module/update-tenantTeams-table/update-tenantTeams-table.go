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
	ctx, root := xray.BeginSegment(context.TODO(), "update-tenant-teams")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	dynamodbClient := dynamodb.NewFromConfig(cfg)
	logger := log.New(os.Stdout, "", log.LstdFlags)

	employeeSvc := companylib.CreateEmployeeService(ctx, dynamodbClient, nil, logger)
	employeeSvc.TenantTeamsTable = os.Getenv("TENANT_TEAMS_TABLE")

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		employeeSvc: *employeeSvc,
	}

	lambda.Start(svc.handleEmployeeEvents)
}

func (svc *Service) handleEmployeeEvents(ctx context.Context, event events.CloudWatchEvent) error {
	svc.logger.Println("Events:", event)
	svc.logger.Println("Detail : ", string(event.Detail))
	svc.logger.Println("Updating the Tenant teams")

	// Map the event detail to the EmployeeDynamodbData struct
	var employeeData companylib.EmployeeDynamodbData
	if err := json.Unmarshal(event.Detail, &employeeData); err != nil {
		svc.logger.Println("Error unmarshaling employee data:", err)
		return err
	}

	err := svc.employeeSvc.UpdateTenantTeams(employeeData)
	if err != nil {
		svc.logger.Println("Error in UpdateTenantTeams:", err)
		return err
	}

	return nil
}
