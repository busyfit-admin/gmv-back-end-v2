package Companylib

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

// TeamAttributeType represents the type of attribute (skill, value, milestone, metric)
type TeamAttributeType string

const (
	AttributeTypeSkill     TeamAttributeType = "SKILL"
	AttributeTypeValue     TeamAttributeType = "VALUE"
	AttributeTypeMilestone TeamAttributeType = "MILESTONE"
	AttributeTypeMetric    TeamAttributeType = "METRIC"
)

// TeamAttribute represents a unified structure for skills, values, milestones, and metrics
type TeamAttribute struct {
	AttributeId   string            `dynamodbav:"AttributeId" json:"attributeId"`     // PK: Format ATTR-{uuid}
	TeamId        string            `dynamodbav:"TeamId" json:"teamId"`               // SK: Format TEAM-{id}
	AttributeType TeamAttributeType `dynamodbav:"AttributeType" json:"attributeType"` // SKILL, VALUE, MILESTONE, METRIC
	Name          string            `dynamodbav:"Name" json:"name"`
	Description   string            `dynamodbav:"Description" json:"description"`
	IsDefault     bool              `dynamodbav:"IsDefault" json:"isDefault"` // True for system-created defaults
	CreatedAt     string            `dynamodbav:"CreatedAt" json:"createdAt"`
	CreatedBy     string            `dynamodbav:"CreatedBy" json:"createdBy"` // Username of creator
	UpdatedAt     string            `dynamodbav:"UpdatedAt" json:"updatedAt"`
}

// TeamAttributeService handles operations for team attributes
type TeamAttributeServiceV2 struct {
	ctx                           context.Context
	dynamodbClient                awsclients.DynamodbClient
	logger                        *log.Logger
	TeamAttributesTable           string // Table name
	TeamAttributesTeamIdIndex     string // GSI: TeamId-AttributeType-index
	TenantEngagementTable         string // For engagements
	TenantEngagementEntityIdIndex string
}

// CreateTeamAttributeService creates a new service instance
func CreateTeamAttributeServiceV2(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *TeamAttributeServiceV2 {
	return &TeamAttributeServiceV2{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

// InitializeDefaultAttributes creates the 3 default attributes for each type for a team
func (svc *TeamAttributeServiceV2) InitializeDefaultAttributes(teamId string, createdBy string) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	defaultAttributes := []TeamAttribute{
		// Default Skills
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeSkill,
			Name:          "Leadership",
			Description:   "Demonstrated ability to lead and inspire team members",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeSkill,
			Name:          "Communication",
			Description:   "Effective verbal and written communication skills",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeSkill,
			Name:          "Problem Solving",
			Description:   "Ability to analyze and resolve complex challenges",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		// Default Values
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeValue,
			Name:          "Integrity",
			Description:   "Consistently demonstrates honesty and strong moral principles",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeValue,
			Name:          "Teamwork",
			Description:   "Works collaboratively and supports team success",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeValue,
			Name:          "Innovation",
			Description:   "Brings creative ideas and embraces new approaches",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		// Default Milestones
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeMilestone,
			Name:          "First Quarter Achievement",
			Description:   "Successfully completed first quarter objectives",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeMilestone,
			Name:          "Project Completion",
			Description:   "Delivered project on time and within scope",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeMilestone,
			Name:          "Team Goal Achievement",
			Description:   "Contributed significantly to achieving team goals",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		// Default Metrics
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeMetric,
			Name:          "Productivity",
			Description:   "Measures output and efficiency in task completion",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeMetric,
			Name:          "Quality",
			Description:   "Measures the standard of work delivered",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
		{
			AttributeId:   fmt.Sprintf("ATTR-%s", uuid.New().String()),
			TeamId:        teamId,
			AttributeType: AttributeTypeMetric,
			Name:          "Engagement",
			Description:   "Measures active participation and involvement",
			IsDefault:     true,
			CreatedAt:     timestamp,
			CreatedBy:     createdBy,
			UpdatedAt:     timestamp,
		},
	}

	// Batch write all default attributes
	var writeRequests []types.WriteRequest
	for _, attr := range defaultAttributes {
		av, err := attributevalue.MarshalMap(attr)
		if err != nil {
			return fmt.Errorf("failed to marshal attribute: %v", err)
		}
		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: av,
			},
		})
	}

	// DynamoDB BatchWriteItem supports max 25 items per request
	batchWriteInput := dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]types.WriteRequest{
			svc.TeamAttributesTable: writeRequests,
		},
	}

	_, err := svc.dynamodbClient.BatchWriteItem(svc.ctx, &batchWriteInput)
	if err != nil {
		return fmt.Errorf("failed to initialize default attributes: %v", err)
	}

	svc.logger.Printf("Initialized default attributes for team: %s", teamId)
	return nil
}

