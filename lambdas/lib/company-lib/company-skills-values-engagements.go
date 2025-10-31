package Companylib

import (
	"context"
	"errors"
	"fmt"
	"log"

	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

type TenantSkillsTable struct {
	SkillId   string `dynamodbav:"SkillId"`
	SkillName string `dynamodbav:"SkillName"`
	SkillDesc string `dynamodbav:"SkillDesc"`
}

type TenantValuesTable struct {
	ValueId   string `dynamodbav:"ValueId"`
	ValueName string `dynamodbav:"ValueName"`
	ValueDesc string `dynamodbav:"ValueDesc"`
}

type TenantMilestonesTable struct {
	MilestoneId   string `dynamodbav:"MilestoneId"`
	MilestoneName string `dynamodbav:"MilestoneName"`
	MilestoneDesc string `dynamodbav:"MilestoneDesc"`
}

type TenantMetricsTable struct {
	MetricId   string `dynamodbav:"MetricId"`
	MetricName string `dynamodbav:"MetricName"`
	MetricDesc string `dynamodbav:"MetricDesc"`
}

type TenantEngagementTable struct {
	EngagementId string `dynamodbav:"EngagementId" json:"EngagementId"` // PK : format : TEAMFEED-xxxx , TEAMEVENT#, GRPFEED#xxxx , GRPDOCS#xxxx, USERAPP#xxxx

	EntityId          string     `dynamodbav:"EntityId" json:"EntityId"`                   // format : USER-xxx , TEAM-xxxx, GROUP-xxxx. This is the entity of Team or User or Group who is being appreciated or FEED Is being created
	ProvidedBy        string     `dynamodbav:"ProvidedBy" json:"ProvidedBy"`               // Source EntityId , format : USER-xxx . This is the entity who provided the appreciation
	ProvidedByContent EntityData `dynamodbav:"ProvidedByContent" json:"ProvidedByContent"` // Source EntityId , format : USER-xxx . This is the entity who provided the appreciation

	Message string `dynamodbav:"Message" json:"Message"` // Message in format of Rich Text

	// Skill, Value, Milestone, Metrics are the fields which are used to store the Values of the Engagement
	Skill     []string `dynamodbav:"Skill" json:"Skill"`
	Value     []string `dynamodbav:"Value" json:"Value"`
	Milestone []string `dynamodbav:"Milestone" json:"Milestone"`
	Metrics   []string `dynamodbav:"Metrics" json:"Metrics"`

	Images []string              `dynamodbav:"Images" json:"Images"` // List of Image URLs
	Likes  map[string]EntityData `dynamodbav:"Likes" json:"Likes"`   // Number of Likes, username as map key and EntityData as value

	Timestamp string `dynamodbav:"Timestamp" json:"Timestamp"` // Timestamp of the Created Engagement

	// Events specific Fields
	EventTitle   string                `dynamodbav:"EventTitle" json:"EventTitle"`     // Event Title
	EventDesc    string                `dynamodbav:"EventDesc" json:"EventDesc"`       // Event Description
	FromDateTime string                `dynamodbav:"FromDateTime" json:"FromDateTime"` // Event Start Date Time
	ToDateTime   string                `dynamodbav:"ToDateTime" json:"ToDateTime"`     // Event End Date Time
	Location     string                `dynamodbav:"Location" json:"Location"`         // Event Location
	MeetingLink  string                `dynamodbav:"MeetingLink" json:"MeetingLink"`   // Event Meeting Link
	RSVP         map[string]EntityData `dynamodbav:"RSVP" json:"RSVP"`                 // RSVP List, username as map key and EntityData as value

	// Transferred Points
	TransferredPoints map[string]int `dynamodbav:"TransferredPoints" json:"TransferredPoints"` // Transferred Points to other entities
	TaggedUsers       []string       `dynamodbav:"TaggedUsers" json:"TaggedUsers"`
}

type EntityData struct {
	DisplayName string `dynamodbav:"DisplayName" json:"DisplayName"`
	Designation string `dynamodbav:"Designation" json:"Designation"`
	ProfilePic  string `dynamodbav:"ProfilePic" json:"ProfilePic"`

	// For RSVP
	IsRSVP bool `dynamodbav:"IsRSVP" json:"IsRSVP"`

	// For Likes
	IsLike bool `dynamodbav:"IsLike" json:"IsLike"`
}

type TenantEngagementService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	TenantSkillsTable     string
	TenantValuesTable     string
	TenantMilestonesTable string
	TenantMetricsTable    string

	TenantEngagementTable                  string //  EngagementId Hash Index and EntityId Range Index
	TenantEngagementEntityIdIndex          string // EntityId Hash Index and EngagementId Range Index
	TenantEngagementEntityIdTimestampIndex string // EntityId Hash Index and Timestamp Range Index
}

