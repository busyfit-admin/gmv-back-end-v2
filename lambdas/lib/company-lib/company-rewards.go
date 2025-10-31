package Companylib

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	utils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type RewardsService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	EmployeeRewardRulesTable                               string
	EmployeeRewardRulesTable_RuleStatusIndex               string
	EmployeeRewardRulesTables_RewardRuleLogUpdateDateIndex string
}

func CreateRewardsService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *RewardsService {
	return &RewardsService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

type RewardStatus struct {
	RewardTypeId string `json:"RewardTypeId" dynamodbav:"RewardTypeId"`
	RewardName   string `json:"RewardName" dynamodbav:"RewardName"`
	RewardDesc   string `json:"RewardDesc" dynamodbav:"RewardDesc"`
	Active       bool   `json:"Active" dynamodbav:"Active"`
}

type RewardUnits struct {
	Points       string `json:"Points" dynamodbav:"Points"`
	EqualAmount  string `json:"EqualAmount" dynamodbav:"EqualAmount"`
	CurrencyType string `json:"CurrencyType" dynamodbav:"CurrencyType"`
}

type RewardsRuleDynamodbData struct {
	RuleId   string `json:"RuleId" dynamodbav:"RuleId"`     // pk
	RuleType string `json:"RuleType" dynmoadbav:"RuleType"` // sk

	// Fields only for Reward types settings
	// RuleId : 0000AAAA , RuleType: "RewardTypeStatus"
	RewardTypeStatus map[string]RewardStatus `json:"RewardTypeStatus" dynamodbav:"RewardTypeStatus"`

	// Fields only for Reward Units settings
	// RuleId : 0000BBBB , RuleType: "RewardUnits"
	RewardUnits RewardUnits `json:"RewardUnits" dynamodbav:"RewardUnits"`

	// Fields only for Reward automated rules
	// RuleId : rr-ab23das , RuleType: "RewardRules"
	RuleName string `json:"RuleName" dynamodbav:"RuleName"`
	RuleDesc string `json:"RuleDesc" dynamodbav:"RuleDesc"`

	RuleForImplementation  string `json:"RuleForImplementation" dynamodbav:"RuleForImplementation"`
	RuleRewardPoints       string `json:"RuleRewardAmount" dynamodbav:"RuleRewardPoints"`
	RuleRewardType         string `json:"RuleRewardType" dynamodbav:"RuleRewardType"`
	RuleWhenImplementation string `json:"RuleWhenImplementation" dynamodbav:"RuleWhenImplementation"`

	RuleStartDate string `json:"RuleStartDate" dynamodbav:"RuleStartDate"`
	RuleEndDate   string `json:"RuleEndDate" dynamodbav:"RuleEndDate"`

	RuleStatus      string `json:"RuleStatus" dynamodbav:"RuleStatus"`
	RuleLastUpdated string `json:"RuleLastUpdated" dynamodbav:"RuleLastUpdated"`

	// Fields Only for RewardUpdateLogs
	// RuleId : ru-0000BBBB , RuleType: "RewardUpdateLogs"
	RewardRuleLogData       string `json:"RewardRuleLogData" dynamodbav:"RewardRuleLogData"`
	RewardRuleLogUpdateDate string `json:"RewardRuleLogUpdateDate" dynamodbav:"RewardRuleLogUpdateDate"`
	RewardRuleUpdateBy      string `json:"RewardRuleUpdateBy" dynamodbav:"RewardRuleUpdateBy"`
}

const (
	REWARD_TYPE_General = "RD00"

	REWARD_TYPE_Health          = "RD01"
	REWARD_TYPE_Skills          = "RD02"
	REWARD_TYPE_EmployeeSupport = "RD03"
)

const (
	REWARDPOINTS_INCEPTION = "RewardsInception"
	REWARDS_DEFAULT_ADMIN  = "RewardsAdminUser"
)

func ConvertEmpRewardTypeToRewardNames(empRewardData map[string]EmployeeRewards) map[string]EmployeeRewards {
	rewardNames := make(map[string]EmployeeRewards)

	for key, reward := range empRewardData {
		switch key {
		case "RD00":
			rewardNames["General Rewards"] = reward
		case "RD01":
			rewardNames["Health Rewards"] = reward
		case "RD02":
			rewardNames["Skills Rewards"] = reward
		case "RD03":
			rewardNames["Employee Support Rewards"] = reward
		}
	}

	return rewardNames
}

func ConvertEmpRewardTypeToRewardName(rewardId string) string {
	switch rewardId {
	case "RD00":
		return "General Rewards"
	case "RD01":
		return "Health Rewards"
	case "RD02":
		return "Skills Rewards"
	case "RD03":
		return "Employee Support Rewards"
	default:
		return ""
	}
}

/* The Get Reward Rules will get all the reward settings for the rewards management page.

The expected output of the reward rules as below:


 {
	"TopLevelRewardSettings": {
		"RewardTypeStatus" : {
			"HealthRewards" : {
				"RewardName": "Health Rewards",
				"RewardDesc": "Health Rewards desc",
				"Active": true
			},
			"SkillDevRewards" : {
				"RewardName": "Skill Dev Rewards",
				"RewardDesc": "Skill Dev desc",
				"Active": true
			}
		}
		"RewardUnits": {
			"Points": "10",
			"EqualAmount": "1",
			CurrencyType: "AUD"
		}
	},
	"RewardRules": {
		"Active": {
			{
				"RuleName": "Yearly Rewards",
				"RuleForImplementation": "Everyone",
				"RuleRewardType": "Health Rewards",
				"RuleStartDate": "01-01-24",
				"RuleWhenImplementation": "Yearly",
				"RuleRewardPoints": "100"

			}
		},
		"Draft": {
			{
				"RuleName": "Yearly Rewards",
				"RuleForImplementation": "Everyone",
				"RuleRewardType": "Health Rewards",
				"RuleStartDate": "01-02-24",
				"RuleWhenImplementation": "Yearly",
				"RuleRewardPoints": "200"
			},
		}
	},
	"RewardUpdateLogs": {
		{
			"RewardRuleLogData": "Reward Name: abc has been added with status : Active ",
			"RewardRuleLogUpdateDate": "01-01-2024",
			"RewardRuleUpdateBy": "admin"
		},
	}
 }

*/

const (
	RULE_ID____RewardTypeStatus = "0000AAAA"
	RULE_TYPE__RewardTypeStatus = "RewardTypeStatus"

	RULE_ID____RewardUnits = "0000BBBB"
	RULE_TYPE__RewardUnits = "RewardUnits"

	RULE_ID____RewardRules_example = "rr-0000cccc"
	RULE_TYPE__RewardRules         = "RewardRules"

	RULE_ID____RewardUpdateLogs_example = "ru-0000dddd"
	RULE_TYPE__RewardUpdateLogs         = "RewardUpdateLogs"
)

const (
	REWARD_RULE_STATUS____Active   = "Active"
	REWARD_RULE_STATUS____Draft    = "Draft"
	REWARD_RULE_STATUS____Inactive = "Inactive"
)

type TopLevelRewardSettings struct {
	RewardTypeStatus map[string]RewardStatus `json:"RewardTypeStatus"`
	RewardUnits      RewardUnits             `json:"RewardUnits"`
}

type RuleData struct {
	RuleId   string `json:"RuleId" dynamodbav:"RuleId"`
	RuleName string `json:"RuleName" dynamodbav:"RuleName"`

	RuleForImplementation  string `json:"RuleForImplementation" dynamodbav:"RuleForImplementation"`
	RuleRewardPoints       string `json:"RuleRewardPoints" dynamodbav:"RuleRewardPoints"`
	RuleRewardType         string `json:"RuleRewardType" dynamodbav:"RuleRewardType"`
	RuleWhenImplementation string `json:"RuleWhenImplementation" dynamodbav:"RuleWhenImplementation"`

	RuleStartDate string `json:"RuleStartDate" dynamodbav:"RuleStartDate"`
	RuleEndDate   string `json:"RuleEndDate" dynamodbav:"RuleEndDate"`

	RuleStatus      string `json:"RuleStatus" dynamodbav:"RuleStatus"`
	RuleLastUpdated string `json:"RuleLastUpdated" dynamodbav:"RuleLastUpdated"`
}

type RewardRules struct {
	Active []RuleData `json:"Active"`
	Draft  []RuleData `json:"Draft"`
}

type RewardUpdateLogs struct {
	RewardRuleLogData       string `json:"RewardRuleLogData" dynamodbav:"RewardRuleLogData"`
	RewardRuleLogUpdateDate string `json:"RewardRuleLogUpdateDate" dynamodbav:"RewardRuleLogUpdateDate"`
	RewardRuleUpdateBy      string `json:"RewardRuleUpdateBy" dynamodbav:"RewardRuleUpdateBy"`
}

type GetAllRewardRules struct {
	RewardAdminPoints      EmployeeDynamodbData   `json:"RewardAdminPoints"`
	TopLevelRewardSettings TopLevelRewardSettings `json:"TopLevelRewardSettings"`
	RewardRules            RewardRules            `json:"RewardRules"`
	RewardUpdateLogs       []RewardUpdateLogs     `json:"RewardUpdateLogs"`
}

func (svc *RewardsService) GetAllRewardRules() (GetAllRewardRules, error) {

	var allRewardsRulesData GetAllRewardRules

	// 1. Get Top Level Reward Settings
	topLevelData, err := svc.GetTopLevelRewardSettings()
	if err != nil {
		return GetAllRewardRules{}, err
	}
	allRewardsRulesData.TopLevelRewardSettings = topLevelData

	// 2a. Get Active Reward Rules
	allRewardsRulesData.RewardRules, err = svc.GetRewardRules()
	if err != nil {
		return GetAllRewardRules{}, err
	}
	// 3. Get Reward Update Logs

	updateLogs, err := svc.GetRewardUpdateLogsData()
	if err != nil {
		return GetAllRewardRules{}, err
	}
	allRewardsRulesData.RewardUpdateLogs = updateLogs

	return allRewardsRulesData, nil
}

func (svc *RewardsService) GetTopLevelRewardSettings() (TopLevelRewardSettings, error) {

	topLevelSettingData := TopLevelRewardSettings{
		RewardTypeStatus: map[string]RewardStatus{},
		RewardUnits:      RewardUnits{},
	}

	// 1. Get Reward Type Settings

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.EmployeeRewardRulesTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus}, // PK for Get Reward Type Settings
			"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus}, // SK
		},
	})
	if err != nil {
		svc.logger.Printf("Unable to perform Get Operation on Reward Rules Table")
		return TopLevelRewardSettings{}, err
	}

	var ddbData RewardsRuleDynamodbData
	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &ddbData)
	if err != nil {
		svc.logger.Printf("Unable to Unmarshal the output from Get Operation on Reward Rules Table")
		return TopLevelRewardSettings{}, err
	}

	topLevelSettingData.RewardTypeStatus = ddbData.RewardTypeStatus

	// Set the ddb output data to the Top level Settings Data struct

	// 2. Get Reward Unit Settings
	output, err = svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.EmployeeRewardRulesTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardUnits}, // PK for Get Reward Units
			"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardUnits}, // SK
		},
	})
	if err != nil {
		svc.logger.Printf("Unable to perform Get Operation on Reward Rules Table")
		return TopLevelRewardSettings{}, err
	}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &ddbData)
	if err != nil {
		svc.logger.Printf("Unable to Unmarshal the output from Get Operation on Reward Rules Table")
		return TopLevelRewardSettings{}, err
	}
	topLevelSettingData.RewardUnits = ddbData.RewardUnits

	return topLevelSettingData, nil
}

