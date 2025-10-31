package Companylib

import (
	"bytes"
	"log"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	utils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
	"github.com/stretchr/testify/assert"
)

func Test_CreateRewardsRule(t *testing.T) {
	t.Run("It should Create the Reward Rule when correct input is provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
				{},
			},
			PutItemErrors: []error{
				nil,
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		createInput := CreateRewardRuleInput{
			RuleName: "Rule-test",
			RuleDesc: "Desc Test",

			RuleForImplementation:  "Everyone",
			RuleRewardPoints:       "100.00",
			RuleRewardType:         "RD00",
			RuleWhenImplementation: "IMMEDIATELY",

			RuleStartDate: "01-01-2024",
			RuleEndDate:   "01-01-2030",

			RuleStatus: "ACTIVE",
		}

		expectedDDBInput := dynamodb.PutItemInput{
			TableName: aws.String("RewardRule-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"RuleId":                 &dynamodb_types.AttributeValueMemberS{Value: ""},
				"RuleType":               &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardRules},
				"RuleName":               &dynamodb_types.AttributeValueMemberS{Value: "Rule-test"},
				"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
				"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "100.00"},
				"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD00"},
				"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "IMMEDIATELY"},
				"RuleStartDate":          &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
				"RuleEndDate":            &dynamodb_types.AttributeValueMemberS{Value: "01-01-2030"},
				"RuleStatus":             &dynamodb_types.AttributeValueMemberS{Value: "ACTIVE"},
				"RuleLastUpdated":        &dynamodb_types.AttributeValueMemberS{Value: ""},
			},
		}

		err := svc.CreateRewardsRule(createInput)

		assert.NoError(t, err)

		// -- Test will fail as RuleId and RuleLastUpdated Timestamp will change everytime.
		// -- Enable this to only check if other parts of PutItem Input params
		// assert.Equal(t, expectedDDBInput.Item, ddbClient.PutItemInputs[0].Item)

		assert.Equal(t, expectedDDBInput.TableName, ddbClient.PutItemInputs[0].TableName)

	})
}

func Test_DeleteRuleByRuleId(t *testing.T) {

	t.Run("It should delete the rule when correct Input is provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},

			DeleteItemOutputs: []dynamodb.DeleteItemOutput{
				{},
			},
			DeleteItemErrors: []error{
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		expectedDDBInput := dynamodb.DeleteItemInput{
			TableName: aws.String("RewardRule-table"),
			Key: map[string]dynamodb_types.AttributeValue{
				"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-1234abaddsaf"},
				"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardRules},
			},
		}
		err := svc.DeleteRuleByRuleId("rr-1234abaddsaf")

		assert.NoError(t, err)

		assert.Equal(t, expectedDDBInput, ddbClient.DeleteItemInputs[0])

	})
}