func CreateEngagementService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *TenantEngagementService {
	return &TenantEngagementService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

// Performs both  Put , delete Skill actions based on ReqType(Allowed Values: "POST", "DELETE")
func (svc *TenantEngagementService) PutSkillsToDynamoDB(skills []TenantSkillsTable, reqType string) error {
	// Create the batch write request

	if !(reqType == "POST" || reqType == "DELETE") {
		return fmt.Errorf("ReqType Not found")
	}

	var writeRequests []types.WriteRequest

	for _, skill := range skills {
		av, err := attributevalue.MarshalMap(skill)
		if err != nil {
			return fmt.Errorf("failed to marshal skill: %v", err)
		}
		if reqType == "POST" {
			writeRequests = append(writeRequests, types.WriteRequest{
				PutRequest: &types.PutRequest{
					Item: av,
				},
			})
		} else if reqType == "DELETE" {
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"SkillId": &types.AttributeValueMemberS{Value: skill.SkillId},
					},
				},
			})
		}

	}

	batchWriteInput := dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			svc.TenantSkillsTable: writeRequests,
		},
	}

	// Perform the batch write
	_, err := svc.dynamodbClient.BatchWriteItem(svc.ctx, &batchWriteInput)
	if err != nil {
		return fmt.Errorf("failed to write batch to DynamoDB: %v", err)
	}

	svc.logger.Printf("Batch Put/Delete op completed for skills")

	return nil
}

// Performs both  Put , delete Values actions based on ReqType(Allowed Values: "POST", "DELETE")
func (svc *TenantEngagementService) PutValuesToDynamoDB(values []TenantValuesTable, reqType string) error {
	// Create the batch write request

	if !(reqType == "POST" || reqType == "DELETE") {
		return fmt.Errorf("ReqType Not found")
	}

	var writeRequests []types.WriteRequest

	for _, value := range values {
		av, err := attributevalue.MarshalMap(value)
		if err != nil {
			return fmt.Errorf("failed to marshal skill: %v", err)
		}
		if reqType == "POST" {
			writeRequests = append(writeRequests, types.WriteRequest{
				PutRequest: &types.PutRequest{
					Item: av,
				},
			})
		} else if reqType == "DELETE" {
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"ValueId": &types.AttributeValueMemberS{Value: value.ValueId},
					},
				},
			})
		}

	}

	batchWriteInput := dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			svc.TenantValuesTable: writeRequests,
		},
	}

	// Perform the batch write
	_, err := svc.dynamodbClient.BatchWriteItem(svc.ctx, &batchWriteInput)
	if err != nil {
		return fmt.Errorf("failed to write batch to DynamoDB: %v", err)
	}

	svc.logger.Printf("Batch Put/Delete op completed for skills")

	return nil
}

// Performs both  Put , delete Milestones actions based on ReqType(Allowed Values: "POST", "DELETE")
func (svc *TenantEngagementService) PutMilestonesToDynamoDB(milestones []TenantMilestonesTable, reqType string) error {
	// Create the batch write request

	if !(reqType == "POST" || reqType == "DELETE") {
		return fmt.Errorf("ReqType Not found")
	}

	var writeRequests []types.WriteRequest

	for _, value := range milestones {
		av, err := attributevalue.MarshalMap(value)
		if err != nil {
			return fmt.Errorf("failed to marshal skill: %v", err)
		}
		if reqType == "POST" {
			writeRequests = append(writeRequests, types.WriteRequest{
				PutRequest: &types.PutRequest{
					Item: av,
				},
			})
		} else if reqType == "DELETE" {
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"MilestoneId": &types.AttributeValueMemberS{Value: value.MilestoneId},
					},
				},
			})
		}

	}

	batchWriteInput := dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			svc.TenantValuesTable: writeRequests,
		},
	}

	// Perform the batch write
	_, err := svc.dynamodbClient.BatchWriteItem(svc.ctx, &batchWriteInput)
	if err != nil {
		return fmt.Errorf("failed to write batch to DynamoDB: %v", err)
	}

	svc.logger.Printf("Batch Put/Delete op completed for skills")

	return nil
}

