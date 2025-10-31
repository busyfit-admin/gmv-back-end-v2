package adminlib

import (
	"context"
	"errors"
	"log"

	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"

	utils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

/*
- The Supplier stages define the stage at which a certain Supplier is currently on-boarded.
- Each stage has a defined set of statuses and only after a certain stage is completed, a supplier can move to another stage.


- The Supplier Stage table data can be added only when a Supplier has been created in the SupplierDetails Table.
The Supplier Stages are defined under the supplier lifecycle file.

*/

// -------------- DDB Table Supplier Stage Details ----------------
type SupplierStages struct {
	SupplierId string `dynamodbav:"SupplierId" json:"SupplierId"` // Ref SupplierId , Required Field.
	StageName  string `dynamodbav:"StageName" json:"StageName" `
	StageId    string `dynamodbav:"StageId" json:"StageId"` // Unique stage ID defined in the supplier Lifecycle

	StageStatus   string        `dynamodbav:"StageStatus" json:"StageStatus"` // Overall Status of the Stage
	CommentsCount int           `dynamodbav:"CommentsCount" json:"CommentsCount"`
	StageComments []CommentData `dynamodbav:"StageComments" json:"StageComments"` // Admin added comments for each stage. This cannot be deleted.
}
type CommentData struct {
	Comment         string `dynamodbav:"Comment" json:"Comment"`
	CommentBy       string `dynamodbav:"CommentBy" json:"CommentBy"`
	UpdateTimeStamp string `dynamodbav:"UpdateTimeStamp" json:"UpdateTimeStamp"`
}
type SupplierStageService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	SupplierStagesTable            string
	SupplierStages_SupplierIdIndex string
}

// Create Supplier Stage service function.
func CreateSupplierStageService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *SupplierStageService {
	return &SupplierStageService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

func (svc *SupplierStageService) GetAllSupplierStages(supplierId string) ([]SupplierStages, error) {

	query_output, err := svc.dynamodbClient.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.SupplierStagesTable),
		IndexName:              aws.String(svc.SupplierStages_SupplierIdIndex),
		KeyConditionExpression: aws.String("SupplierId = :SupplierId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":SupplierId": &dynamodb_types.AttributeValueMemberS{Value: supplierId},
		},
	})
	if err != nil {
		var conditionFailed *dynamodb_types.ConditionalCheckFailedException
		if !errors.As(err, &conditionFailed) {
			svc.logger.Printf("Could not Query Supplier Stages Table for SupplierId:" + supplierId + " Failed with Error: " + err.Error())
			return []SupplierStages{}, err
		}
	}

	if query_output.Count == 0 {
		svc.logger.Printf("No Stages found for this SupplierId:" + supplierId)
		return []SupplierStages{}, nil
	}

	allSupplierStageData := []SupplierStages{}
	for _, stageItem := range query_output.Items {
		stageData := SupplierStages{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &stageData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal supplier stage data :" + supplierId + " Error : " + err.Error())
			return []SupplierStages{}, err
		}
		allSupplierStageData = append(allSupplierStageData, stageData)
	}

	return allSupplierStageData, nil

}

/*
Appending data to Dynamodb List ex using boto3:

		table = get_dynamodb_resource().Table("table_name")
		result = table.update_item(
			Key={
				'hash_key': hash_key,
				'range_key': range_key
			},
			UpdateExpression="SET some_attr = list_append(some_attr, :i)",
			ExpressionAttributeValues={
				':i': [some_value],
			},
			ReturnValues="UPDATED_NEW"
		)
		if result['ResponseMetadata']['HTTPStatusCode'] == 200 and 'Attributes' in result:
			return result['Attributes']['some_attr']

*/

/*
Input from Admin Portal to add new comments / status of a certain stage

ex Input from Front end:

	{
		SupplierId: "abc-123",
		StageId : "STG-1",

		StageStatus : "Assigned",

		Comment : "comment-abc",
		CommentBy : "Name of person logged in "

}
*/
type PostReqNewStageData struct {
	SupplierId string `json:"SupplierId"`
	StageName  string `json:"StageName"`
	StageId    string `json:"StageId"` // Ref StageIDs as per const defined under - suppliers lifecycle

	StageStatus string `json:"StageStatus"` // Status can be as per const defined under - suppliers lifecycle

	Comment   string `json:"Comment"`
	CommentBy string `json:"CommentBy"`
}

// Add new comments or update the status
func (svc *SupplierStageService) AddNewStageData(stageData PostReqNewStageData) error {

	currentTimeStamp := utils.GenerateTimestamp()

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.SupplierStagesTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"StageId": &dynamodb_types.AttributeValueMemberS{
				Value: stageData.StageId,
			},
			"SupplierId": &dynamodb_types.AttributeValueMemberS{
				Value: stageData.SupplierId,
			},
		},

		UpdateExpression: aws.String("ADD CommentsCount :One SET StageComments = list_append(if_not_exists(StageComments, :EmptyList), :Comment), StageStatus = :StageStatus, LastModifiedDate = :LastModifiedDate"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":Comment": &dynamodb_types.AttributeValueMemberL{
				Value: []dynamodb_types.AttributeValue{
					&dynamodb_types.AttributeValueMemberM{
						Value: map[string]dynamodb_types.AttributeValue{
							"Comment": &dynamodb_types.AttributeValueMemberS{
								Value: stageData.Comment,
							},
							"CommentBy": &dynamodb_types.AttributeValueMemberS{
								Value: stageData.CommentBy,
							},
							"UpdateTimeStamp": &dynamodb_types.AttributeValueMemberS{
								Value: currentTimeStamp,
							},
						},
					},
				},
			},
			":EmptyList": &dynamodb_types.AttributeValueMemberL{},
			":One":       &dynamodb_types.AttributeValueMemberN{Value: "1"},
			":LastModifiedDate": &dynamodb_types.AttributeValueMemberS{
				Value: currentTimeStamp,
			},
			":StageStatus": &dynamodb_types.AttributeValueMemberS{Value: stageData.StageStatus},
		},
		ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)

	if err != nil {
		svc.logger.Printf("[ERROR] Failed to Update the SupplierStage Table for SupplierId :%s and Incoming Data: %v", stageData.SupplierId, stageData)
		return err
	}

	return nil
}
