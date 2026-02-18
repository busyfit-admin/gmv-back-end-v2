# AWS IAM User Setup for GitHub Actions - Best Practices

This guide walks you through creating a secure AWS IAM user specifically for GitHub Actions CI/CD with minimal required permissions.

## 🛡️ Security Best Practices Overview

### 1. **Use Dedicated Service Account**
- Create a separate IAM user only for GitHub Actions
- Never use your personal AWS credentials
- Name it clearly: `github-actions-gmv-ci`

### 2. **Principle of Least Privilege**  
- Grant only the minimum permissions needed
- Use specific resource ARNs when possible
- Regularly audit and rotate credentials

### 3. **Enable Security Features**
- Use IAM conditions for enhanced security
- Enable CloudTrail for audit logging
- Set up billing alerts for cost monitoring

## 🚀 Step-by-Step Setup

### Step 1: Create IAM User via AWS Console

1. **Navigate to IAM Console**:
   ```
   AWS Console → IAM → Users → Create user
   ```

2. **Configure User Details**:
   ```yaml
   Username: github-actions-gmv-ci
   Access type: Programmatic access (no console access)
   Description: "GitHub Actions service account for GMV project CI/CD"
   ```

3. **Skip Password Configuration**:
   - Uncheck "Provide user access to the AWS Management Console"
   - This user should only have programmatic access

### Step 2: Create Custom IAM Policy