// Performs both  Put , delete Milestones actions based on ReqType(Allowed Values: "POST", "DELETE")
func (svc *TenantEngagementService) PutMetricsToDynamoDB(metrics []TenantMetricsTable, reqType string) error {
	// Create the batch write request

	if !(reqType == "POST" || reqType == "DELETE") {
		return fmt.Errorf("ReqType Not found")
	}

	var writeRequests []types.WriteRequest

	for _, value := range metrics {
		av, err := attributevalue.MarshalMap(value)
		if err != nil {
			return fmt.Errorf("failed to marshal skill: %v", err)
		}
		if reqType == "POST" {
			writeRequests = append(writeRequests, types.WriteRequest{
				PutRequest: &types.PutRequest{
					Item: av,
				},
			})
		} else if reqType == "DELETE" {
			writeRequests = append(writeRequests, types.WriteRequest{
				DeleteRequest: &types.DeleteRequest{
					Key: map[string]types.AttributeValue{
						"MetricId": &types.AttributeValueMemberS{Value: value.MetricId},
					},
				},
			})
		}

	}

	batchWriteInput := dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			svc.TenantValuesTable: writeRequests,
		},
	}

	// Perform the batch write
	_, err := svc.dynamodbClient.BatchWriteItem(svc.ctx, &batchWriteInput)
	if err != nil {
		return fmt.Errorf("failed to write batch to DynamoDB: %v", err)
	}

	svc.logger.Printf("Batch Put/Delete op completed for skills")

	return nil
}

// Create an appreciation to an entityID
func (svc *TenantEngagementService) CreateEngagement(newEngagement TenantEngagementTable) error {

	// Enter Empty map for Likes and RSVP
	newEngagement.Likes = make(map[string]EntityData)
	newEngagement.RSVP = make(map[string]EntityData)

	av, err := attributevalue.MarshalMap(newEngagement)
	if err != nil {
		return err
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx,
		&dynamodb.PutItemInput{
			TableName: aws.String(svc.TenantEngagementTable),
			Item:      av,
		})
	if err != nil {
		return err
	}

	return nil
}

// Updates an Engagement event RSVP
func (svc *TenantEngagementService) UpdateEngagementRSVP(engagementId string, entityId string, entityUserName string, rsvp EntityData) error {

	// Query the EngagementId and get the existing RSVP count
	output, err := svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.TenantEngagementTable),
		Key: map[string]types.AttributeValue{
			"EngagementId": &types.AttributeValueMemberS{Value: engagementId},
			"EntityId":     &types.AttributeValueMemberS{Value: entityId},
		},
	})
	if err != nil {
		return err
	}

	if output.Item == nil {
		return fmt.Errorf("engagement not found: %v", engagementId)
	}

	_, found := output.Item["RSVP"]
	if !found {
		// Create the empty RSVP map, but only if it doesn't exist yet.
		_, err = svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
			TableName:           aws.String(svc.TenantEngagementTable),
			ConditionExpression: aws.String("attribute_not_exists(RSVP)"),
			Key: map[string]dynamodb_types.AttributeValue{
				"EngagementId": &dynamodb_types.AttributeValueMemberS{Value: engagementId},
			},
			UpdateExpression: aws.String("SET RSVP = :RSVP"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":RSVP": &dynamodb_types.AttributeValueMemberM{},
			},
			ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
		})
		if err != nil {
			var conditionFailed *dynamodb_types.ConditionalCheckFailedException
			if !errors.As(err, &conditionFailed) {
				// ConditionalCheckFailedException means that item already exists, so we can continue, otherwise it's
				// a real error which we should return.
				return err
			}
		}
	}

	// Create the entity data map
	entityDataDDB, err := attributevalue.MarshalMap(rsvp)
	if err != nil {
		svc.logger.Printf("failed to marshal entity data: %v", err)
		return err
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantEngagementTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"EngagementId": &dynamodb_types.AttributeValueMemberS{Value: engagementId},
		},
		UpdateExpression: aws.String("SET RSVP.#USERNAME = :RSVP_DATA"),
		ExpressionAttributeNames: map[string]string{
			"#USERNAME": entityUserName,
		},
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":RSVP_DATA": &dynamodb_types.AttributeValueMemberM{
				Value: entityDataDDB,
			},
		},
		ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return err
	}

	svc.logger.Printf("RSVP updated for engagement: %s from %s", engagementId, entityUserName)

	return nil
}

