

@end2endAdmin


Feature: Manage Tenant Profile - Patch APIs

    Scenario Outline: PATCH API call made to the Tenant Profiles the top level info
        Given the lambda ManageTenantProfileLambda
        And DynamoDB table TenantDetailsTable
        And row for DynamoDB table TenantDetailsTable has the data tenant_data
            |tenant_data                                                |
            | testdata/tenant/manage-tenant-profiles/scenario-4/table_data1.json |
        When the lambda ManageTenantProfileLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageTenantProfileLambda should finish successfully
        And DynamoDB table TenantDetailsTable with keys TenantId should have <table_item_path>

     Examples: API calls to the Lambda
        |base_api_call_path                                                   | body_input                                                                | table_item_path                                                             |
        |testdata/tenant/manage-tenant-profiles/scenario-4/api_update_top_level.json | testdata/tenant/manage-tenant-profiles/scenario-4/update-top-level-input.json     | testdata/tenant/manage-tenant-profiles/scenario-4/output_update_top_level_ddb.json |


    Scenario Outline: PATCH API call made to the Tenant Profiles the Add contact
        Given the lambda ManageTenantProfileLambda
        And DynamoDB table TenantDetailsTable
        And row for DynamoDB table TenantDetailsTable has the data tenant_data
            |tenant_data                                                |
            | testdata/tenant/manage-tenant-profiles/scenario-4/table_data2.json |
        When the lambda ManageTenantProfileLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageTenantProfileLambda should finish successfully
        And DynamoDB table TenantDetailsTable with keys TenantId should have <table_item_path>

     Examples: API calls to the Lambda
        |base_api_call_path                                                   | body_input                                                        | table_item_path                                                             |
        |testdata/tenant/manage-tenant-profiles/scenario-4/api_update_contact.json | testdata/tenant/manage-tenant-profiles/scenario-4/update-contact.json     | testdata/tenant/manage-tenant-profiles/scenario-4/output_update_contact_ddb.json |

    Scenario Outline: PATCH API call made to the Tenant Profiles to patch stage
        Given the lambda ManageTenantProfileLambda
        And DynamoDB table TenantDetailsTable
        And row for DynamoDB table TenantDetailsTable has the data tenant_data
            |tenant_data                                                |
            | testdata/tenant/manage-tenant-profiles/scenario-4/table_data3.json |
        When the lambda ManageTenantProfileLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageTenantProfileLambda should finish successfully
        And DynamoDB table TenantDetailsTable with keys TenantId should have <table_item_path>

     Examples: API calls to the Lambda
        |base_api_call_path                                                   | body_input                                             | table_item_path                                                             |
        |testdata/tenant/manage-tenant-profiles/scenario-4/api_update_stage.json | testdata/tenant/manage-tenant-profiles/scenario-4/update-stage.json | testdata/tenant/manage-tenant-profiles/scenario-4/output_update_stage_ddb.json |