func (svc *RewardsService) GetRewardRules() (RewardRules, error) {

	var rewardRulesData RewardRules
	// 2a. Get Active Reward Rules
	queryGetActiveRules := "SELECT RuleId, RuleName, RuleForImplementation, RuleRewardPoints, RuleRewardType, RuleWhenImplementation, RuleStartDate, RuleEndDate, RuleStatus, RuleLastUpdated FROM \"" + svc.EmployeeRewardRulesTable + "\".\"" + svc.EmployeeRewardRulesTable_RuleStatusIndex + "\" WHERE RuleStatus = 'Active'"

	activeRewardRules, err := svc.GetRewardRuleData(queryGetActiveRules)
	if err != nil {
		return RewardRules{}, err
	}
	rewardRulesData.Active = activeRewardRules
	// 2b. Get Draft Reward Rules
	queryGetDraftRules := "SELECT RuleId, RuleName, RuleForImplementation, RuleRewardPoints, RuleRewardType, RuleWhenImplementation, RuleStartDate, RuleEndDate, RuleStatus, RuleLastUpdated FROM \"" + svc.EmployeeRewardRulesTable + "\".\"" + svc.EmployeeRewardRulesTable_RuleStatusIndex + "\" WHERE RuleStatus IN ('Draft', 'Inactive')"

	draftRewardRules, err := svc.GetRewardRuleData(queryGetDraftRules)
	if err != nil {
		return RewardRules{}, err
	}
	rewardRulesData.Draft = draftRewardRules

	return rewardRulesData, nil
}

