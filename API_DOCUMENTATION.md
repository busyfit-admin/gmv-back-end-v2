# Organization Management API Documentation

## Overview

This API provides comprehensive organization management functionality for a multi-tenant SaaS platform. It includes client details capture, subscription plan switching, promotional code management, and team limits enforcement.

## Base URL
```
https://api.tenant.gmv.com/v2
```

## Authentication

All endpoints require AWS Cognito JWT authentication. Include the token in the Authorization header:
```
Authorization: Bearer <jwt-token>
```

## Core Features

### 1. Organization Management
- **Client Details Capture**: Store comprehensive client information including company size, industry, contact details, and business model
- **Industry-Specific Details**: Custom fields for different industry requirements
- **Admin Management**: Track organization administrators and their permissions

### 2. Subscription Management
- **Plan Types**: BASIC, PROFESSIONAL, ENTERPRISE with different feature sets and team limits
- **Billing Cycles**: Monthly and yearly billing with automatic discounts for annual plans
- **Plan Switching**: Seamless upgrade/downgrade with validation checks
- **Team Limits**: Enforce maximum team counts based on subscription level

### 3. Promotional Codes
- **Discount Types**: Percentage discounts and free trial periods
- **Usage Tracking**: Monitor promo code usage and prevent abuse
- **Eligibility Validation**: Check organization qualification for specific promotions

## API Endpoints

### Create Organization
**POST** `/organization`

Creates a new organization with the authenticated user as admin.

**Request Body:**
```json
{
  "organizationName": "Tech Solutions Inc",
  "displayName": "Tech Solutions",
  "industry": "Technology",
  "description": "Leading provider of innovative tech solutions",
  "clientDetails": {
    "companySize": "50-200",
    "headquarters": "San Francisco, CA",
    "website": "https://techsolutions.com",
    "primaryContact": {
      "name": "John Smith",
      "email": "john.smith@techsolutions.com",
      "phone": "+1-555-0123"
    },
    "businessType": "B2B",
    "annualRevenue": "$10M-50M"
  },
  "subscriptionPlan": "PROFESSIONAL",
  "billingCycle": "MONTHLY"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Organization created successfully",
  "organizationId": "org-12345",
  "data": {
    // Full organization details
  }
}
```

### Get Organization Details
**GET** `/organization/{organizationId}`

Retrieves comprehensive organization information. Only admins can access.

**Response (200):**
```json
{
  "organizationId": "org-12345",
  "organizationName": "Tech Solutions Inc",
  "displayName": "Tech Solutions",
  "industry": "Technology",
  "description": "Leading provider of innovative tech solutions",
  "clientDetails": {
    "companySize": "50-200",
    "headquarters": "San Francisco, CA",
    "website": "https://techsolutions.com",
    "primaryContact": {
      "name": "John Smith",
      "email": "john.smith@techsolutions.com",
      "phone": "+1-555-0123"
    },
    "businessType": "B2B",
    "annualRevenue": "$10M-50M"
  },
  "planType": "PROFESSIONAL",
  "billingCycle": "MONTHLY",
  "teamLimit": 10,
  "currentTeamCount": 7,
  "isActive": true,
  "createdAt": "2024-01-15T10:30:00Z",
  "lastUpdated": "2024-01-20T15:45:00Z",
  "admins": ["user-123", "user-456"],
  "activePromoCode": "WELCOME2024"
}
```

### Update Organization
**PUT** `/organization/{organizationId}`

Updates organization details. Only admins can perform this operation.

**Request Body:**
```json
{
  "displayName": "Tech Solutions Updated",
  "industry": "Software Technology",
  "description": "Updated description",
  "clientDetails": {
    "companySize": "200-1000",
    "headquarters": "New York, NY",
    // ... other client details
  }
}
```

### List User Organizations
**GET** `/organization/list`

Lists all organizations where the user is an admin with summary information.

**Response (200):**
```json
{
  "success": true,
  "organizations": [
    {
      "organizationId": "org-12345",
      "organizationName": "Tech Solutions Inc",
      "displayName": "Tech Solutions",
      "industry": "Technology",
      "planType": "PROFESSIONAL",
      "currentTeamCount": 7,
      "teamLimit": 10,
      "isActive": true,
      "role": "ADMIN"
    }
  ]
}
```

