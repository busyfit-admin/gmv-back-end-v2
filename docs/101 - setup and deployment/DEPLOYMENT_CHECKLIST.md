# Deployment Checklist

Use this checklist to ensure a successful deployment of the Tenant Portal infrastructure.

---

## Pre-Deployment Checklist

### Prerequisites
- [ ] AWS CLI installed (`aws --version`)
- [ ] AWS credentials configured (`aws sts get-caller-identity`)
- [ ] Make utility installed (`make --version`)
- [ ] Git repository cloned and up-to-date
- [ ] Python 3.11 available (for Lambda functions)
- [ ] Appropriate IAM permissions verified

### AWS Account Information
- [ ] AWS Account ID recorded: `_________________________`
- [ ] Target region confirmed: `ap-south-1` (tenant) / `ap-southeast-2` (cloudfront keys)
- [ ] Environment determined: `dev` / `test` / `uat` / `prod`

---

## Route 53 Setup (One-Time)

### Domain Configuration
- [ ] Domain name decided: `_________________________`
- [ ] Domain registered or transferred to Route 53
- [ ] Hosted Zone created in Route 53
- [ ] Hosted Zone ID recorded: `_________________________`
- [ ] Nameservers retrieved from Route 53
- [ ] Nameservers updated at domain registrar
- [ ] DNS propagation verified (can take 24-48 hours)
  ```bash
  dig NS your-domain.com
  ```

### Subdomain Strategy (if applicable)
- [ ] Dev subdomain: `dev.your-domain.com`
- [ ] UAT subdomain: `uat.your-domain.com`
- [ ] Prod domain: `www.your-domain.com` or `your-domain.com`

---

## Template Configuration

### Update cfn/tenant-cfn/template.yaml
- [ ] File opened: `cfn/tenant-cfn/template.yaml`
- [ ] Navigated to Mappings section (around line 43)
- [ ] AWS Account ID added/verified in AccountMappings
- [ ] APSouthDomainName updated with your domain
- [ ] APSouthHostedZoneId updated (without `/hostedzone/` prefix)
- [ ] TenantClientTokenValidityHRS configured (10-24 hours recommended)
- [ ] TenantClientAuthSessionValidityMin configured (5-15 minutes recommended)
- [ ] Changes saved
- [ ] Changes committed to git (optional but recommended)
  ```bash
  git add cfn/tenant-cfn/template.yaml
  git commit -m "Update domain mappings for [environment]"
  ```

---

## Step 1: CloudFront Key Stack Deployment

### Pre-Deployment
- [ ] Changed to cloudfront-key-cfn directory
  ```bash
  cd cfn/cloudfront-key-cfn
  ```
- [ ] Environment variable set
  ```bash
  export ENVIRONMENT=dev
  ```

### Deployment
- [ ] Deployment command executed:
  ```bash
  make deploy ENVIRONMENT=$ENVIRONMENT AWS_REGION=ap-southeast-2
  ```
- [ ] Lambda layer built successfully
- [ ] Stack deployment initiated
- [ ] Deployment completed without errors

### Verification
- [ ] Stack status checked:
  ```bash
  aws cloudformation describe-stacks \
    --stack-name cloudfront-public-key-stack-$ENVIRONMENT \
    --region ap-southeast-2 \
    --query 'Stacks[0].StackStatus'
  ```
  Expected: `"CREATE_COMPLETE"` or `"UPDATE_COMPLETE"`

- [ ] Stack outputs retrieved and recorded:
  ```bash
  aws cloudformation describe-stacks \
    --stack-name cloudfront-public-key-stack-$ENVIRONMENT \
    --region ap-southeast-2 \
    --query 'Stacks[0].Outputs' \
    --output table
  ```
  
  - [ ] PublicKeyId: `_________________________`
  - [ ] PrivateKeySecretArn: `_________________________`
  - [ ] PublicKeyValue visible (PEM format)

### Troubleshooting (if needed)
- [ ] CloudWatch Logs checked for Lambda errors
- [ ] Stack events reviewed for failure reasons
- [ ] Build script permissions verified (`chmod +x build-layer.sh`)

---

## Step 2: Tenant Portal Stack Deployment

### Pre-Deployment
- [ ] Changed to tenant-cfn directory
  ```bash
  cd cfn/tenant-cfn
  ```
- [ ] Environment variable still set
  ```bash
  echo $ENVIRONMENT
  ```
- [ ] CloudFront Key stack outputs confirmed available

### Deployment
- [ ] Deployment command executed:
  ```bash
  make deploy ENVIRONMENT=$ENVIRONMENT
  ```
- [ ] CloudFront keys automatically fetched (check output)
- [ ] Lambda functions packaged
- [ ] Template uploaded to S3
- [ ] Stack deployment initiated
- [ ] Deployment completed (10-15 minutes)