// Updates a Feed Likes
func (svc *TenantEngagementService) UpdateEngagementFeedLikes(engagementId string, entityId string, entityUserName string, like EntityData) error {

	// Query the EngagementId and get the existing RSVP count
	output, err := svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.TenantEngagementTable),
		Key: map[string]types.AttributeValue{
			"EngagementId": &types.AttributeValueMemberS{Value: engagementId},
			"EntityId":     &types.AttributeValueMemberS{Value: entityId},
		},
	})
	if err != nil {
		return err
	}

	if output.Item == nil {
		return fmt.Errorf("engagement Id : %s not found", engagementId)
	}

	_, found := output.Item["Likes"]
	if !found {
		// Create the empty Likes map, but only if it doesn't exist yet.
		_, err = svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
			TableName:           aws.String(svc.TenantEngagementTable),
			ConditionExpression: aws.String("attribute_not_exists(Likes)"),
			Key: map[string]dynamodb_types.AttributeValue{
				"EngagementId": &dynamodb_types.AttributeValueMemberS{Value: engagementId},
				"EntityId":     &types.AttributeValueMemberS{Value: entityId},
			},
			UpdateExpression: aws.String("SET Likes = :Likes"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":Likes": &dynamodb_types.AttributeValueMemberM{},
			},
			ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
		})
		if err != nil {
			var conditionFailed *dynamodb_types.ConditionalCheckFailedException
			if !errors.As(err, &conditionFailed) {
				// ConditionalCheckFailedException means that item already exists, so we can continue, otherwise it's
				// a real error which we should return.
				return err
			}
		}
	}

	// Create the entity data map
	entityDataDDB, err := attributevalue.MarshalMap(like)
	if err != nil {
		svc.logger.Printf("failed to marshal entity data: %v", err)
		return err
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantEngagementTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"EngagementId": &dynamodb_types.AttributeValueMemberS{Value: engagementId},
			"EntityId":     &types.AttributeValueMemberS{Value: entityId},
		},
		UpdateExpression: aws.String("SET Likes.#USERNAME = :LIKE_DATA"),
		ExpressionAttributeNames: map[string]string{
			"#USERNAME": entityUserName,
		},
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":LIKE_DATA": &dynamodb_types.AttributeValueMemberM{
				Value: entityDataDDB,
			},
		},
		ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return err
	}

	svc.logger.Printf("Likes updated for engagement: %s from %s", engagementId, entityUserName)

	return nil
}

type AppreciationObject struct {
	ObjId      string `json:"ObjectId"`
	ObjectType string `json:"ObjectType"`
	Value      string `json:"Value"`
	Desc       string `json:"Desc"`
}

// Get all Skills Object
func (svc *TenantEngagementService) GetAllSkills() ([]AppreciationObject, error) {

	output, err := svc.dynamodbClient.Scan(svc.ctx, &dynamodb.ScanInput{
		TableName: &svc.TenantSkillsTable,
		AttributesToGet: []string{
			"SkillName",
			"SkillId",
			"SkillDesc",
		},
	})
	if err != nil {
		return []AppreciationObject{}, err
	}

	if output.Count == 0 {
		return []AppreciationObject{}, nil
	}

	var allskills []AppreciationObject
	for _, item := range output.Items {
		var skill TenantSkillsTable
		err := attributevalue.UnmarshalMap(item, &skill)
		if err != nil {
			return []AppreciationObject{}, err
		}
		allskills = append(allskills, AppreciationObject{
			ObjectType: "Skill",
			Value:      skill.SkillName,
			ObjId:      skill.SkillId,
			Desc:       skill.SkillDesc,
		})
	}

	return allskills, nil
}

