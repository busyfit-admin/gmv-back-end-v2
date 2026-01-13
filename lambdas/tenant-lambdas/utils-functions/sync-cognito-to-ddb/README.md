# Sync Cognito to DynamoDB Lambda

This Lambda function synchronizes Cognito user data to the EmployeeDataTable in DynamoDB.

## Features

- **PostConfirmation Trigger**: Creates new user in DynamoDB when user confirms signup
- **PreAuthentication Trigger**: Updates user data on every login
- **PreTokenGeneration Trigger**: Syncs user attributes before generating/refreshing tokens

## Architecture

The function handles three Cognito trigger events:
1. `PostConfirmation_ConfirmSignUp` - Creates new user
2. `PreAuthentication_Authentication` - Updates existing user on login
3. `TokenGeneration_Authentication` / `TokenGeneration_RefreshTokens` - Updates user on token operations

## Environment Variables

- `EMPLOYEE_TABLE` - Name of the DynamoDB table to sync users to (required)

## Testing

Run unit tests:

```bash
# Install dependencies
pip install -r requirements.txt

# Run tests
python -m pytest test_sync_cognito_to_ddb.py -v

# Run with coverage
python -m pytest test_sync_cognito_to_ddb.py --cov=sync_cognito_to_ddb --cov-report=html
```

Or run with unittest:

```bash
python -m unittest test_sync_cognito_to_ddb.py -v
```

## Test Coverage

The test suite includes:
- User data extraction tests
- User creation tests
- User update tests
- Trigger handler tests
- Main handler routing tests
- Error handling tests
- Integration tests simulating full user lifecycle

## IAM Permissions Required

The Lambda execution role needs:
- `dynamodb:PutItem` - Create new users
- `dynamodb:UpdateItem` - Update existing users
- `dynamodb:GetItem` - Check if user exists
- `logs:CreateLogGroup`, `logs:CreateLogStream`, `logs:PutLogEvents` - CloudWatch Logs

## Error Handling

The function is designed to never block user authentication. If an error occurs during sync:
- Error is logged to CloudWatch
- Full stack trace is printed
- Event is still returned to Cognito to allow authentication to proceed
