#!/usr/bin/python3

import boto3
import json

# installed
from behave import given, then
from hamcrest import assert_that, equal_to

from steps.cfn_step import get_logical_resource_details
from steps.common import read_file_path

@given("the SQS queue {queue_name}")
def step_impl_get_sqs_queue(context, queue_name):
    try:
        if queue_name not in context.aws["sqs"]:
            queue_details = get_logical_resource_details(context, queue_name)
            context.aws["sqs"][queue_name] = queue_details["PhysicalResourceId"]
    except Exception as exc:
        raise Exception(f"error getting finding SQS queue {queue_name}") from exc

    purge_queue(context, queue_name)

# delete leftover messages from previous test iteratively.
# the sqs_client.purge function has a rate limit 1 request/60s so we can't use it.
def purge_queue(context, queue_name):
    sqs_client = create_sqs_client(context)
    queue_url = context.aws["sqs"][queue_name]

    response = sqs_client.receive_message(
        QueueUrl=queue_url,
        MaxNumberOfMessages=10
    )
    if "Messages" in response:
        for message in response["Messages"]:
            sqs_client.delete_message(
                QueueUrl=queue_url,
                ReceiptHandle=message["ReceiptHandle"]
            )

@when("the sqs queue {queue_name} has message body {message_body_path}")
def step_impl_get_sqs_queue(context, queue_name, message_body_path):
    contents = json.loads(read_file_path(message_body_path))
    sqs_client = create_sqs_client(context)
    queue_url = context.aws["sqs"][queue_name]
    try:
        # Send message to SQS queue
        response = sqs.send_message(
            QueueUrl=queue_url,
            MessageBody=contents
        )
        return response
    except (BotoCoreError, ClientError) as error:
        print(f"An error occurred: {error}")
        return None



def create_sqs_client(context):
    if "sqs_client" in context.aws:
        return context.aws["sqs_client"]

    if 'sts_creds' in context.aws:
        events_client = boto3.client('sqs',
                                     aws_access_key_id=context.aws["sts_creds"]["AccessKeyId"],
                                     aws_secret_access_key=context.aws["sts_creds"]["SecretAccessKey"],
                                     aws_session_token=context.aws["sts_creds"]["SessionToken"],
                                     )
    else:
        events_client = boto3.client('sqs')
    context.aws["sqs_client"] = events_client
    return events_client