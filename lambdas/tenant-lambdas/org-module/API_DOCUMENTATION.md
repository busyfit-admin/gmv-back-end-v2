# Organization Management API Documentation

## Overview
The Organization Management API provides comprehensive endpoints for managing organizations, subscriptions, billing, and promotional codes within the tenant API Gateway. All endpoints are prefixed with `/v2/` and require JWT authentication via AWS Cognito.

## Organization Performance APIs

The Organization Performance APIs are now exposed under `/v2` and mapped to the `manage-org-performance` Lambda. They include:

- Performance Cycles CRUD
- Quarters CRUD
- KPI CRUD, Sub-KPI, KPI value updates
- OKR CRUD, Key Result progress updates
- Meeting Notes CRUD
- Cycle/Quarter analytics
- Goals endpoints (details, value history, tagged teams, sub-items, ladder-up workflow, private tasks)

### Implementation Source Links

- Swagger Paths: `swagger-docs/tenant/tenant-apis.yaml`
- CFN Lambda Wiring: `cfn/tenant-cfn/template.yaml`
- Lambda Handler: `lambdas/tenant-lambdas/org-module/manage-org-performance/manage-org-performance.go`
- Core Business Logic: `lambdas/lib/company-lib/company-org-performance.go`

## Base URL
All endpoints are relative to your API Gateway base URL with the `/v2/` prefix:
```
https://{api-gateway-id}.execute-api.{region}.amazonaws.com/{stage}/v2/
```

## Authentication
All endpoints require a valid Cognito JWT token in the Authorization header:
```
Authorization: Bearer <cognito-jwt-token>
```

## API Endpoints

### 1. Create Organization
**Endpoint:** `POST /v2/organization`  
**Function:** Creates a new organization with the requesting user as owner/admin

**Request Body:**
```json
{
  "organizationName": "string (required)",
  "displayName": "string (optional)",
  "industry": "string (optional)",
  "description": "string (optional)",
  "clientDetails": {
    "companySize": "string (optional)", // "1-10", "11-50", "51-200", "201-1000", "1000+"
    "headquarters": "string (optional)",
    "website": "string (optional)",
    "primaryContact": {
      "name": "string (optional)",
      "email": "string (optional)",
      "phone": "string (optional)"
    },
    "businessType": "string (optional)", // "B2B", "B2C", "B2B2C"
    "annualRevenue": "string (optional)" // "<$1M", "$1M-10M", "$10M-50M", "$50M-100M", "$100M+"
  },
  "subscriptionPlan": "string (required)", // "BASIC", "PROFESSIONAL", "ENTERPRISE"
  "billingCycle": "string (required)" // "MONTHLY", "YEARLY"
}
```

**Success Response (201):**
```json
{
  "message": "Organization created successfully",
  "organization": {
    "organizationId": "ORG#uuid",
    "orgName": "string",
    "orgDesc": "string",
    "clientName": "string",
    "industry": "string",
    "companySize": "string",
    "website": "string",
    "contactEmail": "string",
    "contactPhone": "string",
    "address": "string",
    "city": "string",
    "state": "string",
    "country": "string",
    "zipCode": "string",
    "taxId": "string",
    "billingMode": "FREE",
    "subscriptionType": "TRIAL",
    "billingPlan": "MONTHLY",
    "orgBillingStatus": "TRIAL",
    "currentPlanId": "starter",
    "planType": "starter",
    "currentTeamCount": 0,
    "maxTeamsAllowed": 5,
    "maxMembersAllowed": 25,
    "appliedPromoCode": "",
    "promoDiscountPercent": 0,
    "promoValidUntil": "",
    "trialStartDate": "2024-01-01T00:00:00Z",
    "trialEndDate": "2024-01-31T00:00:00Z",
    "billingStartDate": "",
    "nextBillingDate": "",
    "lastPaymentDate": "",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z",
    "creatorUserName": "user-123"
  }
}
```

**Note:** Admin users are now stored separately. Use the "Get Organization Admins" endpoint to retrieve admin information.

---

