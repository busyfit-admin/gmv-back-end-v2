# GitHub Actions Secret Setup

This document provides the exact secrets and variables you need to configure in your GitHub repository for the workflows to function properly.

## 🔐 Required GitHub Secrets

Navigate to your repository: `Settings > Secrets and variables > Actions`

### AWS Credentials (Required for all workflows)
```
Name: AWS_ACCESS_KEY_ID
Value: Your AWS access key ID
Description: AWS access key for CloudFormation and S3 operations

Name: AWS_SECRET_ACCESS_KEY  
Value: Your AWS secret access key
Description: AWS secret access key for CloudFormation and S3 operations
```

### S3 Bucket for Lambda Artifacts (Required for build workflow)
```
Name: LAMBDA_ARTIFACTS_BUCKET
Value: your-lambda-artifacts-bucket-name
Description: S3 bucket for storing compiled Lambda packages
```

### Environment URLs (Required for health checks)
```
Name: STAGING_API_URL
Value: https://api-staging.yourdomain.com
Description: Base URL for staging API environment

Name: PROD_API_URL
Value: https://api.yourdomain.com  
Description: Base URL for production API environment
```

### Cognito Configuration - Staging
```
Name: STAGING_COGNITO_CLIENT_ID
Value: your-staging-cognito-client-id
Description: Cognito App Client ID for staging environment

Name: STAGING_USER_POOL_ID
Value: your-staging-user-pool-id
Description: Cognito User Pool ID for staging environment
```

### Cognito Configuration - Production
```
Name: PROD_COGNITO_CLIENT_ID
Value: your-production-cognito-client-id
Description: Cognito App Client ID for production environment

Name: PROD_USER_POOL_ID
Value: your-production-user-pool-id
Description: Cognito User Pool ID for production environment
```

## 🏗️ AWS Infrastructure Requirements

### 1. IAM User/Role for GitHub Actions

