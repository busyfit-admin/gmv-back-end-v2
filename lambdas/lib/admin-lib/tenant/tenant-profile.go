package adminlib

import (
	"context"
	"fmt"
	"log"

	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type TenantDetailsTable struct {
	TenantId   string `dynamodbav:"TenantId" json:"TenantId"`     // PK . To be generated during Tenant Creation
	TenantName string `dynamodbav:"TenantName" json:"TenantName"` // Name of the Tenant joining the program

	TenantDesc string `dynamodbav:"TenantDesc" json:"TenantDesc"`
	Industry   string `dynamodbav:"Industry" json:"Industry"`

	TenantContacts map[string]TenantContacts `dynamodbav:"TenantContacts" json:"TenantContacts"` // Map of all tenant Contacts. At-least one is required.

	TenantCreationDate string `dynamodbav:"TenantCreationDate" json:"TenantCreationDate"`
	TenantStageId      string `dynamodbav:"TenantStageId"` // Can take values as per Tenant Lifecycle stages
	TenantStageName    string `dynamodbav:"TenantStageName" json:"TenantStageName"`
}

type TenantContacts struct {
	TenantEmail string `dynamodbav:"TenantEmail" json:"TenantEmail"` // should be unique

	TenantContactName string `dynamodbav:"TenantContactName" json:"TenantContactName"`
	TenantPh          string `dynamodbav:"TenantPh" json:"TenantPh"`

	IsPrimary bool `dynamodbav:"IsPrimary" json:"IsPrimary"`
}

// -------------------- Tenant Profile Functions -------------

type TenantDetailsSvc struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	TenantDetailsTable              string
	TenantDetails_TenantStatusIndex string
}

func CreateTenantDetailsSvc(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *TenantDetailsSvc {
	return &TenantDetailsSvc{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

// -------------------- Tenant Get Profile(s) API related functions -------------

// Accepts (TenantId) and returns all TenantDetailsTable data
func (svc *TenantDetailsSvc) GetTenantProfileById(TenantId string) (TenantDetailsTable, error) {

	getItemInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"TenantId": &dynamodb_types.AttributeValueMemberS{Value: TenantId},
		},
		TableName:      aws.String(svc.TenantDetailsTable),
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)
	if err != nil {
		svc.logger.Printf("Get Tenant Failed with error :%v", err)
		return TenantDetailsTable{}, err
	}

	TenantData := TenantDetailsTable{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &TenantData)
	if err != nil {
		svc.logger.Printf("Get Tenant Unmarshal failed with error :%v", err)
		return TenantDetailsTable{}, err
	}

	SetStageIdFromStageName_GetTenantData(&TenantData)

	return TenantData, nil
}

// Required Output for Listing Tenant data in Admin List Tenants page
type TenantDetails struct {
	TenantId   string `dynamodbav:"TenantId" json:"TenantId"`
	TenantName string `dynamodbav:"TenantName" json:"TenantName"`

	Industry string `dynamodbav:"Industry" json:"Industry"`

	TenantCreationDate string `dynamodbav:"TenantCreationDate" json:"TenantCreationDate"`
	TenantStageId      string `dynamodbav:"TenantStageId" ` // Can take values as per Tenant Lifecycle stages
	TenantStageName    string `dynamodbav:"TenantStageName" json:"TenantStageName"`
}
type ListInProgActiveTenants struct {
	OnboardingInProg []TenantDetails `json:"OnboardingInProg"`
	Active           []TenantDetails `json:"Active"`
}

func (svc *TenantDetailsSvc) GetAllTenantDetails() (ListInProgActiveTenants, error) {

	var AllTenantsData ListInProgActiveTenants
	// 1) Query Tenant Details table to find all Onboarding InProg Tenants
	queryGetInProgTenants := "SELECT TenantId, TenantName, Industry, TenantCreationDate, TenantStageId FROM \"" + svc.TenantDetailsTable + "\".\"" + svc.TenantDetails_TenantStatusIndex + "\" WHERE TenantStageId IN ('STG01', 'STG02','STG03','STG04','STG05','STG06','STG07') ORDER BY TenantStageId ASC"

	OnboardingTenantsDetails, err := svc.ExecuteQueryDDB(queryGetInProgTenants)
	if err != nil {
		return ListInProgActiveTenants{}, err
	}
	AllTenantsData.OnboardingInProg = OnboardingTenantsDetails

	// 2) Query Tenant Details table to find all Active Tenants

	queryGetActiveTenants := "SELECT TenantId, TenantName, Industry, TenantCreationDate, TenantStageId FROM \"" + svc.TenantDetailsTable + "\".\"" + svc.TenantDetails_TenantStatusIndex + "\" WHERE TenantStageId = 'STG08'"

	ActiveTenantsDetails, err := svc.ExecuteQueryDDB(queryGetActiveTenants)
	if err != nil {
		return ListInProgActiveTenants{}, err
	}
	AllTenantsData.Active = ActiveTenantsDetails

	return AllTenantsData, nil
}

func (svc *TenantDetailsSvc) ExecuteQueryDDB(stmt string) ([]TenantDetails, error) {

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(stmt),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []TenantDetails{}, err
	}

	// Logging read units stats
	svc.logger.Printf("[DDB_USAGE_STATS] Read Units : %v", output.ConsumedCapacity)

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for Onboarding InProg Tenants")
		return []TenantDetails{}, nil
	}

	allTenantDetailsData := []TenantDetails{}
	for _, stageItem := range output.Items {
		tenantData := TenantDetails{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &tenantData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal tenant Details data  Error : %v", err)
			return []TenantDetails{}, err
		}
		// Set TenantStage Name
		SetStageIdFromStageName(&tenantData)
		// Append data to the overall tenant data
		allTenantDetailsData = append(allTenantDetailsData, tenantData)
	}

	return allTenantDetailsData, nil

}

