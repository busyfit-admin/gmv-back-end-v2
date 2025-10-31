# common Cloudformation related functions.

def get_logical_resource_details(context, cfn_logical_name):
    """
    Retrieve logical details about the resource from Cloudformation stack.
    """
    resource_list = context.aws["cfn_client"].describe_stack_resources(
        StackName=context.aws["cfn_stack_name"],
        LogicalResourceId=cfn_logical_name
    )
    resource_details = list(
        filter(lambda s: (s["LogicalResourceId"] == cfn_logical_name), resource_list["StackResources"])).pop()
    return resource_details


def get_logical_resource_details_on_datastack(context, cfn_logical_name):
    """
    Retrieve logical details about the resource from Cloudformation data stack.
    """
    resource_list = context.aws["cfn_client"].describe_stack_resources(
        StackName=context.aws["cfn_active_data_stack_name"],
        LogicalResourceId=cfn_logical_name
    )
    resource_details = list(
        filter(lambda s: (s["LogicalResourceId"] == cfn_logical_name), resource_list["StackResources"])).pop()
    return resource_details
