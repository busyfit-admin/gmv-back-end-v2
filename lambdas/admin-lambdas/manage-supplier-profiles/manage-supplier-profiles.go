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

	adminlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/admin-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	adminSVC adminlib.SupplierDetailsSvc
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-supplier-profiles")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	adminsvc := adminlib.CreateSupplierDetailsSvc(ctx, ddbclient, logger)
	adminsvc.SupplierDetailsTable = os.Getenv("SUPPLIER_DETAILS_TABLE")
	adminsvc.SupplierDetails_SupplierStatusIndex = os.Getenv("SUPPLIER_DETAILS_INDEX_SUPPLIERSTAGEID")

	svc := Service{
		ctx:    ctx,
		logger: logger,

		adminSVC: *adminsvc,
	}

	lambda.Start(svc.handleAPIRequests)

}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	svc.logger.Printf("Multi Value Headers : %v\n", request.MultiValueHeaders)
	svc.logger.Printf("Request Context: %v\n", request.RequestContext)
	svc.logger.Printf("HTTP METHOD: %s\n", request.HTTPMethod)
	svc.logger.Printf("Headers Passed: %v\n", request.Headers)
	switch request.HTTPMethod {
	case "GET":
		return svc.GETRequestHandler(request)
	case "POST":
		return svc.POSTRequestHandler(request)
	case "PATCH":
		return svc.PATCHReqHandler(request)

	default:
		svc.logger.Printf("entered Default section of the switch, Erroring by returning 500")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, nil
	}

}

func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	supplierId := request.Headers["supplierid"]

	if supplierId == "" {
		svc.logger.Printf("No Supplier ID is passed in header. Performing GET All Supplier Details Logic")
		return svc.GETAllSuppliers(request)
	}
	svc.logger.Printf("Supplier ID is passed in header and performing GET on the SupplierID : %s", supplierId)
	return svc.GETSupplierDetails(supplierId)
}

func (svc *Service) GETAllSuppliers(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	responseOutput, err := svc.adminSVC.GetAllSupplierDetails()
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	respBody, err := json.Marshal(responseOutput)
	if err != nil {
		svc.logger.Printf("Unable to Marshal the response output from DDB")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "supplierid,SupplierId, Supplierid, X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
		Body:       string(respBody)}, nil

}

func (svc *Service) GETSupplierDetails(supplierId string) (events.APIGatewayProxyResponse, error) {

	supplierData, err := svc.adminSVC.GetSupplierProfileById(supplierId)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	respBody, err := json.Marshal(supplierData)
	if err != nil {
		svc.logger.Printf("Unable to Marshal the response output from DDB")
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "SupplierId, Supplierid,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
		Body:       string(respBody)}, nil

}

// Func to handle Creation of new Suppliers
func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var ReqBody adminlib.CreateSupplierProfile

	err := json.Unmarshal([]byte(request.Body), &ReqBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshal the request")
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	err = svc.adminSVC.CreateSupplierProfile(ReqBody)

	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
			Headers: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "*",
				"Access-Control-Allow-Headers": "SupplierId, Supplierid,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
			},
			StatusCode: 200},
		nil
}

func (svc *Service) PATCHReqHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	svc.logger.Printf("PatchType Passed: %v", request.Headers["patch_type"])
	switch request.Headers["patch_type"] {
	case "update_top_level":
		svc.logger.Printf("Triggering HandleUpdateTopLevelInfo")
		return svc.HandleUpdateTopLevelInfo(request)
	case "add_contact":
		svc.logger.Printf("Triggering Add Contact")
		return svc.HandleAddContact(request)
	case "update_stage":
		svc.logger.Printf("Triggering Handle Update Stage")
		return svc.HandleUpdateStage(request)

	default:
		svc.logger.Printf("Incorrect header sent for patch :%v", request.Headers["patch_type"])
		return events.APIGatewayProxyResponse{StatusCode: 500}, nil

	}
}

func (svc *Service) HandleUpdateTopLevelInfo(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Unmarshal request input to required struct
	var TopLevelUpdateInfo adminlib.PatchTopLevelInfo

	err := json.Unmarshal([]byte(request.Body), &TopLevelUpdateInfo)
	if err != nil {
		svc.logger.Printf("unable to unmarshal the incoming request body: %v", request.Body)
	}

	err = svc.adminSVC.PatchTopLevelInfo(TopLevelUpdateInfo)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "patch_type, SupplierId, Supplierid,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
	}, nil

}
func (svc *Service) HandleAddContact(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Unmarshal request input to required struct
	var AddContact adminlib.PatchSupplierContacts

	err := json.Unmarshal([]byte(request.Body), &AddContact)
	if err != nil {
		svc.logger.Printf("unable to unmarshal the incoming request body: %v", request.Body)
	}

	err = svc.adminSVC.PatchSupplierContacts(AddContact)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "patch_type, SupplierId, Supplierid,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
	}, nil

}

func (svc *Service) HandleUpdateStage(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Unmarshal request input to required struct
	var UpdateStage adminlib.PatchSupplierOverallStage

	err := json.Unmarshal([]byte(request.Body), &UpdateStage)
	if err != nil {
		svc.logger.Printf("unable to unmarshal the incoming request body: %v", request.Body)
	}

	err = svc.adminSVC.PatchOverallStageId(UpdateStage)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "patch_type, SupplierId, Supplierid,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
		StatusCode: 200,
	}, nil

}