### Get Subscription Plans
**GET** `/organization/{organizationId}/subscription`

Retrieves available subscription plans with pricing and current plan details.

**Response (200):**
```json
{
  "success": true,
  "currentPlan": {
    "planType": "PROFESSIONAL",
    "billingCycle": "MONTHLY",
    "price": 49.99,
    "teamLimit": 10,
    "isActive": true,
    "nextBillingDate": "2024-02-15T10:30:00Z"
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
        "Custom integrations",
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

### Update Subscription
**PUT** `/organization/{organizationId}/subscription`

Updates subscription plan and billing cycle with validation.

**Request Body:**
```json
{
  "planType": "PROFESSIONAL",
  "billingCycle": "YEARLY"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Subscription updated successfully",
  "subscription": {
    "planType": "PROFESSIONAL",
    "billingCycle": "YEARLY",
    "price": 499.99,
    "teamLimit": 10,
    "isActive": true,
    "nextBillingDate": "2025-01-20T15:45:00Z"
  }
}
```

### Apply Promo Code
**POST** `/organization/{organizationId}/promo-code`

Applies promotional codes for discounts or free trials.

**Request Body:**
```json
{
  "promoCode": "WELCOME2024"
}
```

**Response (200):**
```json
{
  "success": true,
  "message": "Promo code applied successfully",
  "promoDetails": {
    "promoCode": "WELCOME2024",
    "discountType": "PERCENTAGE",
    "discountValue": 20,
    "description": "20% off for new customers",
    "expiresAt": "2024-12-31T23:59:59Z",
    "isActive": true
  }
}
```

### Get Active Promo Code
**GET** `/organization/{organizationId}/promo-code`

Retrieves details of the currently active promo code.

**Response (200):**
```json
{
  "success": true,
  "message": "Active promo code found",
  "promoDetails": {
    "promoCode": "WELCOME2024",
    "discountType": "PERCENTAGE",
    "discountValue": 20,
    "description": "20% off for new customers",
    "expiresAt": "2024-12-31T23:59:59Z",
    "isActive": true
  }
}
```

## Subscription Plans

### BASIC Plan
- **Team Limit**: 3 teams
- **Monthly Price**: $19.99
- **Yearly Price**: $199.99 (16.67% discount)
- **Features**: Basic reporting, Email support

### PROFESSIONAL Plan
- **Team Limit**: 10 teams
- **Monthly Price**: $49.99
- **Yearly Price**: $499.99 (16.67% discount)
- **Features**: Advanced reporting, Priority support, Custom integrations

### ENTERPRISE Plan
- **Team Limit**: Unlimited
- **Monthly Price**: $99.99
- **Yearly Price**: $999.99 (16.67% discount)
- **Features**: Enterprise reporting, 24/7 support, Advanced security

## Team Limits and Validation

The system enforces team limits based on subscription plans:
- When creating teams, the system checks if the current count exceeds the plan limit
- When downgrading plans, the system validates that existing teams don't exceed the new limit
- Organizations cannot downgrade if they have more teams than the new plan allows

## Promo Code Types

### Percentage Discounts
- Apply percentage discounts to subscription fees
- Can be one-time or recurring discounts
- Usage tracking prevents multiple applications

### Free Trial Periods
- Extend trial periods for new organizations
- Track trial start and end dates
- Automatic conversion to paid plan after trial

## Error Handling

### Common Error Codes

- **400 Bad Request**: Invalid request data or parameters
- **401 Unauthorized**: Missing or invalid authentication token
- **403 Forbidden**: User lacks permission to perform the operation
- **404 Not Found**: Organization or resource not found
- **409 Conflict**: Operation conflicts with current state (e.g., exceeding team limits)

### Error Response Format
```json
{
  "success": false,
  "error": "INVALID_PLAN",
  "message": "The specified subscription plan is not valid"
}
```

## Client Details Schema

### Company Size Options
- "1-10"
- "11-50" 
- "51-200"
- "201-1000"
- "1000+"

### Business Type Options
- "B2B" (Business to Business)
- "B2C" (Business to Consumer)
- "B2B2C" (Business to Business to Consumer)

### Annual Revenue Options
- "<$1M"
- "$1M-10M"
- "$10M-50M"
- "$50M-100M"
- "$100M+"

## Frontend Integration Examples

### React Hook for Organization Management
```javascript
import { useState, useEffect } from 'react';
import { API } from 'aws-amplify';

