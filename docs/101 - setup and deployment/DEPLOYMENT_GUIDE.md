# Tenant Portal - Deployment Guide

Complete deployment guide for the Tenant Portal infrastructure on AWS.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Initial Route 53 Setup](#initial-route-53-setup)
3. [Configure Template Mappings](#configure-template-mappings)
4. [Step 1: Deploy CloudFront Key Stack](#step-1-deploy-cloudfront-key-stack)
5. [Step 2: Deploy Tenant Portal Stack](#step-2-deploy-tenant-portal-stack)
6. [Verification](#verification)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

Before starting the deployment, ensure you have:

- **AWS CLI** installed and configured (`aws --version`)
- **AWS Credentials** with permissions for:
  - CloudFormation
  - Lambda
  - S3
  - DynamoDB
  - Cognito
  - CloudFront
  - Route 53
  - ACM (Certificate Manager)
  - Secrets Manager
- **Make** utility installed
- **Git** access to the repository
- **Python 3.11** for Lambda functions

---

## Initial Route 53 Setup

Before deploying any CloudFormation stacks, you must configure your domain in Route 53.

### Option A: Purchase New Domain Through Route 53

1. **Register domain via AWS Console**:
   - Go to AWS Console → Route 53 → Domain Registration
   - Search for available domain
   - Complete registration process

2. **Hosted Zone is automatically created** when you register through Route 53

### Option B: Use Existing Domain

1. **Create Hosted Zone**:
   ```bash
   aws route53 create-hosted-zone \
     --name your-domain.com \
     --caller-reference $(date +%s) \
     --hosted-zone-config Comment="Tenant Portal domain"
   ```

2. **Get the Hosted Zone ID**:
   ```bash
   aws route53 list-hosted-zones \
     --query "HostedZones[?Name=='your-domain.com.'].{Name:Name,Id:Id}" \
     --output table
   ```

   Example output:
   ```
   -------------------------------------------------------
   |                  ListHostedZones                    |
   +---------------------------------+-------------------+
   |              Id                 |       Name        |
   +---------------------------------+-------------------+
   |  /hostedzone/Z07064771YL69HXR2  |  example.com.     |
   +---------------------------------+-------------------+
   ```

3. **Update DNS at domain registrar**:
   - Get Route 53 nameservers:
     ```bash
     aws route53 get-hosted-zone --id Z07064771YL69HXR2 \
       --query "DelegationSet.NameServers" --output table
     ```
   
   - Update your domain registrar's DNS settings with these nameservers
   - Wait for DNS propagation (can take 24-48 hours)

4. **Verify DNS propagation**:
   ```bash
   dig NS your-domain.com
   # or
   nslookup -type=NS your-domain.com
   ```

### Domain Setup for Different Environments

For different environments, you can use subdomains:

- **Development**: `dev.your-domain.com`
- **UAT**: `uat.your-domain.com`
- **Production**: `your-domain.com` or `www.your-domain.com`

---

## Configure Template Mappings

After setting up Route 53, you need to configure the CloudFormation template with your domain details.

### Edit `cfn/tenant-cfn/template.yaml`

1. **Open the template**:
   ```bash
   cd cfn/tenant-cfn
   vim template.yaml  # or use your preferred editor
   ```

2. **Locate the `Mappings` section** (around line 43):

   ```yaml
   Mappings:
     AccountMappings:
       "622778846370": # dev AWS Account ID
         APSouthDomainName: mvp-dev.4cl-tech.com.au
         APSouthHostedZoneId: Z07064771YL69HXR2PLAL
         TenantClientTokenValidityHRS: 10
         TenantClientAuthSessionValidityMin: 5
   ```

3. **Add your AWS account mappings**:

   ```yaml
   Mappings:
     AccountMappings:
       "622778846370": # dev account
         APSouthDomainName: dev.your-domain.com
         APSouthHostedZoneId: Z07064771YL69HXR2PLAL
         TenantClientTokenValidityHRS: 10
         TenantClientAuthSessionValidityMin: 5
       
       "YOUR_UAT_ACCOUNT_ID": # uat account
         APSouthDomainName: uat.your-domain.com
         APSouthHostedZoneId: Z08123456EXAMPLE
         TenantClientTokenValidityHRS: 12
         TenantClientAuthSessionValidityMin: 10
       
       "YOUR_PROD_ACCOUNT_ID": # prod account
         APSouthDomainName: your-domain.com
         APSouthHostedZoneId: Z09123456EXAMPLE
         TenantClientTokenValidityHRS: 24
         TenantClientAuthSessionValidityMin: 15
   ```

4. **Get your AWS Account ID**:
   ```bash
   aws sts get-caller-identity --query Account --output text
   ```

5. **Save the file** and commit changes:
   ```bash
   git add cfn/tenant-cfn/template.yaml
   git commit -m "Update domain mappings for deployment"
   ```

### Mapping Fields Explanation

- **APSouthDomainName**: Your domain or subdomain for the API Gateway
- **APSouthHostedZoneId**: Route 53 Hosted Zone ID (without `/hostedzone/` prefix)
- **TenantClientTokenValidityHRS**: Cognito access token validity in hours
- **TenantClientAuthSessionValidityMin**: Cognito auth session validity in minutes

---

## Step 1: Deploy CloudFront Key Stack

The CloudFront Key stack must be deployed **FIRST** as it generates RSA keys for CloudFront signed URLs, which are required by the Tenant Portal stack.

### Why Deploy This First?

The tenant portal uses CloudFront signed URLs for secure content delivery. The CloudFront Key stack:
- Generates RSA 2048 key pairs
- Creates CloudFront Public Key resources
- Stores private keys in Secrets Manager
- Outputs Public Key ID and Secret ARN for use by tenant stack

### Deployment Steps

1. **Navigate to CloudFront Key CFN directory**:
   ```bash
   cd cfn/cloudfront-key-cfn
   ```

2. **Set environment** (dev, uat, or prod):
   ```bash
   export ENVIRONMENT=dev
   ```

3. **Deploy the stack**:
   ```bash
   make deploy ENVIRONMENT=$ENVIRONMENT AWS_REGION=ap-southeast-2
   ```

   This command will:
   - Build the Lambda layer with cryptography library
   - Package the CloudFormation template
   - Deploy the stack
   - Create CloudFront public key
   - Generate and store RSA key pair

4. **Monitor deployment progress**:
   ```bash
   # Watch stack events
   aws cloudformation describe-stack-events \
     --stack-name cloudfront-public-key-stack-$ENVIRONMENT \
     --region ap-southeast-2 \
     --max-items 10
   ```

5. **Wait for completion** (typically 3-5 minutes)

6. **Verify deployment**:
   ```bash
   aws cloudformation describe-stacks \
     --stack-name cloudfront-public-key-stack-$ENVIRONMENT \
     --region ap-southeast-2 \
     --query 'Stacks[0].StackStatus'
   ```

   Expected output: `"CREATE_COMPLETE"`

7. **Retrieve stack outputs**:
   ```bash
   aws cloudformation describe-stacks \
     --stack-name cloudfront-public-key-stack-$ENVIRONMENT \
     --region ap-southeast-2 \
     --query 'Stacks[0].Outputs' \
     --output table
   ```

   You should see:
   - **PublicKeyId**: CloudFront Public Key ID (e.g., `K2JHDKAHS123`)
   - **PrivateKeySecretArn**: Secrets Manager ARN for private key
   - **PublicKeyValue**: The actual public key (PEM format)

---

## Step 2: Deploy Tenant Portal Stack

After successfully deploying the CloudFront Key stack, deploy the main Tenant Portal stack.

### Prerequisites Check

Before deploying, verify:

1. **CloudFront Key stack is deployed**:
   ```bash
   aws cloudformation describe-stacks \
     --stack-name cloudfront-public-key-stack-$ENVIRONMENT \
     --region ap-southeast-2 \
     --query 'Stacks[0].StackStatus'
   ```

2. **Route 53 domain is configured** (from earlier steps)

3. **Template mappings are updated** with your domain details

### Deployment Steps

1. **Navigate to Tenant CFN directory**:
   ```bash
   cd cfn/tenant-cfn
   ```

2. **Set environment**:
   ```bash
   export ENVIRONMENT=dev
   ```

3. **Deploy the stack**:
   ```bash
   make deploy ENVIRONMENT=$ENVIRONMENT
   ```

   The Makefile will automatically:
   - Fetch CloudFront Public Key ID from cloudfront-key stack
   - Fetch CloudFront Private Key Secret ARN
   - Package Lambda functions
   - Upload to S3 artifactory bucket
   - Deploy CloudFormation stack
   - Pass CloudFront parameters automatically

4. **Monitor deployment**:
   ```bash
   # Watch stack events
   aws cloudformation describe-stack-events \
     --stack-name tenant-portal-apis-$ENVIRONMENT \
     --max-items 20
   ```

5. **Wait for completion** (typically 10-15 minutes due to CloudFront distribution)

### What Gets Deployed

The tenant portal stack creates:

**Compute**:
- 15+ Lambda functions (employees, rewards, teams, surveys, etc.)
- Cognito sync Lambda (Python 3.11)

**Storage**:
- 15+ DynamoDB tables (EmployeeData, Rewards, Teams, etc.)
- S3 buckets for employee data uploads and content
- Secrets Manager for API keys

**Networking & API**:
- API Gateway REST API with custom domain
- CloudFront distribution for S3 content
- Route 53 DNS records

**Authentication**:
- Cognito User Pool
- Cognito User Pool Client
- Lambda triggers (PostConfirmation, PreAuthentication, PreTokenGeneration)

---

## Verification

After both stacks are deployed, verify the infrastructure is working correctly.

### 1. Verify CloudFront Keys Integration

```bash
# Check tenant stack parameters include CloudFront keys
aws cloudformation describe-stacks \
  --stack-name tenant-portal-apis-$ENVIRONMENT \
  --query 'Stacks[0].Parameters[?ParameterKey==`CloudFrontPublicKeyIdParam`]'
```

### 2. Verify API Gateway

```bash
# Get API Gateway URL
aws cloudformation describe-stacks \
  --stack-name tenant-portal-apis-$ENVIRONMENT \
  --query 'Stacks[0].Outputs[?OutputKey==`ApiUrl`].OutputValue' \
  --output text
```

### 3. Verify Custom Domain

```bash
# Check Route 53 record
aws route53 list-resource-record-sets \
  --hosted-zone-id Z07064771YL69HXR2PLAL \
  --query "ResourceRecordSets[?Name=='dev.your-domain.com.']"
```

---

## Troubleshooting

### Common Issues

**Issue**: Certificate validation pending
- **Solution**: Manually add validation CNAME to Route53

**Issue**: CloudFront public key not found
- **Solution**: Verify cloudfront-key stack is deployed and check outputs

**Issue**: Lambda function errors
- **Solution**: Check CloudWatch Logs

---

## Deployment Checklist

- [ ] Route 53 domain configured
- [ ] Hosted Zone ID obtained
- [ ] Template mappings updated
- [ ] CloudFront Key stack deployed
- [ ] CloudFront outputs verified
- [ ] Tenant Portal stack deployed
- [ ] API Gateway accessible
- [ ] DynamoDB tables active
- [ ] Cognito User Pool created
- [ ] Test user synced to DynamoDB

---

**Last Updated**: January 13, 2026
