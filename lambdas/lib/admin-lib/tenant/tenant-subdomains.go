/*
Tenant SubDomains are created for each Tenant and their related env.
This allows Tenants to login via a unique front end url.

The following are features that are part of this package.

1. Admin can provision Tenant's new environment
2. Admin can create Tenant's admin users.


PK Identifier : TenantId#EnvId

Resources Used:
Tables :
	TenantSubDomains
		pk: SubDomain
		rk: TenantId

Cognito UserPool:
	TenantUserPool:
		Pk: cognitoId


		APIs + Tasks:
			1. Check SubDomain - do a get item on pk : subdomain
			2. Create SubDomain - perform a update on SubDomain, TenantId.
					a. Update the table and put status to InProg
					b. DevOps team to deploy a new stack for the related Tenant
			3. Update SubDomain with new stack information ( stackName + cognito userpool Id )
					c. Move status to "Deployed Stack"
			4. List Tenant SubDomains - perform query on Pk : TenantId
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

	TenantSubDomainsTable           string
	TenantSubDomains_SubDomainIndex string
	TenantSubDomains_TenantIdIndex  string
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
type TenantSubDomainsTable struct {
	SubDomain string `dynamodbav:"SubDomain" json:"SubDomain"`
	TenantId  string `dynamodbav:"TenantId"  json:"TenantId"`

	Status string `dynamodbav:"Status" json:"Status"` // To take values from SUBDOMAIN_STATUS

	EnvName string `dynamodbav:"EnvName" json:"EnvName"` // To take values from ENV_TYPE Constants

	TenantStack      string `dynamodbav:"TenantStackName" json:"TenantStack"`       // To be updated after the Tenant Stack is deployed
	TenantUserPoolId string `dynamodbav:"TenantUserPoolId" json:"TenantUserPoolId"` // To be updated after the Tenant Stack is deployed

	AdminUsers []string `dynamodbav:"AdminUsers" json:"AdminUsers"` // When added the admin user is created in the Tenant Stack
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

// 1. Check Tenant SubDomain Availability - Get Req
func (svc *SubDomainService) CheckSubDomainsAvailability(SubDomain string) (bool, error) {

	queryItemInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.TenantSubDomainsTable),
		IndexName:              aws.String(svc.TenantSubDomains_SubDomainIndex),
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

// 2. Create SubDomain for the tenant - POST Req
type CreateSubDomainInput struct {
	TenantId  string `json:"TenantId"`
	SubDomain string `json:"SubDomain"`
	EnvName   string `json:"EnvName"`
}

func (svc *SubDomainService) CreateTenantSubDomain(subDomainData CreateSubDomainInput) error {

	subdomainItem := map[string]dynamodb_types.AttributeValue{
		"TenantId":  &dynamodb_types.AttributeValueMemberS{Value: subDomainData.TenantId},
		"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: subDomainData.SubDomain},
		"Status":    &dynamodb_types.AttributeValueMemberS{Value: SUBDOMAIN_STATUS_INPROG},
		"EnvName":   &dynamodb_types.AttributeValueMemberS{Value: subDomainData.EnvName},
	}

	putItemInput := dynamodb.PutItemInput{
		Item:      subdomainItem,
		TableName: aws.String(svc.TenantSubDomainsTable),
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput)
	if err != nil {
		svc.logger.Printf("PutItem failed with error :%v", err)
		return err
	}
	svc.logger.Printf("Successfully added SubDomain %s for TenantID: %s", subDomainData.SubDomain, subDomainData.TenantId)

	return nil
}

// 3. Update SubDomain with new stack information ( stackName + cognito userpool Id )
type UpdateSubDomainStackInfo struct {
	SubDomain string `dynamodbav:"SubDomain"`
	TenantId  string `dynamodbav:"TenantId"`

	TenantStackName  string `json:"TenantStackName"`
	TenantUserPoolId string `json:"TenantUserPoolId"`
}

func (svc *SubDomainService) UpdateSubDomainStackInfo(stackInfo UpdateSubDomainStackInfo) error {

	if stackInfo.SubDomain == "" || stackInfo.TenantId == "" || stackInfo.TenantStackName == "" || stackInfo.TenantUserPoolId == "" {
		svc.logger.Printf("Update Stack Information cannot be empty : %v", stackInfo)
		return fmt.Errorf("update Stack info is empty %v", stackInfo)
	}

	ddbUpdateInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantSubDomainsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: stackInfo.SubDomain},
			"TenantId":  &dynamodb_types.AttributeValueMemberS{Value: stackInfo.TenantId},
		},
		UpdateExpression: aws.String("SET TenantStackName = :TenantStackName, TenantUserPoolId = :TenantUserPoolId, Status = :Status"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":TenantStackName":  &dynamodb_types.AttributeValueMemberS{Value: stackInfo.TenantStackName},
			":TenantUserPoolId": &dynamodb_types.AttributeValueMemberS{Value: stackInfo.TenantUserPoolId},
			":Status":           &dynamodb_types.AttributeValueMemberS{Value: SUBDOMAIN_STATUS_STACK_DEPLOYED},
		},
		ReturnValues: dynamodb_types.ReturnValueNone,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &ddbUpdateInput)

	if err != nil {
		svc.logger.Printf("DDB Update failed with error : %v", err)
		return err
	}

	svc.logger.Printf("Successfully Updated stackInfo to the Tenant's Env")

	return nil
}

// 4. List all SubDomains related to the Tenant - Get Req
func (svc *SubDomainService) GetAllTenantSubDomains(tenantId string) ([]TenantSubDomainsTable, error) {

	var alldomainsDDBOutput []TenantSubDomainsTable

	queryTableInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.TenantSubDomainsTable),
		IndexName:              aws.String(svc.TenantSubDomains_TenantIdIndex),
		KeyConditionExpression: aws.String("TenantId = :TenantId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":TenantId": &dynamodb_types.AttributeValueMemberS{Value: tenantId},
		},
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryTableInput)
	if err != nil {
		svc.logger.Printf("Failed to Query the table. Error: %v", err)
		return nil, err
	}

	if output.Count == 0 {
		svc.logger.Printf("No Tenant SubDomains found for TenantId: %s", tenantId)
		return []TenantSubDomainsTable{}, nil
	}

	for _, sudDomainData := range output.Items {
		var subDomain TenantSubDomainsTable
		err = dynamodb_attributevalue.UnmarshalMap(sudDomainData, &subDomain)
		if err != nil {
			svc.logger.Printf("unable to UnMarshall the output from the DDB Query under :GetAllTenantSubDomains ")
			return nil, err
		}
		alldomainsDDBOutput = append(alldomainsDDBOutput, subDomain)
	}

	return alldomainsDDBOutput, nil
}

// 5. Create SubDomain Admins - perform a Update operation on AdminUsers Column(Map)
type SubDomainAdmin struct {
	SubDomain string `json:"SubDomain"`
	TenantId  string `json:"TenantId"`

	AdminUserId string `json:"AdminUserId"`
}

func (svc *SubDomainService) AddSubDomainAdmin(AdminData SubDomainAdmin) error {

	// 1. Get Tenant SubDomain's UserPoolId
	getItemInput := dynamodb.GetItemInput{
		TableName: aws.String(svc.TenantSubDomainsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: AdminData.SubDomain},
			"TenantId":  &dynamodb_types.AttributeValueMemberS{Value: AdminData.TenantId},
		},
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)
	if err != nil {
		svc.logger.Printf("Get Item Failed with error : %v", err)
		return err
	}

	var SubDomainData TenantSubDomainsTable

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &SubDomainData)
	if err != nil {
		svc.logger.Printf("UnMarshall failed : %v", err)
		return err
	}

	svc.logger.Printf("Cognito User Pool for the Tenant's SubDomain[%s] :UserPool %s", SubDomainData.TenantId+" /"+SubDomainData.SubDomain, SubDomainData.TenantUserPoolId)

	if SubDomainData.TenantUserPoolId == "" {
		svc.logger.Print("Cognito User Pool cannot be empty ")
		return fmt.Errorf("cognito UserPoolId cannot be empty to perform Create Admin users")
	}

	// 2. Perform Create Admin user action on the respective Cognito UserPool

	if AdminData.AdminUserId == "" {
		svc.logger.Print("Admin User cannot be Empty ")
		return fmt.Errorf("admin user cannot be empty")
	}
	_, err = svc.cognitoClient.AdminCreateUser(svc.ctx, &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId: aws.String(SubDomainData.TenantUserPoolId),
		Username:   aws.String(AdminData.AdminUserId),
	})
	if err != nil {
		svc.logger.Printf("Unable to create user in the userPoolID :%s , Failed with error : %v", SubDomainData.TenantUserPoolId, err)
		return err
	}

	//3 . Add admin user to the ddb table

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantSubDomainsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SubDomain": &dynamodb_types.AttributeValueMemberS{Value: AdminData.SubDomain},
			"TenantId":  &dynamodb_types.AttributeValueMemberS{Value: AdminData.TenantId},
		},
		UpdateExpression: aws.String("SET AdminUsers = list_append(if_not_exists(AdminUsers, :EmptyList), :AdminUser)"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":AdminUser": &dynamodb_types.AttributeValueMemberL{
				Value: []dynamodb_types.AttributeValue{
					&dynamodb_types.AttributeValueMemberS{Value: AdminData.AdminUserId},
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