// Get all Values
func (svc *TenantEngagementService) GetAllValues() ([]AppreciationObject, error) {

	output, err := svc.dynamodbClient.Scan(svc.ctx, &dynamodb.ScanInput{
		TableName: &svc.TenantValuesTable,
		AttributesToGet: []string{
			"ValueName",
			"ValueId",
			"ValueDesc",
		},
	})
	if err != nil {
		return []AppreciationObject{}, err
	}

	if output.Count == 0 {
		return []AppreciationObject{}, nil
	}

	var allValues []AppreciationObject
	for _, item := range output.Items {
		var value TenantValuesTable
		err := attributevalue.UnmarshalMap(item, &value)
		if err != nil {
			return []AppreciationObject{}, err
		}
		allValues = append(allValues, AppreciationObject{
			ObjectType: "Value",
			Value:      value.ValueName,
			ObjId:      value.ValueId,
			Desc:       value.ValueDesc,
		})
	}

	return allValues, nil
}

// Get all Milestones
func (svc *TenantEngagementService) GetAllMilestones() ([]AppreciationObject, error) {

	output, err := svc.dynamodbClient.Scan(svc.ctx, &dynamodb.ScanInput{
		TableName: &svc.TenantMilestonesTable,
		AttributesToGet: []string{
			"MilestoneName",
			"MilestoneId",
			"MilestoneDesc",
		},
	})
	if err != nil {
		return []AppreciationObject{}, err
	}

	if output.Count == 0 {
		return []AppreciationObject{}, nil
	}

	var allValues []AppreciationObject
	for _, item := range output.Items {
		var value TenantMilestonesTable
		err := attributevalue.UnmarshalMap(item, &value)
		if err != nil {
			return []AppreciationObject{}, err
		}
		allValues = append(allValues, AppreciationObject{
			ObjectType: "Milestone",
			Value:      value.MilestoneName,
			ObjId:      value.MilestoneId,
			Desc:       value.MilestoneDesc,
		})
	}

	return allValues, nil
}

// Get all Metrics
func (svc *TenantEngagementService) GetAllMetrics() ([]AppreciationObject, error) {

	output, err := svc.dynamodbClient.Scan(svc.ctx, &dynamodb.ScanInput{
		TableName: &svc.TenantMetricsTable,
		AttributesToGet: []string{
			"MetricName",
			"MetricId",
			"MetricDesc",
		},
	})
	if err != nil {
		return []AppreciationObject{}, err
	}

	if output.Count == 0 {
		return []AppreciationObject{}, nil
	}

	var allValues []AppreciationObject
	for _, item := range output.Items {
		var value TenantMetricsTable
		err := attributevalue.UnmarshalMap(item, &value)
		if err != nil {
			return []AppreciationObject{}, err
		}
		allValues = append(allValues, AppreciationObject{
			ObjectType: "Metric",
			Value:      value.MetricName,
			ObjId:      value.MetricId,
			Desc:       value.MetricDesc,
		})
	}

	return allValues, nil
}

type AllAppreciationsData struct {
	Skills     []AppreciationObject `json:"Skills"`
	Values     []AppreciationObject `json:"Values"`
	Milestones []AppreciationObject `json:"Milestones"`
	Metrics    []AppreciationObject `json:"Metrics"`
}

// Get All Appreciations Entity Values
func (svc *TenantEngagementService) GetAllAppreciationsObjects() (AllAppreciationsData, error) {

	skills, err := svc.GetAllSkills()
	if err != nil {
		return AllAppreciationsData{}, err
	}

	values, err := svc.GetAllValues()
	if err != nil {
		return AllAppreciationsData{}, err
	}

	milestones, err := svc.GetAllMilestones()
	if err != nil {
		return AllAppreciationsData{}, err
	}

	metrics, err := svc.GetAllMetrics()
	if err != nil {
		return AllAppreciationsData{}, err
	}

	return AllAppreciationsData{
		Skills:     skills,
		Values:     values,
		Milestones: milestones,
		Metrics:    metrics,
	}, nil
}

