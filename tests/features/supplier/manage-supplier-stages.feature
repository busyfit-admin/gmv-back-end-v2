@dynamodb_delete_SupplierStagesTable_StageId_STG01_SupplierId_supplier123
@dynamodb_delete_SupplierStagesTable_StageId_STG02_SupplierId_supplier123
@dynamodb_delete_SupplierStagesTable_StageId_STG03_SupplierId_supplier123
@dynamodb_delete_SupplierStagesTable_StageId_STG04_SupplierId_supplier123
@dynamodb_delete_SupplierStagesTable_StageId_STG08_SupplierId_supplier456

@end2endAdmin

Feature: Manage Supplier Stages APIs

    Scenario Outline: GET API call made to the Supplier Stages to get Empty response when no data is found
        Given the lambda ManageSupplierStagesLambda
        And DynamoDB table SupplierStagesTable
        And row for DynamoDB table SupplierStagesTable has the data Stages_data
            | Stages_data                                                |
            | testdata/supplier/manage-supplier-stages/scenario-1/tabledata1.json |
            | testdata/supplier/manage-supplier-stages/scenario-1/tabledata2.json |
            | testdata/supplier/manage-supplier-stages/scenario-1/tabledata3.json |
            | testdata/supplier/manage-supplier-stages/scenario-1/tabledata4.json |
        When the lambda ManageSupplierStagesLambda receives an input of file <get_api_call>
        Then the lambda ManageSupplierStagesLambda should finish successfully with status code 200 and response body of <lambda_response>

     Examples: API calls to the Lambda
        | get_api_call                                                 | lambda_response                                                 |
        | testdata/supplier/manage-supplier-stages/scenario-1/get_api_call1.json | testdata/supplier/manage-supplier-stages/scenario-1/lambda_response1.json |
        # with no stage data in DDB Table
        | testdata/supplier/manage-supplier-stages/scenario-1/get_api_call2.json | testdata/supplier/manage-supplier-stages/scenario-1/lambda_response2.json |


    Scenario Outline: POST API call made to the Supplier Stages should update the DDB Table in the correct data format
        Given the lambda ManageSupplierStagesLambda
        And DynamoDB table SupplierStagesTable
        When the lambda ManageSupplierStagesLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageSupplierStagesLambda should finish successfully

     Examples: API calls to the Lambda
        | base_api_call_path                   | body_input                                                      |
        | testdata/supplier/common/common-post-api.json | testdata/supplier/manage-supplier-stages/scenario-2/input_body1.json    |
