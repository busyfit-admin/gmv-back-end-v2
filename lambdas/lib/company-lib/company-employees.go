/*

1. Admins can get users data with pagination.


*/

package Companylib

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"log"

	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"

	cognito_types "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

// --------- Employee Profile functions ---------

type EmployeeService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	CognitoClient  awsclients.CognitoClient

	logger *log.Logger

	EmployeeTable                  string
	TenantTeamsTable               string
	EmployeeTable_EmailId_Index    string
	EmployeeTable_ExternalId_Index string
	EmployeeTable_CognitoId_Index  string // Used for employee Profile API

	EmployeeUserPoolId string

	EmployeeGroupsTable string

	RewardsRuleTable string
}

func CreateEmployeeService(ctx context.Context, ddbClient awsclients.DynamodbClient, cognitoClient awsclients.CognitoClient, logger *log.Logger) *EmployeeService {

	return &EmployeeService{
		ctx: ctx,

		dynamodbClient: ddbClient,
		CognitoClient:  cognitoClient,
		logger:         logger,
	}
}

const DEFAULT_PROFILE_PIC = "default.jpg"

func GetDefaultProfilePicPath(username string) string {
	return fmt.Sprintf("users/%s/profilepic/profile.png", username)
}

// Default path for all Profile pics: users/Testvar@gmail.com/profilepic/profile.png
func GetDefaultThumbnailPicPath(username string) string {
	return fmt.Sprintf("users/%s/profilepic/profile.png", username)
}

// --------------- DDB Tables Employees Table-------------

type EmployeeDynamodbData struct {
	CognitoId string `json:"CognitoId,omitempty" dynamodbav:"CognitoId"`

	UserName string `json:"UserName" dynamodbav:"UserName"` // PK in the Employee Table

	// Stored in S3 under /users/profilepics/<username>.png , if empty, default pic is shown from S3 bucket /users/profilepics/default.png
	ProfilePic string `json:"ProfilePic,omitempty" dynamodbav:"ProfilePic"`

	EmailID string `json:"EmailId,omitempty" dynamodbav:"EmailId"` // Secondary Index

	ExternalId string `json:"ExternalId,omitempty" dynamodbav:"E_ID"` // Secondary Index - maps to E_ID in DynamoDB

	DisplayName string `json:"DisplayName,omitempty" dynamodbav:"DisplayName"`

	// Additional fields that exist in DynamoDB but weren't in the original struct
	FirstName string `json:"FirstName,omitempty" dynamodbav:"FirstName"`
	LastName  string `json:"LastName,omitempty" dynamodbav:"LastName"`
	Status    string `json:"Status,omitempty" dynamodbav:"Status"`
	Source    string `json:"Source,omitempty" dynamodbav:"Source"`
	CreatedAt string `json:"CreatedAt,omitempty" dynamodbav:"CreatedAt"`
	UpdatedAt string `json:"UpdatedAt,omitempty" dynamodbav:"UpdatedAt"`

	Designation string `json:"Designation,omitempty" dynamodbav:"Designation"`
	PhoneNumber string `json:"PhoneNumber,omitempty" dynamodbav:"PhoneNumber"`

	LoginType string `json:"LoginType,omitempty" dynamodbav:"LoginType"` // Allowed Values : "password", "sso"
	IsActive  string `json:"Active,omitempty" dynamodbav:"Active"`       // Allowed Values : "Active", "Inactive"

	IsManager string `json:"IsManager,omitempty" dynamodbav:"IsManager"` //Allowed Values : 'Y'/'N'

	MgrUserName string `json:"MgrUserName,omitempty" dynamodbav:"MgrUserName"`

	// StartDate string `json:"StartDate,omitempty" dynamodbav:"StartDate"`
	// EndDate   string `json:"EndDate,omitempty" dynamodbav:"EndDate"`

	TopLevelGroupName string `json:"TopLevelGroupName,omitempty" dynamodbav:"TopLevelGroupName"` // '|' separated Group Names

	Location string `json:"Location,omitempty" dynamodbav:"Location"`

	// RewardsData map[string]EmployeeRewards `json:"RewardsData,omitempty" dynamodbav:"RewardsData"`

	// RedeemedCards map[string]RewardCards `json:"RedeemedCards,omitempty" dynamodbav:"RedeemedCards"`

	// CertificatesData map[string]EmployeeCertificates `json:"CertificatesData,omitempty" dynamodbav:"CertificatesData"`

	RolesData map[string]bool `json:"RolesData,omitempty" dynamodbav:"RolesData"`

	CurrentTeamId string `json:"CurrentTeamId,omitempty" dynamodbav:"CurrentTeamId"` // Currently logged-in team
}