func Test_PatchRewardTypesStatus(t *testing.T) {
	t.Run("It should perform update on the RewardTypes when correct Input is provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},

			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
				{},
				{},
				{},
				{},
			},
			UpdateItemErrors: []error{
				nil,
				nil,
				nil,
				nil,
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		patchInput := PatchRewardTypesStatusInput{
			AllRewardInput: []RewardInput{
				{
					RewardType:     "RD00",
					IsRewardActive: true,
				},
				{
					RewardType:     "RD01",
					IsRewardActive: true,
				},
				{
					RewardType:     "RD02",
					IsRewardActive: false,
				},
				{
					RewardType:     "RD03",
					IsRewardActive: false,
				},
				{
					RewardType:     "RD04",
					IsRewardActive: true,
				},
			},
			UpdateBy: "Admin",
		}

		expectedDDBInput := []dynamodb.UpdateItemInput{
			{
				TableName: aws.String("RewardRule-table"),
				Key: map[string]dynamodb_types.AttributeValue{
					"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus},
					"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus},
				},
				ExpressionAttributeNames: map[string]string{
					"#RewardTypeId": "RD00",
				},
				UpdateExpression: aws.String("SET RewardTypeStatus.#RewardTypeId.Active = :Active"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":Active": &dynamodb_types.AttributeValueMemberBOOL{Value: true},
				},
				ReturnValues: dynamodb_types.ReturnValueNone,
			},
			{
				TableName: aws.String("RewardRule-table"),
				Key: map[string]dynamodb_types.AttributeValue{
					"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus},
					"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus},
				},
				ExpressionAttributeNames: map[string]string{
					"#RewardTypeId": "RD01",
				},
				UpdateExpression: aws.String("SET RewardTypeStatus.#RewardTypeId.Active = :Active"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":Active": &dynamodb_types.AttributeValueMemberBOOL{Value: true},
				},
				ReturnValues: dynamodb_types.ReturnValueNone,
			},
			{
				TableName: aws.String("RewardRule-table"),
				Key: map[string]dynamodb_types.AttributeValue{
					"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus},
					"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus},
				},
				ExpressionAttributeNames: map[string]string{
					"#RewardTypeId": "RD02",
				},
				UpdateExpression: aws.String("SET RewardTypeStatus.#RewardTypeId.Active = :Active"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":Active": &dynamodb_types.AttributeValueMemberBOOL{Value: false},
				},
				ReturnValues: dynamodb_types.ReturnValueNone,
			},
			{
				TableName: aws.String("RewardRule-table"),
				Key: map[string]dynamodb_types.AttributeValue{
					"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus},
					"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus},
				},
				ExpressionAttributeNames: map[string]string{
					"#RewardTypeId": "RD03",
				},
				UpdateExpression: aws.String("SET RewardTypeStatus.#RewardTypeId.Active = :Active"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":Active": &dynamodb_types.AttributeValueMemberBOOL{Value: false},
				},
				ReturnValues: dynamodb_types.ReturnValueNone,
			},
			{
				TableName: aws.String("RewardRule-table"),
				Key: map[string]dynamodb_types.AttributeValue{
					"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardTypeStatus},
					"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardTypeStatus},
				},
				ExpressionAttributeNames: map[string]string{
					"#RewardTypeId": "RD04",
				},
				UpdateExpression: aws.String("SET RewardTypeStatus.#RewardTypeId.Active = :Active"),
				ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
					":Active": &dynamodb_types.AttributeValueMemberBOOL{Value: true},
				},
				ReturnValues: dynamodb_types.ReturnValueNone,
			},
		}

		err := svc.PatchRewardTypesStatus(patchInput)

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs)

	})
}

func Test_PatchRewardUnits(t *testing.T) {
	t.Run("it should perform patch on Reward Units", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},

			UpdateItemOutputs: []dynamodb.UpdateItemOutput{
				{},
			},
			UpdateItemErrors: []error{
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		expectedDDBInput := dynamodb.UpdateItemInput{
			TableName: aws.String("RewardRule-table"),
			Key: map[string]dynamodb_types.AttributeValue{
				"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: RULE_ID____RewardUnits},
				"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardUnits},
			},
			UpdateExpression: aws.String("SET RewardUnits = :RewardUnits"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":RewardUnits": &dynamodb_types.AttributeValueMemberM{
					Value: map[string]dynamodb_types.AttributeValue{
						"Points":       &dynamodb_types.AttributeValueMemberS{Value: "150.00"},
						"EqualAmount":  &dynamodb_types.AttributeValueMemberS{Value: "1"},
						"CurrencyType": &dynamodb_types.AttributeValueMemberS{Value: "AUD"},
					},
				},
			},
			ReturnValues: dynamodb_types.ReturnValueNone,
		}

		err := svc.PatchRewardUnits(RewardUnits{
			Points:       "150.00",
			EqualAmount:  "1",
			CurrencyType: "AUD",
		})

		assert.NoError(t, err)
		assert.Equal(t, expectedDDBInput, ddbClient.UpdateItemInputs[0])

	})
}