export const useOrganization = (organizationId) => {
  const [organization, setOrganization] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchOrganization();
  }, [organizationId]);

  const fetchOrganization = async () => {
    try {
      setLoading(true);
      const response = await API.get('tenant-api', `/organization/${organizationId}`);
      setOrganization(response);
    } catch (err) {
      setError(err.response?.data?.message || 'Failed to fetch organization');
    } finally {
      setLoading(false);
    }
  };

  const updateSubscription = async (planType, billingCycle) => {
    try {
      const response = await API.put('tenant-api', `/organization/${organizationId}/subscription`, {
        body: { planType, billingCycle }
      });
      setOrganization(prev => ({
        ...prev,
        planType,
        billingCycle,
        teamLimit: response.subscription.teamLimit
      }));
      return response;
    } catch (err) {
      throw new Error(err.response?.data?.message || 'Failed to update subscription');
    }
  };

  const applyPromoCode = async (promoCode) => {
    try {
      const response = await API.post('tenant-api', `/organization/${organizationId}/promo-code`, {
        body: { promoCode }
      });
      setOrganization(prev => ({
        ...prev,
        activePromoCode: promoCode
      }));
      return response;
    } catch (err) {
      throw new Error(err.response?.data?.message || 'Failed to apply promo code');
    }
  };

  return {
    organization,
    loading,
    error,
    refetch: fetchOrganization,
    updateSubscription,
    applyPromoCode
  };
};
```

### Organization Creation Form Component
```javascript
import React, { useState } from 'react';
import { API } from 'aws-amplify';

export const CreateOrganizationForm = ({ onSuccess }) => {
  const [formData, setFormData] = useState({
    organizationName: '',
    displayName: '',
    industry: '',
    description: '',
    clientDetails: {
      companySize: '',
      headquarters: '',
      website: '',
      primaryContact: {
        name: '',
        email: '',
        phone: ''
      },
      businessType: '',
      annualRevenue: ''
    },
    subscriptionPlan: 'BASIC',
    billingCycle: 'MONTHLY'
  });

  const handleSubmit = async (e) => {
    e.preventDefault();
    try {
      const response = await API.post('tenant-api', '/organization', {
        body: formData
      });
      onSuccess(response.organizationId);
    } catch (error) {
      console.error('Failed to create organization:', error);
    }
  };

  return (
    <form onSubmit={handleSubmit}>
      {/* Form fields implementation */}
    </form>
  );
};
```

## Testing and Validation

### Test Organization Creation
```bash
curl -X POST https://api.tenant.gmv.com/v2/organization \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "organizationName": "Test Company",
    "subscriptionPlan": "BASIC",
    "billingCycle": "MONTHLY"
  }'
```

### Test Subscription Update
```bash
curl -X PUT https://api.tenant.gmv.com/v2/organization/org-123/subscription \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "planType": "PROFESSIONAL",
    "billingCycle": "YEARLY"
  }'
```

## Security Considerations

1. **Authentication**: All endpoints require valid AWS Cognito JWT tokens
2. **Authorization**: Only organization admins can modify organization data
3. **Input Validation**: All inputs are validated for type, length, and format
4. **Rate Limiting**: API calls are rate-limited to prevent abuse
5. **Data Encryption**: All data is encrypted at rest and in transit

## Support and Troubleshooting

### Common Issues

1. **Team Limit Exceeded**: Cannot create more teams than plan allows
   - Solution: Upgrade subscription plan or remove existing teams

2. **Promo Code Already Used**: Promo code has already been applied
   - Solution: Use a different promo code or contact support

3. **Invalid Subscription Downgrade**: Cannot downgrade due to existing teams
   - Solution: Reduce team count before downgrading

### Contact Information
For API support and technical questions, contact the GMV Development Team.