Create an IAM user with these permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CloudFormationFullAccess",
      "Effect": "Allow",
      "Action": [
        "cloudformation:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "S3FullAccess",
      "Effect": "Allow", 
      "Action": [
        "s3:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "LambdaFullAccess",
      "Effect": "Allow",
      "Action": [
        "lambda:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "APIGatewayFullAccess",
      "Effect": "Allow",
      "Action": [
        "apigateway:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "CognitoFullAccess", 
      "Effect": "Allow",
      "Action": [
        "cognito-idp:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "IAMPassRole",
      "Effect": "Allow",
      "Action": [
        "iam:PassRole",
        "iam:GetRole",
        "iam:CreateRole",
        "iam:DeleteRole",
        "iam:AttachRolePolicy",
        "iam:DetachRolePolicy",
        "iam:PutRolePolicy",
        "iam:DeleteRolePolicy"
      ],
      "Resource": "*"
    },
    {
      "Sid": "LogsAccess",
      "Effect": "Allow",
      "Action": [
        "logs:*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "DynamoDBAccess",
      "Effect": "Allow",
      "Action": [
        "dynamodb:*"
      ],
      "Resource": "*"
    }
  ]
}
```

### 2. S3 Bucket for Lambda Artifacts

Create an S3 bucket for storing Lambda deployment packages:
```bash
aws s3 mb s3://your-lambda-artifacts-bucket-name --region us-east-1
```

### 3. Environment-Specific Resources

For each environment (staging, production), ensure you have:
- **API Gateway**: Deployed with base URL
- **Cognito User Pool**: With app client configured
- **Lambda Functions**: Deployed and accessible
- **DynamoDB Tables**: Created with proper schema

## 🔧 Configuration Commands

### Get Cognito Details
```bash
# List User Pools
aws cognito-idp list-user-pools --max-results 20

# Get User Pool details
aws cognito-idp describe-user-pool --user-pool-id your-user-pool-id

# List User Pool Clients
aws cognito-idp list-user-pool-clients --user-pool-id your-user-pool-id

# Get App Client details
aws cognito-idp describe-user-pool-client \
  --user-pool-id your-user-pool-id \
  --client-id your-client-id
```

### Get API Gateway URLs
```bash
# List APIs
aws apigateway get-rest-apis

# Get deployment details
aws apigateway get-deployments --rest-api-id your-api-id

# Get stage details
aws apigateway get-stage --rest-api-id your-api-id --stage-name prod
```

### Verify S3 Bucket
```bash
# Check bucket exists
aws s3 ls s3://your-lambda-artifacts-bucket-name/

# Set bucket policy if needed
aws s3api put-bucket-policy \
  --bucket your-lambda-artifacts-bucket-name \
  --policy file://bucket-policy.json
```

## ✅ Validation Checklist

### Pre-deployment Verification:
- [ ] AWS credentials have correct permissions
- [ ] S3 bucket exists and is accessible  
- [ ] Staging environment URLs are reachable
- [ ] Production environment URLs are reachable
- [ ] Cognito User Pools exist in both environments
- [ ] App Clients are configured in Cognito
- [ ] API Gateway stages are deployed

### Test Secret Configuration:
```bash
# Test AWS credentials
aws sts get-caller-identity

# Test S3 access
aws s3 ls s3://your-lambda-artifacts-bucket-name/

# Test Cognito access (staging)
aws cognito-idp describe-user-pool --user-pool-id YOUR_STAGING_USER_POOL_ID

# Test Cognito access (production)  
aws cognito-idp describe-user-pool --user-pool-id YOUR_PROD_USER_POOL_ID

# Test API connectivity
curl -I https://api-staging.yourdomain.com/health
curl -I https://api.yourdomain.com/health
```

## 🚨 Security Best Practices

### 1. Credential Rotation
- Rotate AWS access keys every 90 days
- Use IAM roles instead of users when possible
- Enable MFA for AWS accounts

### 2. Principle of Least Privilege
- Grant minimum required permissions
- Use resource-specific ARNs where possible
- Regular audit of permissions

### 3. Secret Management
- Never commit secrets to repository
- Use GitHub's secret scanning
- Monitor for exposed credentials

### 4. Environment Isolation
- Use separate AWS accounts for environments
- Implement cross-account roles for deployment
- Network isolation with VPCs

## 📞 Troubleshooting

### Common Issues:

1. **CloudFormation Permission Denied**
   ```bash
   # Check IAM permissions
   aws iam simulate-principal-policy \
     --policy-source-arn arn:aws:iam::ACCOUNT:user/github-actions \
     --action-names cloudformation:CreateStack
   ```

2. **S3 Access Denied**
   ```bash
   # Check bucket permissions
   aws s3api get-bucket-acl --bucket your-lambda-artifacts-bucket-name
   ```

3. **Cognito Access Issues**
   ```bash
   # Verify User Pool exists
   aws cognito-idp describe-user-pool --user-pool-id YOUR_USER_POOL_ID
   ```

4. **API Gateway Not Found**
   ```bash
   # List all APIs
   aws apigateway get-rest-apis
   ```

### Debugging Workflows:
1. Check the Actions tab for detailed logs
2. Verify secret names match exactly
3. Confirm AWS region settings
4. Test API endpoints manually
5. Validate JSON formatting in environments

## 📋 Secret Template

Copy this template and fill in your values:
```yaml
# AWS Configuration
AWS_ACCESS_KEY_ID: "AKIA..."
AWS_SECRET_ACCESS_KEY: "..."
LAMBDA_ARTIFACTS_BUCKET: "your-lambda-artifacts-bucket"

# Staging Environment  
STAGING_API_URL: "https://api-staging.yourdomain.com"
STAGING_COGNITO_CLIENT_ID: "..."
STAGING_USER_POOL_ID: "us-east-1_..."

# Production Environment
PROD_API_URL: "https://api.yourdomain.com"  
PROD_COGNITO_CLIENT_ID: "..."
PROD_USER_POOL_ID: "us-east-1_..."
```