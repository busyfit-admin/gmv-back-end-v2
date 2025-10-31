@end2endAdmin

Feature: Manage Supplier Profile - Patch APIs

    Scenario Outline: PATCH API call made to the Supplier Profiles to update the top level info
        Given the lambda ManageSupplierProfileLambda
        And DynamoDB table SupplierDetailsTable
        And row for DynamoDB table SupplierDetailsTable has the data supplier_data
            | supplier_data                                               |
            | testdata/supplier/manage-supplier-profiles/scenario-4/table_data1.json |
        When the lambda ManageSupplierProfileLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageSupplierProfileLambda should finish successfully
        And DynamoDB table SupplierDetailsTable with keys SupplierId should have <table_item_path>

     Examples: API calls to the Lambda
        | base_api_call_path                                                   | body_input                                                          | table_item_path                                                           |
        | testdata/supplier/manage-supplier-profiles/scenario-4/api_update_top_level.json | testdata/supplier/manage-supplier-profiles/scenario-4/update-top-level-input.json | testdata/supplier/manage-supplier-profiles/scenario-4/output_update_top_level_ddb.json |


    Scenario Outline: PATCH API call made to the Supplier Profiles to add contact
        Given the lambda ManageSupplierProfileLambda
        And DynamoDB table SupplierDetailsTable
        And row for DynamoDB table SupplierDetailsTable has the data supplier_data
            | supplier_data                                               |
            | testdata/supplier/manage-supplier-profiles/scenario-4/table_data2.json |
        When the lambda ManageSupplierProfileLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageSupplierProfileLambda should finish successfully
        And DynamoDB table SupplierDetailsTable with keys SupplierId should have <table_item_path>

     Examples: API calls to the Lambda
        | base_api_call_path                                                   | body_input                                                      | table_item_path                                                           |
        | testdata/supplier/manage-supplier-profiles/scenario-4/api_update_contact.json | testdata/supplier/manage-supplier-profiles/scenario-4/update-contact.json | testdata/supplier/manage-supplier-profiles/scenario-4/output_update_contact_ddb.json |

    Scenario Outline: PATCH API call made to the Supplier Profiles to patch stage
        Given the lambda ManageSupplierProfileLambda
        And DynamoDB table SupplierDetailsTable
        And row for DynamoDB table SupplierDetailsTable has the data supplier_data
            | supplier_data                                               |
            | testdata/supplier/manage-supplier-profiles/scenario-4/table_data3.json |
        When the lambda ManageSupplierProfileLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageSupplierProfileLambda should finish successfully
        And DynamoDB table SupplierDetailsTable with keys SupplierId should have <table_item_path>

     Examples: API calls to the Lambda
        | base_api_call_path                                                   | body_input                                                | table_item_path                                                           |
        | testdata/supplier/manage-supplier-profiles/scenario-4/api_update_stage.json | testdata/supplier/manage-supplier-profiles/scenario-4/update-stage.json | testdata/supplier/manage-supplier-profiles/scenario-4/output_update_stage_ddb.json |
