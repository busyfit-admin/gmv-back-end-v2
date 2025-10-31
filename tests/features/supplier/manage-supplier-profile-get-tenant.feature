@dynamodb_delete_SupplierDetailsTable_SupplierId_supplier123
@dynamodb_delete_SupplierDetailsTable_SupplierId_supplier456

@end2endAdmin

Feature: Manage Supplier Profiles APIs - Get Supplier Details

    Scenario Outline: GET API call made to the Supplier Details with SupplierId in Header
        Given the lambda ManageSupplierProfileLambda
        And DynamoDB table SupplierDetailsTable
        And row for DynamoDB table SupplierDetailsTable has the data supplier_data
            | supplier_data                                               |
            | testdata/supplier/manage-supplier-profiles/scenario-2/tabledata1.json |
            | testdata/supplier/manage-supplier-profiles/scenario-2/tabledata2.json |
        When the lambda ManageSupplierProfileLambda receives an input of file <get_api_call>
        Then the lambda ManageSupplierProfileLambda should finish successfully with status code 200 and response body of <lambda_response>

     Examples: API calls to the Lambda
        | get_api_call                                                | lambda_response                                                |
        | testdata/supplier/manage-supplier-profiles/scenario-2/get_api_call1.json | testdata/supplier/manage-supplier-profiles/scenario-2/lambda_response1.json |
        | testdata/supplier/manage-supplier-profiles/scenario-2/get_api_call2.json | testdata/supplier/manage-supplier-profiles/scenario-2/lambda_response2.json |
