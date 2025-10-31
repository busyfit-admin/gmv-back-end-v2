package Companylib

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

type CompanyProfileTable struct {
	CognitoCompanyUserName string `dynamodbav:"CognitoCompanyUserName"`

	CompanyName string `dynamodbav:"CompanyName"`

	CompanyPrimaryEmail   string `dynamodbav:"CompanyPrimaryEmail"`
	CompanySecondaryEmail string `dynamodbav:"CompanySecondaryEmail"`

	CompanyDesc string   `dynamodbav:"CompanyDesc"`
	Categories  []string `dynamodbav:"Categories"`

	CompanyBranchs map[string]CompanyBranch `dynamodbav:"CompanyBranchs"`
}

type CompanyBranch struct {
	BranchId string `dynamodbav:"BranchId"`
	IsActive string `dynamodbav:"IsActive"`

	BranchName          string `dynamodbav:"BranchName"`
	BranchAddressField1 string `dynamodbav:"BranchAddressField1"`
	BranchAddressField2 string `dynamodbav:"BranchAddressField2"`
	BranchArea          string `dynamodbav:"BranchArea"`
	BranchCity          string `dynamodbav:"BranchCity"`
	BranchState         string `dynamodbav:"BranchState"`
	BranchPinCode       string `dynamodbav:"BranchPinCode"`

	BranchLocLat string `dynamodbav:"BranchLocLat"`
	BranchLocLng string `dynamodbav:"BranchLocLng"`

	BranchPrimaryContactName string `dynamodbav:"BranchPrimaryContactName"`
	BranchPrimaryPh          string `dynamodbav:"BranchPrimaryPh"`
	BranchPrimaryEmail       string `dynamodbav:"BranchPrimaryEmail"`

	BranchSecondaryContactName string `dynamodbav:"BranchSecondaryContactName"`
	BranchSecondaryPh          string `dynamodbav:"BranchSecondaryPh"`
	BranchSecondaryEmail       string `dynamodbav:"BranchSecondaryEmail"`
}

// -------------------- Company Profile Functions -------------

type CompanyService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	CompanyTable string

	EmployeeTable                  string
	EmployeeTable_UserName_Index   string
	EmployeeTable_ExternalId_Index string

	EmployeeGroupsTable string
}

func CreateCompanyService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger, CompanyTable string) *CompanyService {
	return &CompanyService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
		CompanyTable:   CompanyTable,
	}
}

// Get CompanyId from API Request
func (svc *CompanyService) GetCompanyIdFromAPIReq(request events.APIGatewayProxyRequest) (string, error) {

	cognitoClaims := request.RequestContext.Authorizer["claims"].(map[string]interface{})
	cognitoUsername := cognitoClaims["cognito:username"].(string)

	svc.logger.Printf("Request Context data: %v\n", cognitoUsername)
	if cognitoUsername == "" {
		return "", fmt.Errorf("error :Cognito username is empty")
	}

	return cognitoUsername, nil
}

// Accepts (CompanyId) and returns all CompanyProfileTable data
func (svc *CompanyService) GetCompanyProfileById(CompanyId string) (CompanyProfileTable, error) {

	getItemInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"CognitoCompanyUserName": &dynamodb_types.AttributeValueMemberS{Value: CompanyId},
		},
		TableName:      aws.String(svc.CompanyTable),
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)
	if err != nil {
		svc.logger.Printf("Get Company Failed with error :%v", err)
		return CompanyProfileTable{}, err
	}

	CompanyData := CompanyProfileTable{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &CompanyData)
	if err != nil {
		svc.logger.Printf("Get Company Unmarshal failed with error :%v", err)
		return CompanyProfileTable{}, err
	}

	return CompanyData, nil
}

type CompanyUpdateData struct {
	CompanyName string `json:"CompanyName"`

	CompanyPrimaryEmail   string `json:"CompanyPrimaryEmail"`
	CompanySecondaryEmail string `json:"CompanySecondaryEmail"`

	CompanyDesc string   `json:"CompanyDesc"`
	Categories  []string `json:"Categories"` // Todo for future
}

// Updates the CompanyProfile Information
// Accepts (CompanyId , CompanyUpdateData) returns All Updated New Values of CompanyProfileTable
func (svc *CompanyService) UpdateCompanyProfileById(CompanyId string, compData CompanyUpdateData) (CompanyProfileTable, error) {

	UpdateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.CompanyTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"CognitoCompanyUserName": &dynamodb_types.AttributeValueMemberS{Value: CompanyId},
		},
		UpdateExpression: aws.String("set CompanyName = :CompanyName, CompanyPrimaryEmail = :CompanyPrimaryEmail, CompanySecondaryEmail = :CompanySecondaryEmail, CompanyDesc = :CompanyDesc"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":CompanyName":           &dynamodb_types.AttributeValueMemberS{Value: compData.CompanyName},
			":CompanyPrimaryEmail":   &dynamodb_types.AttributeValueMemberS{Value: compData.CompanyPrimaryEmail},
			":CompanySecondaryEmail": &dynamodb_types.AttributeValueMemberS{Value: compData.CompanySecondaryEmail},
			":CompanyDesc":           &dynamodb_types.AttributeValueMemberS{Value: compData.CompanyDesc},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	svc.logger.Printf("dfdsfs %v", UpdateItemInput)

	output, err := svc.dynamodbClient.UpdateItem(svc.ctx, &UpdateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Company Table :%v", err)
		return CompanyProfileTable{}, err
	}

	// unmarshal the new values and return
	UpdatedTableData := CompanyProfileTable{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Attributes, &UpdatedTableData)
	if err != nil {
		svc.logger.Printf("Failed to Unmarshal the Company Table in UpdateCompanyFunction :%v", err)
		return CompanyProfileTable{}, err
	} 

	return UpdatedTableData, nil
}

// Adds and Updates the Branch Information
func (svc *CompanyService) UpdateCompanyBranchById(BranchId string) (CompanyBranch, error) {

	return CompanyBranch{}, nil
}

// Remove the branch from the Company
func (svc *CompanyService) DeleteCompanyBranchById(BranchId string) (CompanyBranch, error) {

	return CompanyBranch{}, nil
}