// -------------------- Tenant POST and PATCH Profile(s) API related functions -------------

/*
 Types of POST and PATCH Requests handled by manage-tenant-profiles

 POST :
  1. Create the Tenant Profile ( This is the first step in onboarding a Tenant )
	  - Should consist of all required details from the Onboarding Tenant Profile page.

	  NOTE: Once a Tenant is created, we cannot update the TenantID.
	        If Tenant ID needs to be changed, we'll need to create another Tenant from start.


  PATCH API options:

  1. Update Name, Desc, Industry - Top level Information
  2. Add new Contact ID - Add new contactID in the TenantContacts ( currently not taking contactID delete option)
  3. Update the TenantStage to next Level

*/

// Required Fields from the Tenant Onboarding Page
type CreateTenantProfile struct {
	TenantName string `json:"TenantName"`
	TenantDesc string `json:"TenantDesc"`
	Industry   string `json:"Industry"`

	EnvType string `json:"EnvType"`

	PrimaryContactName  string `json:"PrimaryContactName"`
	PrimaryContactEmail string `json:"PrimaryContactEmail"`
	PrimaryContactPh    string `json:"PrimaryContactPh"`
}

func (svc *TenantDetailsSvc) CreateTenantProfile(tenantData CreateTenantProfile) error {

	//1.  Generate Unique ID for Tenant
	newTenantUUID := utils.GenerateRandomString(10)

	//2. Default Starting Stage of the Tenant is STG01 (Onboarding) + Create CreationDate
	defaultStartStage := INITIAL_ONBOARDING_STAGE_ID
	creationDate := utils.GenerateTimestamp()

	//3. Validate if all required fields are present ( TenantName, PrimaryContact as Min requirement )
	if tenantData.TenantName == "" || tenantData.PrimaryContactName == "" || tenantData.PrimaryContactEmail == "" || tenantData.PrimaryContactPh == "" {
		return fmt.Errorf("required fields for creation of tenant are missing")
	}

	//4. Use UpdateItem to ensure TenantID's are not overwritten
	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantDetailsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"TenantId": &dynamodb_types.AttributeValueMemberS{Value: newTenantUUID},
		},
		ConditionExpression: aws.String("attribute_not_exists(TenantContacts)"),
		UpdateExpression:    aws.String("SET TenantName = :TenantName, TenantDesc = :TenantDesc, Industry = :Industry, EnvType = :EnvType, TenantContacts = :TenantContacts, TenantCreationDate = :TenantCreationDate, TenantStageId = :TenantStageId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":TenantName": &dynamodb_types.AttributeValueMemberS{Value: tenantData.TenantName},
			":TenantDesc": &dynamodb_types.AttributeValueMemberS{Value: tenantData.TenantDesc},
			":Industry":   &dynamodb_types.AttributeValueMemberS{Value: tenantData.Industry},
			":TenantContacts": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					tenantData.PrimaryContactEmail: &dynamodb_types.AttributeValueMemberM{
						Value: map[string]dynamodb_types.AttributeValue{
							"TenantEmail":       &dynamodb_types.AttributeValueMemberS{Value: tenantData.PrimaryContactEmail},
							"TenantContactName": &dynamodb_types.AttributeValueMemberS{Value: tenantData.PrimaryContactName},
							"TenantPh":          &dynamodb_types.AttributeValueMemberS{Value: tenantData.PrimaryContactPh},
							"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: true},
						},
					},
				}},
			":EnvType":            &dynamodb_types.AttributeValueMemberS{Value: tenantData.EnvType},
			":TenantCreationDate": &dynamodb_types.AttributeValueMemberS{Value: creationDate},
			":TenantStageId":      &dynamodb_types.AttributeValueMemberS{Value: defaultStartStage},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}

	//5. Necessary Logging in cloudwatch logs
	svc.logger.Printf("[LOGGER] New Tenant has been added successfully. New TenantID: %s ", newTenantUUID)

	//6. Return Success
	return nil
}

