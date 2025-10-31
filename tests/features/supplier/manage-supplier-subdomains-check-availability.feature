@end2endAdmin
@end2endSupplierSubDomain

Feature: Manage Supplier SubDomains - GET API - Check availability

    Scenario Outline: GET API call made to the Supplier SubDomain should check the availability of the domain
        Given the lambda ManageSupplierSubDomainsLambda
        And DynamoDB table SupplierSubdomainsTable
        And row for DynamoDB table SupplierSubdomainsTable has the data supplier_subdomain_data
            | supplier_subdomain_data                                       |
            | testdata/supplier/manage-supplier-subdomains/scenario-3/tabledata1.json  |
            | testdata/supplier/manage-supplier-subdomains/scenario-3/tabledata2.json  |
            | testdata/supplier/manage-supplier-subdomains/scenario-3/tabledata3.json  |
        When the lambda ManageSupplierSubDomainsLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageSupplierSubDomainsLambda should finish successfully with status code 200 and response body of <lambda_response>

     Examples: API calls to the Lambda
        | base_api_call_path                                                                | body_input                                                        | lambda_response                                                       |
        | testdata/supplier/manage-supplier-subdomains/scenario-3/api_get_subdomains-availability.json | testdata/supplier/manage-supplier-subdomains/scenario-3/input_body.json     | testdata/supplier/manage-supplier-subdomains/scenario-3/expected_lambdares.json |
