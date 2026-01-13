# CloudFront Keys Integration - Quick Reference

## Overview
The tenant stack automatically fetches CloudFront keys from the `cloudfront-public-key-stack` when deploying.

## How It Works

### 1. Makefile Variables (Automatically Set)
```makefile
CLOUDFRONT_KEY_STACK := cloudfront-public-key-stack-$(ENVIRONMENT)
CLOUDFRONT_PUBLIC_KEY_ID := Fetched from stack outputs
CLOUDFRONT_PRIVATE_KEY_SECRET_ARN := Fetched from stack outputs
```

### 2. Template Parameters
The template accepts these parameters:
- `CloudFrontPublicKeyIdParam` - The public key ID
- `CloudFrontPrivateKeySecretArn` - ARN of the secret containing private key

### 3. Resources Configuration

**CloudFrontPublicKeyId** (SSM Parameter)
- Uses `CloudFrontPublicKeyIdParam` if provided
- Falls back to hardcoded value if not provided

**PrivateKeySecretsCloudfront** (SSM Parameter)
- Stores the ARN reference to the Secrets Manager secret
- Uses `CloudFrontPrivateKeySecretArn` if provided
- Falls back to constructing default ARN if not provided

### 4. Lambda Usage
All lambdas access these via environment variables:
```yaml
SECRETS_CND_PK_ARN: !GetAtt PrivateKeySecretsCloudfront.Value  # ARN of the secret
PUBLIC_KEY_ID: !GetAtt CloudFrontPublicKeyId.Value             # Public key ID
```

## Deployment Steps

### Normal Deployment (Automatic)
```bash
# 1. Deploy CloudFront keys first
cd cfn/cloudfront-key-cfn
make deploy ENVIRONMENT=test AWS_REGION=ap-southeast-2

# 2. Deploy tenant stack (automatically picks up keys)
cd ../tenant-cfn
make deploy ENVIRONMENT=test
```

The Makefile will:
- Query the cloudfront-key stack for outputs
- Pass CloudFront keys to the tenant stack deployment
- Display the keys being used

### Manual Override
You can manually specify keys if needed:
```bash
make deploy ENVIRONMENT=test \
  CLOUDFRONT_PUBLIC_KEY_ID=K1234567890ABC \
  CLOUDFRONT_PRIVATE_KEY_SECRET_ARN=arn:aws:secretsmanager:ap-southeast-2:123456789:secret:name
```

## Verification

Check if keys were fetched:
```bash
make deploy ENVIRONMENT=test
# Output will show:
# CloudFront Public Key ID: K12345...
# CloudFront Private Key Secret ARN: arn:aws:secretsmanager:...
```

Check CloudFront key stack outputs:
```bash
aws cloudformation describe-stacks \
  --stack-name cloudfront-public-key-stack-test \
  --region ap-southeast-2 \
  --query 'Stacks[0].Outputs'
```

## Important Notes

1. **Region**: CloudFront key stack is in `ap-southeast-2`, while tenant stack defaults to `ap-south-1`
2. **Stack Name**: CloudFront stack must be named `cloudfront-public-key-stack-{ENVIRONMENT}`
3. **Backward Compatible**: If CloudFront keys aren't found, uses default hardcoded values
4. **Secret Access**: Lambdas retrieve the private key from Secrets Manager using the ARN
