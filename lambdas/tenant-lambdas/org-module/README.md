# Organization Module

This module contains Lambda functions for organization management including:

## API Endpoints

### 1. Create Organization
- **Path**: `/org`
- **Method**: `POST`
- **Description**: Creates a new organization with the requesting user as owner
- **Body**: Organization details including client info and industry data

### 2. Manage Organization
- **Path**: `/org/{orgId}`
- **Methods**: `GET`, `PUT`
- **Description**: Get or update organization details (admin only for updates)

### 3. Manage Subscription
- **Path**: `/org/{orgId}/subscription`
- **Methods**: `GET`, `PUT`
- **Description**: View available plans or update subscription (admin only)

### 4. Manage Promo Codes
- **Path**: `/org/{orgId}/promo`
- **Methods**: `POST`
- **Description**: Apply promo codes to organization (admin only)

### 5. List User Organizations
- **Path**: `/org/my-organizations`
- **Method**: `GET`
- **Description**: List all organizations where user is an admin

## Environment Variables

- `ORGANIZATION_TABLE`: DynamoDB table for organizations
- `PROMO_CODES_TABLE`: DynamoDB table for promo codes
- `EMPLOYEE_TABLE`: Employee table for user details
- `EMPLOYEE_TABLE_COGNITO_ID_INDEX`: GSI for employee lookup

## Authentication

All endpoints require Cognito JWT authentication. The Cognito ID is extracted from the authorizer context.