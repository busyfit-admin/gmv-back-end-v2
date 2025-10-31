@dynamodb_delete_TenantStagesTable_StageId_STG01_TenantId_testtenantId1
@dynamodb_delete_TenantStagesTable_StageId_STG02_TenantId_testtenantId1
@dynamodb_delete_TenantStagesTable_StageId_STG03_TenantId_testtenantId1
@dynamodb_delete_TenantStagesTable_StageId_STG04_TenantId_testtenantId1
@dynamodb_delete_TenantStagesTable_StageId_STG08_TenantId_testtenantId2

@end2endAdmin


Feature: Manage Tenant Stages APIs

    Scenario Outline: GET API call made to the TenantStages to get Empty response when no data is found
        Given the lambda ManageTenantStagesLambda
        And DynamoDB table TenantStagesTable
        And row for DynamoDB table TenantStagesTable has the data Stages_data
            | Stages_data                                               |
            | testdata/tenant/manage-tenant-stages/scenario-1/tabledata1.json  |
            | testdata/tenant/manage-tenant-stages/scenario-1/tabledata2.json  |
            | testdata/tenant/manage-tenant-stages/scenario-1/tabledata3.json  |
            | testdata/tenant/manage-tenant-stages/scenario-1/tabledata4.json  |
        When the lambda ManageTenantStagesLambda receives an input of file <get_api_call>
        Then the lambda ManageTenantStagesLambda should finish successfully with status code 200 and response body of <lambda_response>

     Examples: API calls to the Lambda
        |get_api_call                                                | lambda_response                                                |
        |testdata/tenant/manage-tenant-stages/scenario-1/get_api_call1.json | testdata/tenant/manage-tenant-stages/scenario-1/lambda_response1.json |
        #with no stage data in DDB Table
        |testdata/tenant/manage-tenant-stages/scenario-1/get_api_call2.json | testdata/tenant/manage-tenant-stages/scenario-1/lambda_response2.json |


    Scenario Outline: POST API call made to the TenantStages should update the DDB Table in the correct data format
        Given the lambda ManageTenantStagesLambda
        And DynamoDB table TenantStagesTable
        When the lambda ManageTenantStagesLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageTenantStagesLambda should finish successfully
     Examples: API calls to the Lambda
        |base_api_call_path                   | body_input                                                      |
        |testdata/tenant/common/common-post-api.json | testdata/tenant/manage-tenant-stages/scenario-2/input_body1.json       |