// PATCH API - Update Name, Desc, Industry, EnvType - Top level Information
type PatchTopLevelInfo struct {
	TenantId string `json:"TenantId"`

	TenantName string `json:"TenantName"`
	TenantDesc string `json:"TenantDesc"`
	Industry   string `json:"Industry"`
	EnvType    string `json:"EnvType"`
}

func (svc *TenantDetailsSvc) PatchTopLevelInfo(TopLevelInfo PatchTopLevelInfo) error {

	// 1. Ensure Name , EnvType is not empty string in incoming Update set
	if TopLevelInfo.TenantName == "" || TopLevelInfo.EnvType == "" || TopLevelInfo.TenantId == "" {
		return fmt.Errorf("tenant name or env type cannot be empty for update set")
	}

	ddbUpdateItem := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantDetailsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"TenantId": &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.TenantId},
		},
		UpdateExpression:    aws.String("SET TenantName = :TenantName, TenantDesc = :TenantDesc, Industry = :Industry, EnvType = :EnvType"),
		ConditionExpression: aws.String("attribute_exists(TenantId)"), // This will prevent new tenant creation when performing updates
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":TenantName": &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.TenantName},
			":TenantDesc": &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.TenantDesc},
			":Industry":   &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.Industry},
			":EnvType":    &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.EnvType},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &ddbUpdateItem)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}

	svc.logger.Printf("[LOGGER] Top level Tenant details have been updated successfully. TenantID: %s ", TopLevelInfo.TenantId)

	return nil
}

// PATCH API - Add new ContactId in the Tenant Contacts
type PatchTenantContacts struct {
	TenantId string `json:"TenantId"`

	ContactName  string `json:"ContactName"`
	ContactEmail string `json:"ContactEmail"`
	ContactPh    string `json:"ContactPh"`
	IsPrimary    bool   `json:"IsPrimary"`
}

