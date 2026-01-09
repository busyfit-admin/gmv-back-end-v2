package supplierlib

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	utils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type SupplierBranch struct {
	BranchId string `dynamodbav:"BranchId" json:"BranchId"` //PK , ID can be either provided by Supplier or UUID is created
	IsActive string `dynamodbav:"IsActive" json:"IsActive"`

	BranchType string `dynamodbav:"BranchType" json:"BranchType"` // Online, Store

	BranchName          string `dynamodbav:"BranchName" json:"BranchName"`
	BranchAddressField1 string `dynamodbav:"BranchAddressField1" json:"BranchAddressField1"`
	BranchAddressField2 string `dynamodbav:"BranchAddressField2" json:"BranchAddressField2"`
	BranchArea          string `dynamodbav:"BranchArea" json:"BranchArea"`
	BranchCity          string `dynamodbav:"BranchCity" json:"BranchCity"`
	BranchState         string `dynamodbav:"BranchState" json:"BranchState"`
	BranchPinCode       string `dynamodbav:"BranchPinCode" json:"BranchPinCode"`

	BranchLocLat string `dynamodbav:"BranchLocLat" json:"BranchLocLat"`
	BranchLocLng string `dynamodbav:"BranchLocLng" json:"BranchLocLng"`

	BranchPrimaryContactName string `dynamodbav:"BranchPrimaryContactName" json:"BranchPrimaryContactName"`
	BranchPrimaryPh          string `dynamodbav:"BranchPrimaryPh" json:"BranchPrimaryPh"`
	BranchPrimaryEmail       string `dynamodbav:"BranchPrimaryEmail" json:"BranchPrimaryEmail"`

	BranchSecondaryContactName string `dynamodbav:"BranchSecondaryContactName" json:"BranchSecondaryContactName"`
	BranchSecondaryPh          string `dynamodbav:"BranchSecondaryPh" json:"BranchSecondaryPh"`
	BranchSecondaryEmail       string `dynamodbav:"BranchSecondaryEmail" json:"BranchSecondaryEmail"`
}

const (
	BRANCH_TYPE_ONLINE = "ONLINE"
	BRANCH_TYPE_STORE  = "STORE"

	BRANCH_ISACTIVE_FALSE = "INACTIVE"
	BRANCH_ISACTIVE_TRUE  = "ACTIVE"
)

// -------------------- Supplier Profile Functions -------------

type SupplierService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	SupplierBranchTable               string
	SupplierBranchTable_IsActiveIndex string
}