// Get All Engagements for an EntityId
func (svc *TenantEngagementService) FilterEngagementsForEntity(entityId string) ([]TenantEngagementTable, error) {

	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.TenantEngagementTable),
		KeyConditionExpression: aws.String("EntityId = :EntityId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":EntityId": &types.AttributeValueMemberS{Value: entityId},
		},
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryInput)
	if err != nil {
		return []TenantEngagementTable{}, err
	}

	if output.Count == 0 {
		return []TenantEngagementTable{}, nil
	}

	var allEngagements []TenantEngagementTable
	for _, item := range output.Items {
		var appreciation TenantEngagementTable
		err := attributevalue.UnmarshalMap(item, &appreciation)
		if err != nil {
			return []TenantEngagementTable{}, err
		}
		allEngagements = append(allEngagements, appreciation)
	}

	return allEngagements, nil
}

// Get GetAllEngagementsFeed for an EntityId in Feeds Section
type EngagementFeed struct {
	EngagementFeed   []TenantEngagementTable `json:"EngagementFeed"`
	NextEngagementId string                  `json:"NextEngagementId"`
}

func (svc *TenantEngagementService) GetAllEngagementsFeed(entityId string, next string, filter string) (EngagementFeed, error) {

	/*
		SELECT * FROM "TenantEngagementTable-test"."EntityId_Timestamp_Index"
		WHERE
		EntityId = 'TEAMID-2uZPMC8a'
		AND
		begins_with(EngagementId, 'TEAMFEED-')
		ORDER BY "EntityId","Timestamp" DESC
	*/
	stmt := fmt.Sprintf("SELECT * FROM \"%s\".\"%s\" WHERE EntityId = '%s' AND begins_with(EngagementId, '%s') ORDER BY \"EntityId\",\"Timestamp\" DESC", svc.TenantEngagementTable, svc.TenantEngagementEntityIdTimestampIndex, entityId, filter)

	svc.logger.Printf("EntityIdsPartiql: %s", stmt)

	queryInput := dynamodb.ExecuteStatementInput{
		Statement: aws.String(stmt),
		Limit:     aws.Int32(10),
	}

	if next != "" {
		queryInput.NextToken = aws.String(next)
	}

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &queryInput)
	if err != nil {
		svc.logger.Printf("Error in ExecuteStatement: %v", err)
		return EngagementFeed{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Engagements found")
		return EngagementFeed{}, nil
	}

	var allEngagements []TenantEngagementTable
	for _, item := range output.Items {
		var appreciation TenantEngagementTable
		err := attributevalue.UnmarshalMap(item, &appreciation)
		if err != nil {
			svc.logger.Printf("Error in UnmarshalMap: %v", err)
			return EngagementFeed{}, err
		}
		allEngagements = append(allEngagements, appreciation)
	}

	if output.NextToken != nil {
		return EngagementFeed{
			EngagementFeed:   allEngagements,
			NextEngagementId: *output.NextToken,
		}, nil
	}

	svc.logger.Printf("All Engagements fetched")

	return EngagementFeed{
		EngagementFeed:   allEngagements,
		NextEngagementId: "",
	}, nil
}

func (svc *TenantEngagementService) GetUserProfileAppreciations(entityId string, next string) (EngagementFeed, error) {

	// Get all Engagements for the Entity. Query on the EntityId
	svc.logger.Printf("GetUserProfileAppreciations: %s", entityId)

	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.TenantEngagementTable),
		IndexName:              aws.String(svc.TenantEngagementEntityIdIndex),
		KeyConditionExpression: aws.String("EntityId = :EntityId AND begins_with(EngagementId, :UserAppreciationPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":EntityId":               &types.AttributeValueMemberS{Value: entityId},
			":UserAppreciationPrefix": &types.AttributeValueMemberS{Value: "USER"},
		},
		Limit: aws.Int32(5),
	}

	if next != "" {
		queryInput.ExclusiveStartKey = map[string]types.AttributeValue{
			"EngagementId": &types.AttributeValueMemberS{Value: next},
		}
	}

	svc.logger.Printf("QueryInput: %v", queryInput)

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryInput)
	if err != nil {
		return EngagementFeed{}, err
	}
	if output.Count == 0 {
		return EngagementFeed{}, nil
	}

	var allEngagements []TenantEngagementTable
	for _, item := range output.Items {
		var appreciation TenantEngagementTable
		err := attributevalue.UnmarshalMap(item, &appreciation)
		if err != nil {
			return EngagementFeed{}, err
		}
		allEngagements = append(allEngagements, appreciation)
	}
	if output.LastEvaluatedKey != nil {
		return EngagementFeed{
			EngagementFeed:   allEngagements,
			NextEngagementId: output.LastEvaluatedKey["EngagementId"].(*types.AttributeValueMemberS).Value,
		}, nil
	}

	svc.logger.Printf("All Engagements fetched")

	return EngagementFeed{
		EngagementFeed:   allEngagements,
		NextEngagementId: "",
	}, nil
}

