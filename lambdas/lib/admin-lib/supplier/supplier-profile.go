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

type SupplierDetailsTable struct {
	SupplierId   string `dynamodbav:"SupplierId" json:"SupplierId"`     // PK . To be generated during Supplier Creation
	SupplierName string `dynamodbav:"SupplierName" json:"SupplierName"` // Name of the Supplier joining the program

	SupplierDesc string `dynamodbav:"SupplierDesc" json:"SupplierDesc"`
	Industry     string `dynamodbav:"Industry" json:"Industry"`

	SupplierContacts map[string]SupplierContacts `dynamodbav:"SupplierContacts" json:"SupplierContacts"` // Map of all supplier Contacts. At-least one is required.

	SupplierCreationDate string `dynamodbav:"SupplierCreationDate" json:"SupplierCreationDate"`
	SupplierStageId      string `dynamodbav:"SupplierStageId"` // Can take values as per Supplier Lifecycle stages
	SupplierStageName    string `dynamodbav:"SupplierStageName" json:"SupplierStageName"`
}

type SupplierContacts struct {
	SupplierEmail string `dynamodbav:"SupplierEmail" json:"SupplierEmail"` // should be unique

	SupplierContactName string `dynamodbav:"SupplierContactName" json:"SupplierContactName"`
	SupplierPh          string `dynamodbav:"SupplierPh" json:"SupplierPh"`

	IsPrimary bool `dynamodbav:"IsPrimary" json:"IsPrimary"`
}

// -------------------- Supplier Profile Functions -------------

type SupplierDetailsSvc struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	SupplierDetailsTable                string
	SupplierDetails_SupplierStatusIndex string
}

func CreateSupplierDetailsSvc(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *SupplierDetailsSvc {
	return &SupplierDetailsSvc{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

// -------------------- Supplier Get Profile(s) API related functions -------------

// Accepts (SupplierId) and returns all SupplierDetailsTable data
func (svc *SupplierDetailsSvc) GetSupplierProfileById(SupplierId string) (SupplierDetailsTable, error) {

	getItemInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: SupplierId},
		},
		TableName:      aws.String(svc.SupplierDetailsTable),
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)
	if err != nil {
		svc.logger.Printf("Get Supplier Failed with error :%v", err)
		return SupplierDetailsTable{}, err
	}

	SupplierData := SupplierDetailsTable{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &SupplierData)
	if err != nil {
		svc.logger.Printf("Get Supplier Unmarshal failed with error :%v", err)
		return SupplierDetailsTable{}, err
	}

	SetStageIdFromStageName_GetSupplierData(&SupplierData)

	return SupplierData, nil
}

// Required Output for Listing Supplier data in Admin List Suppliers page
type SupplierDetails struct {
	SupplierId   string `dynamodbav:"SupplierId" json:"SupplierId"`
	SupplierName string `dynamodbav:"SupplierName" json:"SupplierName"`

	Industry string `dynamodbav:"Industry" json:"Industry"`

	SupplierCreationDate string `dynamodbav:"SupplierCreationDate" json:"SupplierCreationDate"`
	SupplierStageId      string `dynamodbav:"SupplierStageId" ` // Can take values as per Supplier Lifecycle stages
	SupplierStageName    string `dynamodbav:"SupplierStageName" json:"SupplierStageName"`
}
type ListInProgActiveSuppliers struct {
	OnboardingInProg []SupplierDetails `json:"OnboardingInProg"`
	Active           []SupplierDetails `json:"Active"`
}

