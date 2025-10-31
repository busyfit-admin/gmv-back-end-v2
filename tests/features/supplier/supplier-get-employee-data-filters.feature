@end2endSupplier

Feature: Get Supplier Employee Data with and without filters

    Scenario Outline: POST API call made to the GetEmployeeData should get data as per filter provided
        Given the lambda GetEmployeeDataLambda
        And DynamoDB table SupplierEmployeeDataTable
        And row for DynamoDB table SupplierEmployeeDataTable has the data employee_data
            | employee_data                                           |
            | testdata/supplier/supplier-get-employee-data/ddb-item-1.json     |
            | testdata/supplier/supplier-get-employee-data/ddb-item-2.json     |
            | testdata/supplier/supplier-get-employee-data/ddb-item-3.json     |
            | testdata/supplier/supplier-get-employee-data/ddb-item-4.json     |
        When the lambda GetEmployeeDataLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda GetEmployeeDataLambda should finish successfully with status code 200 and response body of <lambda_response>

    Examples: API calls to the Lambda
        | base_api_call_path                                             | body_input                                                     | lambda_response                                                |
        | testdata/supplier/supplier-get-employee-data/post-api-base-call.json     | testdata/supplier/supplier-get-employee-data/scenario-1/input-body.json | testdata/supplier/supplier-get-employee-data/scenario-1/expected-output.json |
        | testdata/supplier/supplier-get-employee-data/post-api-base-call.json     | testdata/supplier/supplier-get-employee-data/scenario-2/input-body.json | testdata/supplier/supplier-get-employee-data/scenario-2/expected-output.json |
        | testdata/supplier/supplier-get-employee-data/post-api-base-call.json     | testdata/supplier/supplier-get-employee-data/scenario-3/input-body.json | testdata/supplier/supplier-get-employee-data/scenario-3/expected-output.json |
        | testdata/supplier/supplier-get-employee-data/post-api-base-call.json     | testdata/supplier/supplier-get-employee-data/scenario-4/input-body.json | testdata/supplier/supplier-get-employee-data/scenario-4/expected-output.json |