func (svc *RewardsService) GetRewardRuleData(stmt string) ([]RuleData, error) {

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(stmt),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []RuleData{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for Rule Data Query %s", stmt)
		return []RuleData{}, nil
	}

	allRuleData := []RuleData{}
	for _, stageItem := range output.Items {
		ruleData := RuleData{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &ruleData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal Rule data. Failed with  Error : %v", err)
			return []RuleData{}, err
		}

		// Append data to the overall rule data
		allRuleData = append(allRuleData, ruleData)
	}

	return allRuleData, nil
}

func (svc *RewardsService) GetRewardUpdateLogsData() ([]RewardUpdateLogs, error) {

	queryLatestUpdateLogs := "SELECT RewardRuleLogData, RewardRuleLogUpdateDate, RewardRuleUpdateBy FROM \"" + svc.EmployeeRewardRulesTable + "\".\"" + svc.EmployeeRewardRulesTables_RewardRuleLogUpdateDateIndex + "\" WHERE RuleType = 'RewardUpdateLogs' ORDER BY RewardRuleLogUpdateDate"

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(queryLatestUpdateLogs),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []RewardUpdateLogs{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for RewardUpdateLogs Data Query %s", queryLatestUpdateLogs)
		return []RewardUpdateLogs{}, nil
	}

	allUpdateLogData := []RewardUpdateLogs{}
	for _, stageItem := range output.Items {
		logData := RewardUpdateLogs{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &logData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal Reward Rules Update Log data. Failed with  Error : %v", err)
			return []RewardUpdateLogs{}, err
		}

		// Append data to the overall rule data
		allUpdateLogData = append(allUpdateLogData, logData)
	}

	return allUpdateLogData, nil
}

