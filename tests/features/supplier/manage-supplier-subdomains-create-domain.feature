@end2endAdmin
@end2endSupplierSubDomain

Feature: Manage Supplier SubDomains - POST API - Create Subdomain

    Scenario Outline: POST API call made to the Supplier SubDomain should create the subdomain successfully
        Given the lambda ManageSupplierSubDomainsLambda
        And DynamoDB table SupplierSubdomainsTable
        When the lambda ManageSupplierSubDomainsLambda receives a base api call <base_api_call_path> input body <body_input>
        Then the lambda ManageSupplierSubDomainsLambda should finish successfully
        And DynamoDB table SupplierSubdomainsTable with keys SubDomain, SupplierId should have <table_item_path>

     Examples: API calls to the Lambda
        | base_api_call_path                                                        | body_input                                                               | table_item_path                                                         |
        | testdata/supplier/manage-supplier-subdomains/scenario-1/api_post_create-subdomain.json | testdata/supplier/manage-supplier-subdomains/scenario-1/input_body.json       | testdata/supplier/manage-supplier-subdomains/scenario-1/expected_table_data.json |
