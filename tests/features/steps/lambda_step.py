#!/usr/bin/python3

import copy
import os
import boto3
import json
import copy
import time

from behave import given, when, then
from hamcrest import assert_that, equal_to
from steps.common import get_absolute_filepath, read_file_path


@given("the lambda {lambda_name}")
def step_impl_get_cfn_stack(context, lambda_name):
    print(context)
    try:
        lambda_list = context.aws["cfn_client"].describe_stack_resources(
            StackName=context.aws["cfn_stack_name"],
            LogicalResourceId=lambda_name
        )
        print("describe stack completed")
        lambda_details = list(
            filter(lambda s: (s["LogicalResourceId"] == lambda_name), lambda_list["StackResources"])).pop()

        lambda_client = create_lambda_client(context)

        context.aws['lambda'][lambda_name] = {
            "name": lambda_details["PhysicalResourceId"]
        }
        context.aws['lambda'][lambda_name]["details"] = lambda_client.get_function(
            FunctionName=context.aws['lambda'][lambda_name]["name"])
    except Exception as exc:
        stack_name = context.aws["cfn_stack_name"]
        raise Exception(
            f"error getting finding lambda {lambda_name} in stack {stack_name}") from exc


def wait_for_lambda_update(context, lambda_name):
    lambda_id = context.aws["lambda"][lambda_name]["name"]
    lambda_client = create_lambda_client(context)

    status = "InProgress"
    while status == "InProgress":
        response = lambda_client.get_function_configuration(
            FunctionName=lambda_id
        )

        status = response["LastUpdateStatus"]
        if status == "Failed":
            raise Exception(
                f'Lambda configuration update failed: {response["LastUpdateStatusReasonCode"]}:{response["LastUpdateStatusReason"]}')
        elif status == "Successful":
            return

        time.sleep(1)


@given("the environment variable {env_var_name} of lambda {lambda_name} has value of {value}")
def step_impl_override_lambda_env(context, lambda_name, env_var_name, value):
    lambda_id = context.aws["lambda"][lambda_name]["name"]
    lambda_client = create_lambda_client(context)

    response = lambda_client.get_function_configuration(
        FunctionName=lambda_id
    )
    lambda_env = response["Environment"]
    lambda_new_env = copy.deepcopy(lambda_env)
    lambda_new_env["Variables"][env_var_name] = value

    result = lambda_client.update_function_configuration(
        FunctionName=lambda_id,
        Environment=lambda_new_env
    )

    if result["LastUpdateStatus"] == "Failed":
        raise Exception(
            f'Could not update environment of lambda {lambda_name}: {result["LastUpdateStatusReasonCode"]}:{result["LastUpdateStatusReason"]}')

    wait_for_lambda_update(context, lambda_name)

    # restore to original env at the end of all tests
    context.aws["cleanup"]["lambda_envs"].append({
        "lambda_id": lambda_id,
        "lambda_env": lambda_env
    })


@when("the lambda {lambda_name} receives an input of file {input_file}")
def step_impl_pass_file_input_to_lambda(context, lambda_name, input_file):
    input_path = get_absolute_filepath(input_file)

    with open(input_path, 'rb') as input_file:
        context.aws['lambda'][lambda_name]["response"] = create_lambda_client(context).invoke(
            FunctionName=context.aws['lambda'][lambda_name]["name"],
            InvocationType='RequestResponse',
            LogType='Tail',
            Payload=input_file,
        )


@when("the lambda {lambda_name} receives a base api call {base_api_call_path} input body {body_input}")
def step_invoke_lambda_with_api_call_format(context, lambda_name, base_api_call_path, body_input):
    # Read base api call file
    base_api_call_json = json.loads(read_file_path(base_api_call_path))

    # Read required Body section for the api call
    api_body_path = get_absolute_filepath(body_input)
    with open(api_body_path, 'r') as file:
        contents = file.read()

    #Add body section contents to the base api json
    base_api_call_json["body"] = contents

    lambdaPayload = json.dumps(base_api_call_json)

    #Invoke lambda with updated api input
    context.aws['lambda'][lambda_name]["response"] = create_lambda_client(context).invoke(
            FunctionName=context.aws['lambda'][lambda_name]["name"],
            InvocationType='RequestResponse',
            LogType='Tail',
            Payload= lambdaPayload,
        )




@then("the lambda {lambda_name} should finish successfully")
def step_impl_lambda_has_empty_response(context, lambda_name):
    assert_that(context.aws['lambda'][lambda_name]
                ["response"]["StatusCode"], is_(200))


@then("the lambda {lambda_name} should finish successfully with status code {status_code} and response body of {file_name}")
def step_impl_lambda_has_response(context, lambda_name, status_code , file_name):
    lambdarespBody = json.loads(read_file_path(file_name))
    response_stream = context.aws['lambda'][lambda_name]["response"]["Payload"]
    response_object = json.loads(response_stream.read())

    assert_that(context.aws['lambda'][lambda_name]
                ["response"]["StatusCode"], status_code)
    assert_that(lambdarespBody, equal_to(json.loads(response_object["body"])))

@then("the lambda {lambda_name} should finish successfully with response of {file_name}")
def step_impl_lambda_has_response(context, lambda_name, file_name):
    lambdaresp = json.loads(read_file_path(file_name))
    response_stream = context.aws['lambda'][lambda_name]["response"]["Payload"]
    response_object = json.loads(response_stream.read())

    assert_that(context.aws['lambda'][lambda_name]
                ["response"]["StatusCode"], is_(200))
    assert_that(lambdaresp, equal_to(response_object))


def create_lambda_client(context):
    if "client" in context.aws["lambda"]:
        return context.aws["lambda"]["client"]

    if 'sts_creds' in context.aws:
        client = boto3.client('lambda',
                              aws_access_key_id=context.aws["sts_creds"]["AccessKeyId"],
                              aws_secret_access_key=context.aws["sts_creds"]["SecretAccessKey"],
                              aws_session_token=context.aws["sts_creds"]["SessionToken"],
                              )
    else:
        client = boto3.client('lambda')

    context.aws["lambda"]["client"] = client

    return client