func (svc *RewardsService) GetRulesByRuleId(RuleId string) (RewardsRuleDynamodbData, error) {

	getItemInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RuleId},
			"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardRules},
		},
		TableName:      aws.String(svc.EmployeeRewardRulesTable),
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)

	if err != nil {
		svc.logger.Printf("Query on Employee data failed with error :%v", err)
		return RewardsRuleDynamodbData{}, err
	}

	RewardRuleData := RewardsRuleDynamodbData{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &RewardRuleData)
	if err != nil {
		svc.logger.Printf("Query on Employee data Unmarshal failed with error :%v", err)
		return RewardsRuleDynamodbData{}, err
	}

	return RewardRuleData, nil
}

// ---------------- Handle Create Reward Rules Functions ----

type CreateRewardRuleInput struct {
	RuleName string `json:"RuleName"`
	RuleDesc string `json:"RuleDesc"`

	RuleForImplementation  string `json:"RuleForImplementation"`
	RuleRewardPoints       string `json:"RuleRewardPoints"`
	RuleRewardType         string `json:"RuleRewardType"`
	RuleWhenImplementation string `json:"RuleWhenImplementation"`

	RuleStartDate string `json:"RuleStartDate"`
	RuleEndDate   string `json:"RuleEndDate"`

	RuleStatus string `json:"RuleStatus"`
}