type EmployeeRewards struct {
	IsActive     bool   `json:"IsActive" dynamodbav:"IsActive"`
	RewardId     string `json:"RewardId" dynamodbav:"RewardId"`
	RewardPoints int    `json:"RewardPoints" dynamodbav:"RewardPoints"`

	TransferablePoints int `json:"TransferablePoints" dynamodbav:"TransferablePoints"` // Points which can be transferred to other employees

	// RewardsProvidedDate string `json:"RewardsProvidedDate" dynamodbav:"RewardsProvidedDate"` // Format : "YYYY-MM-DD"
	RewardsExpiryDate string `json:"RewardsExpiryDate" dynamodbav:"RewardsExpiryDate"` // Format : "YYYY-MM-DD"
}
type RewardCards struct {
	CardId     string `json:"CardId" dynamodbav:"CardId"`         // SK , to Identify the card in the Cards Meta Data table
	CardNumber string `json:"CardNumber" dynamodbav:"CardNumber"` // Card Number

	CardName string `json:"CardName" dynamodbav:"CardName"` // Name of the Card
	CardDesc string `json:"CardDesc" dynamodbav:"CardDesc"` // Description of the Card

	RewardPoints int `json:"RewardPoints" dynamodbav:"RewardPoints"` // Points used to redeem the card

	Status string `json:"Status" dynamodbav:"Status"` // Allowed Values : "Active", "Expired" , "InProgress"

	CreatedDate string `json:"CreatedDate" dynamodbav:"CreatedDate"` // Date when the card was redeemed
	ExpiryDate  string `json:"ExpiryDate" dynamodbav:"ExpiryDate"`   // Date when the card expires

	// Additional Parameters but not required.
	// Notes      string `json:"Notes" dynamodbav:"Notes"`           // Notes for the Redemption, or adding appreciations etc.( Entered when Card Generations Module )
}
type EmployeeCertificates struct {
	CertificatesId  string `json:"CertificatesId" dynamodbav:"CertificatesId"`
	CertificateName string `json:"CertificateName" dynamodbav:"CertificateName"`
	DateAwarded     string `json:"DateAwarded" dynamodbav:"DateAwarded"`
	CertificatesImg string `json:"CertificatesImg" dynamodbav:"CertificatesImg"`
	Title           string `json:"Title" dynamodbav:"Title"`
}
type RoleNames struct {
	AdminRole               bool `json:"AdminRole" dynamodbav:"AdminRole"`
	UserManagementRole      bool `json:"UserManagementRole" dynamodbav:"UserManagementRole"`
	AnalyticsRole           bool `json:"AnalyticsRole" dynamodbav:"AnalyticsRole"`
	RewardsManagerRole      bool `json:"RewardsManagerRole" dynamodbav:"RewardsManagerRole"`
	AppreciationManagerRole bool `json:"AppreciationManagerRole" dynamodbav:"AppreciationManagerRole"`
	TeamsManagerRole        bool `json:"TeamsManagerRole" dynamodbav:"TeamsManagerRole"`
	User                    bool `json:"User" dynamodbav:"User"`
}

// ------------- Tenant Admin Related Functions

//  1. Get Employee Data functions

func (svc *EmployeeService) GetEmployeeDataByUserName(EmployeeUserName string) (EmployeeDynamodbData, error) {

	getItemInput := dynamodb.GetItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: EmployeeUserName},
		},
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)

	if err != nil {
		svc.logger.Printf("Query on Employee data failed with error :%v", err)
		return EmployeeDynamodbData{}, err
	}

	EmployeeData := EmployeeDynamodbData{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &EmployeeData)
	if err != nil {
		svc.logger.Printf("Query on Employee data Unmarshal failed with error :%v", err)
		return EmployeeDynamodbData{}, err
	}
	if EmployeeData.ProfilePic == "" {
		EmployeeData.ProfilePic = DEFAULT_PROFILE_PIC
	}

	return EmployeeData, nil
}