### 2. Get Organization Details
**Endpoint:** `GET /v2/organization/{organizationId}`  
**Function:** Retrieves organization details (admin only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID

**Success Response (200):**
```json
{
  "organization": {
    // Same structure as create organization response
  }
}
```

---

### 3. Update Organization
**Endpoint:** `PUT /v2/organization/{organizationId}`  
**Function:** Updates organization details (admin only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID

**Request Body:**
```json
{
  "displayName": "string (optional)",
  "industry": "string (optional)",
  "description": "string (optional)",
  "clientDetails": {
    "companySize": "string (optional)",
    "headquarters": "string (optional)",
    "website": "string (optional)",
    "primaryContact": {
      "name": "string (optional)",
      "email": "string (optional)",
      "phone": "string (optional)"
    },
    "businessType": "string (optional)",
    "annualRevenue": "string (optional)"
  }
}
```

**Success Response (200):**
```json
{
  "message": "Organization updated successfully",
  "organization": {
    // Updated organization object
  }
}
```

---

### 4. Get Subscription Plans
**Endpoint:** `GET /v2/organization/{organizationId}/subscription`  
**Function:** Retrieves current subscription details and available plans (admin only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID

**Success Response (200):**
```json
{
  "success": true,
  "currentPlan": {
    "planType": "PROFESSIONAL",
    "billingCycle": "MONTHLY",
    "price": 49.99,
    "teamLimit": 10,
    "isActive": true,
    "nextBillingDate": "2024-02-01T00:00:00Z"
  },
  "availablePlans": [
    {
      "planType": "BASIC",
      "features": [
        "Up to 3 teams",
        "Basic reporting",
        "Email support"
      ],
      "teamLimit": 3,
      "pricing": {
        "monthly": 19.99,
        "yearly": 199.99,
        "yearlyDiscount": 16.67
      }
    },
    {
      "planType": "PROFESSIONAL",
      "features": [
        "Up to 10 teams",
        "Advanced reporting",
        "Priority support",
        "Custom integrations"
      ],
      "teamLimit": 10,
      "pricing": {
        "monthly": 49.99,
        "yearly": 499.99,
        "yearlyDiscount": 16.67
      }
    },
    {
      "planType": "ENTERPRISE",
      "features": [
        "Unlimited teams",
        "Enterprise reporting",
        "24/7 support",
        "Advanced security"
      ],
      "teamLimit": -1,
      "pricing": {
        "monthly": 99.99,
        "yearly": 999.99,
        "yearlyDiscount": 16.67
      }
    }
  ]
}
```

---

### 5. Update Subscription
**Endpoint:** `PUT /v2/organization/{organizationId}/subscription`  
**Function:** Updates organization subscription plan (admin only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID

**Request Body:**
```json
{
  "planType": "string (required)", // "BASIC", "PROFESSIONAL", "ENTERPRISE"
  "billingCycle": "string (required)" // "MONTHLY" or "YEARLY"
}
```

**Success Response (200):**
```json
{
  "message": "Subscription updated successfully",
  "subscription": {
    "planId": "professional",
    "planName": "Professional Plan",
    "billingMode": "PAID",
    "subscriptionType": "SUBSCRIPTION",
    "billingPlan": "YEARLY",
    "billingStatus": "ACTIVE",
    "maxTeams": 25,
    "maxMembers": 150,
    "nextBillingDate": "2025-01-01T00:00:00Z"
  }
}
```

---

### 6. Apply Promo Code
**Endpoint:** `POST /v2/organization/{organizationId}/promo-code`  
**Function:** Applies a promotional code to an organization (admin only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID

**Request Body:**
```json
{
  "promoCode": "string (required)" // e.g., "SAVE20", "TRIAL30"
}
```

**Success Response (200):**
```json
{
  "message": "Promo code applied successfully",
  "organization": {
    "orgId": "ORG#uuid",
    "appliedPromoCode": "SAVE20",
    "promoDiscountPercent": 20.0,
    "promoValidUntil": "2024-12-31T23:59:59Z",
    "trialEndDate": "2024-02-15T00:00:00Z", // Extended if promo includes trial days
    "orgBillingStatus": "ACTIVE",
    "nextBillingDate": "2024-02-01T00:00:00Z"
  },
  "promoCodeDetails": {
    "promoCode": "SAVE20",
    "discountPercent": 20.0,
    "discountAmount": 0,
    "freeTrialDays": 0,
    "validUntil": "2024-12-31T23:59:59Z"
  }
}
```

---

### 7. Get Active Promo Code Details
**Endpoint:** `GET /v2/organization/{organizationId}/promo-code`  
**Function:** Gets details of currently active promo code for the organization (admin only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID

**Success Response (200):**
```json
{
  "promoCode": {
    "code": "SAVE20",
    "discountPercent": 20.0,
    "discountAmount": 0,
    "validFrom": "2024-01-01T00:00:00Z",
    "validUntil": "2024-12-31T23:59:59Z",
    "maxUsages": 1000,
    "currentUsages": 245,
    "freeTrialDays": 0,
    "applicablePlans": ["starter", "professional"],
    "isActive": true,
    "isEligible": true,
    "isApplicable": true
  },
  "organization": {
    "currentPlan": "starter",
    "appliedPromoCode": "",
    "billingStatus": "TRIAL"
  }
}
```

---

### 8. Get Organization Admins
**Endpoint:** `GET /v2/organization/{organizationId}/admins`  
**Function:** Lists all active admins for an organization (admin only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID

**Success Response (200):**
```json
{
  "admins": [
    {
      "userName": "user-123",
      "displayName": "John Doe",
      "role": "OWNER",
      "addedAt": "2024-01-01T00:00:00Z",
      "isActive": true,
      "updatedAt": "2024-01-01T00:00:00Z"
    },
    {
      "userName": "user-456",
      "displayName": "Jane Smith",
      "role": "ADMIN",
      "addedAt": "2024-01-15T10:30:00Z",
      "isActive": true,
      "updatedAt": "2024-01-15T10:30:00Z"
    }
  ],
  "totalCount": 2
}
```

**Admin Roles:**
- `OWNER`: Full organization control, can manage all settings and admins
- `ADMIN`: Can manage organization settings and subscriptions
- `BILLING_ONLY`: Can only manage billing and subscription details

---

### 9. Add Organization Admin
**Endpoint:** `POST /v2/organization/{organizationId}/admins`  
**Function:** Adds a new admin to the organization (owner only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID

**Request Body:**
```json
{
  "userName": "string (required)",
  "role": "string (required)" // "OWNER", "ADMIN", or "BILLING_ONLY"
}
```

**Success Response (201):**
```json
{
  "message": "Admin added successfully",
  "admin": {
    "userName": "user-789",
    "displayName": "Bob Johnson",
    "role": "ADMIN",
    "addedAt": "2024-01-20T14:00:00Z",
    "isActive": true
  }
}
```

**Business Rules:**
- Only organization owners can add admins
- Users can only belong to one organization (enforced automatically)
- If user is already in another organization, they must be removed first

---

### 10. Remove Organization Admin
**Endpoint:** `DELETE /v2/organization/{organizationId}/admins/{userName}`  
**Function:** Removes (deactivates) an admin from the organization (owner only)

**Path Parameters:**
- `organizationId` (string, required): Organization ID
- `userName` (string, required): Username of admin to remove

**Success Response (200):**
```json
{
  "message": "Admin removed successfully"
}
```

**Business Rules:**
- Only organization owners can remove admins
- Cannot remove the last owner from an organization
- Removing an admin deactivates them rather than deleting the record

---

### 11. List User Organizations
**Endpoint:** `GET /v2/organization/list`  
**Function:** Lists all organizations where the user is a member

**Success Response (200):**
```json
{
  "organizations": [
    {
      "organizationId": "ORG#uuid1",
      "orgName": "Acme Corporation",
      "orgDesc": "Technology solutions company",
      "clientName": "Acme Corp",
      "industry": "Technology",
      "companySize": "51-200",
      "website": "https://acme.com",
      "contactEmail": "admin@acme.com",
      "contactPhone": "+1-555-0123",
      "address": "123 Main St",
      "city": "San Francisco",
      "state": "CA",
      "country": "USA",
      "zipCode": "94105",
      "taxId": "12-3456789",
      "billingMode": "PAID",
      "subscriptionType": "SUBSCRIPTION",
      "billingPlan": "YEARLY",
      "orgBillingStatus": "ACTIVE",
      "currentPlanId": "professional",
      "planType": "professional",
      "currentTeamCount": 8,
      "maxTeamsAllowed": 25,
      "maxMembersAllowed": 150,
      "appliedPromoCode": "SAVE20",
      "promoDiscountPercent": 20.0,
      "promoValidUntil": "2024-12-31T23:59:59Z",
      "trialStartDate": "2024-01-01T00:00:00Z",
      "trialEndDate": "2024-01-31T00:00:00Z",
      "billingStartDate": "2024-02-01T00:00:00Z",
      "nextBillingDate": "2025-02-01T00:00:00Z",
      "lastPaymentDate": "2024-12-01T00:00:00Z",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-15T10:30:00Z",
      "creatorUserName": "user-123"
    }
  ],
  "totalCount": 1,
  "availablePlans": [
    {
      "planId": "starter",
      "planName": "Starter Plan",
      "planDescription": "Perfect for small teams just getting started",
      "maxTeams": 5,
      "maxMembers": 25,
      "monthlyPrice": 29.99,
      "yearlyPrice": 299.99,
      "features": [
        "Basic team management",
        "Email support",
        "5 teams",
        "25 members"
      ]
    },
    {
      "planId": "professional",
      "planName": "Professional Plan",
      "planDescription": "Great for growing organizations",
      "maxTeams": 25,
      "maxMembers": 150,
      "monthlyPrice": 79.99,
      "yearlyPrice": 799.99,
      "features": [
        "Advanced team management",
        "Priority support",
        "25 teams",
        "150 members",
        "Analytics dashboard"
      ]
    },
    {
      "planId": "enterprise",
      "planName": "Enterprise Plan",
      "planDescription": "For large organizations with advanced needs",
      "maxTeams": -1,
      "maxMembers": -1,
      "monthlyPrice": 199.99,
      "yearlyPrice": 1999.99,
      "features": [
        "Unlimited teams",
        "Unlimited members",
        "24/7 support",
        "Custom integrations",
        "Advanced analytics"
      ]
    }
  ]
}
```

**Note:** Users can only belong to one organization in this release. The API enforces this constraint automatically.

---

### 12. Check if User is Organization Admin
**Endpoint:** `GET /v2/organization/check-admin`  
**Function:** Checks if the authenticated user is an organization admin and returns organization details if true

**Success Response (200) - User IS an Org Admin:**
```json
{
  "isOrgAdmin": true,
  "organizationId": "ORG#uuid",
  "orgName": "Acme Corporation",
  "orgDesc": "Technology solutions company",
  "role": "OWNER"
}
```

**Success Response (200) - User IS NOT an Org Admin:**
```json
{
  "isOrgAdmin": false
}
```

**Role Values:**
- `OWNER`: Full organization control
- `ADMIN`: Can manage organization settings
- `BILLING_ONLY`: Can only manage billing

**Use Cases:**
- Frontend conditional rendering (show/hide admin features)
- Quick permission checks before operations
- Navigation guards and route protection
- Dashboard initialization
- Feature flag evaluation

---

### 13. Manage Organization Users
**Base Endpoint:** `/v2/organization/users`  
**Function:** Comprehensive user management for organizations (admins and regular users)

#### 13.1 List All Organization Users
**Endpoint:** `GET /v2/organization/users`  
**Headers Required:**
- `Organization-Id`: Organization identifier
- `Authorization`: Bearer token

**Success Response (200):**
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
    },
    {
      "userName": "user2@example.com",
      "displayName": "Jane Smith",
      "role": "member",
      "userType": "user",
      "joinedAt": "2024-01-05T00:00:00Z",
      "isActive": true
    }
  ],
  "totalCount": 2,
  "adminCount": 1,
  "userCount": 1
}
```

**User Types:**
- `admin`: Organization administrator with elevated privileges
- `user`: Regular organization user

**Admin Roles:**
- `owner`: Full organization control
- `admin`: Administrative access
- `manager`: Management-level access

---

#### 13.2 Add User to Organization
**Endpoint:** `POST /v2/organization/users`  
**Headers Required:**
- `Organization-Id`: Organization identifier
- `Authorization`: Bearer token

**Request Body:**
```json
{
  "userName": "newuser@example.com",
  "role": "admin",
  "userType": "admin"
}
```

**Field Descriptions:**
- `userName` (required): Email address or username of the user to add
- `role` (required): Role to assign (owner/admin/manager for admin type)
- `userType` (optional): "admin" or "user" (defaults to "user")

**Success Response (201):**
```json
{
  "message": "User added successfully",
  "userName": "newuser@example.com",
  "role": "admin",
  "userType": "admin"
}
```

**Permissions:**
- Only organization admins can add users
- Only organization owners can add other admins

---

#### 13.3 Update User Role
**Endpoint:** `PUT /v2/organization/users`  
**Status:** Not yet implemented  
**Headers Required:**
- `Organization-Id`: Organization identifier
- `Authorization`: Bearer token

**Request Body:**
```json
{
  "userName": "user@example.com",
  "role": "manager"
}
```

---

#### 13.4 Remove User from Organization
**Endpoint:** `DELETE /v2/organization/users`  
**Headers Required:**
- `Organization-Id`: Organization identifier
- `Authorization`: Bearer token

**Request Body:**
```json
{
  "userName": "user@example.com"
}
```

**Alternate Method:** Query parameter
```
DELETE /v2/organization/users?userName=user@example.com
```

**Success Response (200):**
```json
{
  "message": "User removed successfully",
  "userName": "user@example.com"
}
```

**Permissions:**
- Only organization admins can remove users
- Only organization owners can remove other admins
- Cannot remove the last owner from the organization

---

## Error Responses

All endpoints return consistent error responses:

### 400 Bad Request
```json
{
  "error": "Bad Request",
  "message": "Organization name is required"
}
```

### 401 Unauthorized
```json
{
  "error": "Unauthorized",
  "message": "Cognito ID not found in request"
}
```

### 403 Forbidden
```json
{
  "error": "Forbidden",
  "message": "Access denied: Not an organization admin"
}
```

### 404 Not Found
```json
{
  "error": "Not Found",
  "message": "Organization not found"
}
```

### 500 Internal Server Error
```json
{
  "error": "Internal Server Error",
  "message": "Failed to create organization"
}
```

---

## Usage Examples

### Frontend JavaScript Examples

#### Create Organization
```javascript
const createOrganization = async (orgData) => {
  const response = await fetch('/v2/organization', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${cognitoToken}`
    },
    body: JSON.stringify(orgData)
  });
  
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }
  
  return await response.json();
};

// Usage
const newOrg = await createOrganization({
  organizationName: "My Company",
  displayName: "My Company",
  industry: "Technology",
  subscriptionPlan: "PROFESSIONAL",
  billingCycle: "MONTHLY"
});
```

#### Get User Organizations
```javascript
const getUserOrganizations = async () => {
  const response = await fetch('/v2/organization/list', {
    headers: {
      'Authorization': `Bearer ${cognitoToken}`
    }
  });
  
  return await response.json();
};
```

#### Check if User is Organization Admin
```javascript
const checkIsOrgAdmin = async () => {
  const response = await fetch('/v2/organization/check-admin', {
    headers: {
      'Authorization': `Bearer ${cognitoToken}`
    }
  });
  
  const data = await response.json();
  return data;
};

// Usage example
const adminStatus = await checkIsOrgAdmin();
if (adminStatus.isOrgAdmin) {
  console.log(`User is ${adminStatus.role} of ${adminStatus.orgName}`);
  // Show admin features
  enableAdminFeatures(adminStatus.organizationId);
} else {
  console.log('User is not an organization admin');
  // Hide admin features
}
```

#### Update Subscription
```javascript
const updateSubscription = async (organizationId, planType, billingCycle) => {
  const response = await fetch(`/v2/organization/${organizationId}/subscription`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${cognitoToken}`
    },
    body: JSON.stringify({
      planType: planType,
      billingCycle: billingCycle
    })
  });
  
  return await response.json();
};
```

#### Apply Promo Code
```javascript
const applyPromoCode = async (organizationId, promoCode) => {
  const response = await fetch(`/v2/organization/${organizationId}/promo-code`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${cognitoToken}`
    },
    body: JSON.stringify({
      promoCode: promoCode
    })
  });
  
  return await response.json();
};
```

#### Get Organization Admins
```javascript
const getOrgAdmins = async (organizationId) => {
  const response = await fetch(`/v2/organization/${organizationId}/admins`, {
    headers: {
      'Authorization': `Bearer ${cognitoToken}`
    }
  });
  
  return await response.json();
};
```

#### Add Organization Admin
```javascript
const addOrgAdmin = async (organizationId, userName, role) => {
  const response = await fetch(`/v2/organization/${organizationId}/admins`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${cognitoToken}`
    },
    body: JSON.stringify({
      userName: userName,
      role: role // "OWNER", "ADMIN", or "BILLING_ONLY"
    })
  });
  
  return await response.json();
};
```

#### Remove Organization Admin
```javascript
const removeOrgAdmin = async (organizationId, userName) => {
  const response = await fetch(`/v2/organization/${organizationId}/admins/${userName}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${cognitoToken}`
    }
  });
  
  return await response.json();
};
```

#### List All Organization Users
```javascript
const listOrgUsers = async (organizationId) => {
  const response = await fetch(`/v2/organization/users`, {
    headers: {
      'Organization-Id': organizationId,
      'Authorization': `Bearer ${cognitoToken}`
    }
  });
  
  const data = await response.json();
  console.log(`Total users: ${data.totalCount} (${data.adminCount} admins, ${data.userCount} users)`);
  return data;
};
```

#### Add User to Organization
```javascript
const addOrgUser = async (organizationId, userName, role, userType = 'admin') => {
  const response = await fetch(`/v2/organization/users`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Organization-Id': organizationId,
      'Authorization': `Bearer ${cognitoToken}`
    },
    body: JSON.stringify({
      userName: userName,
      role: role, // "owner", "admin", "manager"
      userType: userType // "admin" or "user"
    })
  });
  
  return await response.json();
};