func (svc *TenantEngagementService) GetAllFeedGroups(entityIds []string, next string) (EngagementFeed, error) {
	// Get all Engagements for the Entity. Query on the EntityId

	// construct partiql stmt for dynamodb with all entity IDs in the list
	var entityIdsPartiql string
	for i, entityId := range entityIds {
		entityIdsPartiql += fmt.Sprintf("'%s'", entityId)
		if i != len(entityIds)-1 {
			entityIdsPartiql += ", "
		}
	}
	// SELECT * FROM "TenantEngagementTable-test"."EntityId-Timestamp-index" WHERE EntityId IN ('dev-group', 'test-group', 'GROUP-0') ORDER BY "EntityId","Timestamp" DESC
	stmt := fmt.Sprintf("SELECT * FROM \"%s\".\"%s\" WHERE EntityId IN (%s) ORDER BY \"EntityId\",\"Timestamp\" DESC", svc.TenantEngagementTable, svc.TenantEngagementEntityIdTimestampIndex, entityIdsPartiql)

	svc.logger.Printf("EntityIdsPartiql: %s", stmt)

	queryInput := dynamodb.ExecuteStatementInput{
		Statement: aws.String(stmt),
		Limit:     aws.Int32(10),
	}

	if next != "" {
		queryInput.NextToken = aws.String(next)
	}

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &queryInput)
	if err != nil {
		svc.logger.Printf("Error in ExecuteStatement: %v", err)
		return EngagementFeed{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Engagements found")
		return EngagementFeed{}, nil
	}

	var allEngagements []TenantEngagementTable
	for _, item := range output.Items {
		var appreciation TenantEngagementTable
		err := attributevalue.UnmarshalMap(item, &appreciation)
		if err != nil {
			svc.logger.Printf("Error in UnmarshalMap: %v", err)
			return EngagementFeed{}, err
		}
		allEngagements = append(allEngagements, appreciation)
	}

	if output.NextToken != nil {
		return EngagementFeed{
			EngagementFeed:   allEngagements,
			NextEngagementId: *output.NextToken,
		}, nil
	}

	svc.logger.Printf("All Engagements fetched")

	return EngagementFeed{
		EngagementFeed:   allEngagements,
		NextEngagementId: "",
	}, nil

}

func (svc *TenantEngagementService) GetAllEngagementEvents(entityId string, next string) (EngagementFeed, error) {

	// Get all Engagements for the Entity. Query on the EntityId

	queryInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.TenantEngagementTable),
		IndexName:              aws.String(svc.TenantEngagementEntityIdIndex),
		KeyConditionExpression: aws.String("EntityId = :EntityId AND begins_with(EngagementId, :TeamEventPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":EntityId":        &types.AttributeValueMemberS{Value: entityId},
			":TeamEventPrefix": &types.AttributeValueMemberS{Value: "TEAMEVENT-"},
		},
		Limit: aws.Int32(5),
	}

	if next != "" {
		queryInput.ExclusiveStartKey = map[string]types.AttributeValue{
			"EngagementId": &types.AttributeValueMemberS{Value: next},
		}
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryInput)
	if err != nil {
		return EngagementFeed{}, err
	}
	if output.Count == 0 {
		return EngagementFeed{}, nil
	}

	var allEngagements []TenantEngagementTable
	for _, item := range output.Items {
		var appreciation TenantEngagementTable
		err := attributevalue.UnmarshalMap(item, &appreciation)
		if err != nil {
			return EngagementFeed{}, err
		}
		allEngagements = append(allEngagements, appreciation)
	}
	if output.LastEvaluatedKey != nil {
		return EngagementFeed{
			EngagementFeed:   allEngagements,
			NextEngagementId: output.LastEvaluatedKey["EngagementId"].(*types.AttributeValueMemberS).Value,
		}, nil
	}

	return EngagementFeed{
		EngagementFeed:   allEngagements,
		NextEngagementId: "",
	}, nil
}

