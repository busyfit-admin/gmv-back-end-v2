from __future__ import print_function
import os
import time
import json
import boto3
import csv
import codecs

s3_client = boto3.resource('s3')
ddbClient = boto3.resource('dynamodb')

employee_table = os.getenv("EMPLOYEE_DATA_TABLE")

def create_employees_in_table(event, context):
    bucket = event['Records'][0]['s3']['bucket']['name']
    csv_file_name = event['Records'][0]['s3']['object']['key']
    print("Onboarding Employee Data from file: " + csv_file_name)
    try:
        csv_object = s3_client.Object(bucket_name=bucket,key=csv_file_name).get()["Body"]
        obj = codecs.getreader('utf-8')(csv_object)
        fileContents = csv.DictReader(obj)
        table=ddbClient.Table(employee_table)
        for employee in fileContents:
            table.put_item(Item=employee)
        print("successfully Inserted/Updated the Employee Data from File " + csv_file_name)
        return 'success'
    except Exception as e:
        print(e)
        print('Error getting object {} from bucket {}. Make sure they exist and your bucket is in the same region as this function.'.format(csv_file_name, bucket))
        raise e