func (svc *EmployeeService) GetEmployeeDataRewardSettingsByUserName(EmployeeUserName string) (EmployeeDynamodbData, error) {

	getItemInput := dynamodb.GetItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: EmployeeUserName},
		},
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)

	if err != nil {
		svc.logger.Printf("Query on Employee data failed with error :%v", err)
		return EmployeeDynamodbData{}, err
	}

	EmployeeData := EmployeeDynamodbData{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &EmployeeData)
	if err != nil {
		svc.logger.Printf("Query on Employee data Unmarshal failed with error :%v", err)
		return EmployeeDynamodbData{}, err
	}
	if EmployeeData.ProfilePic == "" {
		EmployeeData.ProfilePic = DEFAULT_PROFILE_PIC
	}

	// Rewards Information
	output, err = svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.RewardsRuleTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus}, // PK for Get Reward Type Settings
			"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus}, // SK
		},
	})
	if err != nil {
		svc.logger.Printf("Unable to perform Get Operation on Reward Rules Table")
		return EmployeeDynamodbData{}, err
	}

	var ddbData RewardsRuleDynamodbData
	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &ddbData)
	if err != nil {
		svc.logger.Printf("Unable to Unmarshal the output from Get Operation on Reward Rules Table")
		return EmployeeDynamodbData{}, err
	}

	// if EmployeeData.RewardsData == nil {
	// 	EmployeeData.RewardsData = make(map[string]EmployeeRewards)
	// }

	// for _, id := range []string{"RD00", "RD01", "RD02", "RD03"} {
	// 	if status, ok := ddbData.RewardTypeStatus[id]; ok {
	// 		data := EmployeeData.RewardsData[id] // this is a copy
	// 		data.IsActive = status.Active        // modify the copy
	// 		EmployeeData.RewardsData[id] = data  // reassign it back
	// 	}
	// }

	return EmployeeData, nil
}

type GetBasicEmployeeData struct {
	UserName    string `json:"UserName" dynamodbav:"UserName"`
	EmailID     string `json:"EmailId" dynamodbav:"EmailId"`
	ExternalId  string `json:"ExternalId" dynamodbav:"ExternalId"`
	DisplayName string `json:"DisplayName" dynamodbav:"DisplayName"`
	Designation string `json:"Designation" dynamodbav:"Designation"`
	ProfilePic  string `json:"ProfilePic" dynamodbav:"ProfilePic"`
	IsManager   string `json:"IsManager" dynamodbav:"IsManager"`
}

