import boto3


def describe_lambda_function(stack_name, lambda_logical_id):
    # Create a CloudFormation client
    cfn_client = boto3.client('cloudformation')

    try:
        # List all stacks
        response = cfn_client.describe_stacks()

        # Extract and print stack names
        stacks = response['Stacks']
        print("Available Stacks:")
        for stack in stacks:
            print(stack['StackName'])

    except Exception as e:
        print(f"Error: {e}")

    # Describe stack resources to get Lambda function details
    try:
        response = cfn_client.describe_stack_resources(StackName=stack_name)
        for resource in response['StackResources']:
            if resource['LogicalResourceId'] == lambda_logical_id and resource['ResourceType'] == 'AWS::Lambda::Function':
                function_name = resource['PhysicalResourceId']

                # Create a Lambda client
                lambda_client = boto3.client('lambda')

                # Describe the Lambda function
                lambda_function = lambda_client.get_function(
                    FunctionName=function_name)
                return lambda_function

        print(
            f"No Lambda function with logical ID '{lambda_logical_id}' found in stack '{stack_name}'.")
        return None

    except Exception as e:
        print(f"Error describing Lambda function: {e}")
        return None


stack_name = 'company-portal-apis-vishal'
lambda_logical_id = 'ManageRewardRulesLambda'

lambda_function_details = describe_lambda_function(
    stack_name, lambda_logical_id)

if lambda_function_details:
    print("Lambda Function Details:")
    print(
        f"Function Name: {lambda_function_details['Configuration']['FunctionName']}")
    print(f"Runtime: {lambda_function_details['Configuration']['Runtime']}")
    print(f"Handler: {lambda_function_details['Configuration']['Handler']}")
    # Add more details as needed