func (svc *SupplierDetailsSvc) GetAllSupplierDetails() (ListInProgActiveSuppliers, error) {

	var AllSuppliersData ListInProgActiveSuppliers
	// 1) Query Supplier Details table to find all Onboarding InProg Suppliers
	queryGetInProgSuppliers := "SELECT SupplierId, SupplierName, Industry, SupplierCreationDate, SupplierStageId FROM \"" + svc.SupplierDetailsTable + "\".\"" + svc.SupplierDetails_SupplierStatusIndex + "\" WHERE SupplierStageId IN ('STG01', 'STG02','STG03','STG04','STG05','STG06','STG07') ORDER BY SupplierStageId ASC"

	OnboardingSuppliersDetails, err := svc.ExecuteQueryDDB(queryGetInProgSuppliers)
	if err != nil {
		return ListInProgActiveSuppliers{}, err
	}
	AllSuppliersData.OnboardingInProg = OnboardingSuppliersDetails

	// 2) Query Supplier Details table to find all Active Suppliers

	queryGetActiveSuppliers := "SELECT SupplierId, SupplierName, Industry, SupplierCreationDate, SupplierStageId FROM \"" + svc.SupplierDetailsTable + "\".\"" + svc.SupplierDetails_SupplierStatusIndex + "\" WHERE SupplierStageId = 'STG08'"

	ActiveSuppliersDetails, err := svc.ExecuteQueryDDB(queryGetActiveSuppliers)
	if err != nil {
		return ListInProgActiveSuppliers{}, err
	}
	AllSuppliersData.Active = ActiveSuppliersDetails

	return AllSuppliersData, nil
}

func (svc *SupplierDetailsSvc) ExecuteQueryDDB(stmt string) ([]SupplierDetails, error) {

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(stmt),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []SupplierDetails{}, err
	}

	// Logging read units stats
	svc.logger.Printf("[DDB_USAGE_STATS] Read Units : %v", output.ConsumedCapacity)

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for Onboarding InProg Suppliers")
		return []SupplierDetails{}, nil
	}

	allSupplierDetailsData := []SupplierDetails{}
	for _, stageItem := range output.Items {
		supplierData := SupplierDetails{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &supplierData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal supplier Details data  Error : %v", err)
			return []SupplierDetails{}, err
		}
		// Set SupplierStage Name
		SetStageIdFromStageName(&supplierData)
		// Append data to the overall supplier data
		allSupplierDetailsData = append(allSupplierDetailsData, supplierData)
	}

	return allSupplierDetailsData, nil

}

// -------------------- Supplier POST and PATCH Profile(s) API related functions -------------

/*
 Types of POST and PATCH Requests handled by manage-supplier-profiles

 POST :
  1. Create the Supplier Profile ( This is the first step in onboarding a Supplier )
	  - Should consist of all required details from the Onboarding Supplier Profile page.

	  NOTE: Once a Supplier is created, we cannot update the SupplierID.
	        If Supplier ID needs to be changed, we'll need to create another Supplier from start.


  PATCH API options:

  1. Update Name, Desc, Industry - Top level Information
  2. Add new Contact ID - Add new contactID in the SupplierContacts ( currently not taking contactID delete option)
  3. Update the SupplierStage to next Level

*/

// Required Fields from the Supplier Onboarding Page
type CreateSupplierProfile struct {
	SupplierName string `json:"SupplierName"`
	SupplierDesc string `json:"SupplierDesc"`
	Industry     string `json:"Industry"`

	EnvType string `json:"EnvType"`

	PrimaryContactName  string `json:"PrimaryContactName"`
	PrimaryContactEmail string `json:"PrimaryContactEmail"`
	PrimaryContactPh    string `json:"PrimaryContactPh"`
}