type EventAndAppreciationCount struct {
	Month             string `json:"Month"`
	EventsCount       int    `json:"EventsCount"`
	AppreciationCount int    `json:"AppreciationCount"`
	TotalAppreciation int    `json:"TotalAppreciation"`
}

type MonthlyStats struct {
	MonthData []EventAndAppreciationCount `json:"MonthData"`
}

func (svc *TenantEngagementService) CountOfEventsAndAppreciations(entityId string) (MonthlyStats, error) {
	svc.logger.Printf("Starting CountOfEventsAndAppreciations for EntityId: %s", entityId)

	monthlyStats := MonthlyStats{}
	now := time.Now()
	for i := 0; i < 3; i++ {
		startOfMonth := now.AddDate(0, -i, -now.Day()+1).Format(time.RFC3339)
		endOfMonth := now.AddDate(0, -i+1, -now.Day()).Format(time.RFC3339)
		month := now.AddDate(0, -i, 0).Format("2006-01")

		svc.logger.Printf("Processing data for month: %s (Start: %s, End: %s)", month, startOfMonth, endOfMonth)

		// Query for appreciations
		queryAppreciations := `SELECT * FROM "` + svc.TenantEngagementTable + `"."EntityId_Timestamp_Index"
		WHERE "EntityId" = ? AND ("Timestamp" BETWEEN ? AND ?)`

		svc.logger.Println("Executing PartiQL query for appreciations...")
		outputAppreciations, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
			Statement: aws.String(queryAppreciations),
			Parameters: []types.AttributeValue{
				&types.AttributeValueMemberS{Value: entityId},
				&types.AttributeValueMemberS{Value: startOfMonth},
				&types.AttributeValueMemberS{Value: endOfMonth},
			},
		})
		if err != nil {
			svc.logger.Printf("Error executing PartiQL query for appreciations: %v", err)
			return MonthlyStats{}, err
		}
		svc.logger.Printf("Appreciations query returned %d items", len(outputAppreciations.Items))

		// Query for events
		queryEvents := `SELECT * FROM "` + svc.TenantEngagementTable + `"."EntityId_Timestamp_Index"
		WHERE "EntityId" = ? AND ("Timestamp" BETWEEN ? AND ?) AND contains("EngagementId", 'TEAMEVENT')`

		svc.logger.Println("Executing PartiQL query for events...")
		outputEvents, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
			Statement: aws.String(queryEvents),
			Parameters: []types.AttributeValue{
				&types.AttributeValueMemberS{Value: entityId},
				&types.AttributeValueMemberS{Value: startOfMonth},
				&types.AttributeValueMemberS{Value: endOfMonth},
			},
		})
		if err != nil {
			svc.logger.Printf("Error executing PartiQL query for events: %v", err)
			return MonthlyStats{}, err
		}
		svc.logger.Printf("Events query returned %d items", len(outputEvents.Items))

		var appreciationCount, totalAppreciation int
		for _, item := range outputAppreciations.Items {
			var appreciation TenantEngagementTable
			err := attributevalue.UnmarshalMap(item, &appreciation)
			if err != nil {
				svc.logger.Printf("Error unmarshalling appreciation record: %v", err)
				continue
			}
			svc.logger.Printf("Processing appreciation item: %+v", appreciation)
			if strings.HasPrefix(appreciation.EngagementId, "TEAMFEED-") {
				appreciationCount++
			}
			if appreciation.TransferredPoints != nil {
				for _, points := range appreciation.TransferredPoints {
					totalAppreciation += points
				}
			}
		}

		eventCount := len(outputEvents.Items)
		svc.logger.Printf("Finalized counts for %s - Events: %d, Appreciations: %d, Total Appreciation: %d", month, eventCount, appreciationCount, totalAppreciation)

		monthlyStats.MonthData = append(monthlyStats.MonthData, EventAndAppreciationCount{
			Month:             month,
			AppreciationCount: appreciationCount,
			EventsCount:       eventCount,
			TotalAppreciation: totalAppreciation,
		})
	}

	svc.logger.Println("Final Monthly Stats:", monthlyStats)

	return monthlyStats, nil
}