// CreateCustomAttribute creates a new custom attribute for a team (admin only)
func (svc *TeamAttributeServiceV2) CreateCustomAttribute(attr TeamAttribute) error {
	// Generate new ID if not provided
	if attr.AttributeId == "" {
		attr.AttributeId = fmt.Sprintf("ATTR-%s", uuid.New().String())
	}

	// Set timestamps
	timestamp := time.Now().UTC().Format(time.RFC3339)
	attr.CreatedAt = timestamp
	attr.UpdatedAt = timestamp
	attr.IsDefault = false // Custom attributes are never defaults

	// Validate attribute type
	switch attr.AttributeType {
	case AttributeTypeSkill, AttributeTypeValue, AttributeTypeMilestone, AttributeTypeMetric:
		// Valid type
	default:
		return fmt.Errorf("invalid attribute type: %s", attr.AttributeType)
	}

	av, err := attributevalue.MarshalMap(attr)
	if err != nil {
		return fmt.Errorf("failed to marshal attribute: %v", err)
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.TeamAttributesTable),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("failed to create custom attribute: %v", err)
	}

	svc.logger.Printf("Created custom attribute: %s for team: %s", attr.AttributeId, attr.TeamId)
	return nil
}

// ListTeamAttributes lists all attributes for a specific team, optionally filtered by type
func (svc *TeamAttributeServiceV2) ListTeamAttributes(teamId string, attributeType *TeamAttributeType) ([]TeamAttribute, error) {
	var queryInput *dynamodb.QueryInput

	if attributeType != nil {
		// Filter by both TeamId and AttributeType
		queryInput = &dynamodb.QueryInput{
			TableName:              aws.String(svc.TeamAttributesTable),
			IndexName:              aws.String(svc.TeamAttributesTeamIdIndex),
			KeyConditionExpression: aws.String("TeamId = :teamId AND AttributeType = :attrType"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":teamId":   &types.AttributeValueMemberS{Value: teamId},
				":attrType": &types.AttributeValueMemberS{Value: string(*attributeType)},
			},
		}
	} else {
		// Get all attributes for the team
		queryInput = &dynamodb.QueryInput{
			TableName:              aws.String(svc.TeamAttributesTable),
			IndexName:              aws.String(svc.TeamAttributesTeamIdIndex),
			KeyConditionExpression: aws.String("TeamId = :teamId"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":teamId": &types.AttributeValueMemberS{Value: teamId},
			},
		}
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query team attributes: %v", err)
	}

	if output.Count == 0 {
		return []TeamAttribute{}, nil
	}

	var attributes []TeamAttribute
	for _, item := range output.Items {
		var attr TeamAttribute
		err := attributevalue.UnmarshalMap(item, &attr)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal attribute: %v", err)
		}
		attributes = append(attributes, attr)
	}

	svc.logger.Printf("Retrieved %d attributes for team: %s", len(attributes), teamId)
	return attributes, nil
}

// GetAttributesByType returns attributes grouped by type for a team
type GroupedAttributes struct {
	Skills     []TeamAttribute `json:"skills"`
	Values     []TeamAttribute `json:"values"`
	Milestones []TeamAttribute `json:"milestones"`
	Metrics    []TeamAttribute `json:"metrics"`
}

