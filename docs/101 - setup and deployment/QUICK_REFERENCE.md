# Quick Deployment Reference

Quick commands for deploying the Tenant Portal infrastructure.

## Prerequisites

```bash
# Verify AWS CLI
aws --version

# Verify authentication
aws sts get-caller-identity

# Get your AWS Account ID
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
echo "AWS Account: $AWS_ACCOUNT_ID"
```

## Route 53 Setup (One-time)

```bash
# Create Hosted Zone
aws route53 create-hosted-zone \
  --name your-domain.com \
  --caller-reference $(date +%s)

# Get Hosted Zone ID
export HOSTED_ZONE_ID=$(aws route53 list-hosted-zones \
  --query "HostedZones[?Name=='your-domain.com.'].Id" \
  --output text | cut -d'/' -f3)

echo "Hosted Zone ID: $HOSTED_ZONE_ID"

# Get nameservers (update at domain registrar)
aws route53 get-hosted-zone --id $HOSTED_ZONE_ID \
  --query "DelegationSet.NameServers" \
  --output table
```

## Update Template Mappings

Edit `cfn/tenant-cfn/template.yaml` (lines 43-50):

```yaml
Mappings:
  AccountMappings:
    "YOUR_ACCOUNT_ID":
      APSouthDomainName: dev.your-domain.com
      APSouthHostedZoneId: YOUR_HOSTED_ZONE_ID
      TenantClientTokenValidityHRS: 10
      TenantClientAuthSessionValidityMin: 5
```

## Deployment Commands

### Step 1: Deploy CloudFront Key Stack

```bash
cd cfn/cloudfront-key-cfn
make deploy ENVIRONMENT=dev AWS_REGION=ap-southeast-2
```

**Verify:**
```bash
aws cloudformation describe-stacks \
  --stack-name cloudfront-public-key-stack-dev \
  --region ap-southeast-2 \
  --query 'Stacks[0].Outputs' --output table
```

### Step 2: Deploy Tenant Portal Stack

```bash
cd ../tenant-cfn
make deploy ENVIRONMENT=dev
```

**Monitor:**
```bash
watch aws cloudformation describe-stacks \
  --stack-name tenant-portal-apis-dev \
  --query 'Stacks[0].StackStatus'
```

## Verification

```bash
# Check API Gateway
aws cloudformation describe-stacks \
  --stack-name tenant-portal-apis-dev \
  --query 'Stacks[0].Outputs[?OutputKey==`ApiUrl`].OutputValue' \
  --output text

# Test Cognito sync
aws cognito-idp admin-create-user \
  --user-pool-id <USER_POOL_ID> \
  --username test@example.com \
  --user-attributes Name=email,Value=test@example.com

# Check DynamoDB
aws dynamodb get-item \
  --table-name EmployeeDataTable-dev \
  --key '{"UserName": {"S": "test@example.com"}}'
```

## Cleanup

```bash
# Delete tenant stack
cd cfn/tenant-cfn
make undeploy ENVIRONMENT=dev

# Delete cloudfront key stack
cd ../cloudfront-key-cfn
make delete ENVIRONMENT=dev AWS_REGION=ap-southeast-2
```

## Deployment Order

**IMPORTANT**: Always deploy in this order:

1. ✅ Route 53 setup (one-time)
2. ✅ Update template mappings
3. ✅ Deploy CloudFront Key stack
4. ✅ Deploy Tenant Portal stack

For detailed instructions, see [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md)
