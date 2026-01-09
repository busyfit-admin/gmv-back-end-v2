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
	"Access-Control-Allow-Headers": "get_type,card-id,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
}

type Service struct {
	ctx    context.Context
	logger *log.Logger

	supplierSvc supplierlib.SupplierCardsService
}

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-supplier-cards")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	supplierSvc := supplierlib.CreateSupplierCardsService(ctx, logger, ddbclient)
	supplierSvc.SupplierCardsTable = os.Getenv("SUPPLIER_CARDS_TABLE")
	supplierSvc.SupplierCardsTable_IsActiveIndex = os.Getenv("SUPPLIER_CARDS_TABLE_ISACTIVE_INDEX")

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
 1. Get All Supplier Cards Data ( Active + Inactive)
 2. Get a single Card Data
*/
// Get Request types header filters
const (
	GET_ALL_CARDS_DATA = "get-all-cards"
	GET_CARD_DATA      = "get-card"
)

func (svc *Service) GetRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	getType := request.Headers["get_type"]

	switch getType {
	case GET_ALL_CARDS_DATA:
		return svc.GetAllCards(request)
	case GET_CARD_DATA:
		return svc.GetCard(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}
func (svc *Service) GetAllCards(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allCards, err := svc.supplierSvc.GetAllCards()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allCards)
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
func (svc *Service) GetCard(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	cardId := request.Headers["card-id"]

	cardData, err := svc.supplierSvc.GetCardDetails(cardId)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(cardData)
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

	var SupplierCard supplierlib.SupplierCards
	err := json.Unmarshal([]byte(request.Body), &SupplierCard)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	err = svc.supplierSvc.CreateSupplierCards(SupplierCard)

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

	var SupplierCard supplierlib.SupplierCards
	err := json.Unmarshal([]byte(request.Body), &SupplierCard)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	err = svc.supplierSvc.UpdateSupplierCard(SupplierCard)

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

	cardId := request.Headers["card-id"]

	err := svc.supplierSvc.DeleteSupplierCard(cardId)

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
