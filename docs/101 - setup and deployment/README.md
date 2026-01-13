# Tenant Portal - Documentation

Welcome to the Tenant Portal deployment documentation.

## Available Guides

### 📘 [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)
Complete step-by-step deployment guide covering:
- Prerequisites and AWS setup
- Route 53 domain configuration
- CloudFormation template mappings
- CloudFront Key stack deployment
- Tenant Portal stack deployment
- Verification procedures
- Troubleshooting common issues

**Use this for**: First-time deployment or when you need detailed explanations

---

### ⚡ [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
Quick command reference for experienced users:
- Essential commands only
- Deployment order checklist
- Verification commands
- Cleanup procedures

**Use this for**: Quick deployments when you're familiar with the process

---

## Deployment Order

**CRITICAL**: Always follow this sequence:

```
1. Route 53 Setup (one-time)
   └─> Create hosted zone
   └─> Update domain registrar nameservers
   └─> Wait for DNS propagation (24-48 hrs)

2. Configure Template
   └─> Update cfn/tenant-cfn/template.yaml Mappings section
   └─> Add your AWS Account ID
   └─> Add your domain and hosted zone ID

3. Deploy CloudFront Key Stack
   └─> cd cfn/cloudfront-key-cfn
   └─> make deploy ENVIRONMENT=dev AWS_REGION=ap-southeast-2

4. Deploy Tenant Portal Stack
   └─> cd cfn/tenant-cfn
   └─> make deploy ENVIRONMENT=dev
```

## Quick Start

```bash
# 1. Verify prerequisites
aws --version
aws sts get-caller-identity

# 2. Deploy CloudFront keys (FIRST!)
cd cfn/cloudfront-key-cfn
make deploy ENVIRONMENT=dev AWS_REGION=ap-southeast-2

# 3. Deploy Tenant Portal (SECOND!)
cd ../tenant-cfn
make deploy ENVIRONMENT=dev
```

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     User / Client                        │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │   Route 53 (Custom Domain) │
        │   dev.your-domain.com      │
        └────────────┬───────────────┘
                     │
                     ▼
        ┌────────────────────────────┐
        │   API Gateway (REST API)   │
        │   + ACM Certificate        │
        └────────────┬───────────────┘
                     │
         ┌───────────┴───────────┐
         │                       │
         ▼                       ▼
┌────────────────┐    ┌─────────────────────┐
│  Lambda Funcs  │    │  Cognito User Pool  │
│  (15+)         │    │  + Sync Lambda      │
└────────┬───────┘    └──────────┬──────────┘
         │                       │
         ▼                       ▼
┌────────────────────────────────────────┐
│        DynamoDB Tables (15+)           │
│  EmployeeData, Rewards, Teams, etc.    │
└────────────────────────────────────────┘

┌────────────────────────────────────────┐
│      CloudFront + S3 (Content)         │
│  Signed URLs (CloudFront Key Stack)    │
└────────────────────────────────────────┘
```

## Key Components

### CloudFront Key Stack
- **Purpose**: Generate RSA keys for signed URLs
- **Region**: ap-southeast-2
- **Outputs**: PublicKeyId, PrivateKeySecretArn
- **Must deploy FIRST**

### Tenant Portal Stack
- **Purpose**: Main application infrastructure
- **Region**: ap-south-1 (default)
- **Depends on**: CloudFront Key Stack outputs
- **Deploy SECOND**

## Resources Created

### CloudFront Key Stack
- Lambda function (key generator)
- CloudFront Public Key
- Secrets Manager secret (private key)
- SSM Parameters

### Tenant Portal Stack
- **API**: 1 API Gateway with custom domain
- **Compute**: 15+ Lambda functions
- **Storage**: 15+ DynamoDB tables, 2 S3 buckets
- **Auth**: Cognito User Pool + Client
- **Networking**: CloudFront distribution, Route 53 records
- **Monitoring**: CloudWatch Log Groups

## Environment Variables

Common environments:
- `dev` - Development
- `test` - Testing  
- `uat` - User Acceptance Testing
- `prod` - Production

Set before deployment:
```bash
export ENVIRONMENT=dev
```

## Getting Help

1. **Check the guides**:
   - Detailed: [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)
   - Quick: [QUICK_REFERENCE.md](QUICK_REFERENCE.md)

2. **Check CloudWatch Logs**:
   ```bash
   aws logs tail /aws/lambda/<function-name> --follow
   ```

3. **Check CloudFormation Events**:
   ```bash
   aws cloudformation describe-stack-events \
     --stack-name tenant-portal-apis-dev \
     --max-items 20
   ```

4. **Verify Prerequisites**:
   - AWS CLI configured
   - Route 53 domain setup complete
   - Template mappings updated
   - CloudFront Key stack deployed

## Related Documentation

- Cognito Sync Lambda: `lambdas/tenant-lambdas/utils-functions/sync-cognito-to-ddb/README.md`
- CloudFront Keys: `cfn/cloudfront-key-cfn/README.md`
- Tenant CFN: `cfn/tenant-cfn/CLOUDFRONT_KEYS_INTEGRATION.md`

## Support

For issues:
1. Review deployment guide troubleshooting section
2. Check AWS CloudWatch Logs
3. Verify all prerequisites
4. Contact DevOps team

---

**Last Updated**: January 13, 2026