Create a custom policy with minimal required permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "CloudFormationAccess",
      "Effect": "Allow",
      "Action": [
        "cloudformation:CreateStack",
        "cloudformation:UpdateStack",
        "cloudformation:DeleteStack",
        "cloudformation:DescribeStacks",
        "cloudformation:DescribeStackEvents",
        "cloudformation:DescribeStackResources",
        "cloudformation:GetTemplate",
        "cloudformation:ValidateTemplate",
        "cloudformation:ListStacks"
      ],
      "Resource": [
        "arn:aws:cloudformation:ap-southeast-2:*:stack/tenant-portal-apis-test/*",
        "arn:aws:cloudformation:ap-southeast-2:*:stack/tenant-portal-apis-test"
      ]
    },
    {
      "Sid": "S3DeploymentBucketAccess",
      "Effect": "Allow",
      "Action": [
        "s3:CreateBucket",
        "s3:DeleteBucket",
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:ListBucket",
        "s3:PutBucketVersioning",
        "s3:PutBucketPublicAccessBlock"
      ],
      "Resource": [
        "arn:aws:s3:::gmv-test-deployment-bucket",
        "arn:aws:s3:::gmv-test-deployment-bucket/*"
      ]
    },
    {
      "Sid": "LambdaAccess",
      "Effect": "Allow",
      "Action": [
        "lambda:CreateFunction",
        "lambda:UpdateFunctionCode",
        "lambda:UpdateFunctionConfiguration",
        "lambda:DeleteFunction",
        "lambda:GetFunction",
        "lambda:ListFunctions",
        "lambda:AddPermission",
        "lambda:RemovePermission",
        "lambda:InvokeFunction"
      ],
      "Resource": "arn:aws:lambda:ap-southeast-2:*:function:tenant-portal-*"
    },
    {
      "Sid": "APIGatewayAccess",
      "Effect": "Allow",
      "Action": [
        "apigateway:GET",
        "apigateway:POST",
        "apigateway:PUT",
        "apigateway:DELETE",
        "apigateway:PATCH"
      ],
      "Resource": "arn:aws:apigateway:ap-southeast-2::/*"
    },
    {
      "Sid": "CognitoAccess",
      "Effect": "Allow",
      "Action": [
        "cognito-idp:CreateUserPool",
        "cognito-idp:UpdateUserPool",
        "cognito-idp:DeleteUserPool",
        "cognito-idp:DescribeUserPool",
        "cognito-idp:CreateUserPoolClient",
        "cognito-idp:UpdateUserPoolClient",
        "cognito-idp:DeleteUserPoolClient",
        "cognito-idp:DescribeUserPoolClient",
        "cognito-idp:AdminCreateUser",
        "cognito-idp:AdminSetUserPassword",
        "cognito-idp:AdminDeleteUser",
        "cognito-idp:ListUsers",
        "cognito-idp:ListUserPoolClients"
      ],
      "Resource": [
        "arn:aws:cognito-idp:ap-southeast-2:*:userpool/*"
      ]
    },
    {
      "Sid": "DynamoDBAccess",
      "Effect": "Allow",
      "Action": [
        "dynamodb:CreateTable",
        "dynamodb:UpdateTable",
        "dynamodb:DeleteTable",
        "dynamodb:DescribeTable",
        "dynamodb:ListTables",
        "dynamodb:PutItem",
        "dynamodb:GetItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan",
        "dynamodb:UpdateItem"
      ],
      "Resource": [
        "arn:aws:dynamodb:ap-southeast-2:*:table/tenant-portal-*",
        "arn:aws:dynamodb:ap-southeast-2:*:table/tenant-portal-*/index/*"
      ]
    },
    {
      "Sid": "IAMRoleManagement",
      "Effect": "Allow",
      "Action": [
        "iam:CreateRole",
        "iam:DeleteRole",
        "iam:GetRole",
        "iam:PassRole",
        "iam:AttachRolePolicy",
        "iam:DetachRolePolicy",
        "iam:PutRolePolicy",
        "iam:DeleteRolePolicy",
        "iam:GetRolePolicy",
        "iam:ListRolePolicies",
        "iam:ListAttachedRolePolicies"
      ],
      "Resource": [
        "arn:aws:iam::*:role/tenant-portal-*",
        "arn:aws:iam::*:role/lambda-execution-*"
      ]
    },
    {
      "Sid": "CloudWatchLogsAccess",
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams",
        "logs:DeleteLogGroup"
      ],
      "Resource": "arn:aws:logs:ap-southeast-2:*:log-group:/aws/lambda/tenant-portal-*"
    }
  ]
}
```

### Step 3: Attach Policy to User

1. **Create Policy**:
   ```
   IAM Console → Policies → Create policy → JSON tab
   Paste the policy above → Name: "GitHubActions-GMV-CI-Policy"
   ```

2. **Attach to User**:
   ```
   IAM Console → Users → github-actions-gmv-ci
   Permissions → Add permissions → Attach policies directly
   Select "GitHubActions-GMV-CI-Policy"
   ```

### Step 4: Create Access Keys

1. **Generate Access Keys**:
   ```
   IAM Console → Users → github-actions-gmv-ci
   Security credentials → Create access key → Command Line Interface (CLI)
   ✓ I understand the above recommendation
   ```

2. **Download Credentials**:
   - **Access Key ID**: `AKIA...`
   - **Secret Access Key**: `...` (save securely)

3. **⚠️ IMPORTANT**: Save these credentials securely and delete from your local machine after adding to GitHub

## 🔐 GitHub Secrets Configuration

### Add Secrets to GitHub Repository

1. **Navigate to Repository Settings**:
   ```
   GitHub Repository → Settings → Secrets and variables → Actions
   ```

2. **Add Repository Secrets**:
   ```yaml
   AWS_ACCESS_KEY_ID: "AKIA..." 
   AWS_SECRET_ACCESS_KEY: "your-secret-key"
   ```

3. **Verify Secrets**:
   - Secrets should show as "Updated X minutes ago"
   - Never expose these in logs or code

## 🛡️ Enhanced Security Measures

### 1. Add IP Restrictions (Optional)
```json
{
  "Condition": {
    "IpAddress": {
      "aws:SourceIp": [
        "140.82.112.0/20",    
        "192.30.252.0/22",    
        "185.199.108.0/22"    
      ]
    }
  }
}
```

### 2. Add Time-Based Restrictions
```json
{
  "Condition": {
    "DateGreaterThan": {
      "aws:CurrentTime": "2026-01-01T00:00:00Z"
    },
    "DateLessThan": {
      "aws:CurrentTime": "2026-12-31T23:59:59Z"
    }
  }
}
```

### 3. Enable CloudTrail Monitoring
```bash
# Monitor API calls from this user
aws logs filter-log-events \
  --log-group-name CloudTrail/APICallHistory \
  --filter-pattern "{ $.userIdentity.userName = github-actions-gmv-ci }"
