# Organization Management API Documentation

## Overview
The Organization Management API provides comprehensive endpoints for managing organizations, subscriptions, billing, and promotional codes within the tenant API Gateway. All endpoints are prefixed with `/v2/` and require JWT authentication via AWS Cognito.

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
    "organizationId": "string",
    "organizationName": "string",
    "displayName": "string",
    "industry": "string",
    "description": "string",
    "clientDetails": {
      "companySize": "string",
      "headquarters": "string",
      "website": "string",
      "primaryContact": {
        "name": "string",
        "email": "string",
        "phone": "string"
      },
      "businessType": "string",
      "annualRevenue": "string"
    },
    "planType": "PROFESSIONAL",
    "billingCycle": "MONTHLY",
    "teamLimit": 10,
    "currentTeamCount": 0,
    "isActive": true,
    "createdAt": "2024-01-01T00:00:00Z",
    "lastUpdated": "2024-01-01T00:00:00Z",
    "admins": ["user-123"],
    "activePromoCode": null
  }
}
```

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

### 8. List User Organizations
**Endpoint:** `GET /v2/organization/list`  
**Function:** Lists all organizations where the user is an admin

**Success Response (200):**
```json
{
  "organizations": [
    {
      "orgId": "ORG#uuid1",
      "orgName": "Acme Corporation",
      "orgDesc": "Technology solutions company",
      "clientName": "Acme Corp",
      "industry": "Technology",
      "companySize": "51-200",
      "website": "https://acme.com",
      "contactEmail": "admin@acme.com",
      "billingMode": "PAID",
      "subscriptionType": "SUBSCRIPTION",
      "billingPlan": "YEARLY",
      "orgBillingStatus": "ACTIVE",
      "currentPlanId": "professional",
      "currentPlanName": "Professional Plan",
      "currentTeamCount": 8,
      "maxTeamsAllowed": 25,
      "maxMembersAllowed": 150,
      "appliedPromoCode": "",
      "promoDiscountPercent": 0,
      "trialEndDate": "",
      "nextBillingDate": "2024-12-01T00:00:00Z",
      "createdAt": "2024-01-01T00:00:00Z",
      "updatedAt": "2024-01-15T10:30:00Z"
    }
  ],
  "totalCount": 1,
  "availablePlans": [
    // Array of available subscription plans
  ]
}
```

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