func (svc *RewardsService) CreateRewardsRule(rewardsRuleData CreateRewardRuleInput) error {

	RuleId := "rr-" + utils.GenerateRandomString(12)

	ruleItem := map[string]dynamodb_types.AttributeValue{
		"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RuleId},
		"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardRules},

		"RuleName": &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleName},
		"RuleDesc": &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleDesc},

		"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleForImplementation},
		"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleRewardPoints},
		"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleRewardType},
		"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleWhenImplementation},

		"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleStartDate},
		"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleEndDate},

		"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: rewardsRuleData.RuleStatus},
		"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: utils.GenerateTimestamp()},
	}

	putItemInput := dynamodb.PutItemInput{
		Item:      ruleItem,
		TableName: aws.String(svc.EmployeeRewardRulesTable),
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput)
	if err != nil {
		svc.logger.Printf("Reward Rule PutItem failed with error :%v", err)
		return err
	}
	svc.logger.Print("Put item success")

	// Update the logs
	svc.PutRewardRuleUpdateLogs(PutRewardRuleUpdateLogs{
		UpdateType:     UPDATE_TYPE_Create,
		RewardRuleType: RULE_TYPE__RewardRules,
		RuleId:         RuleId,
		UpdatedBy:      "Admin", // TBC once we implement role based access
	})

	return nil
}

// ----------- Handle Patch Requests for RewardType Status ---------
type RewardInput struct {
	RewardType     string `json:"RewardType"`
	IsRewardActive bool   `json:"IsRewardActive"`
}
type PatchRewardTypesStatusInput struct {
	AllRewardInput []RewardInput `json:"AllRewardInput"`
	UpdateBy       string        `json:"UpdateBy"`
}

func (svc *RewardsService) PatchRewardTypesStatus(patchInputData PatchRewardTypesStatusInput) error {

	for _, patchData := range patchInputData.AllRewardInput {

		_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
			TableName: aws.String(svc.EmployeeRewardRulesTable),
			Key: map[string]dynamodb_types.AttributeValue{
				"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus},
				"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus},
			},
			ExpressionAttributeNames: map[string]string{
				"#RewardTypeId": patchData.RewardType,
			},
			UpdateExpression: aws.String("SET RewardTypeStatus.#RewardTypeId.Active = :Active"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":Active": &dynamodb_types.AttributeValueMemberBOOL{Value: patchData.IsRewardActive},
			},
			ReturnValues: dynamodb_types.ReturnValueNone,
		})
		if err != nil {
			svc.logger.Printf("Error Updating the Reward Type status . Failed with error : %v", err)
			return err
		}
	}

	svc.PutRewardRuleUpdateLogs(PutRewardRuleUpdateLogs{
		UpdateType:     UPDATE_TYPE_Update,
		RewardRuleType: RULE_TYPE__RewardTypeStatus,
		RuleId:         RULE_ID____RewardTypeStatus,
		UpdatedBy:      "Admin", // TBC once we implement role based access
	})

	return nil
}

// ----------- Handle Patch Requests for Reward Units ---------

func (svc *RewardsService) PatchRewardUnits(patchInputData RewardUnits) error {

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.EmployeeRewardRulesTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardUnits},
			"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardUnits},
		},
		UpdateExpression: aws.String("SET RewardUnits = :RewardUnits"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":RewardUnits": &dynamodb_types.AttributeValueMemberM{
				Value: map[string]dynamodb_types.AttributeValue{
					"Points":       &dynamodb_types.AttributeValueMemberS{Value: patchInputData.Points},
					"EqualAmount":  &dynamodb_types.AttributeValueMemberS{Value: patchInputData.EqualAmount},
					"CurrencyType": &dynamodb_types.AttributeValueMemberS{Value: patchInputData.CurrencyType},
				},
			},
		},
		ReturnValues: dynamodb_types.ReturnValueNone,
	})

	if err != nil {
		svc.logger.Printf("Failed to Update the Reward Units . Error: %v ", err)
		return err
	}

	svc.PutRewardRuleUpdateLogs(PutRewardRuleUpdateLogs{
		UpdateType:     UPDATE_TYPE_Update,
		RewardRuleType: RULE_TYPE__RewardUnits,
		RuleId:         RULE_ID____RewardUnits,
		UpdatedBy:      "Admin", // TBC once we implement role based access
	})

	return nil
}

// ----------- Handle Patch Requests for Reward Rules ---------

