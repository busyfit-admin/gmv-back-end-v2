# CloudFront Public Key Generator

This CloudFormation stack automatically generates an RSA 2048 key pair for CloudFront signed URLs.

## What it does

1. **Generates RSA 2048 Key Pair**: Creates a cryptographically secure RSA 2048 key pair
2. **CloudFront Public Key**: Creates a CloudFront Public Key resource with the generated public key
3. **Secrets Manager**: Stores both the private and public keys securely in AWS Secrets Manager
4. **SSM Parameters**: Stores the Public Key ID and Secret ARN in Parameter Store for easy reference

## Prerequisites

- AWS CLI configured with appropriate credentials
- Permissions to create CloudFormation stacks, Lambda functions, IAM roles, CloudFront resources, and Secrets Manager secrets

## Deployment

### Deploy the stack

```bash
# Deploy to dev environment
make deploy ENVIRONMENT=dev

# Deploy to test environment  
make deploy ENVIRONMENT=test

# Deploy to prod environment
make deploy ENVIRONMENT=prod
```

### View stack outputs

```bash
make outputs ENVIRONMENT=dev
```

### Delete the stack

```bash
make delete ENVIRONMENT=dev
```

### Validate template

```bash
make validate
```

## Outputs

After deployment, the stack provides:

- **PublicKeyId**: The CloudFront Public Key ID (use this for signed URLs)
- **PrivateKeySecretArn**: ARN of the Secrets Manager secret containing the private key
- **PublicKeyIdParameterName**: SSM Parameter path for the Public Key ID
- **PrivateKeySecretArnParameterName**: SSM Parameter path for the Secret ARN

## Using the Keys

### Get the Public Key ID

```bash
aws ssm get-parameter --name /cloudfront/public-key-id/dev --query Parameter.Value --output text
```

### Get the Private Key

```bash
aws secretsmanager get-secret-value \
  --secret-id cloudfront-private-key-dev \
  --query SecretString \
  --output text | jq -r .private_key
```

## Architecture

The stack uses:
- **AWS Lambda**: Custom resource to generate RSA keys using Python cryptography library
- **CloudFront Public Key**: Stores the public key for signed URL verification
- **Secrets Manager**: Securely stores the private key
- **SSM Parameter Store**: Stores references for easy access
- **IAM Role**: Minimal permissions for Lambda execution

## Notes

- The Lambda function uses the Klayers cryptography layer for RSA key generation
- Keys are generated only once during stack creation
- Private keys are stored encrypted in Secrets Manager
- Stack deletion will remove all resources including keys (use caution in production)

## Security Considerations

- Private keys are never exposed in CloudFormation outputs or logs
- Keys are encrypted at rest in Secrets Manager
- IAM roles follow least privilege principle
- For production, consider additional key rotation policies
