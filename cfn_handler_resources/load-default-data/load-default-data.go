package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/cfn"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

type CFNRequestData struct {
	DDBTableName string `json:"DDBTableName"` // TableName of the Dynamodb table
	Data         string `json:"Data"`         // json data as string that needs to be added
	PrimaryKeys  string `json:"PrimaryKeys"`  // map of PK and SK for Delete operations
	/*
		ex:
		{
			"fieldName" : "fieldValue",
			"fieldName" : "fieldValue"
		}
	*/
}
type DefaultDataService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	cfnData CFNRequestData
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Cannot load configuration: %v\n", err)
	}

	svc := &DefaultDataService{
		logger:         log.New(os.Stdout, "", log.LstdFlags),
		dynamodbClient: dynamodb.NewFromConfig(cfg),
	}

	lambda.Start(cfn.LambdaWrap(svc.handler))
}

func (svc *DefaultDataService) handler(ctx context.Context, cfnEvent cfn.Event) (string, map[string]interface{}, error) {

	svc.ctx = ctx

	response := make(map[string]interface{})

	switch cfnEvent.RequestType {
	case cfn.RequestCreate:
		svc.logger.Printf("Create Request initiated")
		svc.SetRequestData(cfnEvent)
		err := svc.handleCreateDDBData()
		if err != nil {
			return "incomplete", response, err
		}

	case cfn.RequestUpdate:
		// Will overwrite the data as its a put operation. But if we are changing the PK, we'll need to delete the entries manually
		svc.logger.Printf("Update Request initiated")
		svc.SetRequestData(cfnEvent)
		err := svc.handleCreateDDBData()
		if err != nil {
			return "incomplete", response, err
		}
	}

	return "completed", response, nil
}

func (svc *DefaultDataService) SetRequestData(cfnEvent cfn.Event) error {

	svc.logger.Printf("DDBTableName: %s\n , Data : %s \n PrimaryKeys : %s", cfnEvent.ResourceProperties["DDBTableName"].(string), cfnEvent.ResourceProperties["Data"].(string), cfnEvent.ResourceProperties["PrimaryKeys"].(string))

	data := CFNRequestData{
		DDBTableName: cfnEvent.ResourceProperties["DDBTableName"].(string),
		Data:         cfnEvent.ResourceProperties["Data"].(string),
		PrimaryKeys:  cfnEvent.ResourceProperties["PrimaryKeys"].(string),
	}

	svc.cfnData = data

	return nil
}

func (svc *DefaultDataService) handleCreateDDBData() error {

	var inputData interface{}

	_ = json.Unmarshal([]byte(svc.cfnData.Data), &inputData)

	putItemInput, err := dynamodb_attributevalue.MarshalMap(inputData)
	if err != nil {
		svc.logger.Printf("Unable to Marshal to putItem Input")
		return err
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.cfnData.DDBTableName),
		Item:      putItemInput,
	})
	if err != nil {
		return err
	}

	return nil
}

func (svc *DefaultDataService) HandleDeleteDDBData() error {

	var inputData interface{}

	_ = json.Unmarshal([]byte(svc.cfnData.PrimaryKeys), &inputData)

	deleteItemKey, _ := dynamodb_attributevalue.MarshalMap(inputData)

	_, err := svc.dynamodbClient.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.cfnData.DDBTableName),
		Key:       deleteItemKey,
	})
	if err != nil {
		return err
	}

	return nil
}
