# Cognito User Creation with First Name, Last Name, and Phone Number

This document shows how to create users with the new fields that will sync to DynamoDB.

## Fields Supported

The Cognito User Pool now supports the following fields:

### Standard Attributes
- `email` - Email address (required, used as username)
- `given_name` - First name (synced to `FirstName` in DynamoDB)
- `family_name` - Last name (synced to `LastName` in DynamoDB)
- `name` - Full name (synced to `DisplayName` in DynamoDB)
- `phone_number` - Phone number (synced to `PhoneNumber` in DynamoDB)

### Custom Attributes
- `custom:userName` - Custom username
- `custom:E_ID` - Employee ID

## Creating a User via AWS CLI

### Example 1: User with First Name and Last Name

```bash
aws cognito-idp admin-create-user \
  --user-pool-id <USER_POOL_ID> \
  --username john.doe@example.com \
  --user-attributes \
    Name=email,Value=john.doe@example.com \
    Name=given_name,Value=John \
    Name=family_name,Value=Doe \
    Name=phone_number,Value=+1234567890 \
    Name=custom:E_ID,Value=EMP001 \
  --message-action SUPPRESS
```

**What syncs to DynamoDB:**
- `UserName`: john.doe@example.com
- `EmailId`: john.doe@example.com
- `FirstName`: John
- `LastName`: Doe
- `DisplayName`: John Doe (auto-built from given_name + family_name)
- `PhoneNumber`: +1234567890
- `E_ID`: EMP001

### Example 2: User with Full Name

```bash
aws cognito-idp admin-create-user \
  --user-pool-id <USER_POOL_ID> \
  --username jane.smith@example.com \
  --user-attributes \
    Name=email,Value=jane.smith@example.com \
    Name=name,Value="Jane Smith" \
    Name=phone_number,Value=+9876543210 \
  --message-action SUPPRESS
```

**What syncs to DynamoDB:**
- `UserName`: jane.smith@example.com
- `EmailId`: jane.smith@example.com
- `DisplayName`: Jane Smith
- `FirstName`: (empty)
- `LastName`: (empty)
- `PhoneNumber`: +9876543210

### Example 3: Complete User with All Fields

```bash
aws cognito-idp admin-create-user \
  --user-pool-id <USER_POOL_ID> \
  --username alice.johnson@example.com \
  --user-attributes \
    Name=email,Value=alice.johnson@example.com \
    Name=given_name,Value=Alice \
    Name=family_name,Value=Johnson \
    Name=name,Value="Alice M. Johnson" \
    Name=phone_number,Value=+15551234567 \
    Name=custom:userName,Value=ajohnson \
    Name=custom:E_ID,Value=EMP123 \
  --message-action SUPPRESS
```

**What syncs to DynamoDB:**
- `UserName`: alice.johnson@example.com
- `EmailId`: alice.johnson@example.com
- `FirstName`: Alice
- `LastName`: Johnson
- `DisplayName`: Alice M. Johnson
- `PhoneNumber`: +15551234567
- `E_ID`: EMP123

## Setting User Password

After creating the user, set a permanent password:

```bash
aws cognito-idp admin-set-user-password \
  --user-pool-id <USER_POOL_ID> \
  --username john.doe@example.com \
  --password YourSecurePassword123! \
  --permanent
```

## Updating User Attributes

You can update user attributes which will sync to DynamoDB on next login:

```bash
aws cognito-idp admin-update-user-attributes \
  --user-pool-id <USER_POOL_ID> \
  --username john.doe@example.com \
  --user-attributes \
    Name=given_name,Value=Jonathan \
    Name=phone_number,Value=+19998887777
```

## Sync Behavior

### When Sync Happens

1. **PostConfirmation** - When user confirms their account (creates in DDB)
2. **PreAuthentication** - Every time user logs in (updates in DDB)
3. **PreTokenGeneration** - When generating or refreshing tokens (updates in DDB)

### Auto-Build Name Logic