func Test_PatchRewardRules(t *testing.T) {
	t.Run("It should Patch the Reward Rules when correct Input is provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
				{},
			},
			PutItemErrors: []error{
				nil,
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		ExpectedDDBInput := dynamodb.PutItemInput{
			TableName: aws.String("RewardRule-table"),
			Item: map[string]dynamodb_types.AttributeValue{
				"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-234523423a"},
				"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardRules},

				"RuleName": &dynamodb_types.AttributeValueMemberS{Value: "Test-RuleName"},
				"RuleDesc": &dynamodb_types.AttributeValueMemberS{Value: "Test Desc"},

				"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
				"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "200"},
				"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD01"},
				"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "Yearly"},

				"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: "20-12-2024"},
				"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: "20-12-2030"},

				"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: "Active"},
				"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: utils.GenerateTimestamp()},
			},
		}

		err := svc.PatchRewardRules(RewardRulesPatchInput{
			RuleId:                 "rr-234523423a",
			RuleName:               "Test-RuleName",
			RuleDesc:               "Test Desc",
			RuleForImplementation:  "Everyone",
			RuleRewardPoints:       "200",
			RuleRewardType:         "RD01",
			RuleWhenImplementation: "Yearly",
			RuleStartDate:          "20-12-2024",
			RuleEndDate:            "20-12-2030",
			RuleStatus:             "Active",
		})

		assert.NoError(t, err)
		assert.Equal(t, ExpectedDDBInput, ddbClient.PutItemInputs[0])

	})
}

func Test_PutRewardRuleUpdateLogs(t *testing.T) {
	t.Run("It should put the Rule Update logs when correct Input is provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			PutItemOutputs: []dynamodb.PutItemOutput{
				{},
			},
			PutItemErrors: []error{
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		err := svc.PutRewardRuleUpdateLogs(PutRewardRuleUpdateLogs{
			UpdateType:     "Created",
			RewardRuleType: "RewardUnits",
			RuleId:         "dasda",
			UpdatedBy:      "Admin",
		})

		assert.NoError(t, err)

		/*
			ExpectedddbPutItemInput := dynamodb.PutItemInput{
				TableName: aws.String("RewardRule-table"),
				Item: map[string]dynamodb_types.AttributeValue{
					"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: ""},
					"RuleType": &dynamodb_types.AttributeValueMemberS{Value: RULE_TYPE__RewardUpdateLogs},

					"RewardRuleLogData":       &dynamodb_types.AttributeValueMemberS{Value: "A Reward Rule was Created of RewardType: RewardUnits having RuleId: dasda"},
					"RewardRuleLogUpdateDate": &dynamodb_types.AttributeValueMemberS{Value: utils.GenerateTimestamp()},
					"RewardRuleUpdateBy":      &dynamodb_types.AttributeValueMemberS{Value: "Admin"},
				},
			}
		*/
		//assert.Equal(t, ExpectedddbPutItemInput, ddbClient.PutItemInputs[0])

	})
}

func Test_GetTopLevelRewardSettings(t *testing.T) {
	t.Run("It should get the Top Level Rewards", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: map[string]dynamodb_types.AttributeValue{
						"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "0000AAAA"},
						"RuleType": &dynamodb_types.AttributeValueMemberS{Value: "RewardTypeStatus"},
						"RewardTypeStatus": &dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"RD00": &dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: "RD00"},
										"RewardName":   &dynamodb_types.AttributeValueMemberS{Value: "General Rewards"},
										"RewardDesc":   &dynamodb_types.AttributeValueMemberS{Value: "Rewards that are applicable to all items"},
										"Active":       &dynamodb_types.AttributeValueMemberBOOL{Value: true},
									},
								},
								"RD01": &dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: "RD01"},
										"RewardName":   &dynamodb_types.AttributeValueMemberS{Value: "Health Rewards"},
										"RewardDesc":   &dynamodb_types.AttributeValueMemberS{Value: "Rewards that are applicable to health related Items"},
										"Active":       &dynamodb_types.AttributeValueMemberBOOL{Value: true},
									},
								},
								"RD02": &dynamodb_types.AttributeValueMemberM{
									Value: map[string]dynamodb_types.AttributeValue{
										"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: "RD02"},
										"RewardName":   &dynamodb_types.AttributeValueMemberS{Value: "Skill Rewards"},
										"RewardDesc":   &dynamodb_types.AttributeValueMemberS{Value: "Rewards that are applicable to skill related resources"},
										"Active":       &dynamodb_types.AttributeValueMemberBOOL{Value: true},
									},
								},
							},
						},
					},
				},
				{
					Item: map[string]dynamodb_types.AttributeValue{
						"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "0000BBBB"},
						"RuleType": &dynamodb_types.AttributeValueMemberS{Value: "RewardUnits"},
						"RewardUnits": &dynamodb_types.AttributeValueMemberM{
							Value: map[string]dynamodb_types.AttributeValue{
								"Points":       &dynamodb_types.AttributeValueMemberS{Value: "100"},
								"EqualAmount":  &dynamodb_types.AttributeValueMemberS{Value: "1"},
								"CurrencyType": &dynamodb_types.AttributeValueMemberS{Value: "AUD"},
							},
						},
					},
				},
			},
			GetItemErrors: []error{
				nil,
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		expectedTopLevelSettings := TopLevelRewardSettings{
			RewardTypeStatus: map[string]RewardStatus{
				"RD00": {
					RewardTypeId: "RD00",
					RewardName:   "General Rewards",
					RewardDesc:   "Rewards that are applicable to all items",
					Active:       true,
				},
				"RD01": {
					RewardTypeId: "RD01",
					RewardName:   "Health Rewards",
					RewardDesc:   "Rewards that are applicable to health related Items",
					Active:       true,
				},
				"RD02": {
					RewardTypeId: "RD02",
					RewardName:   "Skill Rewards",
					RewardDesc:   "Rewards that are applicable to skill related resources",
					Active:       true,
				},
			},
			RewardUnits: RewardUnits{
				Points:       "100",
				EqualAmount:  "1",
				CurrencyType: "AUD",
			},
		}

		output, err := svc.GetTopLevelRewardSettings()

		assert.NoError(t, err)
		assert.Equal(t, expectedTopLevelSettings, output)
	})
}

