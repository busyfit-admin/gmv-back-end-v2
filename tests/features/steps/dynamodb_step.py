import boto3
import json
import time

from behave import *
from hamcrest import assert_that, equal_to

from steps.cfn_step import get_logical_resource_details, get_logical_resource_details_on_datastack
from steps.common import read_file_path


@given('row for DynamoDB table {table_name} has the data {table_item_path}')
def step_impl_write_ddb_item_row(context, table_name, table_item_path):
    for row in context.table:
        step_impl_write_ddb_item(context, table_name, row[table_item_path])


@given('DynamoDB table {table_name} has the data {table_item_path}')
def step_impl_write_ddb_item(context, table_name, table_item_path):
    row = json.loads(read_file_path(table_item_path))
    dynamodb_client = create_dynamodb_client(context)
    item = None
    response = dynamodb_client.put_item(
        TableName=context.aws["dynamodb_tables"][table_name],
        Item=row,
        ReturnConsumedCapacity='TOTAL'
    )


@given('DynamoDB table {table_name} has the multiple data of {table_item_path} with rowcount {row_count} and pk {PrimaryKey}')
def step_impl_write_ddb_multiple_item(context, table_name, table_item_path, row_count, PrimaryKey):
    item_data = json.loads(read_file_path(table_item_path))
    dynamodb_client = create_dynamodb_client(context)
    row = 1
    while row <= int(row_count):
        item_data[PrimaryKey]['S'] = '{}'.format(row)
        response = dynamodb_client.put_item(
            TableName=context.aws["dynamodb_tables"][table_name],
            Item=item_data,
            ReturnConsumedCapacity='TOTAL'
        )
        row += 1


@given("DynamoDB table {table_name}")
def step_dynamodb_table_impl(context, table_name):
    try:
        if "dynamodb_tables" not in context.aws:
            context.aws["dynamodb_tables"] = {}

        table_details = get_logical_resource_details_on_datastack(
            context, table_name)
        context.aws["dynamodb_tables"][table_name] = table_details["PhysicalResourceId"]

    except Exception as exc:
        stack_name = context.aws["cfn_stack_name"]
        raise Exception(
            f"error getting finding DynamoDB table {table_name} in stack {stack_name}") from exc


@then('DynamoDB table {table_name} can be queried on GSI {gsi_name} with pk {pk_name} and response is {table_item_path}')
def step_impl(context, table_name, gsi_name, pk_name, table_item_path):
    dynamodb_client = create_dynamodb_client(context)

    expected_contents = json.loads(read_file_path(table_item_path))

    res = dynamodb_client.query(
        TableName=context.aws["dynamodb_tables"][table_name],
        IndexName=gsi_name,
        KeyConditionExpression=f'{pk_name} = :value',
        ExpressionAttributeValues={
            ':value': expected_contents[pk_name]
        }
    )

    actual_value = sorted(res["Items"][0].items(), reverse=True)
    expected_value = sorted(expected_contents.items(), reverse=True)

    assert_that(actual_value, equal_to(expected_value))


@then('row for DynamoDB table {table_name} with keys {table_keys} should have {table_item_path}')
def step_impl_read_ddb_item_row(context, table_name, table_keys, table_item_path):
    for row in context.table:
        step_impl_read_ddb_item(context, table_name,
                                table_keys, row[table_item_path])


@then('DynamoDB table {table_name} with keys {table_keys} should have {table_item_path}')
def step_impl_read_ddb_item(context, table_name, table_keys, table_item_path):
    time.sleep(1)

    expected_contents = json.loads(read_file_path(table_item_path))
    dynamodb_client = create_dynamodb_client(context)

    keys = {}
    for table_key in table_keys.split(", "):
        keys[table_key] = expected_contents[table_key]

    item = None
    response = dynamodb_client.get_item(
        TableName=context.aws["dynamodb_tables"][table_name],
        Key=keys,
        ConsistentRead=True,
    )

    if "Item" in response:
        item = response["Item"]
    else:
        print(f"Item key not found in response: {response}")

    actual_value = sorted(item.items(), reverse=True)
    expected_value = sorted(expected_contents.items(), reverse=True)

    assert_that(actual_value, equal_to(expected_value))


def create_dynamodb_client(context):
    if "dynamodb_client" in context.aws:
        return context.aws["dynamodb_client"]

    if 'sts_creds' in context.aws:
        client = boto3.client('dynamodb',
                              aws_access_key_id=context.aws["sts_creds"]["AccessKeyId"],
                              aws_secret_access_key=context.aws["sts_creds"]["SecretAccessKey"],
                              aws_session_token=context.aws["sts_creds"]["SessionToken"],
                              )
    else:
        client = boto3.client('dynamodb')

    context.aws["dynamodb_client"] = client

    return client


def dynamodb_tag(context, tag):
    dynamodb_client = create_dynamodb_client(context)
    # retreive tablename, key name(s) and value(s) from tag string
    # example: @dynamodb_delete_EmployeeRewardRulesTable_RuleName_test
    try:
        if tag.startswith("dynamodb_delete_"):

            tag_components = tag.split("_")

            try:
                table_name = tag_components[2]
            except IndexError:
                return

            keys = {}

            # partition key is required
            try:
                keys[tag_components[3]] = {
                    'S': tag_components[4]
                }
            except IndexError:
                return

            # sort key is optional
            try:
                keys[tag_components[5]] = {
                    'S': tag_components[6]
                }
            except IndexError:
                pass

            dynamodb_client.delete_item(
                TableName=get_logical_resource_details_on_datastack(context, table_name)[
                    "PhysicalResourceId"],
                Key=keys
            )

    except Exception:
        pass
