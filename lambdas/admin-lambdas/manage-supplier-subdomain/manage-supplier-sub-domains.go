package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/aws/aws-lambda-go/lambda"

	adminlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/admin-lib"
)

type Service struct {
	ctx          context.Context
	logger       *log.Logger
	subDomainSVC adminlib.SubDomainService
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-supplier-sub-domains")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	cognitoClient := cognitoidentityprovider.NewFromConfig(cfg)

	subDomainSVC := adminlib.CreateSubDomainService(ctx, ddbclient, cognitoClient, logger)
	subDomainSVC.SupplierSubDomainsTable = os.Getenv("SUPPLIER_SUBDOMAIN_TABLE")
	subDomainSVC.SupplierSubDomains_SubDomainIndex = os.Getenv("SUPPLIER_SUBDOMAIN_INDEX_SUBDOMAIN")
	subDomainSVC.SupplierSubDomains_SupplierIdIndex = os.Getenv("SUPPLIER_SUBDOMAIN_INDEX_SUPPLIERID")

	svc := Service{
		ctx:          ctx,
		logger:       logger,
		subDomainSVC: *subDomainSVC,
	}

	lambda.Start(svc.handleAPIRequests)
}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	svc.ctx = ctx

	switch request.HTTPMethod {
	case "POST":
		return svc.POSTRequestHandler(request)
	case "PATCH":
		return svc.PATCHReqHandler(request)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, fmt.Errorf("HTTP METHOD not recognized for manage-supplier-sub-domains")
	}

}

type SubDomainCheckInput struct {
	SubDomain string `json:"SubDomain"`
}
type SubDomainCheckOutput struct {
	IsAvailable bool `json:"isAvailable"`
}

func (svc *Service) CheckSubDomainsAvailability(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var inputBody SubDomainCheckInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	isAvailable, err := svc.subDomainSVC.CheckSubDomainsAvailability(strings.ToLower(inputBody.SubDomain))
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	jsonBytes, err := json.Marshal(SubDomainCheckOutput{
		IsAvailable: isAvailable,
	})
	if err != nil {
		svc.logger.Printf("Unable to Marshal the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       string(jsonBytes),
		StatusCode: 200,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "get_type, X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
	}, nil
}

type GetAllSubDomainsInput struct {
	SupplierId string `json:"SupplierId"`
}

func (svc *Service) GetAllSupplierSubdomains(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var inputBody GetAllSubDomainsInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	allSubDomains, err := svc.subDomainSVC.GetAllSupplierSubDomains(inputBody.SupplierId)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	jsonBytes, err := json.Marshal(allSubDomains)
	if err != nil {
		svc.logger.Printf("Unable to Marshal the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       string(jsonBytes),
		StatusCode: 200,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "get_type, X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
	}, nil
}

// ------------------------- Handle POST Requests ------

func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.Headers["get_type"] {
	case "check_availability":
		return svc.CheckSubDomainsAvailability(request)
	case "get_subdomains":
		return svc.GetAllSupplierSubdomains(request)
	}
	var createSubDomainData adminlib.CreateSubDomainInput
	if err := json.Unmarshal([]byte(request.Body), &createSubDomainData); err != nil {
		svc.logger.Printf("Failed to Unmarshal the input is: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, nil
	}

	err := svc.subDomainSVC.CreateSupplierSubDomain(createSubDomainData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
	}, nil
}

// ------------------------- Handle PATCH Requests ------

func (svc *Service) PATCHReqHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	switch request.Headers["patch_type"] {
	case "update_stack_info":
		return svc.UpdateStackInfo(request)
	case "add_admin_user":
		return svc.AddAdminUser(request)
	default:
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil
	}

}

func (svc *Service) UpdateStackInfo(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var stackInfo adminlib.UpdateSubDomainStackInfo

	err := json.Unmarshal([]byte(request.Body), &stackInfo)
	if err != nil {
		svc.logger.Printf("unable to UnMarshal the input request, error :%v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	err = svc.subDomainSVC.UpdateSubDomainStackInfo(stackInfo)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "patch_type, X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
	}, nil
}

func (svc *Service) AddAdminUser(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var adminUserData adminlib.SubDomainAdmin

	err := json.Unmarshal([]byte(request.Body), &adminUserData)
	if err != nil {
		svc.logger.Printf("unable to UnMarshal the input request, error :%v", err)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	err = svc.subDomainSVC.AddSubDomainAdmin(adminUserData)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "patch_type, X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
	}, nil
}