func CreateSupplierService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *SupplierService {
	return &SupplierService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

func (s *SupplierService) GetSupplierBranchesFunctions(branchId string) (interface{}, error) {

	if branchId == "" {
		s.logger.Printf("Get all Branches Data")
		return s.GetAllBranches()
	}
	s.logger.Printf("Get Branch data for branchID :%s", branchId)
	return s.GetBranchDetails(branchId)
}

type AllBranches struct {
	ActiveBranches   []SupplierBranchShort `json:"ActiveBranches"`
	InactiveBranches []SupplierBranchShort `json:"InActiveBranches"`
}

type SupplierBranchShort struct {
	BranchId string `dynamodbav:"BranchId" json:"BranchId"` //PK , ID can be either provided by Supplier or UUID is created
	IsActive string `dynamodbav:"IsActive" json:"IsActive"`

	BranchType string `dynamodbav:"BranchType" json:"BranchType"` // Online, Store

	BranchName          string `dynamodbav:"BranchName" json:"BranchName"`
	BranchAddressField1 string `dynamodbav:"BranchAddressField1" json:"BranchAddressField1"`

	BranchCity string `dynamodbav:"BranchCity" json:"BranchCity"`
}

func (s *SupplierService) GetAllBranches() (AllBranches, error) {

	allBranches := AllBranches{}

	// 1. Get All Active Branches
	queryGetActiveBranches := "SELECT BranchId, IsActive, BranchType, BranchName, BranchAddressField1, BranchCity FROM \"" + s.SupplierBranchTable + "\".\"" + s.SupplierBranchTable_IsActiveIndex + "\" WHERE IsActive = 'ACTIVE' ORDER BY BranchType ASC"
	activeBranchesData, err := s.ExecuteQueryDDB(queryGetActiveBranches)
	if err != nil {
		return AllBranches{}, err
	}
	allBranches.ActiveBranches = activeBranchesData

	// 2. Get All Inactive Branches
	queryGetInActiveBranches := "SELECT BranchId, IsActive, BranchType, BranchName, BranchAddressField1, BranchCity FROM \"" + s.SupplierBranchTable + "\".\"" + s.SupplierBranchTable_IsActiveIndex + "\" WHERE IsActive = 'INACTIVE' ORDER BY BranchType ASC"
	inActiveBranchesData, err := s.ExecuteQueryDDB(queryGetInActiveBranches)
	if err != nil {
		return AllBranches{}, err
	}
	allBranches.InactiveBranches = inActiveBranchesData

	return allBranches, nil

}

// Used for drop down selection for creating Cards
func (s *SupplierService) GetActiveBranches() ([]SupplierBranchShort, error) {

	// 1. Get All Active Branches
	queryGetActiveBranches := "SELECT BranchId, IsActive, BranchType, BranchName, BranchAddressField1, BranchCity FROM \"" + s.SupplierBranchTable + "\".\"" + s.SupplierBranchTable_IsActiveIndex + "\" WHERE IsActive = 'ACTIVE' ORDER BY BranchType ASC"
	activeBranchesData, err := s.ExecuteQueryDDB(queryGetActiveBranches)
	if err != nil {
		return activeBranchesData, err
	}

	return activeBranchesData, nil
}

func (s *SupplierService) ExecuteQueryDDB(query string) ([]SupplierBranchShort, error) {

	output, err := s.dynamodbClient.ExecuteStatement(s.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(query),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		s.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []SupplierBranchShort{}, err
	}
	if len(output.Items) == 0 {
		s.logger.Printf("No Items found for branches in state")
		return []SupplierBranchShort{}, nil
	}

	allBranchesData := []SupplierBranchShort{}
	for _, stageItem := range output.Items {
		branchData := SupplierBranchShort{}
		err = attributevalue.UnmarshalMap(stageItem, &branchData)
		if err != nil {
			s.logger.Printf("Couldn't unmarshal Branch Details data  Error : %v", err)
			return []SupplierBranchShort{}, err
		}
		// Append data to the overall branch data
		allBranchesData = append(allBranchesData, branchData)
	}

	return allBranchesData, nil
}

func (s *SupplierService) GetBranchDetails(branchId string) (SupplierBranch, error) {

	getItemInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"BranchId": &dynamodb_types.AttributeValueMemberS{Value: branchId},
		},
		TableName:      aws.String(s.SupplierBranchTable),
		ConsistentRead: aws.Bool(true),
	}

	output, err := s.dynamodbClient.GetItem(s.ctx, &getItemInput)
	if err != nil {
		s.logger.Printf("Get SupplierBranch Failed with error :%v", err)
		return SupplierBranch{}, err
	}
	BranchData := SupplierBranch{}

	err = attributevalue.UnmarshalMap(output.Item, &BranchData)
	if err != nil {
		s.logger.Printf("Get SupplierBranch Unmarshal failed with error :%v", err)
		return SupplierBranch{}, err
	}

	return BranchData, nil
}

// -----_ DML Ops on Supplier Branches -------

func (s *SupplierService) CreateSupplierBranch(branch SupplierBranch) error {
	if branch.BranchId == "" {
		branch.BranchId = utils.GenerateRandomString(10)
	}

	av, err := attributevalue.MarshalMap(branch)
	if err != nil {
		s.logger.Printf("Failed to marshal supplier branch: %v", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName:           aws.String(s.SupplierBranchTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(BranchId)"), // To ensure it does not create a new branch by overriding existing one.
	}

	_, err = s.dynamodbClient.PutItem(s.ctx, input)
	if err != nil {
		s.logger.Printf("Failed to put item in DynamoDB: %v", err)
		return err
	}

	return nil
}

func (s *SupplierService) UpdateSupplierBranch(branch SupplierBranch) error {
	if branch.BranchId == "" {
		return errors.New("branchId is required for update")
	}

	av, err := attributevalue.MarshalMap(branch)
	if err != nil {
		s.logger.Printf("Failed to marshal supplier branch: %v", err)
		return err
	}

	input := &dynamodb.PutItemInput{
		TableName:           aws.String(s.SupplierBranchTable),
		Item:                av,
		ConditionExpression: aws.String("attribute_exists(BranchId)"), // to ensure it updates an existing BranchId
	}

	_, err = s.dynamodbClient.PutItem(s.ctx, input)
	if err != nil {
		s.logger.Printf("Failed to put item in DynamoDB: %v", err)
		return err
	}

	return nil
}

// Delete a Branch only after making it InActive
func (s *SupplierService) DeleteSupplierBranch(branchId string) error {
	if branchId == "" {
		return errors.New("branchId is required for delete")
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(s.SupplierBranchTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"BranchId": &dynamodb_types.AttributeValueMemberS{Value: branchId},
		},
		ConditionExpression: aws.String("IsActive = :InActive"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			"InActive": &dynamodb_types.AttributeValueMemberS{Value: BRANCH_ISACTIVE_FALSE},
		},
	}

	_, err := s.dynamodbClient.DeleteItem(s.ctx, input)
	if err != nil {
		s.logger.Printf("Failed to delete item from DynamoDB: %v", err)
		return err
	}

	return nil
}
