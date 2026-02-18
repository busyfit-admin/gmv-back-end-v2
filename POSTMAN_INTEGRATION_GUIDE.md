# Postman Integration Guide for GMV Back-End APIs

## Overview
This guide helps you set up comprehensive Postman testing for your multi-tenant SaaS API endpoints including authentication, environment management, and automated testing.

## 1. Environment Setup

### Create Postman Environments

Create separate environments for different deployment stages:

#### Development Environment
```json
{
  "name": "GMV-Dev",
  "values": [
    {
      "key": "base_url",
      "value": "https://mvp-dev.4cl-tech.com.au",
      "enabled": true
    },
    {
      "key": "auth_url", 
      "value": "https://cognito-idp.ap-south-1.amazonaws.com",
      "enabled": true
    },
    {
      "key": "client_id",
      "value": "YOUR_COGNITO_CLIENT_ID",
      "enabled": true
    },
    {
      "key": "username",
      "value": "test@example.com",
      "enabled": true
    },
    {
      "key": "password",
      "value": "TestPassword123!",
      "enabled": true,
      "type": "secret"
    },
    {
      "key": "access_token",
      "value": "",
      "enabled": true,
      "type": "secret"
    },
    {
      "key": "refresh_token",
      "value": "",
      "enabled": true,
      "type": "secret"
    }
  ]
}
```

#### UAT Environment
```json
{
  "name": "GMV-UAT",
  "values": [
    {
      "key": "base_url",
      "value": "https://uat.your-domain.com",
      "enabled": true
    }
    // ... similar structure to dev
  ]
}
```

## 2. Authentication Setup

### Pre-request Script for Cognito Authentication

Create a collection-level pre-request script:

```javascript
// Collection Pre-request Script
const username = pm.environment.get("username");
const password = pm.environment.get("password");
const clientId = pm.environment.get("client_id");
const authUrl = pm.environment.get("auth_url");

// Check if we need to authenticate
const accessToken = pm.environment.get("access_token");
const tokenExpiry = pm.environment.get("token_expiry");

// Function to authenticate with Cognito
function authenticateWithCognito() {
    const authRequest = {
        url: authUrl,
        method: 'POST',
        header: {
            'X-Amz-Target': 'AWSCognitoIdentityProviderService.InitiateAuth',
            'Content-Type': 'application/x-amz-json-1.1'
        },
        body: {
            mode: 'raw',
            raw: JSON.stringify({
                "AuthFlow": "USER_PASSWORD_AUTH",
                "ClientId": clientId,
                "AuthParameters": {
                    "USERNAME": username,
                    "PASSWORD": password
                }
            })
        }
    };
    
    pm.sendRequest(authRequest, (err, response) => {
        if (err) {
            console.error('Authentication failed:', err);
            return;
        }
        
        const responseJson = response.json();
        if (responseJson.AuthenticationResult) {
            const accessToken = responseJson.AuthenticationResult.AccessToken;
            const refreshToken = responseJson.AuthenticationResult.RefreshToken;
            const expiresIn = responseJson.AuthenticationResult.ExpiresIn;
            
            // Store tokens in environment
            pm.environment.set("access_token", accessToken);
            pm.environment.set("refresh_token", refreshToken);
            
            // Calculate expiry time
            const expiryTime = Date.now() + (expiresIn * 1000);
            pm.environment.set("token_expiry", expiryTime);
            
            console.log("Authentication successful");
        } else {
            console.error("Authentication failed:", responseJson);
        }
    });
}

// Check if token is expired or missing
if (!accessToken || (tokenExpiry && Date.now() > tokenExpiry)) {
    console.log("Token expired or missing, authenticating...");
    authenticateWithCognito();
}
```

### Authorization Header Setup

Add this to your request headers (or collection-level headers):

```
Authorization: Bearer {{access_token}}
```

## 3. Collection Structure

### Organize collections by domain:

```
📁 GMV API Collections
├── 📁 01 - Authentication
│   ├── Login with Cognito
│   ├── Refresh Token
│   └── Get User Info
├── 📁 02 - Organization Management  
│   ├── Create Organization
│   ├── Get Organization
│   ├── Update Organization
│   ├── List User Organizations
│   └── Manage Admins
├── 📁 03 - Employee Management
│   ├── Create Employee
│   ├── Get Employee by Username
│   ├── Get All Employees
│   └── Update Employee Roles
├── 📁 04 - Teams Management
│   ├── Create Team
│   ├── List User Teams
│   ├── Add Team Members
│   └── Get Team Details
├── 📁 05 - Subscription Management
│   ├── Get Available Plans
│   ├── Update Subscription
│   ├── Apply Promo Code
│   └── Get Billing Info
└── 📁 06 - System Health
    ├── Health Check
    └── Version Info
```

## 4. Sample Request Templates

### Authentication Request

```json
{
  "name": "Cognito Login",
  "request": {
    "method": "POST",
    "header": [
      {
        "key": "X-Amz-Target",
        "value": "AWSCognitoIdentityProviderService.InitiateAuth"
      },
      {
        "key": "Content-Type", 
        "value": "application/x-amz-json-1.1"
      }
    ],
    "body": {
      "mode": "raw",
      "raw": "{\n  \"AuthFlow\": \"USER_PASSWORD_AUTH\",\n  \"ClientId\": \"{{client_id}}\",\n  \"AuthParameters\": {\n    \"USERNAME\": \"{{username}}\",\n    \"PASSWORD\": \"{{password}}\"\n  }\n}"
    },
    "url": {
      "raw": "{{auth_url}}",
      "host": ["{{auth_url}}"]
    }
  }
}
```