func (svc *EmployeeService) GetEmployeeDataByUserNameBasicData(EmployeeUserName string) (GetBasicEmployeeData, error) {

	getItemInput := dynamodb.GetItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: EmployeeUserName},
		},
		ProjectionExpression: aws.String("UserName, EmailId, ExternalId, DisplayName, Designation, ProfilePic, IsManager"),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)

	if err != nil {
		svc.logger.Printf("Query on Employee data failed with error :%v", err)
		return GetBasicEmployeeData{}, err
	}

	EmployeeData := GetBasicEmployeeData{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &EmployeeData)
	if err != nil {
		svc.logger.Printf("Query on Employee data Unmarshal failed with error :%v", err)
		return GetBasicEmployeeData{}, err
	}

	return EmployeeData, nil
}
func (svc *EmployeeService) GetEmployeeDataByCognitoId(EmployeeCognitoId string) (EmployeeDynamodbData, error) {

	queryItemInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.EmployeeTable),
		IndexName:              aws.String(svc.EmployeeTable_CognitoId_Index),
		KeyConditionExpression: aws.String("CognitoId = :CognitoId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":CognitoId": &dynamodb_types.AttributeValueMemberS{Value: EmployeeCognitoId},
		},
	}
	svc.logger.Println("the user ID", EmployeeCognitoId)
	output, err := svc.dynamodbClient.Query(svc.ctx, &queryItemInput)

	svc.logger.Println("Query log", output)

	if err != nil {
		svc.logger.Printf("Query on Employee data using CognitoId failed with error :%v", err)
		return EmployeeDynamodbData{}, err
	}

	if output.Count == 0 {
		return EmployeeDynamodbData{}, fmt.Errorf("no data found for the employee with cognito-id: %v", EmployeeCognitoId)
	}
	if output.Count > 1 {
		return EmployeeDynamodbData{}, fmt.Errorf("multiple records found for the employee with same cognito-id: %v", EmployeeCognitoId)
	}

	EmployeeData := EmployeeDynamodbData{}

	// Add safety check for nil Items[0] before unmarshaling
	if output.Items[0] == nil {
		svc.logger.Printf("Query returned nil item for employee with cognito-id: %v", EmployeeCognitoId)
		return EmployeeDynamodbData{}, fmt.Errorf("query returned nil item for employee with cognito-id: %v", EmployeeCognitoId)
	}

	// Log the raw item before unmarshaling for debugging
	svc.logger.Printf("Raw DynamoDB item for cognito-id %s: %+v", EmployeeCognitoId, output.Items[0])

	// Unmarshal the first item from the query result
	err = dynamodb_attributevalue.UnmarshalMap(output.Items[0], &EmployeeData)
	if err != nil {
		svc.logger.Printf("Query on Employee data Unmarshal failed with error :%v", err)
		return EmployeeDynamodbData{}, err
	}

	if EmployeeData.ProfilePic == "" {
		EmployeeData.ProfilePic = DEFAULT_PROFILE_PIC
	}

	return EmployeeData, nil
}
func (svc *EmployeeService) GetEmployeeDataByEmail(EmployeeEmailId string) (EmployeeDynamodbData, error) {

	queryItemInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.EmployeeTable),
		IndexName:              aws.String(svc.EmployeeTable_EmailId_Index),
		KeyConditionExpression: aws.String("EmailId = :EmailId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":EmailId": &dynamodb_types.AttributeValueMemberS{Value: EmployeeEmailId},
		},
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryItemInput)
	if err != nil {
		var conditionFailed *dynamodb_types.ConditionalCheckFailedException
		if !errors.As(err, &conditionFailed) {
			svc.logger.Printf("Could not query on Employee Data for EmployeeEmailId:" + EmployeeEmailId + " Failed with Error: " + err.Error())
			return EmployeeDynamodbData{}, err
		}
	}
	// Returns nil if Employee Data not found
	if output.Count == 0 {
		return EmployeeDynamodbData{}, nil
	}

	EmployeeData := EmployeeDynamodbData{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Items[0], &EmployeeData)
	if err != nil {
		svc.logger.Printf("Could not Unmarshal Employee Data for EmployeeEmailId:" + EmployeeEmailId + " Failed with Error: " + err.Error())
		return EmployeeDynamodbData{}, err
	}
	if EmployeeData.ProfilePic == "" {
		EmployeeData.ProfilePic = DEFAULT_PROFILE_PIC
	}

	return EmployeeData, nil
}
func (svc *EmployeeService) GetEmployeeDataByExternalId(EmployeeExternalId string) (EmployeeDynamodbData, error) {

	queryItemInput := dynamodb.QueryInput{
		TableName:              aws.String(svc.EmployeeTable),
		IndexName:              aws.String(svc.EmployeeTable_ExternalId_Index),
		KeyConditionExpression: aws.String("ExternalId = :ExternalId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":ExternalId": &dynamodb_types.AttributeValueMemberS{Value: EmployeeExternalId},
		},
	}

	output, err := svc.dynamodbClient.Query(svc.ctx, &queryItemInput)
	if err != nil {
		var conditionFailed *dynamodb_types.ConditionalCheckFailedException
		if !errors.As(err, &conditionFailed) {
			svc.logger.Printf("Could not query on Employee Data for EmployeeExternalId:" + EmployeeExternalId + " Failed with Error: " + err.Error())
			return EmployeeDynamodbData{}, err
		}
	}
	// Returns nil if Employee Data not found
	if output.Count == 0 {
		return EmployeeDynamodbData{}, nil
	}

	EmployeeData := EmployeeDynamodbData{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Items[0], &EmployeeData)
	if err != nil {
		svc.logger.Printf("Could not Unmarshal Employee Data for EmployeeExternalId:" + EmployeeExternalId + " Failed with Error: " + err.Error())
		return EmployeeDynamodbData{}, err
	}
	if EmployeeData.ProfilePic == "" {
		EmployeeData.ProfilePic = DEFAULT_PROFILE_PIC
	}

	return EmployeeData, nil
}

/*
Running scan on whole table is not efficient,
hence currently we have implemented to get only first 100 Items from the Employee Table.

If needed, implement future changes to capture LastEvaluatedKey form the DDB to perform Paginated Responses
*/
func (svc *EmployeeService) GetAllEmployeeData() ([]EmployeeDynamodbData, error) {

	scanOutput, err := svc.dynamodbClient.Scan(svc.ctx, &dynamodb.ScanInput{
		TableName:            aws.String(svc.EmployeeTable),
		Limit:                aws.Int32(1000),
		ProjectionExpression: aws.String("UserName, EmailId, ExternalId, DisplayName, Designation, IsManager, MgrUserName, StartDate, EndDate, IsActive, RolesData, ProfilePic"),
	})

	if err != nil {
		svc.logger.Printf("Error running scan on the Employee Table. Failed with error: %v", err)
		return []EmployeeDynamodbData{}, err
	}

	if len(scanOutput.Items) == 0 {
		svc.logger.Println("No Data found on the Employee Table")
		return []EmployeeDynamodbData{}, nil
	}

	var allEmployeeData []EmployeeDynamodbData

	for _, data := range scanOutput.Items {
		var EmployeeData EmployeeDynamodbData
		err = dynamodb_attributevalue.UnmarshalMap(data, &EmployeeData)
		if err != nil {
			svc.logger.Printf("Failed to Unmarshal DDB Output data. Failed with error: %v", err)
			return []EmployeeDynamodbData{}, err
		}
		if EmployeeData.ProfilePic == "" {
			EmployeeData.ProfilePic = DEFAULT_PROFILE_PIC
		}
		allEmployeeData = append(allEmployeeData, EmployeeData)
	}

	return allEmployeeData, nil
}

