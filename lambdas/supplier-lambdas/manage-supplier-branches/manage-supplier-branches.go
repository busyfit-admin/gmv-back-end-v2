package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	supplierlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/supplier-lib"
)

var RESP_HEADERS = map[string]string{
	"Access-Control-Allow-Origin":  "*",
	"Access-Control-Allow-Methods": "*",
	"Access-Control-Allow-Headers": "get_type,branch-id,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
}

type Service struct {
	ctx    context.Context
	logger *log.Logger

	supplierSvc supplierlib.SupplierService
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-supplier-branches")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	supplierSvc := supplierlib.CreateSupplierService(ctx, ddbclient, logger)
	supplierSvc.SupplierBranchTable = os.Getenv("SUPPLIER_BRANCHES_TABLE")
	supplierSvc.SupplierBranchTable_IsActiveIndex = os.Getenv("SUPPLIER_BRANCHES_TABLE_ISACTIVE_INDEX")

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		supplierSvc: *supplierSvc,
	}

	lambda.Start(svc.handleAPIRequests)
}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	svc.ctx = ctx

	switch request.HTTPMethod {

	case "GET":
		return svc.GetRequestHandler(request)
	case "POST":
		return svc.PostRequestHandler(request)
	case "PATCH":
		return svc.PatchRequestHandler(request)
	case "DELETE":
		return svc.DeleteRequestHandler(request)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, fmt.Errorf("HTTP METHOD not recognized for manage-supplier-branches")
	}

}

/*
Get Request can handle:
 1. Get All Supplier Branches Data ( Active + Inactive)
 2. Get a single Branch Data
 3. Get all Active Branches ( for cards Creation Drop down)
*/
// Get Request types header filters
const (
	GET_ALL_BRANCHES_DATA        = "get-all-branches"
	GET_BRANCHES_DATA            = "get-branch"
	GET_ALL_ACTIVE_BRANCHES_DATA = "get-all-active-branches"
)

func (svc *Service) GetRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	getType := request.Headers["get_type"]

	switch getType {
	case GET_ALL_BRANCHES_DATA:
		return svc.GetAllBranches(request)
	case GET_BRANCHES_DATA:
		return svc.GetBranch(request)
	case GET_ALL_ACTIVE_BRANCHES_DATA:
		return svc.GetAllActiveBranches(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}
func (svc *Service) GetAllBranches(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allBranches, err := svc.supplierSvc.GetAllBranches()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allBranches)
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
func (svc *Service) GetBranch(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	branchId := request.Headers["branch-id"]

	BranchData, err := svc.supplierSvc.GetBranchDetails(branchId)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(BranchData)
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
func (svc *Service) GetAllActiveBranches(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allBranches, err := svc.supplierSvc.GetActiveBranches()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allBranches)
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

// ------- POST Requests - To handle Creation of Supplier Branches -----
func (svc *Service) PostRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var SupplierBranch supplierlib.SupplierBranch
	err := json.Unmarshal([]byte(request.Body), &SupplierBranch)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	err = svc.supplierSvc.CreateSupplierBranch(SupplierBranch)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

// ------- PATCH Requests - To handle Updating of Supplier Branches -----
func (svc *Service) PatchRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var SupplierBranch supplierlib.SupplierBranch
	err := json.Unmarshal([]byte(request.Body), &SupplierBranch)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	err = svc.supplierSvc.UpdateSupplierBranch(SupplierBranch)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

// ------- DELETE Requests - To handle Delete of Supplier Branches -----
func (svc *Service) DeleteRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	branchId := request.Headers["branch-id"]

	err := svc.supplierSvc.DeleteSupplierBranch(branchId)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}
