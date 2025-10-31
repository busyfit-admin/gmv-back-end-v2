@end2endAdmin
@end2endTenantSubDomain

Feature: Manage Tenant SubDomains - POST API - Create Subdomain

    Scenario Outline: POST API call made to the Tenant SubDomain should create the subdomain successfully
        Given the lambda ManageTenantSubDomainsLambda
        And DynamoDB table TenantSubdomainsTable
        When the lambda ManageTenantSubDomainsLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageTenantSubDomainsLambda should finish successfully
        And DynamoDB table TenantSubdomainsTable with keys SubDomain, TenantId should have <table_item_path>
     Examples: API calls to the Lambda
        |base_api_call_path                                                         | body_input                                                                 | table_item_path                                                       |
        |testdata/tenant/manage-tenant-subdomains/scenario-1/api_post_create-subdomain.json | testdata/tenant/manage-tenant-subdomains/scenario-1/input_body.json     | testdata/tenant/manage-tenant-subdomains/scenario-1/expected_table_data.json |