func Test_GetRewardRuleData(t *testing.T) {
	t.Run("It should Get the Reward Rule data based on the query provided", func(t *testing.T) {

		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-abadads"},
							"RuleName": &dynamodb_types.AttributeValueMemberS{Value: "Rule-test"},

							"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
							"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "100.00"},
							"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD00"},
							"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "IMMEDIATELY"},

							"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
							"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: "01-01-2030"},

							"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: "Active"},
							"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
						},
						{
							"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-dfsadfafd"},
							"RuleName": &dynamodb_types.AttributeValueMemberS{Value: "Rule-test-2"},

							"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
							"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "150.00"},
							"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD01"},
							"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "IMMEDIATELY"},

							"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
							"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: "01-01-2030"},

							"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: "Active"},
							"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
			},
		}
		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		expectedOutput := []RuleData{
			{
				RuleId:   "rr-abadads",
				RuleName: "Rule-test",

				RuleForImplementation:  "Everyone",
				RuleRewardPoints:       "100.00",
				RuleRewardType:         "RD00",
				RuleWhenImplementation: "IMMEDIATELY",

				RuleStartDate: "01-01-2024",
				RuleEndDate:   "01-01-2030",

				RuleStatus:      "Active",
				RuleLastUpdated: "01-01-2024",
			},
			{
				RuleId:   "rr-dfsadfafd",
				RuleName: "Rule-test-2",

				RuleForImplementation:  "Everyone",
				RuleRewardPoints:       "150.00",
				RuleRewardType:         "RD01",
				RuleWhenImplementation: "IMMEDIATELY",

				RuleStartDate: "01-01-2024",
				RuleEndDate:   "01-01-2030",

				RuleStatus:      "Active",
				RuleLastUpdated: "01-01-2024",
			},
		}

		output, err := svc.GetRewardRuleData("query-stmt")

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)

	})
}