func (svc *SupplierDetailsSvc) CreateSupplierProfile(supplierData CreateSupplierProfile) error {

	//1.  Generate Unique ID for Supplier
	newSupplierUUID := utils.GenerateRandomString(10)

	//2. Default Starting Stage of the Supplier is STG01 (Onboarding) + Create CreationDate
	defaultStartStage := INITIAL_ONBOARDING_STAGE_ID
	creationDate := utils.GenerateTimestamp()

	//3. Validate if all required fields are present ( SupplierName, PrimaryContact as Min requirement )
	if supplierData.SupplierName == "" || supplierData.PrimaryContactName == "" || supplierData.PrimaryContactEmail == "" || supplierData.PrimaryContactPh == "" {
		return fmt.Errorf("required fields for creation of supplier are missing")
	}

	//4. Use UpdateItem to ensure SupplierID's are not overwritten
	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.SupplierDetailsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: newSupplierUUID},
		},
		ConditionExpression: aws.String("attribute_not_exists(SupplierContacts)"),
		UpdateExpression:    aws.String("SET SupplierName = :SupplierName, SupplierDesc = :SupplierDesc, Industry = :Industry, EnvType = :EnvType, SupplierContacts = :SupplierContacts, SupplierCreationDate = :SupplierCreationDate, SupplierStageId = :SupplierStageId"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":SupplierName": &dynamodb_types.AttributeValueMemberS{Value: supplierData.SupplierName},
			":SupplierDesc": &dynamodb_types.AttributeValueMemberS{Value: supplierData.SupplierDesc},
			":Industry":     &dynamodb_types.AttributeValueMemberS{Value: supplierData.Industry},
			":SupplierContacts": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					supplierData.PrimaryContactEmail: &dynamodb_types.AttributeValueMemberM{
						Value: map[string]dynamodb_types.AttributeValue{
							"SupplierEmail":       &dynamodb_types.AttributeValueMemberS{Value: supplierData.PrimaryContactEmail},
							"SupplierContactName": &dynamodb_types.AttributeValueMemberS{Value: supplierData.PrimaryContactName},
							"SupplierPh":          &dynamodb_types.AttributeValueMemberS{Value: supplierData.PrimaryContactPh},
							"IsPrimary":           &dynamodb_types.AttributeValueMemberBOOL{Value: true},
						},
					},
				}},
			":EnvType":              &dynamodb_types.AttributeValueMemberS{Value: supplierData.EnvType},
			":SupplierCreationDate": &dynamodb_types.AttributeValueMemberS{Value: creationDate},
			":SupplierStageId":      &dynamodb_types.AttributeValueMemberS{Value: defaultStartStage},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Update the Supplier Table :%v", err)
		return err
	}

	//5. Necessary Logging in cloudwatch logs
	svc.logger.Printf("[LOGGER] New Supplier has been added successfully. New SupplierID: %s ", newSupplierUUID)

	//6. Return Success
	return nil
}

// PATCH API - Update Name, Desc, Industry, EnvType - Top level Information
type PatchTopLevelInfo struct {
	SupplierId string `json:"SupplierId"`

	SupplierName string `json:"SupplierName"`
	SupplierDesc string `json:"SupplierDesc"`
	Industry     string `json:"Industry"`
	EnvType      string `json:"EnvType"`
}

func (svc *SupplierDetailsSvc) PatchTopLevelInfo(TopLevelInfo PatchTopLevelInfo) error {

	// 1. Ensure Name , EnvType is not empty string in incoming Update set
	if TopLevelInfo.SupplierName == "" || TopLevelInfo.EnvType == "" || TopLevelInfo.SupplierId == "" {
		return fmt.Errorf("Supplier name or env type cannot be empty for update set")
	}

	ddbUpdateItem := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.SupplierDetailsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.SupplierId},
		},
		UpdateExpression:    aws.String("SET SupplierName = :SupplierName, SupplierDesc = :SupplierDesc, Industry = :Industry, EnvType = :EnvType"),
		ConditionExpression: aws.String("attribute_exists(SupplierId)"), // This will prevent new Supplier creation when performing updates
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":SupplierName": &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.SupplierName},
			":SupplierDesc": &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.SupplierDesc},
			":Industry":     &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.Industry},
			":EnvType":      &dynamodb_types.AttributeValueMemberS{Value: TopLevelInfo.EnvType},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &ddbUpdateItem)
	if err != nil {
		svc.logger.Printf("Failed to Update the Supplier Table :%v", err)
		return err
	}

	svc.logger.Printf("[LOGGER] Top level Supplier details have been updated successfully. SupplierID: %s ", TopLevelInfo.SupplierId)

	return nil
}

// PATCH API - Add new ContactId in the Supplier Contacts
type PatchSupplierContacts struct {
	SupplierId string `json:"SupplierId"`

	ContactName  string `json:"ContactName"`
	ContactEmail string `json:"ContactEmail"`
	ContactPh    string `json:"ContactPh"`
	IsPrimary    bool   `json:"IsPrimary"`
}