If `name` field is not provided but `given_name` and `family_name` are:
- The sync Lambda will automatically build `DisplayName` as `given_name + " " + family_name`
- Example: given_name="John" + family_name="Doe" → DisplayName="John Doe"

## Verifying Sync in DynamoDB

Check if user synced correctly:

```bash
aws dynamodb get-item \
  --table-name EmployeeDataTable-dev \
  --key '{"UserName": {"S": "john.doe@example.com"}}' \
  --query 'Item.{UserName:UserName.S,Email:EmailId.S,FirstName:FirstName.S,LastName:LastName.S,Phone:PhoneNumber.S}' \
  --output table
```

Expected output:
```
-------------------------------------------------------------------
|                            GetItem                              |
+-----------------+-------------+----------+-----------+-----------+
|      Email      | FirstName   | LastName |   Phone   | UserName  |
+-----------------+-------------+----------+-----------+-----------+
| john@example.com|  John       |  Doe     |+1234567890|john@...   |
+-----------------+-------------+----------+-----------+-----------+
```

## Testing the Changes

After deploying the updated stack:

1. **Create a test user:**
   ```bash
   aws cognito-idp admin-create-user \
     --user-pool-id $(aws cloudformation describe-stack-resources \
       --stack-name tenant-portal-apis-dev \
       --logical-resource-id TenantCognitoUserPool \
       --query 'StackResources[0].PhysicalResourceId' --output text) \
     --username test@example.com \
     --user-attributes \
       Name=email,Value=test@example.com \
       Name=given_name,Value=Test \
       Name=family_name,Value=User \
       Name=phone_number,Value=+1234567890 \
     --message-action SUPPRESS
   ```

2. **Set password:**
   ```bash
   aws cognito-idp admin-set-user-password \
     --user-pool-id <USER_POOL_ID> \
     --username test@example.com \
     --password TestPassword123! \
     --permanent
   ```

3. **Verify in DynamoDB:**
   ```bash
   aws dynamodb get-item \
     --table-name EmployeeDataTable-dev \
     --key '{"UserName": {"S": "test@example.com"}}'
   ```

4. **Check sync worked:**
   - FirstName should be "Test"
   - LastName should be "User"
   - DisplayName should be "Test User"
   - PhoneNumber should be "+1234567890"

## Changes Summary

### Cognito User Pool Schema (`template.yaml`)
- ✅ Added `given_name` (first name)
- ✅ Added `family_name` (last name)
- ✅ Kept `phone_number` (already existed)
- ✅ Kept all custom attributes

### Sync Lambda (`sync_cognito_to_ddb.py`)
- ✅ Extracts `given_name` and `family_name` from Cognito
- ✅ Auto-builds `name` from first + last if not provided
- ✅ Syncs `FirstName` to DynamoDB
- ✅ Syncs `LastName` to DynamoDB
- ✅ Syncs `PhoneNumber` to DynamoDB
- ✅ Updates all fields on login and token generation

### Tests
- ✅ All 16 tests passing
- ✅ Added test for name auto-build from first + last
- ✅ Updated all tests to include new fields

## Phone Number Format

Phone numbers should be in E.164 format:
- ✅ `+1234567890` (correct)
- ✅ `+919876543210` (correct)
- ❌ `1234567890` (missing +)
- ❌ `(123) 456-7890` (formatting not supported)

## Deployment

To apply these changes:

```bash
# Deploy the updated stack
cd cfn/tenant-cfn
make deploy ENVIRONMENT=dev
```

The Lambda function code will be automatically packaged and updated.

## Troubleshooting

### User created but fields not in DynamoDB

Check CloudWatch Logs:
```bash
aws logs tail /aws/lambda/SyncCognitoUserToDynamoDB-dev --follow
```

### Phone number not syncing

- Ensure phone number is in E.164 format (`+1234567890`)
- Check Cognito attribute name is exactly `phone_number`

### First/Last name not syncing

- Use `given_name` and `family_name` (not firstName/lastName)
- These are standard Cognito attributes