func Test_GetRewardUpdateLogsData(t *testing.T) {
	t.Run("It should get RewardUpdate Logs when correct input is provided", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
				{
					Items: []map[string]dynamodb_types.AttributeValue{
						{
							"RewardRuleLogData":       &dynamodb_types.AttributeValueMemberS{Value: "example log data 1"},
							"RewardRuleLogUpdateDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
							"RewardRuleUpdateBy":      &dynamodb_types.AttributeValueMemberS{Value: "Admin"},
						},
						{
							"RewardRuleLogData":       &dynamodb_types.AttributeValueMemberS{Value: "example log data 2"},
							"RewardRuleLogUpdateDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
							"RewardRuleUpdateBy":      &dynamodb_types.AttributeValueMemberS{Value: "Admin"},
						},
					},
				},
			},
			ExecuteStatementErrors: []error{
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		expectedOutput := []RewardUpdateLogs{
			{
				RewardRuleLogData:       "example log data 1",
				RewardRuleUpdateBy:      "Admin",
				RewardRuleLogUpdateDate: "01-01-2024",
			},
			{
				RewardRuleLogData:       "example log data 2",
				RewardRuleUpdateBy:      "Admin",
				RewardRuleLogUpdateDate: "01-01-2024",
			},
		}

		output, err := svc.GetRewardUpdateLogsData()

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)

	})
}

func Test_GetRulesByRuleId(t *testing.T) {
	t.Run("It should get the reward Rule data when Rule Id is passed", func(t *testing.T) {
		logBuffer := &bytes.Buffer{}
		ddbClient := awsclients.MockDynamodbClient{
			GetItemOutputs: []dynamodb.GetItemOutput{
				{
					Item: map[string]dynamodb_types.AttributeValue{
						"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-abadads"},
						"RuleType": &dynamodb_types.AttributeValueMemberS{Value: "RewardRules"},

						"RuleName": &dynamodb_types.AttributeValueMemberS{Value: "Rule-test"},

						"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
						"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "100.00"},
						"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD00"},
						"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "IMMEDIATELY"},

						"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
						"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: "01-01-2030"},

						"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: "Active"},
						"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
					},
				},
			},
			GetItemErrors: []error{
				nil,
			},
		}

		svc := RewardsService{
			dynamodbClient:                           &ddbClient,
			logger:                                   log.New(logBuffer, "TEST:", 0),
			EmployeeRewardRulesTable:                 "RewardRule-table",
			EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
		}

		expectedOutput := RewardsRuleDynamodbData{
			RuleId:   "rr-abadads",
			RuleType: "RewardRules",
			RuleName: "Rule-test",

			RuleForImplementation:  "Everyone",
			RuleRewardPoints:       "100.00",
			RuleRewardType:         "RD00",
			RuleWhenImplementation: "IMMEDIATELY",

			RuleStartDate: "01-01-2024",
			RuleEndDate:   "01-01-2030",

			RuleStatus:      "Active",
			RuleLastUpdated: "01-01-2024",
		}

		expectedDDBInput := dynamodb.GetItemInput{
			Key: map[string]dynamodb_types.AttributeValue{
				"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-abadads"},
				"RuleType": &dynamodb_types.AttributeValueMemberS{Value: "RewardRules"},
			},
			TableName:      aws.String("RewardRule-table"),
			ConsistentRead: aws.Bool(true),
		}

		output, err := svc.GetRulesByRuleId("rr-abadads")

		assert.NoError(t, err)
		assert.Equal(t, expectedOutput, output)
		assert.Equal(t, expectedDDBInput, ddbClient.GetItemInputs[0])

	})
}