```

## 🔄 Credential Rotation

### Monthly Rotation Process:

1. **Create New Access Key**:
   ```
   IAM Console → Users → github-actions-gmv-ci
   Security credentials → Create access key
   ```

2. **Update GitHub Secrets**:
   ```
   Update AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY
   ```

3. **Test New Credentials**:
   ```
   Run a workflow to verify access
   ```

4. **Deactivate Old Key**:
   ```
   IAM Console → Make old key "Inactive"
   Wait 24-48 hours, then delete if no issues
   ```

## 📊 Monitoring and Alerts

### 1. Set Up Cost Alerts
```bash
# Create billing alert for unusual costs
aws budgets create-budget \
  --account-id YOUR_ACCOUNT_ID \
  --budget file://ci-cd-budget.json
```

### 2. Monitor Failed Attempts
```bash
# CloudWatch alarm for failed API calls
aws logs create-metric-filter \
  --log-group-name CloudTrail/APICallHistory \
  --filter-name "FailedGitHubActionsCalls" \
  --filter-pattern "{ $.userIdentity.userName = github-actions-gmv-ci && $.errorCode = * }"
```

### 3. Resource Usage Tracking
```bash
# Track resources created by CI/CD
aws resourcegroupstaggingapi get-resources \
  --tag-filters Key=CreatedBy,Values=github-actions-gmv-ci
```

## 🚨 Security Incident Response

### If Credentials are Compromised:

1. **Immediately Deactivate**:
   ```
   IAM Console → Users → github-actions-gmv-ci 
   Security credentials → Make inactive
   ```

2. **Review CloudTrail Logs**:
   ```bash
   aws logs filter-log-events \
     --log-group-name CloudTrail/APICallHistory \
     --start-time $(date -d "1 day ago" +%s)000 \
     --filter-pattern "{ $.userIdentity.userName = github-actions-gmv-ci }"
   ```

3. **Generate New Credentials**:
   - Create new access key
   - Update GitHub secrets
   - Delete compromised credentials

4. **Audit Resources**:
   ```bash
   # Check for unauthorized resource creation
   aws cloudformation list-stacks --stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE
   ```

## ✅ Validation Checklist

After setup, verify:

- [ ] IAM user created with programmatic access only
- [ ] Custom policy attached with minimal permissions  
- [ ] Access keys generated and stored securely locally
- [ ] GitHub secrets configured correctly
- [ ] Test workflow runs successfully
- [ ] CloudTrail logging enabled
- [ ] Cost monitoring alerts configured
- [ ] Local credential files deleted
- [ ] Credential rotation schedule established
- [ ] Team access to secrets restricted

## 📞 Troubleshooting

### Common Issues:

1. **Permission Denied Errors**:
   ```bash
   # Check current permissions
   aws iam simulate-principal-policy \
     --policy-source-arn arn:aws:iam::ACCOUNT:user/github-actions-gmv-ci \
     --action-names cloudformation:CreateStack
   ```

2. **Policy Too Restrictive**:
   - Add specific permissions as needed
   - Use CloudTrail to see what's being denied
   - Test with broader permissions first, then restrict

3. **Credentials Not Working**:
   ```bash
   # Test credentials locally
   aws sts get-caller-identity
   ```

By following these best practices, you'll have a secure, auditable, and maintainable CI/CD setup for your AWS resources.