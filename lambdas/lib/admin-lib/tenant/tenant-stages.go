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
- The Tenant stages defines the stage at which a certain Tenant is currently on-boarded.
- Each stage has a defined set of status and only after a certain stage is completed, a tenant can move to another stage.


- The Tenant Stage table data can be added only when a Tenant has been created in the TenantDetails Table.
The tenant Tenant Stages can are defined under the tenants lifecycle file.

*/

// -------------- DDB Table Tenant Stage Details ----------------
type TenantStages struct {
	TenantId  string `dynamodbav:"TenantId" json:"TenantId"` // Ref TenantId , Required Field.
	StageName string `dynamodbav:"StageName" json:"StageName" `
	StageId   string `dynamodbav:"StageId" json:"StageId"` // Unique stage ID defined in the tenants Lifecycle

	StageStatus   string        `dynamodbav:"StageStatus" json:"StageStatus"` // Overall Status of the Stage
	CommentsCount int           `dynamodbav:"CommentsCount" json:"CommentsCount"`
	StageComments []CommentData `dynamodbav:"StageComments" json:"StageComments"` // Admin added comments for each stage. This cannot be deleted.
}
type CommentData struct {
	Comment         string `dynamodbav:"Comment" json:"Comment"`
	CommentBy       string `dynamodbav:"CommentBy" json:"CommentBy"`
	UpdateTimeStamp string `dynamodbav:"UpdateTimeStamp" json:"UpdateTimeStamp"`
}
type TenantStageService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	TenantStagesTable          string
	TenantStages_TenantIdIndex string
}

// Create Tenant Stage service function.
func CreateTenantStageService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *TenantStageService {
	return &TenantStageService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

func (svc *TenantStageService) GetAllTenantStages(tenantId string) ([]TenantStages, error) {

	query_output, err := svc.dynamodbClient.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.TenantStagesTable),
		IndexName:              aws.String(svc.TenantStages_TenantIdIndex),
		KeyConditionExpression: aws.String("TenantId = :TenantId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":TenantId": &dynamodb_types.AttributeValueMemberS{Value: tenantId},
		},
	})
	if err != nil {
		var conditionFailed *dynamodb_types.ConditionalCheckFailedException
		if !errors.As(err, &conditionFailed) {
			svc.logger.Printf("Could not Query Tenant Stages Table for TenantId:" + tenantId + " Failed with Error: " + err.Error())
			return []TenantStages{}, err
		}
	}

	if query_output.Count == 0 {
		svc.logger.Printf("No Stages found for this TenantId:" + tenantId)
		return []TenantStages{}, nil
	}

	allTenantStageData := []TenantStages{}
	for _, stageItem := range query_output.Items {
		stageData := TenantStages{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &stageData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal tenant stage data :" + tenantId + " Error : " + err.Error())
			return []TenantStages{}, err
		}
		allTenantStageData = append(allTenantStageData, stageData)
	}

	return allTenantStageData, nil

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
		TenantId: "abc-123",
		StageId : "STG-1",

		StageStatus : "Assigned",

		Comment : "comment-abc",
		CommentBy : "Name of person logged in "

}
*/
type PostReqNewStageData struct {
	TenantId  string `json:"TenantId"`
	StageName string `json:"StageName"`
	StageId   string `json:"StageId"` // Ref StageIDs as per const defined under - tenants lifecycle

	StageStatus string `json:"StageStatus"` // Status can be as per const defined under - tenants lifecycle

	Comment   string `json:"Comment"`
	CommentBy string `json:"CommentBy"`
}

// Add new comments or update the status
func (svc *TenantStageService) AddNewStageData(stageData PostReqNewStageData) error {

	currentTimeStamp := utils.GenerateTimestamp()

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantStagesTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"StageId": &dynamodb_types.AttributeValueMemberS{
				Value: stageData.StageId,
			},
			"TenantId": &dynamodb_types.AttributeValueMemberS{
				Value: stageData.TenantId,
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
		svc.logger.Printf("[ERROR] Failed to Update the TenantStage Table for TenantId :%s and Incoming Data: %v", stageData.TenantId, stageData)
		return err
	}

	return nil
}