func (svc *TenantDetailsSvc) PatchTenantContacts(contactInfo PatchTenantContacts) error {

	// Check if the values provided are not null
	if contactInfo.TenantId == "" || contactInfo.ContactName == "" || contactInfo.ContactEmail == "" {
		return fmt.Errorf("contact info cannot be empty. provide mandate information")
	}

	ddbInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantDetailsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"TenantId": &dynamodb_types.AttributeValueMemberS{Value: contactInfo.TenantId},
		},
		UpdateExpression: aws.String("SET TenantContacts.#ContactId = :Contact"),
		ExpressionAttributeNames: map[string]string{
			"#ContactId": contactInfo.ContactEmail,
		},
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":Contact": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					"TenantEmail":       &dynamodb_types.AttributeValueMemberS{Value: contactInfo.ContactEmail},
					"TenantContactName": &dynamodb_types.AttributeValueMemberS{Value: contactInfo.ContactName},
					"TenantPh":          &dynamodb_types.AttributeValueMemberS{Value: contactInfo.ContactPh},
					"IsPrimary":         &dynamodb_types.AttributeValueMemberBOOL{Value: contactInfo.IsPrimary},
				},
			},
		},
		ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &ddbInput)
	if err != nil {
		svc.logger.Printf("Error performing update on DDB table for input data: %v", contactInfo)
		return err
	}

	return nil
}

// PATCH API - Update Stage - Top level Information
type PatchTenantOverallStage struct {
	TenantId string `json:"TenantId"`

	TenantStageId   string
	TenantStageName string `json:"TenantStageName"`
}

func (svc *TenantDetailsSvc) PatchOverallStageId(patchStageInfo PatchTenantOverallStage) error {

	// Check if Tenant Stage name is not empty:
	if patchStageInfo.TenantStageName == "" || patchStageInfo.TenantId == "" {
		return fmt.Errorf("tenant Id or StageName cannot be empty for update set")
	}

	SetStageIdFromStageName_PatchTenantOverallStage(&patchStageInfo)

	ddbUpdateItem := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantDetailsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"TenantId": &dynamodb_types.AttributeValueMemberS{Value: patchStageInfo.TenantId},
		},
		UpdateExpression:    aws.String("SET TenantStageId = :TenantStageId"),
		ConditionExpression: aws.String("attribute_exists(TenantId)"), // This will prevent new tenant creation when performing updates
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":TenantStageId": &dynamodb_types.AttributeValueMemberS{Value: patchStageInfo.TenantStageId},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &ddbUpdateItem)
	if err != nil {
		svc.logger.Printf("Failed to Update the Tenant Table :%v", err)
		return err
	}

	return nil
}

// -------------------- Tenant APIs helper functions  -------------

func SetStageIdFromStageName(tenantData *TenantDetails) {

	switch tenantData.TenantStageId {
	case INITIAL_ONBOARDING_STAGE_ID:
		tenantData.TenantStageName = INITIAL_ONBOARDING
	case ONBOARDING_DEMO_STAGE_ID:
		tenantData.TenantStageName = ONBOARDING_DEMO
	case TRIAL_SETUP_STAGE_ID:
		tenantData.TenantStageName = TRIAL_SETUP
	case TRIAL_IN_PROG_STAGE_ID:
		tenantData.TenantStageName = TRIAL_IN_PROG
	case TRAIL_DISCONTINUED_STAGE_ID:
		tenantData.TenantStageName = TRAIL_DISCONTINUED
	case PRE_PROVISIONING_CHECKS_STAGE_ID:
		tenantData.TenantStageName = PRE_PROVISIONING_CHECKS
	case PROVISIONING_STAGE_ID:
		tenantData.TenantStageName = PROVISIONING
	case ACTIVE_STAGE_ID:
		tenantData.TenantStageName = ACTIVE
	case INACTIVE_STAGE_ID:
		tenantData.TenantStageName = INACTIVE
	case DEACTIVATED_STAGE_ID:
		tenantData.TenantStageName = DEACTIVATED
	default:
		tenantData.TenantName = "UNDEFINED"
	}

}