type RewardRulesPatchInput struct {
	RuleId string `json:"RuleId"` // Rule is provided from Front end as its an update request

	RuleName string `json:"RuleName"`
	RuleDesc string `json:"RuleDesc"`

	RuleForImplementation  string `json:"RuleForImplementation"`
	RuleRewardPoints       string `json:"RuleRewardAmount"`
	RuleRewardType         string `json:"RuleRewardType"`
	RuleWhenImplementation string `json:"RuleWhenImplementation"`

	RuleStartDate string `json:"RuleStartDate"`
	RuleEndDate   string `json:"RuleEndDate"`

	RuleStatus string `json:"RuleStatus"`
}

func (svc *RewardsService) PatchRewardRules(patchInputData RewardRulesPatchInput) error {

	ruleItem := map[string]dynamodb_types.AttributeValue{
		"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleId},
		"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardRules},

		"RuleName": &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleName},
		"RuleDesc": &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleDesc},

		"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleForImplementation},
		"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleRewardPoints},
		"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleRewardType},
		"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleWhenImplementation},

		"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleStartDate},
		"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleEndDate},

		"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: patchInputData.RuleStatus},
		"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: utils.GenerateTimestamp()},
	}

	putItemInput := dynamodb.PutItemInput{
		Item:      ruleItem,
		TableName: aws.String(svc.EmployeeRewardRulesTable),
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput)
	if err != nil {
		svc.logger.Printf("Reward Rule PutItem failed with error :%v", err)
		return err
	}

	// Update the logs
	svc.PutRewardRuleUpdateLogs(PutRewardRuleUpdateLogs{
		UpdateType:     UPDATE_TYPE_Update,
		RewardRuleType: RULE_TYPE__RewardRules,
		RuleId:         patchInputData.RuleId,
		UpdatedBy:      "Admin", // TBC once we implement role based access
	})

	return nil
}

// -------- Delete Reward Rules
func (svc *RewardsService) DeleteRuleByRuleId(RuleId string) error {

	deleteItemInput := dynamodb.DeleteItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RuleId},
			"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardRules},
		},
		TableName: aws.String(svc.EmployeeRewardRulesTable),
	}

	_, err := svc.dynamodbClient.DeleteItem(svc.ctx, &deleteItemInput)

	if err != nil {
		svc.logger.Printf("Delete on Reward Rule Table failed with error :%v", err)
		return err
	}

	svc.PutRewardRuleUpdateLogs(PutRewardRuleUpdateLogs{
		UpdateType:     UPDATE_TYPE_Delete,
		RewardRuleType: RULE_TYPE__RewardRules,
		RuleId:         RuleId,
		UpdatedBy:      "Admin", // TBC once we implement role based access
	})

	return nil
}

// --------------------- Reward Rule Update Logs ---------

const (
	UPDATE_TYPE_Create = "Created"
	UPDATE_TYPE_Update = "Updated"
	UPDATE_TYPE_Delete = "Deleted"
)

type PutRewardRuleUpdateLogs struct {
	UpdateType     string // Can take Values like "Created", "Updated", "Deleted"
	RewardRuleType string // Can take Values "RewardTypeStatus" , "RewardUnits", "RewardRules"
	RuleId         string // Unique ID of the Rule

	UpdatedBy string // ID of person who has updated

}

func (svc *RewardsService) PutRewardRuleUpdateLogs(updateData PutRewardRuleUpdateLogs) error {

	rewardUpdateId := "ru-" + utils.GenerateRandomString(12)

	LogData := fmt.Sprintf("A Reward Rule was %s of RewardType: %s having RuleId: %s", updateData.UpdateType, updateData.RewardRuleType, updateData.RuleId)

	putItemInput := dynamodb.PutItemInput{
		TableName: aws.String(svc.EmployeeRewardRulesTable),
		Item: map[string]dynamodb_types.AttributeValue{
			"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: rewardUpdateId},
			"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardUpdateLogs},

			"RewardRuleLogData":       &dynamodb_types.AttributeValueMemberS{Value: LogData},
			"RewardRuleLogUpdateDate": &dynamodb_types.AttributeValueMemberS{Value: utils.GenerateTimestamp()},
			"RewardRuleUpdateBy":      &dynamodb_types.AttributeValueMemberS{Value: updateData.UpdatedBy},
		},
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &putItemInput)
	if err != nil {
		svc.logger.Printf("Failed to Put the Reward Update log. Error : %v", err)
		return err
	}

	return nil
}