### Verification
- [ ] Stack status checked:
  ```bash
  aws cloudformation describe-stacks \
    --stack-name tenant-portal-apis-$ENVIRONMENT \
    --query 'Stacks[0].StackStatus'
  ```
  Expected: `"CREATE_COMPLETE"` or `"UPDATE_COMPLETE"`

- [ ] CloudFront parameters verified in stack:
  ```bash
  aws cloudformation describe-stacks \
    --stack-name tenant-portal-apis-$ENVIRONMENT \
    --query 'Stacks[0].Parameters[?ParameterKey==`CloudFrontPublicKeyIdParam`]'
  ```

---

## Post-Deployment Verification

### API Gateway
- [ ] API URL retrieved:
  ```bash
  aws cloudformation describe-stacks \
    --stack-name tenant-portal-apis-$ENVIRONMENT \
    --query 'Stacks[0].Outputs[?OutputKey==`ApiUrl`].OutputValue' \
    --output text
  ```
  API URL: `_________________________`

- [ ] API health check tested:
  ```bash
  curl -I https://dev.your-domain.com/health
  ```
  Expected: `200 OK` or `404` (if no health endpoint)

### Custom Domain
- [ ] Custom domain accessible
- [ ] SSL certificate validated and active
- [ ] Route 53 record created and resolving

### DynamoDB Tables
- [ ] EmployeeDataTable exists and active:
  ```bash
  aws dynamodb describe-table \
    --table-name EmployeeDataTable-$ENVIRONMENT \
    --query 'Table.TableStatus'
  ```
  Expected: `"ACTIVE"`

- [ ] All tables created (15+ tables):
  ```bash
  aws dynamodb list-tables \
    --query 'TableNames[?contains(@, `'$ENVIRONMENT'`)]'
  ```

### Cognito User Pool
- [ ] User Pool created:
  ```bash
  aws cloudformation describe-stack-resources \
    --stack-name tenant-portal-apis-$ENVIRONMENT \
    --logical-resource-id TenantCognitoUserPool
  ```

- [ ] User Pool ID recorded: `_________________________`

- [ ] Lambda triggers configured (PostConfirmation, PreAuthentication, PreTokenGeneration)

### Lambda Functions
- [ ] All Lambda functions deployed:
  ```bash
  aws lambda list-functions \
    --query "Functions[?contains(FunctionName, '$ENVIRONMENT')].[FunctionName, Runtime]" \
    --output table
  ```

- [ ] SyncCognitoUserToDynamoDB function exists

### Test Cognito Sync
- [ ] Test user created in Cognito:
  ```bash
  aws cognito-idp admin-create-user \
    --user-pool-id <USER_POOL_ID> \
    --username test@example.com \
    --user-attributes Name=email,Value=test@example.com Name=name,Value="Test User" \
    --message-action SUPPRESS
  ```

- [ ] Test user password set:
  ```bash
  aws cognito-idp admin-set-user-password \
    --user-pool-id <USER_POOL_ID> \
    --username test@example.com \
    --password TestPassword123! \
    --permanent
  ```

- [ ] User synced to DynamoDB:
  ```bash
  aws dynamodb get-item \
    --table-name EmployeeDataTable-$ENVIRONMENT \
    --key '{"UserName": {"S": "test@example.com"}}'
  ```
  Expected: User record returned

### CloudWatch Logs
- [ ] No critical errors in logs
- [ ] Sync Lambda logs checked:
  ```bash
  aws logs tail /aws/lambda/SyncCognitoUserToDynamoDB-$ENVIRONMENT --follow
  ```

---

## Documentation

- [ ] Deployment details documented (date, time, environment, deployer)
- [ ] API URLs shared with team
- [ ] Cognito User Pool ID shared with team
- [ ] CloudFront distribution ID recorded (if needed)
- [ ] Any issues encountered documented

---

## Post-Deployment Tasks

- [ ] Smoke tests performed
- [ ] Team notified of deployment
- [ ] Monitoring dashboards configured (if applicable)
- [ ] Backup/DR procedures reviewed
- [ ] Cost alerts configured (if applicable)
- [ ] Security groups reviewed
- [ ] IAM roles reviewed for least privilege

---

## Rollback Plan (if needed)

If deployment fails or issues arise:

- [ ] Rollback tenant stack:
  ```bash
  cd cfn/tenant-cfn
  make undeploy ENVIRONMENT=$ENVIRONMENT
  ```

- [ ] Rollback cloudfront key stack (only if needed):
  ```bash
  cd cfn/cloudfront-key-cfn
  make delete ENVIRONMENT=$ENVIRONMENT AWS_REGION=ap-southeast-2
  ```

- [ ] Root cause analysis performed
- [ ] Issues documented for future reference

---

## Sign-Off

**Deployer Name**: `_________________________`

**Date**: `_________________________`

**Environment**: `_________________________`

**Deployment Status**: ☐ Success  ☐ Failed  ☐ Partial

**Notes**:
```
_________________________________________________________________

_________________________________________________________________

_________________________________________________________________
```

---

**For detailed troubleshooting**, see [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)

**For quick commands**, see [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
