@end2endAdmin


Feature: Manage Tenant Profile - POST API

    Scenario Outline: POST API call made to the Tenant Profiles should update the DDB Table in the correct data format
        Given the lambda ManageTenantProfileLambda
        And DynamoDB table TenantDetailsTable
        When the lambda ManageTenantProfileLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageTenantProfileLambda should finish successfully
     Examples: API calls to the Lambda
        |base_api_call_path                   | body_input                                                      |
        |testdata/tenant/common/common-post-api.json | testdata/tenant/manage-tenant-profiles/scenario-3/input_body1.json     |