### Organization API Requests

```json
{
  "name": "Create Organization",
  "request": {
    "method": "POST",
    "header": [
      {
        "key": "Authorization",
        "value": "Bearer {{access_token}}"
      },
      {
        "key": "Content-Type",
        "value": "application/json"
      }
    ],
    "body": {
      "mode": "raw",
      "raw": "{\n  \"orgName\": \"Test Organization\",\n  \"orgDesc\": \"Test organization for API testing\",\n  \"industry\": \"Technology\",\n  \"companySize\": \"11-50\",\n  \"contactEmail\": \"admin@testorg.com\",\n  \"contactPhone\": \"+1-555-0123\",\n  \"website\": \"https://testorg.com\"\n}"
    },
    "url": {
      "raw": "{{base_url}}/v2/organization",
      "host": ["{{base_url}}"],
      "path": ["v2", "organization"]
    }
  }
}
```

## 5. Automated Testing Scripts

### Test Scripts for Organization Creation

```javascript
// Test script for Create Organization endpoint
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

pm.test("Response time is less than 2000ms", function () {
    pm.expect(pm.response.responseTime).to.be.below(2000);
});

pm.test("Response has organization data", function () {
    const responseJson = pm.response.json();
    pm.expect(responseJson).to.have.property('organizationId');
    pm.expect(responseJson).to.have.property('orgName');
    pm.expect(responseJson.orgName).to.eql('Test Organization');
    
    // Store organization ID for subsequent tests
    pm.environment.set('org_id', responseJson.organizationId);
});

pm.test("Response includes billing information", function () {
    const responseJson = pm.response.json();
    pm.expect(responseJson).to.have.property('billingMode');
    pm.expect(responseJson).to.have.property('subscriptionType');
    pm.expect(responseJson.billingMode).to.eql('FREE');
});

// Check response headers
pm.test("Content-Type is application/json", function () {
    pm.expect(pm.response.headers.get("Content-Type")).to.include("application/json");
});
```

### Test Scripts for Employee Management

```javascript
// Test script for Get Employee endpoint
pm.test("Employee data is valid", function () {
    const responseJson = pm.response.json();
    pm.expect(responseJson).to.have.property('userName');
    pm.expect(responseJson).to.have.property('emailId');
    pm.expect(responseJson).to.have.property('displayName');
    pm.expect(responseJson).to.have.property('rolesData');
    
    // Validate email format
    pm.test("Email is valid format", function () {
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        pm.expect(responseJson.emailId).to.match(emailRegex);
    });
});
```

## 6. Environment Variables Management

### Dynamic Variables Script

```javascript
// Pre-request script to generate dynamic data
pm.environment.set("timestamp", Date.now());
pm.environment.set("random_email", "test+" + Math.random().toString(36).substring(7) + "@example.com");
pm.environment.set("random_org_name", "Test Org " + Math.random().toString(36).substring(7));
```

## 7. Collection Runner Setup

### Test Data File (CSV)

Create a CSV file for data-driven testing:

```csv
org_name,industry,company_size,contact_email
"Tech Solutions Inc","Technology","11-50","tech@example.com"
"Marketing Agency","Marketing","1-10","marketing@example.com"
"Finance Corp","Finance","51-200","finance@example.com"
```

### Newman CLI Integration

Install Newman for CI/CD integration:

```bash
npm install -g newman

# Run collection
newman run "GMV API Collection.postman_collection.json" \
  -e "GMV-Dev.postman_environment.json" \
  -d "test-data.csv" \
  --reporters cli,json \
  --reporter-json-export results.json
```

## 8. Monitor and Alerts

### Collection Monitors

Set up Postman monitors to run your tests on a schedule:

1. Go to Monitors in Postman
2. Create new monitor
3. Select your collection and environment
4. Set schedule (hourly, daily, weekly)
5. Configure notifications

## 9. Mock Server Setup

Create mock servers for development:

```javascript
// Mock response for organization creation
{
  "organizationId": "ORG#{{$randomUUID}}",
  "orgName": "{{org_name}}",
  "orgDesc": "Mock organization for testing",
  "industry": "{{industry}}",
  "billingMode": "FREE",
  "subscriptionType": "TRIAL",
  "createdAt": "{{$isoTimestamp}}",
  "adminUsers": [
    {
      "userName": "{{username}}",
      "role": "OWNER",
      "isActive": true
    }
  ]
}
```

## 10. Best Practices

### Request Naming Convention
- Use descriptive names: `GET User Organization Info`
- Include HTTP method: `POST Create Team`
- Group related requests: `01 - Auth`, `02 - Users`, etc.

### Variable Management
- Use environment variables for all URLs and IDs
- Store sensitive data as secret variables
- Use pre-request scripts to generate dynamic data

### Test Coverage
- Test happy path scenarios
- Test error cases (400, 401, 403, 404, 500)
- Test edge cases and boundary conditions
- Validate response schemas

### Documentation
- Add detailed descriptions to requests
- Document expected responses
- Include example payloads
- Add troubleshooting notes

This setup provides comprehensive API testing coverage for your multi-tenant SaaS application with proper authentication, environment management, and automated testing capabilities.