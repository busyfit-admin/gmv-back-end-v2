import boto3


def list_stacks():

    # Create a CloudFormation client
    cf_client = boto3.client('cloudformation')

    try:
        # List all stacks
        response = cf_client.describe_stacks()

        # Extract and print stack names
        stacks = response['Stacks']
        print("Available Stacks:")
        for stack in stacks:
            print(stack['StackName'])

    except Exception as e:
        print(f"Error: {e}")


# Call the function to list stacks
list_stacks()
