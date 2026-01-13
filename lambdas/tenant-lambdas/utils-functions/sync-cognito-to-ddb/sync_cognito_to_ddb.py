"""
Cognito to DynamoDB Sync Lambda
Syncs Cognito user data to EmployeeDataTable on various triggers:
- PostConfirmation: Creates new user
- PreAuthentication: Updates user on login
- PreTokenGeneration: Updates user on token generation/refresh
"""

import json
import boto3
import os
from datetime import datetime
from botocore.exceptions import ClientError

# Initialize DynamoDB resource
dynamodb = boto3.resource('dynamodb')
table_name = os.environ.get('EMPLOYEE_TABLE')
table = dynamodb.Table(table_name) if table_name else None


def extract_user_data(event):
    """Extract user data from Cognito event."""
    user_attributes = event['request']['userAttributes']
    
    return {
        'cognito_username': event['userName'],
        'cognito_id': user_attributes.get('sub'),
        'email': user_attributes.get('email', ''),
        'user_name': user_attributes.get('custom:userName', event['userName']),
        'name': user_attributes.get('name', ''),
        'e_id': user_attributes.get('custom:E_ID', ''),
        'phone_number': user_attributes.get('phone_number', '')
    }


def create_user_in_ddb(user_data, source='Cognito-PostConfirmation'):
    """Create a new user in DynamoDB."""
    if not table:
        raise ValueError("EMPLOYEE_TABLE environment variable not set")
    
    item = {
        'UserName': user_data['cognito_username'],
        'CognitoId': user_data['cognito_id'],
        'EmailId': user_data['email'],
        'DisplayName': user_data['name'] or user_data['user_name'],
        'E_ID': user_data['e_id'],
        'PhoneNumber': user_data['phone_number'],
        'Status': 'Active',
        'CreatedAt': datetime.utcnow().isoformat(),
        'UpdatedAt': datetime.utcnow().isoformat(),
        'Source': source
    }
    
    table.put_item(Item=item)
    print(f"Created user {user_data['cognito_username']} (CognitoId: {user_data['cognito_id']}) in DynamoDB")
    return item


def update_user_in_ddb(user_data):
    """Update existing user in DynamoDB."""
    if not table:
        raise ValueError("EMPLOYEE_TABLE environment variable not set")
    
    try:
        # Check if user exists
        response = table.get_item(Key={'UserName': user_data['cognito_username']})
        
        if 'Item' in response:
            # User exists, update attributes
            update_expression = "SET UpdatedAt = :updated, EmailId = :email"
            expression_values = {
                ':updated': datetime.utcnow().isoformat(),
                ':email': user_data['email']
            }
            
            # Update optional fields if provided
            if user_data['name']:
                update_expression += ", DisplayName = :name"
                expression_values[':name'] = user_data['name']
            if user_data['phone_number']:
                update_expression += ", PhoneNumber = :phone"
                expression_values[':phone'] = user_data['phone_number']
            if user_data['e_id']:
                update_expression += ", E_ID = :eid"
                expression_values[':eid'] = user_data['e_id']
            
            table.update_item(
                Key={'UserName': user_data['cognito_username']},
                UpdateExpression=update_expression,
                ExpressionAttributeValues=expression_values
            )
            print(f"Updated user {user_data['cognito_username']} in DynamoDB")
            return True
        else:
            # User doesn't exist, create it
            create_user_in_ddb(user_data, source='Cognito-Sync')
            print(f"Created missing user {user_data['cognito_username']} in DynamoDB during authentication")
            return True
    
    except ClientError as e:
        if e.response['Error']['Code'] != 'ResourceNotFoundException':
            raise
        print(f"User {user_data['cognito_username']} not found in DynamoDB, skipping update")
        return False


def handle_post_confirmation(event):
    """Handle PostConfirmation trigger - create new user."""
    user_data = extract_user_data(event)
    create_user_in_ddb(user_data, source='Cognito-PostConfirmation')


def handle_pre_authentication(event):
    """Handle PreAuthentication trigger - update user on login."""
    user_data = extract_user_data(event)
    update_user_in_ddb(user_data)


def handle_pre_token_generation(event):
    """Handle PreTokenGeneration trigger - update user before token generation."""
    user_data = extract_user_data(event)
    update_user_in_ddb(user_data)


def handler(event, context):
    """Main Lambda handler for Cognito triggers."""
    trigger_source = event.get('triggerSource', '')
    print(f"Trigger Source: {trigger_source}")
    print(f"Event: {json.dumps(event)}")
    
    try:
        if trigger_source == 'PostConfirmation_ConfirmSignUp':
            handle_post_confirmation(event)
        
        elif trigger_source in ['PreAuthentication_Authentication']:
            handle_pre_authentication(event)
        
        elif trigger_source in ['TokenGeneration_Authentication', 'TokenGeneration_RefreshTokens']:
            handle_pre_token_generation(event)
        
        else:
            print(f"Unhandled trigger source: {trigger_source}")
        
        # Return event to Cognito (required)
        return event
    
    except Exception as e:
        print(f"Error syncing user to DynamoDB: {str(e)}")
        import traceback
        traceback.print_exc()
        # Still return event to not block user authentication
        return event