// 2. Update Employee Data functions

type BasicEmployeeDetails struct {
	UserName string `json:"UserName"`

	ExternalId  string `json:"ExternalId"`
	DisplayName string `json:"DisplayName"`
	PhoneNumber string `json:"PhoneNumber"`
	LoginType   string `json:"LoginType"`

	IsManager       string `json:"IsManager"`
	ManagerUserName string `json:"MgrUserName"`

	StartDate string `json:"StartDate"`
	EndDate   string `json:"EndDate"`

	IsActive string `json:"IsActive"`
}

func (svc *EmployeeService) UpdateEmployeeDetailsByUserName(updateDetails BasicEmployeeDetails) error {

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: updateDetails.UserName},
		},
		ConditionExpression: aws.String("attribute_exists(UserName)"),
		UpdateExpression:    aws.String("SET ExternalId = :ExternalId, DisplayName = :DisplayName, PhoneNumber = :PhoneNumber, LoginType = :LoginType, IsManager = :IsManager, MgrUserName = :MgrUserName, StartDate = :StartDate, EndDate = :EndDate, IsActive = :IsActive"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":ExternalId":  &dynamodb_types.AttributeValueMemberS{Value: updateDetails.ExternalId},
			":DisplayName": &dynamodb_types.AttributeValueMemberS{Value: updateDetails.DisplayName},
			":PhoneNumber": &dynamodb_types.AttributeValueMemberS{Value: updateDetails.PhoneNumber},
			":LoginType":   &dynamodb_types.AttributeValueMemberS{Value: updateDetails.LoginType},
			":IsManager":   &dynamodb_types.AttributeValueMemberS{Value: updateDetails.IsManager},
			":MgrUserName": &dynamodb_types.AttributeValueMemberS{Value: updateDetails.ManagerUserName},
			":StartDate":   &dynamodb_types.AttributeValueMemberS{Value: updateDetails.StartDate},
			":EndDate":     &dynamodb_types.AttributeValueMemberS{Value: updateDetails.EndDate},
			":IsActive":    &dynamodb_types.AttributeValueMemberS{Value: updateDetails.IsActive},
		},
	}

	svc.logger.Printf("updateItemInput: %v", updateItemInput)

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}

	return nil
}

type AllEmployeeDetails struct {
	UserName string `json:"UserName"`

	ExternalId  string `json:"ExternalId"`
	DisplayName string `json:"DisplayName"`
	PhoneNumber string `json:"PhoneNumber"`
	LoginType   string `json:"LoginType"`

	IsManager       string `json:"IsManager"`
	ManagerUserName string `json:"MgrUserName"`

	StartDate string `json:"StartDate"`
	EndDate   string `json:"EndDate"`

	IsActive string `json:"IsActive"`

	RoleData RoleNames `json:"RoleData"`
}

func (svc *EmployeeService) UpdateAllEmployeeDetailsByUserName(updateDetails AllEmployeeDetails) error {

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: updateDetails.UserName},
		},
		ConditionExpression: aws.String("attribute_exists(UserName)"),
		UpdateExpression:    aws.String("SET ExternalId = :ExternalId, DisplayName = :DisplayName, PhoneNumber = :PhoneNumber, LoginType = :LoginType, IsManager = :IsManager, MgrUserName = :MgrUserName, StartDate = :StartDate, EndDate = :EndDate, IsActive = :IsActive, RolesData = :RolesData"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":ExternalId":  &dynamodb_types.AttributeValueMemberS{Value: updateDetails.ExternalId},
			":DisplayName": &dynamodb_types.AttributeValueMemberS{Value: updateDetails.DisplayName},
			":PhoneNumber": &dynamodb_types.AttributeValueMemberS{Value: updateDetails.PhoneNumber},
			":LoginType":   &dynamodb_types.AttributeValueMemberS{Value: updateDetails.LoginType},
			":IsManager":   &dynamodb_types.AttributeValueMemberS{Value: updateDetails.IsManager},
			":MgrUserName": &dynamodb_types.AttributeValueMemberS{Value: updateDetails.ManagerUserName},
			":StartDate":   &dynamodb_types.AttributeValueMemberS{Value: updateDetails.StartDate},
			":EndDate":     &dynamodb_types.AttributeValueMemberS{Value: updateDetails.EndDate},
			":IsActive":    &dynamodb_types.AttributeValueMemberS{Value: updateDetails.IsActive},
			":RolesData": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					"User":                    &dynamodb_types.AttributeValueMemberBOOL{Value: updateDetails.RoleData.User},
					"AdminRole":               &dynamodb_types.AttributeValueMemberBOOL{Value: updateDetails.RoleData.AdminRole},
					"UserManagementRole":      &dynamodb_types.AttributeValueMemberBOOL{Value: updateDetails.RoleData.UserManagementRole},
					"AnalyticsRole":           &dynamodb_types.AttributeValueMemberBOOL{Value: updateDetails.RoleData.AnalyticsRole},
					"RewardsManagerRole":      &dynamodb_types.AttributeValueMemberBOOL{Value: updateDetails.RoleData.RewardsManagerRole},
					"AppreciationManagerRole": &dynamodb_types.AttributeValueMemberBOOL{Value: updateDetails.RoleData.AppreciationManagerRole},
					"TeamsManagerRole":        &dynamodb_types.AttributeValueMemberBOOL{Value: updateDetails.RoleData.TeamsManagerRole},
				},
			},
		},
	}

	svc.logger.Printf("updateItemInput: %v", updateItemInput)

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}
	return nil
}

