/*
Supplier SubDomains are created for each Supplier and their related env.
This allows Suppliers to login via a unique front end url.

The following are features that are part of this package.

1. Admin can provision Supplier's new environment
2. Admin can create Supplier's admin users.


PK Identifier : SupplierId#EnvId

Resources Used:
Tables :
    SupplierSubDomains
        pk: SubDomain
        rk: SupplierId

Cognito UserPool:
    SupplierUserPool:
        Pk: cognitoId


        APIs + Tasks:
            1. Check SubDomain - do a get item on pk : subdomain
            2. Create SubDomain - perform a update on SubDomain, SupplierId.
                    a. Update the table and put status to InProg
                    b. DevOps team to deploy a new stack for the related Supplier
            3. Update SubDomain with new stack information ( stackName + cognito userpool Id )
                    c. Move status to "Deployed Stack"
            4. List Supplier SubDomains - perform query on Pk : SupplierId
            5. Create SubDomain Admins - perform a Update operation on AdminUsers Column(Map)
                    d. Add Admin Users to the Deployed Stack

*/

package adminlib

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

// -------------------- SubDomain Service Functions -------------
type SubDomainService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	cognitoClient  awsclients.CognitoClient
	logger         *log.Logger

	SupplierSubDomainsTable            string
	SupplierSubDomains_SubDomainIndex  string
	SupplierSubDomains_SupplierIdIndex string
}

func CreateSubDomainService(ctx context.Context, ddbClient awsclients.DynamodbClient, congnitoClient awsclients.CognitoClient, logger *log.Logger) *SubDomainService {
	return &SubDomainService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		cognitoClient:  congnitoClient,
		logger:         logger,
	}
}

// DDB Tables Struct
type SupplierSubDomainsTable struct {
	SubDomain  string `dynamodbav:"SubDomain" json:"SubDomain"`
	SupplierId string `dynamodbav:"SupplierId"  json:"SupplierId"`

	Status string `dynamodbav:"Status" json:"Status"` // To take values from SUBDOMAIN_STATUS

	EnvName string `dynamodbav:"EnvName" json:"EnvName"` // To take values from ENV_TYPE Constants

	SupplierStack      string `dynamodbav:"SupplierStackName" json:"SupplierStack"`       // To be updated after the Supplier Stack is deployed
	SupplierUserPoolId string `dynamodbav:"SupplierUserPoolId" json:"SupplierUserPoolId"` // To be updated after the Supplier Stack is deployed

	AdminUsers []string `dynamodbav:"AdminUsers" json:"AdminUsers"` // When added the admin user is created in the Supplier Stack
}

// Constants
const (
	SUBDOMAIN_STATUS_INPROG         = "STACK_INPROG"
	SUBDOMAIN_STATUS_STACK_DEPLOYED = "STACK_DEPLOYED"
)
const (
	ENV_TYPE_SANDBOX = "SANDBOX"
	ENV_TYPE_DEV     = "DEV"
	ENV_TYPE_UAT     = "UAT"
	EVN_TYPE_PRD     = "PROD"
)

// 1. Check Supplier SubDomain Availability - Get Req
func (svc *SubDomainService) CheckSubDomainsAvailability(SubDomain string) (bool, error) {

	queryItemInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.SupplierSubDomainsTable),
		IndexName:              aws.String(svc.SupplierSubDomains_SubDomainIndex),
		KeyConditionExpression: aws.String("SubDomain = :SubDomain"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":SubDomain": &dynamodb_types.AttributeValueMemberS{Value: SubDomain},
		},
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryItemInput)

	if err != nil {
		svc.logger.Printf("Query on Employee data failed with error :%v", err)
		return false, err
	}

	if output.Count == 0 {
		return true, nil
	}

	return false, nil
}

// 2. Create SubDomain for the supplier - POST Req
type CreateSubDomainInput struct {
	SupplierId string `json:"SupplierId"`
	SubDomain  string `json:"SubDomain"`
	EnvName    string `json:"EnvName"`
}

func (svc *SubDomainService) CreateSupplierSubDomain(subDomainData CreateSubDomainInput) error {

	subdomainItem := map[string]dynamodb_types.AttributeValue{
		"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: subDomainData.SupplierId},
		"SubDomain":  &dynamodb_types.AttributeValueMemberS{Value: subDomainData.SubDomain},
		"Status":     &dynamodb_types.AttributeValueMemberS{Value: SUBDOMAIN_STATUS_INPROG},
		"EnvName":    &dynamodb_types.AttributeValueMemberS{Value: subDomainData.EnvName},
	}

	putItemInput := dynamodb.PutItemInput{
		Item:      subdomainItem,
		TableName: aws.String(svc.SupplierSubDomainsTable),
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput)
	if err != nil {
		svc.logger.Printf("PutItem failed with error :%v", err)
		return err
	}
	svc.logger.Printf("Successfully added SubDomain %s for SupplierID: %s", subDomainData.SubDomain, subDomainData.SupplierId)

	return nil
}

// 3. Update SubDomain with new stack information ( stackName + cognito userpool Id )
type UpdateSubDomainStackInfo struct {
	SubDomain  string `dynamodbav:"SubDomain"`
	SupplierId string `dynamodbav:"SupplierId"`

	SupplierStackName  string `json:"SupplierStackName"`
	SupplierUserPoolId string `json:"SupplierUserPoolId"`
}

