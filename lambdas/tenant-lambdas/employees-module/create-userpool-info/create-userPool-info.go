package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	employeeSVC companylib.EmployeeService
}

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "create-userPool-info")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	cognitoClient := cognitoidentityprovider.NewFromConfig(cfg)

	employeesvc := companylib.CreateEmployeeService(ctx, ddbclient, cognitoClient, logger)
	employeesvc.EmployeeUserPoolId = os.Getenv("EMPLOYEE_USER_POOL_ID")
	employeesvc.EmployeeTable = os.Getenv("EMPLOYEE_DDB_TABLE")

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		employeeSVC: *employeesvc,
	}

	svc.logger.Println("employeesvc.EmployeeUserPoolId", employeesvc.EmployeeUserPoolId)

	lambda.Start(svc.handleRequest)
}

func (svc *Service) handleRequest(ctx context.Context, event events.CloudWatchEvent) error {
	svc.logger.Println("event : ", event)
	svc.logger.Println("Detail : ", string(event.Detail))
	var detailMap map[string]interface{}
	if err := json.Unmarshal(event.Detail, &detailMap); err != nil {
		svc.logger.Println("Error unmarshaling event.Detail:", err)
		return err
	}

	for key, value := range detailMap {
		svc.logger.Printf("%s: %v\n", key, value)
	}

	userName, userNameExists := detailMap["UserName"].(string)
	if !userNameExists {
		svc.logger.Println("UserName field not found or not a string")
		return nil
	}

	err := svc.employeeSVC.CreateCognitoUser(userName)
	if err != nil {
		return err
	}


	err = svc.employeeSVC.UpdateEmployeeData(userName)
	if err != nil {
		svc.logger.Println("Error in UpdateEmployeeData:", err)
		return err
	}

	return nil
}