func (svc *SupplierDetailsSvc) PatchSupplierContacts(contactInfo PatchSupplierContacts) error {

	// Check if the values provided are not null
	if contactInfo.SupplierId == "" || contactInfo.ContactName == "" || contactInfo.ContactEmail == "" {
		return fmt.Errorf("contact info cannot be empty. provide mandate information")
	}

	ddbInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.SupplierDetailsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: contactInfo.SupplierId},
		},
		UpdateExpression: aws.String("SET SupplierContacts.#ContactId = :Contact"),
		ExpressionAttributeNames: map[string]string{
			"#ContactId": contactInfo.ContactEmail,
		},
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":Contact": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					"SupplierEmail":       &dynamodb_types.AttributeValueMemberS{Value: contactInfo.ContactEmail},
					"SupplierContactName": &dynamodb_types.AttributeValueMemberS{Value: contactInfo.ContactName},
					"SupplierPh":          &dynamodb_types.AttributeValueMemberS{Value: contactInfo.ContactPh},
					"IsPrimary":           &dynamodb_types.AttributeValueMemberBOOL{Value: contactInfo.IsPrimary},
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
type PatchSupplierOverallStage struct {
	SupplierId string `json:"SupplierId"`

	SupplierStageId   string
	SupplierStageName string `json:"SupplierStageName"`
}

func (svc *SupplierDetailsSvc) PatchOverallStageId(patchStageInfo PatchSupplierOverallStage) error {

	// Check if Supplier Stage name is not empty:
	if patchStageInfo.SupplierStageName == "" || patchStageInfo.SupplierId == "" {
		return fmt.Errorf("Supplier Id or StageName cannot be empty for update set")
	}

	SetStageIdFromStageName_PatchSupplierOverallStage(&patchStageInfo)

	ddbUpdateItem := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.SupplierDetailsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"SupplierId": &dynamodb_types.AttributeValueMemberS{Value: patchStageInfo.SupplierId},
		},
		UpdateExpression:    aws.String("SET SupplierStageId = :SupplierStageId"),
		ConditionExpression: aws.String("attribute_exists(SupplierId)"), // This will prevent new Supplier creation when performing updates
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":SupplierStageId": &dynamodb_types.AttributeValueMemberS{Value: patchStageInfo.SupplierStageId},
		},
		ReturnValues: dynamodb_types.ReturnValueAllNew,
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &ddbUpdateItem)
	if err != nil {
		svc.logger.Printf("Failed to Update the Supplier Table :%v", err)
		return err
	}

	return nil
}

// -------------------- Supplier APIs helper functions  -------------

func SetStageIdFromStageName(supplierData *SupplierDetails) {

	switch supplierData.SupplierStageId {
	case INITIAL_ONBOARDING_STAGE_ID:
		supplierData.SupplierStageName = INITIAL_ONBOARDING
	case ONBOARDING_DEMO_STAGE_ID:
		supplierData.SupplierStageName = ONBOARDING_DEMO
	case TRIAL_SETUP_STAGE_ID:
		supplierData.SupplierStageName = TRIAL_SETUP
	case TRIAL_IN_PROG_STAGE_ID:
		supplierData.SupplierStageName = TRIAL_IN_PROG
	case TRAIL_DISCONTINUED_STAGE_ID:
		supplierData.SupplierStageName = TRAIL_DISCONTINUED
	case PRE_PROVISIONING_CHECKS_STAGE_ID:
		supplierData.SupplierStageName = PRE_PROVISIONING_CHECKS
	case PROVISIONING_STAGE_ID:
		supplierData.SupplierStageName = PROVISIONING
	case ACTIVE_STAGE_ID:
		supplierData.SupplierStageName = ACTIVE
	case INACTIVE_STAGE_ID:
		supplierData.SupplierStageName = INACTIVE
	case DEACTIVATED_STAGE_ID:
		supplierData.SupplierStageName = DEACTIVATED
	default:
		supplierData.SupplierName = "UNDEFINED"
	}

}

