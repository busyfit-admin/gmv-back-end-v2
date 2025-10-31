#!/usr/bin/python3

import boto3

from steps.dynamodb_step import dynamodb_tag
from steps.lambda_step import create_lambda_client


def before_all(context):
    print("Executing Before All!")

    stack_name = context.config.userdata['stack_name']
    data_stack = context.config.userdata['data_stack_name']
    if stack_name is None:
        raise Exception(f"missing configured variable stack_name")
    if data_stack is None:
        raise Exception(f"missing configured variable data_stack")

    context.aws = {
        'cfn_client': boto3.client('cloudformation'),
        'cfn_stack_name': stack_name,
        'cfn_active_data_stack_name': data_stack,
        'lambda': {},
        'cleanup': {
            'lambda_envs': []
        },
    }


# Steps after the behave steps are performed.
def after_all(context):
    # restore lambda envs to original values
    for lambda_env in context.aws["cleanup"]["lambda_envs"]:
        create_lambda_client(context).update_function_configuration(
            FunctionName=lambda_env["lambda_id"],
            Environment=lambda_env["lambda_env"]
        )


def before_feature(context, feature):
    for tag in feature.tags:
        if tag.startswith("dynamodb_"):
            dynamodb_tag(context, tag)


def after_feature(context, feature):
    for tag in feature.tags:
        if tag.startswith("dynamodb_"):
            dynamodb_tag(context, tag)