func (svc *EmployeeService) UpdateRolesofEmployeeByUserName(userName string, roleData RoleNames) error {

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: userName},
		},
		ConditionExpression: aws.String("attribute_exists(UserName)"),
		UpdateExpression:    aws.String("SET RolesData = :RolesData"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":RolesData": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					// Value: map[string]dynamodb_types.AttributeValue{
					"User":                    &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.User},
					"AdminRole":               &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AdminRole},
					"UserManagementRole":      &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.UserManagementRole},
					"AnalyticsRole":           &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AnalyticsRole},
					"RewardsManagerRole":      &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.RewardsManagerRole},
					"AppreciationManagerRole": &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AppreciationManagerRole},
					"TeamsManagerRole":        &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.TeamsManagerRole},
					// },
				},
			},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	svc.logger.Printf("updateItemInput: %v", updateItemInput)

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}

	return nil
}

func (svc *EmployeeService) UpdateRolesofEmployeeByEmailId(emailId string, roleData RoleNames) error {

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"EmailId": &dynamodb_types.AttributeValueMemberS{Value: emailId},
		},
		ConditionExpression: aws.String("attribute_exists(EmailId)"),
		UpdateExpression:    aws.String("SET RolesData = :RolesData"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":RolesData": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					// Value: map[string]dynamodb_types.AttributeValue{
					"User":                    &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.User},
					"AdminRole":               &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AdminRole},
					"UserManagementRole":      &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.UserManagementRole},
					"AnalyticsRole":           &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AnalyticsRole},
					"RewardsManagerRole":      &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.RewardsManagerRole},
					"AppreciationManagerRole": &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AppreciationManagerRole},
					"TeamsManagerRole":        &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.TeamsManagerRole},
					// },
				},
			},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}

	return nil
}

func (svc *EmployeeService) UpdateRolesofEmployeeByExternalId(externalId string, roleData RoleNames) error {

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"ExternalId": &dynamodb_types.AttributeValueMemberS{Value: externalId},
		},
		ConditionExpression: aws.String("attribute_exists(ExternalId)"),
		UpdateExpression:    aws.String("SET RolesData = :RolesData"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":RolesData": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					// Value: map[string]dynamodb_types.AttributeValue{
					"User":                    &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.User},
					"AdminRole":               &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AdminRole},
					"UserManagementRole":      &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.UserManagementRole},
					"AnalyticsRole":           &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AnalyticsRole},
					"RewardsManagerRole":      &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.RewardsManagerRole},
					"AppreciationManagerRole": &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.AppreciationManagerRole},
					"TeamsManagerRole":        &dynamodb_types.AttributeValueMemberBOOL{Value: roleData.TeamsManagerRole},
					// },
				},
			},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}

	return nil
}

// Update Employee Profile Pic
func (svc *EmployeeService) UpdateEmployeeProfilePicByUserName(userName string, profilePic string) error {

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.EmployeeTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"UserName": &dynamodb_types.AttributeValueMemberS{Value: userName},
		},
		ConditionExpression: aws.String("attribute_exists(UserName)"),
		UpdateExpression:    aws.String("SET ProfilePic = :ProfilePic"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":ProfilePic": &dynamodb_types.AttributeValueMemberS{Value: profilePic},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}

	return nil
}

func (svc *EmployeeService) CreateEmployeeFromFrontend(userData EmployeeDynamodbData) error {

	ddbItem, err := dynamodb_attributevalue.MarshalMap(userData)
	if err != nil {
		return err
	}

	putItemInput := dynamodb.PutItemInput{
		Item:      ddbItem,
		TableName: aws.String(svc.EmployeeTable),
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, &putItemInput)
	if err != nil {
		svc.logger.Printf("PutItem failed with error :%v", err)
		return err
	}
	svc.logger.Printf("Successfully performed put operation")

	return nil
}

