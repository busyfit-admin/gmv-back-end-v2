@end2endAdmin

Feature: Manage Supplier Profile - POST API

    Scenario Outline: POST API call made to the Supplier Profiles should update the DDB Table in the correct data format
        Given the lambda ManageSupplierProfileLambda
        And DynamoDB table SupplierDetailsTable
        When the lambda ManageSupplierProfileLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageSupplierProfileLambda should finish successfully

     Examples: API calls to the Lambda
        | base_api_call_path                   | body_input                                                      |
        | testdata/supplier/common/common-post-api.json | testdata/supplier/manage-supplier-profiles/scenario-3/input_body1.json  |