func (svc *TeamAttributeServiceV2) GetAttributesByType(teamId string) (GroupedAttributes, error) {
	allAttributes, err := svc.ListTeamAttributes(teamId, nil)
	if err != nil {
		return GroupedAttributes{}, err
	}

	grouped := GroupedAttributes{
		Skills:     []TeamAttribute{},
		Values:     []TeamAttribute{},
		Milestones: []TeamAttribute{},
		Metrics:    []TeamAttribute{},
	}

	for _, attr := range allAttributes {
		switch attr.AttributeType {
		case AttributeTypeSkill:
			grouped.Skills = append(grouped.Skills, attr)
		case AttributeTypeValue:
			grouped.Values = append(grouped.Values, attr)
		case AttributeTypeMilestone:
			grouped.Milestones = append(grouped.Milestones, attr)
		case AttributeTypeMetric:
			grouped.Metrics = append(grouped.Metrics, attr)
		}
	}

	return grouped, nil
}

func (svc *TeamAttributeServiceV2) GetAttributeById(attributeId string, teamId string) (*TeamAttribute, error) {
	getOutput, err := svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.TeamAttributesTable),
		Key: map[string]types.AttributeValue{
			"AttributeId": &types.AttributeValueMemberS{Value: attributeId},
			"TeamId":      &types.AttributeValueMemberS{Value: teamId},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get attribute: %v", err)
	}

	if getOutput.Item == nil {
		return nil, fmt.Errorf("attribute not found")
	}

	var attr TeamAttribute
	err = attributevalue.UnmarshalMap(getOutput.Item, &attr)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal attribute: %v", err)
	}

	return &attr, nil
}

// UpdateAttribute updates an existing attribute
func (svc *TeamAttributeServiceV2) UpdateAttribute(attributeId string, teamId string, name string, description string) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TeamAttributesTable),
		Key: map[string]types.AttributeValue{
			"AttributeId": &types.AttributeValueMemberS{Value: attributeId},
			"TeamId":      &types.AttributeValueMemberS{Value: teamId},
		},
		UpdateExpression: aws.String("SET #name = :name, #desc = :desc, UpdatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#name": "Name",
			"#desc": "Description",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name":      &types.AttributeValueMemberS{Value: name},
			":desc":      &types.AttributeValueMemberS{Value: description},
			":updatedAt": &types.AttributeValueMemberS{Value: timestamp},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update attribute: %v", err)
	}

	svc.logger.Printf("Updated attribute: %s for team: %s", attributeId, teamId)
	return nil
}

// DeleteAttribute deletes a custom attribute (cannot delete defaults)
func (svc *TeamAttributeServiceV2) DeleteAttribute(attributeId string, teamId string) error {
	// First check if it's a default attribute
	getOutput, err := svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.TeamAttributesTable),
		Key: map[string]types.AttributeValue{
			"AttributeId": &types.AttributeValueMemberS{Value: attributeId},
			"TeamId":      &types.AttributeValueMemberS{Value: teamId},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to get attribute: %v", err)
	}

	if getOutput.Item == nil {
		return fmt.Errorf("attribute not found")
	}

	var attr TeamAttribute
	err = attributevalue.UnmarshalMap(getOutput.Item, &attr)
	if err != nil {
		return fmt.Errorf("failed to unmarshal attribute: %v", err)
	}

	if attr.IsDefault {
		return fmt.Errorf("cannot delete default attributes")
	}

	_, err = svc.dynamodbClient.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.TeamAttributesTable),
		Key: map[string]types.AttributeValue{
			"AttributeId": &types.AttributeValueMemberS{Value: attributeId},
			"TeamId":      &types.AttributeValueMemberS{Value: teamId},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete attribute: %v", err)
	}

	svc.logger.Printf("Deleted attribute: %s for team: %s", attributeId, teamId)
	return nil
}
