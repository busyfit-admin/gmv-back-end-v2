package GISlib

import (
	"context"
	// "encoding/json"
	// "io/ioutil"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/crolly/dyngeo"
	"github.com/gofrs/uuid"
)

type GISService struct {
	ctx            context.Context
	dynamodbClient *dynamodb.DynamoDB
	logger         *log.Logger
	dg             *dyngeo.DynGeo
	GIS_Table      string
}

type Starbucks struct {
	Position Position `json:"position"`
	Name     string   `json:"name"`
	Address  string   `json:"address"`
	Phone    string   `json:"phone"`
}

type Position struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

const BATCH_SIZE = 25

func CreateGISService(ctx context.Context, ddbClient *dynamodb.DynamoDB, logger *log.Logger, GIS_Table string) *GISService {
	dg, err := dyngeo.New(dyngeo.DynGeoConfig{
		DynamoDBClient: ddbClient,
		HashKeyLength:  5,
		TableName:      GIS_Table,
	})
	if err != nil {
		logger.Fatalf("Failed to create DynGeo instance: %v", err)
	}
	return &GISService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
		dg:             dg,
	}
}

// func (svc *GISService) SetupTable() {
// 	createTableInput := dyngeo.GetCreateTableRequest(svc.dg.Config)
// 	createTableInput.ProvisionedThroughput.ReadCapacityUnits = aws.Int64(5)
// 	createTableOutput, err := svc.dg.CreateTable(createTableInput)
// 	if err != nil {
// 		svc.logger.Fatalf("Failed to create table: %v", err)
// 	}
// 	svc.logger.Println("Table created")
// 	svc.logger.Println(createTableOutput)
// }

func (svc *GISService) LoadData(coffeeShops []Starbucks) {
	batchInput := []dyngeo.PutPointInput{}
	for _, s := range coffeeShops {
		id, err := uuid.NewV4()
		if err != nil {
			panic(err)
		}
		input := dyngeo.PutPointInput{
			PutItemInput: dynamodb.PutItemInput{
				Item: map[string]*dynamodb.AttributeValue{
					"name":    &dynamodb.AttributeValue{S: aws.String(s.Name)},
					"address": &dynamodb.AttributeValue{S: aws.String(s.Address)},
				},
			},
		}
		input.RangeKeyValue = id
		input.GeoPoint = dyngeo.GeoPoint{
			Latitude:  s.Position.Latitude,
			Longitude: s.Position.Longitude,
		}
		batchInput = append(batchInput, input)
	}

	batches := [][]dyngeo.PutPointInput{}
	for BATCH_SIZE < len(batchInput) {
		batchInput, batches = batchInput[BATCH_SIZE:], append(batches, batchInput[0:BATCH_SIZE:BATCH_SIZE])
	}
	batches = append(batches, batchInput)

	for count, batch := range batches {
		output, err := svc.dg.BatchWritePoints(batch)
		if err != nil {
			panic(err)
		}
		svc.logger.Printf("Batch %d written: %s", count, output)
	}
}

func (svc *GISService) QueryData(lat float64, long float64, radius int) ([]Starbucks, error) {
	start := time.Now()
	sbs := []Starbucks{}

	err := svc.dg.QueryRadius(dyngeo.QueryRadiusInput{
		CenterPoint: dyngeo.GeoPoint{
			Latitude:  lat,
			Longitude: long,
		},
		RadiusInMeter: radius,
	}, &sbs)
	if err != nil {
		svc.logger.Printf("Failed to query data: %v", err)
		return nil, err
	}

	svc.logger.Printf("Executed in: %v", time.Since(start))
	return sbs, nil
}