// Employee Groups
type EmployeeGroups struct {
	GroupId   string `dynamodbav:"GroupId"`   // PK as GroupName
	GroupName string `dynamodbav:"GroupName"` // GroupName is GroupId if not provided
	GroupDesc string `dynamodbav:"GroupDesc"`
	IsActive  bool   `dynamodbav:"IsActive"` // Allowed Values : true/false
	GroupPic  string `dynamodbav:"GroupPic"` // Path to the S3 Image
}

func (svc *EmployeeService) UpdateEmployeeGroups(groupData EmployeeGroups) error {

	output, err := svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.EmployeeGroupsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"GroupId": &dynamodb_types.AttributeValueMemberS{
				Value: groupData.GroupId,
			},
		},
		UpdateExpression: aws.String("SET GroupName = :GroupName, GroupDesc = :GroupDesc"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":GroupName": &dynamodb_types.AttributeValueMemberS{Value: groupData.GroupName}, // changed from number to string !
			":GroupDesc": &dynamodb_types.AttributeValueMemberS{Value: groupData.GroupDesc},
		},
		ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
	})

	if err != nil {
		svc.logger.Printf("[Error] Failed to Update the Groups Table '%s': %v", svc.EmployeeGroupsTable, err)
		return err
	}

	svc.logger.Printf("Updated Groups Table with : %v", output.Attributes)

	return nil

}

func (svc *EmployeeService) GetAllEmployeeGroupsInMap() (map[string]EmployeeGroups, error) {

	// 1. Scan the DDB table
	groups := make(map[string]EmployeeGroups)
	allGroupsDDBOutput := []map[string]dynamodb_types.AttributeValue{}

	scanTableInput := dynamodb.ScanInput{
		TableName: aws.String(svc.EmployeeGroupsTable),
		Limit:     aws.Int32(1000),
	}

	output, err := svc.dynamodbClient.Scan(svc.ctx, &scanTableInput)
	if err != nil {
		svc.logger.Printf("Failed to scan the Groups table. Error: %v", err)
		return map[string]EmployeeGroups{}, err
	}
	pagination := output.LastEvaluatedKey
	allGroupsDDBOutput = append(allGroupsDDBOutput, output.Items...)

	for pagination != nil {
		next_scan_output, err := svc.dynamodbClient.Scan(svc.ctx, &dynamodb.ScanInput{
			TableName:         aws.String(svc.EmployeeGroupsTable),
			Limit:             aws.Int32(1000),
			ExclusiveStartKey: pagination,
		})
		if err != nil {
			svc.logger.Printf("Failed to scan the Groups table. Error: %v", err)
			return map[string]EmployeeGroups{}, err
		}
		allGroupsDDBOutput = append(allGroupsDDBOutput, next_scan_output.Items...)
		pagination = next_scan_output.LastEvaluatedKey
	}

	// 2. Store All groups in Map struct

	for _, groupItem := range allGroupsDDBOutput {
		group := EmployeeGroups{}
		err := dynamodb_attributevalue.UnmarshalMap(groupItem, &group)
		if err != nil {
			svc.logger.Printf("Unable to Unmarshal the DDB output to struct")
			return map[string]EmployeeGroups{}, err
		}
		groups[group.GroupId] = group

	}

	return groups, nil
}

// ------------ Company Employee Cognito related functions ------------

func (svc *EmployeeService) CheckUserExists(username string) (bool, error) {
	_, err := svc.CognitoClient.AdminGetUser(svc.ctx, &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(svc.EmployeeUserPoolId),
		Username:   aws.String(username),
	})

	var errUserNotFound *cognito_types.UserNotFoundException

	if err == errUserNotFound {
		return false, nil
	}

	if err != nil && err != errUserNotFound {
		svc.logger.Println("Error checking user existence:", err)
		return false, err
	}
	svc.logger.Printf("User %v exists", username)
	return true, nil
}

func (svc *EmployeeService) CreateCognitoUser(userName string) error {

	output, err := svc.CognitoClient.AdminCreateUser(svc.ctx, &cognitoidentityprovider.AdminCreateUserInput{
		UserPoolId: aws.String(svc.EmployeeUserPoolId),
		Username:   aws.String(userName),
	})

	svc.logger.Printf("output: %v", output.User.Username)

	if err != nil {
		svc.logger.Printf("Error creating user in Cognito for user: %v, err: %v", userName, err)
		return err
	}
	svc.logger.Printf("User Creation is successful for username: %v", userName)
	return nil
}

