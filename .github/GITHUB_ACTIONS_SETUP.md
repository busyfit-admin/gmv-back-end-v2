# GitHub Actions Setup Guide

This document explains how to set up and use the GitHub Actions workflows for automated testing and deployment of the GMV API.

## 🚀 Workflows Overview

### 1. Deploy Test Environment and Run API Tests (`deploy-test-and-validate.yml`)
- **Trigger**: PR to main/develop, push to develop, manual dispatch
- **Purpose**: Full end-to-end testing with temporary infrastructure
- **Duration**: ~15-20 minutes
- **Cleanup**: Automatic resource cleanup after tests

### 2. Scheduled API Health Checks (`health-checks.yml`)
- **Trigger**: Every 6 hours, manual dispatch
- **Purpose**: Monitor staging/production API health
- **Duration**: ~2-3 minutes
- **Reports**: Health status with historical tracking

## 🔧 Required Secrets

Configure these secrets in your GitHub repository settings (`Settings > Secrets and variables > Actions`):

### AWS Credentials
```
AWS_ACCESS_KEY_ID          # AWS access key for deployments
AWS_SECRET_ACCESS_KEY      # AWS secret key for deployments
```

### Environment-Specific Secrets
```
# Staging Environment
STAGING_API_URL            # https://api-staging.yourdomain.com
STAGING_COGNITO_CLIENT_ID  # Cognito client ID for staging
STAGING_USER_POOL_ID       # Cognito user pool ID for staging

# Production Environment  
PROD_API_URL               # https://api.yourdomain.com
PROD_COGNITO_CLIENT_ID     # Cognito client ID for production
PROD_USER_POOL_ID          # Cognito user pool ID for production
```

## 📋 Setup Instructions

### 1. AWS IAM Permissions
Create an IAM user/role with these permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudformation:*",
        "s3:*",
        "lambda:*",
        "apigateway:*",
        "cognito-idp:*",
        "iam:*",
        "logs:*"
      ],
      "Resource": "*"
    }
  ]
}
```

### 2. Repository Setup
1. **Add secrets** as listed above
2. **Enable Actions** in repository settings
3. **Configure branch protection** for main/develop branches
4. **Set up notifications** (optional)

### 3. First Run
1. Create a PR or push to develop branch
2. Check the Actions tab for workflow execution
3. Review test reports in artifacts
4. Verify cleanup completed successfully

## 🔄 Workflow Details

### Deploy Test Environment Workflow

#### Jobs Sequence:
1. **deploy-infrastructure**: 
   - Builds Lambda functions
   - Creates CloudFormation stack
   - Outputs API endpoints and Cognito details

2. **setup-test-data**:
   - Creates test users in Cognito
   - Sets up initial test data

3. **run-api-tests**:
   - Runs Postman collection with Newman
   - Generates detailed HTML reports
   - Tests all API endpoints

4. **cleanup-resources**:
   - Deletes CloudFormation stack
   - Removes S3 buckets
   - Cleans up test resources

#### Stack Naming Convention:
```
gmv-tenant-test-{github-run-number}
```

#### Test Reports:
- **Smoke Tests**: Authentication and basic connectivity
- **Organization Management**: CRUD operations for orgs
- **Employee Management**: User management APIs
- **Teams Management**: Team operations
- **Full Integration**: Complete end-to-end scenarios

### Health Checks Workflow

#### Features:
- **Multi-environment**: Tests staging and production
- **Scheduled**: Runs every 6 hours automatically
- **Quick**: Lightweight health checks only
- **Monitoring**: Historical health status tracking

## 📊 Reports and Artifacts

### Test Reports Structure:
```
reports/
├── smoke-test-report.html
├── org-management-report.html
├── employee-management-report.html
├── teams-management-report.html
├── full-integration-report.html
└── integration-results.json
```

### Report Features:
- **HTML Reports**: Interactive with request/response details
- **JSON Results**: Machine-readable for CI/CD integration
- **Test Statistics**: Pass/fail counts and timing
- **Error Details**: Full stack traces and debugging info

## 🚨 Troubleshooting

### Common Issues:

1. **CloudFormation Deployment Fails**
   ```bash
   # Check AWS credentials
   aws sts get-caller-identity
   
   # Verify IAM permissions
   aws iam simulate-principal-policy \
     --policy-source-arn arn:aws:iam::ACCOUNT:user/github-actions \
     --action-names cloudformation:CreateStack
   ```

2. **Lambda Build Failures**
   ```bash
   # Test locally
   cd lambdas
   make build-all
   ```

3. **Postman Tests Fail**
   ```bash
   # Run locally with same environment
   newman run postman/GMV_API_Collection.json \
     --environment postman/test-environment.json \
     --verbose
   ```

4. **Cleanup Issues**
   ```bash
   # Manual cleanup if needed
   aws cloudformation delete-stack --stack-name STACK_NAME
   aws s3 rm s3://BUCKET_NAME --recursive
   ```

### Monitoring Workflow Health:
```bash
# Check recent workflow runs
gh run list --workflow=deploy-test-and-validate.yml

# View specific run details
gh run view RUN_ID --log

# Download artifacts
gh run download RUN_ID
```

## 🔒 Security Best Practices

1. **Secrets Management**:
   - Use GitHub Secrets for sensitive data
   - Rotate AWS keys regularly
   - Limit IAM permissions to minimum required

2. **Resource Cleanup**:
   - Always enable automatic cleanup
   - Monitor for orphaned resources
   - Set resource tagging for easy identification

3. **Environment Isolation**:
   - Use separate AWS accounts for test/staging/prod
   - Implement network isolation
   - Monitor cross-environment access

4. **Access Control**:
   - Limit who can trigger workflows
   - Require reviews for workflow changes
   - Monitor workflow execution logs

## 📈 Performance Optimization

### Workflow Speed:
- **Parallel Jobs**: Tests run in parallel when possible
- **Caching**: Go modules and npm packages cached
- **Selective Triggers**: Only run on relevant file changes

### Cost Management:
- **Resource Cleanup**: Automatic deletion after tests
- **Scheduled Limits**: Health checks limited to 4 times daily
- **Efficient Builds**: Minimal Lambda package sizes

### Monitoring:
- **Execution Time**: Track workflow duration trends
- **Success Rates**: Monitor test pass/fail ratios
- **Resource Usage**: AWS cost tracking for test resources

## 📞 Support

For issues with the GitHub Actions setup:

1. **Check workflow logs** in the Actions tab
2. **Review test reports** in workflow artifacts
3. **Verify AWS permissions** and secrets
4. **Test locally** with same environment configuration

### Integration with External Tools:
- **Slack Notifications**: Add webhook for test results
- **JIRA Integration**: Auto-create issues for test failures
- **Monitoring Tools**: Send metrics to DataDog/NewRelic
- **Status Badges**: Add workflow status to README