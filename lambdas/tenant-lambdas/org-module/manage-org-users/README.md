# Manage Org Users Lambda

This Lambda function provides endpoints for managing organization users (both admins and regular users) within the org-module of the tenant-lambdas.

## Overview

The `manage-org-users` endpoint allows organization admins to:
- List all users in an organization (admins and regular users)
- Add new users to the organization
- Update user roles (planned)
- Remove users from the organization

## HTTP Methods

### GET - List Organization Users
Lists all users (admins and regular users) in the organization.

**Headers:**
- `Organization-Id`: The organization ID (required)
- `Authorization`: Cognito JWT token (required)

**Response:**
```json
{
  "users": [
    {
      "userName": "user@example.com",
      "displayName": "John Doe",
      "role": "owner",
      "userType": "admin",
      "addedAt": "2024-01-01T00:00:00Z",
      "isActive": true
    }
  ],
  "totalCount": 10,
  "adminCount": 3,
  "userCount": 7
}
```

### POST - Add User
Adds a new user (admin or regular user) to the organization.

**Headers:**
- `Organization-Id`: The organization ID (required)
- `Authorization`: Cognito JWT token (required)

**Request Body:**
```json
{
  "userName": "newuser@example.com",
  "role": "admin",
  "userType": "admin"
}
```

**Valid Roles for Admins:**
- `owner` - Full access
- `admin` - Administrative access
- `manager` - Management access

**User Types:**
- `admin` - Organization administrator
- `user` - Regular organization user (not yet implemented)

**Response:**
```json
{
  "message": "User added successfully",
  "userName": "newuser@example.com",
  "role": "admin",
  "userType": "admin"
}
```

### PUT - Update User Role
Updates a user's role in the organization.

**Status:** Not yet implemented

**Headers:**
- `Organization-Id`: The organization ID (required)
- `Authorization`: Cognito JWT token (required)

**Request Body:**
```json
{
  "userName": "user@example.com",
  "role": "manager"
}
```

### DELETE - Remove User
Removes a user from the organization.

**Headers:**
- `Organization-Id`: The organization ID (required)
- `Authorization`: Cognito JWT token (required)

**Request Body:**
```json
{
  "userName": "user@example.com"
}
```

Or use query parameter: `?userName=user@example.com`

**Response:**
```json
{
  "message": "User removed successfully",
  "userName": "user@example.com"
}
```

## Permissions

- Only organization admins can access these endpoints
- Only organization owners can add or remove other admins
- Users can be removed by admins (owners can remove any admin except the last owner)

## Environment Variables

- `EMPLOYEE_TABLE` - DynamoDB table for employee data
- `EMPLOYEE_TABLE_COGNITO_ID_INDEX` - GSI for looking up employees by Cognito ID
- `ORGANIZATION_TABLE` - DynamoDB table for organization data
- `PROMO_CODES_TABLE` - DynamoDB table for promo codes

## Build and Deploy

```bash
# Build the Lambda
make build

# Clean build artifacts
make clean

# Tidy dependencies
make tidy
```

## Notes

- Adding regular users (non-admin) is planned but not yet implemented
- Updating user roles is planned but not yet implemented
- The endpoint follows the existing org-module patterns for authentication and authorization
- All operations require the requesting user to be an admin of the organization