func (svc *EmployeeService) UpdateEmployeeData(userName string) error {
	svc.logger.Println("Retrieve user attributes from Cognito for the specified user")

	userAttributes, err := svc.CognitoClient.AdminGetUser(svc.ctx, &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String(svc.EmployeeUserPoolId),
		Username:   aws.String(userName),
	})

	if err != nil {
		svc.logger.Println("Error fetching user attributes from Cognito:", err)
		return err
	}

	for _, attr := range userAttributes.UserAttributes {
		if *attr.Name == "sub" {
			attributeValue := map[string]dynamodb_types.AttributeValue{
				":CognitoId": &dynamodb_types.AttributeValueMemberS{Value: *attr.Value},
			}
			svc.logger.Println("attributeValue:", attributeValue)
			expressionAttributeNames := map[string]string{
				"#CognitoId": "CognitoId",
			}

			svc.logger.Println("expressionAttributeNames", expressionAttributeNames)

			input := dynamodb.UpdateItemInput{
				TableName: aws.String(svc.EmployeeTable),
				Key: map[string]dynamodb_types.AttributeValue{
					"UserName": &dynamodb_types.AttributeValueMemberS{Value: userName},
				},
				UpdateExpression:          aws.String("SET #CognitoId = :CognitoId"),
				ExpressionAttributeNames:  expressionAttributeNames,
				ExpressionAttributeValues: attributeValue,
			}

			_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &input)
			if err != nil {
				svc.logger.Printf("Error updating EmployeeData table '%s': %v", svc.EmployeeTable, err)
				return err
			}
			svc.logger.Println("Updated the table with CognitoId")
		}
	}

	return nil
}

func (svc *EmployeeService) DeleteEmployeeData(userName string) error {
	// Check if the user exists in Cognito

	userExists, err := svc.CheckUserExists(userName)
	if err != nil {
		return err
	}
	if !userExists {
		svc.logger.Printf("User is not present in the Cognito Pool, No action taken for User %v", userName)
		return nil
	}

	_, err = svc.CognitoClient.AdminDeleteUser(svc.ctx, &cognitoidentityprovider.AdminDeleteUserInput{
		UserPoolId: aws.String(svc.EmployeeUserPoolId),
		Username:   aws.String(userName),
	})
	if err != nil {
		return err
	}

	return nil
}

// ------------ Company Employee Teams related functions ------------

func (svc *EmployeeService) UpdateTenantTeams(employeeData EmployeeDynamodbData) error {
	svc.logger.Println("employeeData in UpdateTenantTeams:", employeeData)
	// When user is IsManager field “Y” , then
	if employeeData.IsManager == "Y" {
		// 1) Create a default Team in Employee Teams Table.
		defaultTeam := map[string]dynamodb_types.AttributeValue{
			"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "TEAM-" + employeeData.UserName},
			"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "TEAM-DEFAULT"},
			"TeamTypeId":      &dynamodb_types.AttributeValueMemberS{Value: TEAM_TYPE_GENERAL},
			"TeamName":        &dynamodb_types.AttributeValueMemberS{Value: employeeData.DisplayName + "'s TEAM (GENERAL)"},
			"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: employeeData.IsActive},
		}
		if _, err := svc.dynamodbClient.PutItem(svc.ctx, &dynamodb.PutItemInput{
			TableName: aws.String(svc.TenantTeamsTable),
			Item:      defaultTeam,
		}); err != nil {
			return err
		}
		// 2) Create a Manager User in the Employee Teams Table
		managerEntry := map[string]dynamodb_types.AttributeValue{
			"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "MNGR-" + employeeData.UserName},
			"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "TEAM-" + employeeData.UserName},
			"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: employeeData.IsActive},
		}
		if _, err := svc.dynamodbClient.PutItem(svc.ctx, &dynamodb.PutItemInput{
			TableName: aws.String(svc.TenantTeamsTable),
			Item:      managerEntry,
		}); err != nil {
			return err
		}
	}
	// When User is MgrUserName is Not Empty , then
	if employeeData.MgrUserName != "" {
		// 1) Create a User in Employee Teams Table
		userEntry := map[string]dynamodb_types.AttributeValue{
			"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "USER-" + employeeData.UserName},
			"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: "TEAM-" + employeeData.MgrUserName},
			"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: employeeData.IsActive},
		}
		if _, err := svc.dynamodbClient.PutItem(svc.ctx,
			&dynamodb.PutItemInput{
				TableName: aws.String(svc.TenantTeamsTable),
				Item:      userEntry,
			}); err != nil {
			return err
		}
	}
	return nil
}