func SetStageIdFromStageName_GetSupplierData(supplierData *SupplierDetailsTable) {

	switch supplierData.SupplierStageId {
	case INITIAL_ONBOARDING_STAGE_ID:
		supplierData.SupplierStageName = INITIAL_ONBOARDING
	case ONBOARDING_DEMO_STAGE_ID:
		supplierData.SupplierStageName = ONBOARDING_DEMO
	case TRIAL_SETUP_STAGE_ID:
		supplierData.SupplierStageName = TRIAL_SETUP
	case TRIAL_IN_PROG_STAGE_ID:
		supplierData.SupplierStageName = TRIAL_IN_PROG
	case TRAIL_DISCONTINUED_STAGE_ID:
		supplierData.SupplierStageName = TRAIL_DISCONTINUED
	case PRE_PROVISIONING_CHECKS_STAGE_ID:
		supplierData.SupplierStageName = PRE_PROVISIONING_CHECKS
	case PROVISIONING_STAGE_ID:
		supplierData.SupplierStageName = PROVISIONING
	case ACTIVE_STAGE_ID:
		supplierData.SupplierStageName = ACTIVE
	case INACTIVE_STAGE_ID:
		supplierData.SupplierStageName = INACTIVE
	case DEACTIVATED_STAGE_ID:
		supplierData.SupplierStageName = DEACTIVATED
	default:
		supplierData.SupplierName = "UNDEFINED"
	}

}

func SetStageIdFromStageName_PatchSupplierOverallStage(supplierData *PatchSupplierOverallStage) {

	switch supplierData.SupplierStageName {
	case INITIAL_ONBOARDING:
		supplierData.SupplierStageId = INITIAL_ONBOARDING_STAGE_ID
	case ONBOARDING_DEMO:
		supplierData.SupplierStageId = ONBOARDING_DEMO_STAGE_ID
	case TRIAL_SETUP:
		supplierData.SupplierStageId = TRIAL_SETUP_STAGE_ID
	case TRIAL_IN_PROG:
		supplierData.SupplierStageId = TRIAL_IN_PROG_STAGE_ID
	case TRAIL_DISCONTINUED:
		supplierData.SupplierStageId = TRAIL_DISCONTINUED_STAGE_ID
	case PRE_PROVISIONING_CHECKS:
		supplierData.SupplierStageId = PRE_PROVISIONING_CHECKS_STAGE_ID
	case PROVISIONING:
		supplierData.SupplierStageId = PROVISIONING_STAGE_ID
	case ACTIVE:
		supplierData.SupplierStageId = ACTIVE_STAGE_ID
	case INACTIVE:
		supplierData.SupplierStageId = INACTIVE_STAGE_ID
	case DEACTIVATED:
		supplierData.SupplierStageId = DEACTIVATED_STAGE_ID
	default:
		supplierData.SupplierStageId = "STG00"
	}

}

// Decommission Func
func (svc *SupplierDetailsSvc) GetSupplierDetailsInMap(filter string) (map[string]SupplierDetails, error) {
	// 1. Scan the DDB table
	groups := make(map[string]SupplierDetails)
	allGroupsDDBOutput := []map[string]dynamodb_types.AttributeValue{}

	scanTableInput := dynamodb.ScanInput{
		TableName: aws.String(svc.SupplierDetailsTable),
		Limit:     aws.Int32(100),
	}

	output, err := svc.dynamodbClient.Scan(svc.ctx, &scanTableInput)
	if err != nil {
		svc.logger.Printf("Failed to scan the Groups table. Error: %v", err)
		return map[string]SupplierDetails{}, err
	}
	pagination := output.LastEvaluatedKey
	allGroupsDDBOutput = append(allGroupsDDBOutput, output.Items...)

	for pagination != nil {
		next_scan_output, err := svc.dynamodbClient.Scan(svc.ctx, &dynamodb.ScanInput{
			TableName:         aws.String(svc.SupplierDetailsTable),
			Limit:             aws.Int32(100),
			ExclusiveStartKey: pagination,
		})
		if err != nil {
			svc.logger.Printf("Failed to scan the Groups table. Error: %v", err)
			return map[string]SupplierDetails{}, err
		}
		allGroupsDDBOutput = append(allGroupsDDBOutput, next_scan_output.Items...)
		pagination = next_scan_output.LastEvaluatedKey
	}

	// 2. Store All groups in Map struct

	for _, groupItem := range allGroupsDDBOutput {
		group := SupplierDetails{}
		err := dynamodb_attributevalue.UnmarshalMap(groupItem, &group)
		if err != nil {
			svc.logger.Printf("Unable to Unmarshal the DDB output to struct")
			return map[string]SupplierDetails{}, err
		}
		groups[group.SupplierId] = group

	}

	return groups, nil
}