// Usage example
const result = await addOrgUser('ORG#123', 'newuser@example.com', 'admin', 'admin');
console.log(result.message); // "User added successfully"
```

#### Remove User from Organization
```javascript
const removeOrgUser = async (organizationId, userName) => {
  const response = await fetch(`/v2/organization/users`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      'Organization-Id': organizationId,
      'Authorization': `Bearer ${cognitoToken}`
    },
    body: JSON.stringify({
      userName: userName
    })
  });
  
  return await response.json();
};

// Alternate method using query parameters
const removeOrgUserAlt = async (organizationId, userName) => {
  const response = await fetch(`/v2/organization/users?userName=${encodeURIComponent(userName)}`, {
    method: 'DELETE',
    headers: {
      'Organization-Id': organizationId,
      'Authorization': `Bearer ${cognitoToken}`
    }
  });
  
  return await response.json();
};
```

---

## Environment Variables Required

For deployment, ensure these environment variables are set:

- `ORGANIZATION_TABLE`: DynamoDB table name for organizations
- `PROMO_CODES_TABLE`: DynamoDB table name for promo codes  
- `EMPLOYEE_TABLE`: DynamoDB table name for employees
- `EMPLOYEE_TABLE_COGNITO_ID_INDEX`: GSI name for Cognito ID lookups

---

## Rate Limiting & Best Practices

1. **Authentication**: Always include valid Cognito JWT tokens
2. **Error Handling**: Check response status codes and handle errors appropriately
3. **Admin Permissions**: Most operations require organization admin privileges
4. **Input Validation**: Validate required fields before making API calls
5. **Caching**: Consider caching organization and plan data on the frontend
6. **Retry Logic**: Implement exponential backoff for failed requests