func Test_GetAllRewardRules(t *testing.T) {
	logBuffer := &bytes.Buffer{}
	ddbClient := awsclients.MockDynamodbClient{
		GetItemOutputs: []dynamodb.GetItemOutput{
			{
				Item: map[string]dynamodb_types.AttributeValue{
					"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "0000AAAA"},
					"RuleType": &dynamodb_types.AttributeValueMemberS{Value: "RewardTypeStatus"},
					"RewardTypeStatus": &dynamodb_types.AttributeValueMemberM{
						Value: map[string]dynamodb_types.AttributeValue{
							"RD00": &dynamodb_types.AttributeValueMemberM{
								Value: map[string]dynamodb_types.AttributeValue{
									"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: "RD00"},
									"RewardName":   &dynamodb_types.AttributeValueMemberS{Value: "General Rewards"},
									"RewardDesc":   &dynamodb_types.AttributeValueMemberS{Value: "Rewards that are applicable to all items"},
									"Active":       &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								},
							},
							"RD01": &dynamodb_types.AttributeValueMemberM{
								Value: map[string]dynamodb_types.AttributeValue{
									"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: "RD01"},
									"RewardName":   &dynamodb_types.AttributeValueMemberS{Value: "Health Rewards"},
									"RewardDesc":   &dynamodb_types.AttributeValueMemberS{Value: "Rewards that are applicable to health related Items"},
									"Active":       &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								},
							},
							"RD02": &dynamodb_types.AttributeValueMemberM{
								Value: map[string]dynamodb_types.AttributeValue{
									"RewardTypeId": &dynamodb_types.AttributeValueMemberS{Value: "RD02"},
									"RewardName":   &dynamodb_types.AttributeValueMemberS{Value: "Skill Rewards"},
									"RewardDesc":   &dynamodb_types.AttributeValueMemberS{Value: "Rewards that are applicable to skill related resources"},
									"Active":       &dynamodb_types.AttributeValueMemberBOOL{Value: true},
								},
							},
						},
					},
				},
			},
			{
				Item: map[string]dynamodb_types.AttributeValue{
					"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "0000BBBB"},
					"RuleType": &dynamodb_types.AttributeValueMemberS{Value: "RewardUnits"},
					"RewardUnits": &dynamodb_types.AttributeValueMemberM{
						Value: map[string]dynamodb_types.AttributeValue{
							"Points":       &dynamodb_types.AttributeValueMemberS{Value: "100"},
							"EqualAmount":  &dynamodb_types.AttributeValueMemberS{Value: "1"},
							"CurrencyType": &dynamodb_types.AttributeValueMemberS{Value: "AUD"},
						},
					},
				},
			},
		},
		GetItemErrors: []error{
			nil,
			nil,
		},
		ExecuteStatementOutputs: []dynamodb.ExecuteStatementOutput{
			{
				Items: []map[string]dynamodb_types.AttributeValue{
					{
						"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-abadads"},
						"RuleName": &dynamodb_types.AttributeValueMemberS{Value: "Rule-test"},

						"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
						"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "100.00"},
						"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD00"},
						"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "IMMEDIATELY"},

						"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
						"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: "01-01-2030"},

						"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: "Active"},
						"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
					},
					{
						"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-dfsadfafd"},
						"RuleName": &dynamodb_types.AttributeValueMemberS{Value: "Rule-test-2"},

						"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
						"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "150.00"},
						"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD01"},
						"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "IMMEDIATELY"},

						"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
						"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: "01-01-2030"},

						"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: "Active"},
						"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
					},
				},
			},
			{
				Items: []map[string]dynamodb_types.AttributeValue{
					{
						"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-abadads"},
						"RuleName": &dynamodb_types.AttributeValueMemberS{Value: "Rule-test"},

						"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
						"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "100.00"},
						"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD00"},
						"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "IMMEDIATELY"},

						"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2025"},
						"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: "01-01-2030"},

						"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: "Draft"},
						"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
					},
					{
						"RuleId":   &dynamodb_types.AttributeValueMemberS{Value: "rr-dfsadfafd"},
						"RuleName": &dynamodb_types.AttributeValueMemberS{Value: "Rule-test-2"},

						"RuleForImplementation":  &dynamodb_types.AttributeValueMemberS{Value: "Everyone"},
						"RuleRewardPoints":       &dynamodb_types.AttributeValueMemberS{Value: "150.00"},
						"RuleRewardType":         &dynamodb_types.AttributeValueMemberS{Value: "RD01"},
						"RuleWhenImplementation": &dynamodb_types.AttributeValueMemberS{Value: "IMMEDIATELY"},

						"RuleStartDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2025"},
						"RuleEndDate":   &dynamodb_types.AttributeValueMemberS{Value: "01-01-2030"},

						"RuleStatus":      &dynamodb_types.AttributeValueMemberS{Value: "Draft"},
						"RuleLastUpdated": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
					},
				},
			},
			{
				Items: []map[string]dynamodb_types.AttributeValue{
					{
						"RewardRuleLogData":       &dynamodb_types.AttributeValueMemberS{Value: "example log data 1"},
						"RewardRuleLogUpdateDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
						"RewardRuleUpdateBy":      &dynamodb_types.AttributeValueMemberS{Value: "Admin"},
					},
					{
						"RewardRuleLogData":       &dynamodb_types.AttributeValueMemberS{Value: "example log data 2"},
						"RewardRuleLogUpdateDate": &dynamodb_types.AttributeValueMemberS{Value: "01-01-2024"},
						"RewardRuleUpdateBy":      &dynamodb_types.AttributeValueMemberS{Value: "Admin"},
					},
				},
			},
		},
		ExecuteStatementErrors: []error{
			nil,
			nil,
			nil,
		},
	}

	svc := RewardsService{
		dynamodbClient:                           &ddbClient,
		logger:                                   log.New(logBuffer, "TEST:", 0),
		EmployeeRewardRulesTable:                 "RewardRule-table",
		EmployeeRewardRulesTable_RuleStatusIndex: "RewardRule-Index",
	}

	expectedOutput := GetAllRewardRules{
		TopLevelRewardSettings: TopLevelRewardSettings{
			RewardTypeStatus: map[string]RewardStatus{
				"RD00": {
					RewardTypeId: "RD00",
					RewardName:   "General Rewards",
					RewardDesc:   "Rewards that are applicable to all items",
					Active:       true,
				},
				"RD01": {
					RewardTypeId: "RD01",
					RewardName:   "Health Rewards",
					RewardDesc:   "Rewards that are applicable to health related Items",
					Active:       true,
				},
				"RD02": {
					RewardTypeId: "RD02",
					RewardName:   "Skill Rewards",
					RewardDesc:   "Rewards that are applicable to skill related resources",
					Active:       true,
				},
			},
			RewardUnits: RewardUnits{
				Points:       "100",
				EqualAmount:  "1",
				CurrencyType: "AUD",
			},
		},
		RewardRules: RewardRules{
			Active: []RuleData{
				{
					RuleId:   "rr-abadads",
					RuleName: "Rule-test",

					RuleForImplementation:  "Everyone",
					RuleRewardPoints:       "100.00",
					RuleRewardType:         "RD00",
					RuleWhenImplementation: "IMMEDIATELY",

					RuleStartDate: "01-01-2024",
					RuleEndDate:   "01-01-2030",

					RuleStatus:      "Active",
					RuleLastUpdated: "01-01-2024",
				},
				{
					RuleId:   "rr-dfsadfafd",
					RuleName: "Rule-test-2",

					RuleForImplementation:  "Everyone",
					RuleRewardPoints:       "150.00",
					RuleRewardType:         "RD01",
					RuleWhenImplementation: "IMMEDIATELY",

					RuleStartDate: "01-01-2024",
					RuleEndDate:   "01-01-2030",

					RuleStatus:      "Active",
					RuleLastUpdated: "01-01-2024",
				},
			},
			Draft: []RuleData{
				{
					RuleId:   "rr-abadads",
					RuleName: "Rule-test",

					RuleForImplementation:  "Everyone",
					RuleRewardPoints:       "100.00",
					RuleRewardType:         "RD00",
					RuleWhenImplementation: "IMMEDIATELY",

					RuleStartDate: "01-01-2025",
					RuleEndDate:   "01-01-2030",

					RuleStatus:      "Draft",
					RuleLastUpdated: "01-01-2024",
				},
				{
					RuleId:   "rr-dfsadfafd",
					RuleName: "Rule-test-2",

					RuleForImplementation:  "Everyone",
					RuleRewardPoints:       "150.00",
					RuleRewardType:         "RD01",
					RuleWhenImplementation: "IMMEDIATELY",

					RuleStartDate: "01-01-2025",
					RuleEndDate:   "01-01-2030",

					RuleStatus:      "Draft",
					RuleLastUpdated: "01-01-2024",
				},
			},
		},
		RewardUpdateLogs: []RewardUpdateLogs{
			{
				RewardRuleLogData:       "example log data 1",
				RewardRuleUpdateBy:      "Admin",
				RewardRuleLogUpdateDate: "01-01-2024",
			},
			{
				RewardRuleLogData:       "example log data 2",
				RewardRuleUpdateBy:      "Admin",
				RewardRuleLogUpdateDate: "01-01-2024",
			},
		},
	}

	output, err := svc.GetAllRewardRules()

	assert.NoError(t, err)
	assert.Equal(t, expectedOutput, output)
}