func (svc *SubDomainService) UpdateSubDomainStackInfo(stackInfo UpdateSubDomainStackInfo) error {

	if stackInfo.SubDomain == "" || stackInfo.SupplierId == "" || stackInfo.SupplierStackName == "" || stackInfo.SupplierUserPoolId == "" {
		svc.logger.Printf("Update Stack Information cannot be empty : %v", stackInfo)
		return fmt.Errorf("update Stack info is empty %v", stackInfo)
	}

	ddbUpdateInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.SupplierSubDomainsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SubDomain":  &dynamodb_types.AttributeValueMemberS{Value: stackInfo.SubDomain},
			"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: stackInfo.SupplierId},
		},
		UpdateExpression: aws.String("SET SupplierStackName = :SupplierStackName, SupplierUserPoolId = :SupplierUserPoolId, Status = :Status"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":SupplierStackName":  &dynamodb_types.AttributeValueMemberS{Value: stackInfo.SupplierStackName},
			":SupplierUserPoolId": &dynamodb_types.AttributeValueMemberS{Value: stackInfo.SupplierUserPoolId},
			":Status":             &dynamodb_types.AttributeValueMemberS{Value: SUBDOMAIN_STATUS_STACK_DEPLOYED},
		},
		ReturnValues: dynamodb_types.ReturnValueNone,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &ddbUpdateInput)

	if err != nil {
		svc.logger.Printf("DDB Update failed with error : %v", err)
		return err
	}

	svc.logger.Printf("Successfully Updated stackInfo to the Supplier's Env")

	return nil
}

// 4. List all SubDomains related to the Supplier - Get Req
func (svc *SubDomainService) GetAllSupplierSubDomains(supplierId string) ([]SupplierSubDomainsTable, error) {

	var alldomainsDDBOutput []SupplierSubDomainsTable

	queryTableInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.SupplierSubDomainsTable),
		IndexName:              aws.String(svc.SupplierSubDomains_SupplierIdIndex),
		KeyConditionExpression: aws.String("SupplierId = :SupplierId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":SupplierId": &dynamodb_types.AttributeValueMemberS{Value: supplierId},
		},
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryTableInput)
	if err != nil {
		svc.logger.Printf("Failed to Query the table. Error: %v", err)
		return nil, err
	}

	if output.Count == 0 {
		svc.logger.Printf("No Supplier SubDomains found for SupplierId: %s", supplierId)
		return []SupplierSubDomainsTable{}, nil
	}

	for _, sudDomainData := range output.Items {
		var subDomain SupplierSubDomainsTable
		err = dynamodb_attributevalue.UnmarshalMap(sudDomainData, &subDomain)
		if err != nil {
			svc.logger.Printf("unable to UnMarshall the output from the DDB Query under :GetAllSupplierSubDomains ")
			return nil, err
		}
		alldomainsDDBOutput = append(alldomainsDDBOutput, subDomain)
	}

	return alldomainsDDBOutput, nil
}

// 5. Create SubDomain Admins - perform a Update operation on AdminUsers Column(Map)
type SubDomainAdmin struct {
	SubDomain  string `json:"SubDomain"`
	SupplierId string `json:"SupplierId"`

	AdminUserId string `json:"AdminUserId"`
}

func (svc *SubDomainService) AddSubDomainAdmin(adminData SubDomainAdmin) error {

	// 1. Get Supplier SubDomain's UserPoolId
	getItemInput := dynamodb.GetItemInput{
		TableName: aws.String(svc.SupplierSubDomainsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SubDomain":  &dynamodb_types.AttributeValueMemberS{Value: adminData.SubDomain},
			"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: adminData.SupplierId},
		},
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)
	if err != nil {
		svc.logger.Printf("Get Item Failed with error : %v", err)
		return err
	}

	var SubDomainData SupplierSubDomainsTable

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &SubDomainData)
	if err != nil {
		svc.logger.Printf("UnMarshall failed : %v", err)
		return err
	}

	svc.logger.Printf("Cognito User Pool for the Supplier's SubDomain[%s] :UserPool %s", SubDomainData.SupplierId+" /"+SubDomainData.SubDomain, SubDomainData.SupplierUserPoolId)

	if SubDomainData.SupplierUserPoolId == "" {
		svc.logger.Print("Cognito User Pool cannot be empty ")
		return fmt.Errorf("cognito UserPoolId cannot be empty to perform Create Admin users")
	}

	// 2. Perform Create Admin user action on the respective Cognito UserPool

	if adminData.AdminUserId == "" {
		svc.logger.Print("Admin User cannot be Empty ")
		return fmt.Errorf("admin user cannot be empty")
	}
	_, err = svc.cognitoClient.AdminCreateUser(svc.ctx, &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId: aws.String(SubDomainData.SupplierUserPoolId),
		Username:   aws.String(adminData.AdminUserId),
	})
	if err != nil {
		svc.logger.Printf("Unable to create user in the userPoolID :%s , Failed with error : %v", SubDomainData.SupplierUserPoolId, err)
		return err
	}

	//3 . Add admin user to the ddb table

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.SupplierSubDomainsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SubDomain":  &dynamodb_types.AttributeValueMemberS{Value: adminData.SubDomain},
			"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: adminData.SupplierId},
		},
		UpdateExpression: aws.String("SET AdminUsers = list_append(if_not_exists(AdminUsers, :EmptyList), :AdminUser)"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":AdminUser": &dynamodb_types.AttributeValueMemberL{
				Value: []dynamodb_types.AttributeValue{
					&dynamodb_types.AttributeValueMemberS{Value: adminData.AdminUserId},
				},
			},
			":EmptyList": &dynamodb_types.AttributeValueMemberL{},
		},
		ReturnValues: dynamodb_types.ReturnValueNone,
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to perform Update Item, err: %v", err)
	}

	return nil
}
