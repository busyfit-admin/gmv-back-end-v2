
@dynamodb_delete_TenantDetailsTable_TenantId_tenant123
@dynamodb_delete_TenantDetailsTable_TenantId_tenant456

@end2endAdmin


Feature: Manage Tenant Profiles APIs - Get Tenant Details

    Scenario Outline: GET API call made to the Tenant Details with TenantId in Header
        Given the lambda ManageTenantProfileLambda
        And DynamoDB table TenantDetailsTable
        And row for DynamoDB table TenantDetailsTable has the data tenant_data
            | tenant_data                                               |
            | testdata/tenant/manage-tenant-profiles/scenario-2/tabledata1.json  |
            | testdata/tenant/manage-tenant-profiles/scenario-2/tabledata2.json  |
        When the lambda ManageTenantProfileLambda receives an input of file <get_api_call>
        Then the lambda ManageTenantProfileLambda should finish successfully with status code 200 and response body of <lambda_response>

     Examples: API calls to the Lambda
        |get_api_call                                                | lambda_response                                                |
        |testdata/tenant/manage-tenant-profiles/scenario-2/get_api_call1.json | testdata/tenant/manage-tenant-profiles/scenario-2/lambda_response1.json |
        |testdata/tenant/manage-tenant-profiles/scenario-2/get_api_call2.json | testdata/tenant/manage-tenant-profiles/scenario-2/lambda_response2.json |
