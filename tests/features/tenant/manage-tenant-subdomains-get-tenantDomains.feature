@end2endAdmin
@end2endTenantSubDomain

Feature: Manage Tenant SubDomains - GET API - List all Subdomains

    Scenario Outline: GET API call made to the Tenant SubDomain should get all tenant related subdomains
        Given the lambda ManageTenantSubDomainsLambda
        And DynamoDB table TenantSubdomainsTable
        And row for DynamoDB table TenantSubdomainsTable has the data tenant_subdomain_data
            | tenant_subdomain_data                                       |
            | testdata/tenant/manage-tenant-subdomains/scenario-2/tabledata1.json  |
            | testdata/tenant/manage-tenant-subdomains/scenario-2/tabledata2.json  |
            | testdata/tenant/manage-tenant-subdomains/scenario-2/tabledata3.json  |
        When the lambda ManageTenantSubDomainsLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageTenantSubDomainsLambda should finish successfully with status code 200 and response body of <lambda_response>
     Examples: API calls to the Lambda
        |base_api_call_path                                                         | body_input                                                        | lambda_response                                                       |
        |testdata/tenant/manage-tenant-subdomains/scenario-2/api_get_tenant-subdomains.json | testdata/tenant/manage-tenant-subdomains/scenario-2/input_body.json     | testdata/tenant/manage-tenant-subdomains/scenario-2/expected_lambdares.json |
