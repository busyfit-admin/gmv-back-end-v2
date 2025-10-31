
@end2endTenant

Feature: Manage Reward Rules API Tests

    Scenario Outline: GET API Call is Made on ManageRewardRules API
        Given the lambda ManageRewardRulesLambda
        And DynamoDB table RewardRulesTable
        When the lambda ManageRewardRulesLambda receives an input of file <get_api_call>
        Then the lambda ManageRewardRulesLambda should finish successfully with status code 200 and response body of <lambda_response>

     Examples: API calls to the Lambda
        |get_api_call                                               | lambda_response                                           |
        |testdata/tenant/manage-reward-rules/scenario-1/get_api_call.json | testdata/tenant/manage-reward-rules/scenario-1/lambda_response.json |


    # Scenario Outline: GET API Call is Made on ManageRewardRules API with header
    #     Given the lambda ManageRewardRulesLambda
    #     And DynamoDB table RewardRulesTable
    #     When the lambda ManageRewardRulesLambda receives an input of file <get_api_call>
    #     Then the lambda ManageRewardRulesLambda should finish successfully with status code 200 and response body of <lambda_response>

    #  Examples: API calls to the Lambda
    #     |get_api_call                                               | lambda_response                                           |
    #     |testdata/tenant/manage-reward-rules/scenario-1/get_api_call.json | testdata/tenant/manage-reward-rules/scenario-1/lambda_response.json |