func SetStageIdFromStageName_GetTenantData(tenantData *TenantDetailsTable) {

	switch tenantData.TenantStageId {
	case INITIAL_ONBOARDING_STAGE_ID:
		tenantData.TenantStageName = INITIAL_ONBOARDING
	case ONBOARDING_DEMO_STAGE_ID:
		tenantData.TenantStageName = ONBOARDING_DEMO
	case TRIAL_SETUP_STAGE_ID:
		tenantData.TenantStageName = TRIAL_SETUP
	case TRIAL_IN_PROG_STAGE_ID:
		tenantData.TenantStageName = TRIAL_IN_PROG
	case TRAIL_DISCONTINUED_STAGE_ID:
		tenantData.TenantStageName = TRAIL_DISCONTINUED
	case PRE_PROVISIONING_CHECKS_STAGE_ID:
		tenantData.TenantStageName = PRE_PROVISIONING_CHECKS
	case PROVISIONING_STAGE_ID:
		tenantData.TenantStageName = PROVISIONING
	case ACTIVE_STAGE_ID:
		tenantData.TenantStageName = ACTIVE
	case INACTIVE_STAGE_ID:
		tenantData.TenantStageName = INACTIVE
	case DEACTIVATED_STAGE_ID:
		tenantData.TenantStageName = DEACTIVATED
	default:
		tenantData.TenantName = "UNDEFINED"
	}

}

func SetStageIdFromStageName_PatchTenantOverallStage(tenantData *PatchTenantOverallStage) {

	switch tenantData.TenantStageName {
	case INITIAL_ONBOARDING:
		tenantData.TenantStageId = INITIAL_ONBOARDING_STAGE_ID
	case ONBOARDING_DEMO:
		tenantData.TenantStageId = ONBOARDING_DEMO_STAGE_ID
	case TRIAL_SETUP:
		tenantData.TenantStageId = TRIAL_SETUP_STAGE_ID
	case TRIAL_IN_PROG:
		tenantData.TenantStageId = TRIAL_IN_PROG_STAGE_ID
	case TRAIL_DISCONTINUED:
		tenantData.TenantStageId = TRAIL_DISCONTINUED_STAGE_ID
	case PRE_PROVISIONING_CHECKS:
		tenantData.TenantStageId = PRE_PROVISIONING_CHECKS_STAGE_ID
	case PROVISIONING:
		tenantData.TenantStageId = PROVISIONING_STAGE_ID
	case ACTIVE:
		tenantData.TenantStageId = ACTIVE_STAGE_ID
	case INACTIVE:
		tenantData.TenantStageId = INACTIVE_STAGE_ID
	case DEACTIVATED:
		tenantData.TenantStageId = DEACTIVATED_STAGE_ID
	default:
		tenantData.TenantStageId = "STG00"
	}

}

// Decommission Func
func (svc *TenantDetailsSvc) GetTenantDetailsInMap(filter string) (map[string]TenantDetails, error) {
	// 1. Scan the DDB table
	groups := make(map[string]TenantDetails)
	allGroupsDDBOutput := []map[string]dynamodb_types.AttributeValue{}

	scanTableInput := dynamodb.ScanInput{
		TableName: aws.String(svc.TenantDetailsTable),
		Limit:     aws.Int32(100),
	}

	output, err := svc.dynamodbClient.Scan(svc.ctx, &scanTableInput)
	if err != nil {
		svc.logger.Printf("Failed to scan the Groups table. Error: %v", err)
		return map[string]TenantDetails{}, err
	}
	pagination := output.LastEvaluatedKey
	allGroupsDDBOutput = append(allGroupsDDBOutput, output.Items...)

	for pagination != nil {
		next_scan_output, err := svc.dynamodbClient.Scan(svc.ctx, &dynamodb.ScanInput{
			TableName:         aws.String(svc.TenantDetailsTable),
			Limit:             aws.Int32(100),
			ExclusiveStartKey: pagination,
		})
		if err != nil {
			svc.logger.Printf("Failed to scan the Groups table. Error: %v", err)
			return map[string]TenantDetails{}, err
		}
		allGroupsDDBOutput = append(allGroupsDDBOutput, next_scan_output.Items...)
		pagination = next_scan_output.LastEvaluatedKey
	}

	// 2. Store All groups in Map struct

	for _, groupItem := range allGroupsDDBOutput {
		group := TenantDetails{}
		err := dynamodb_attributevalue.UnmarshalMap(groupItem, &group)
		if err != nil {
			svc.logger.Printf("Unable to Unmarshal the DDB output to struct")
			return map[string]TenantDetails{}, err
		}
		groups[group.TenantId] = group

	}

	return groups, nil
}
