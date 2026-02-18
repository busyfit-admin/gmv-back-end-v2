# GitHub Deployer IAM User Setup

This directory contains Infrastructure as Code (IaC) to create a secure IAM user for GitHub Actions with minimal required permissions.

## � Prerequisites

**Before using this deployer, you must have AWS credentials configured with sufficient permissions to:**
- Create IAM users and policies
- Create S3 buckets
- Deploy CloudFormation stacks

### AWS Credentials Setup Options:

#### Option 1: AWS CLI Configure (Recommended)
```bash
aws configure
# AWS Access Key ID: [Enter your admin access key]
# AWS Secret Access Key: [Enter your admin secret key]
# Default region name: ap-southeast-2
# Default output format: json
```

#### Option 2: Environment Variables
```bash
export AWS_ACCESS_KEY_ID="your-admin-access-key-id"
export AWS_SECRET_ACCESS_KEY="your-admin-secret-access-key"
export AWS_DEFAULT_REGION="ap-southeast-2"
```

#### Option 3: AWS Profile
```bash
aws configure --profile admin
export AWS_PROFILE=admin
```

### Verify Your Credentials
```bash
aws sts get-caller-identity
```
You should see your AWS account ID and user ARN.

## �🚀 Quick Start

1. **Deploy the IAM User**:
   ```bash
   cd .github/github_deployer
   make deploy
   ```

2. **Get GitHub Secrets**:
   ```bash
   make credentials
   ```

3. **Copy the output to GitHub repository secrets** under `Settings > Secrets and variables > Actions`

## 📁 Files

- **[template.yaml](template.yaml)**: CloudFormation template defining IAM user, policy, and S3 bucket
- **[Makefile](Makefile)**: Automation scripts for deployment and management
- **[README.md](README.md)**: This documentation

## 🔧 Available Commands

```bash
make help              # Show all available commands
make deploy            # Deploy IAM user and resources
make credentials       # Display GitHub secrets to copy
make status           # Check deployment status
make verify           # Verify IAM permissions
make test-credentials # Test if credentials work
make rotate-keys      # Rotate access keys (invalidates current secrets)
make delete           # Clean up all resources
```

## 🛡️ Security Features

### **Principle of Least Privilege**
- Permissions limited to specific resources only
- No wildcard (`*`) permissions on sensitive actions
- Resource-specific ARNs for enhanced security

### **Created Resources**
- **IAM User**: `gmv-github-actions-ci`
- **IAM Policy**: Minimal required permissions
- **S3 Bucket**: `gmv-test-deployment-bucket` for Lambda packages
- **Access Keys**: For programmatic access

### **Permissions Granted**
```yaml
CloudFormation:     # Limited to tenant-portal-apis-test stack only
S3:                # Limited to deployment bucket only
Lambda:            # Limited to tenant-portal-* functions
API Gateway:       # All operations (required for CloudFormation)
Cognito:           # User pool management for testing
DynamoDB:          # Limited to tenant-portal-* tables
IAM:              # Limited to tenant-portal-* and lambda-execution-* roles
CloudWatch Logs:   # Limited to Lambda and API Gateway logs
```

## 📋 Usage Examples

### First Time Setup
```bash
# 1. Navigate to directory
cd .github/github_deployer

# 2. Deploy infrastructure
make deploy

# 3. Get credentials for GitHub
make credentials

# 4. Copy the displayed secrets to GitHub repository settings
# Settings > Secrets and variables > Actions > New repository secret
```

### Verification
```bash
# Check if deployment was successful
make status

# Verify IAM user has correct permissions
make verify

# Test if the credentials work
make test-credentials
```

### Maintenance
```bash
# Rotate access keys (do this every 90 days)
make rotate-keys

# View recent CloudFormation events
make events

# Check all your CloudFormation stacks
make list-stacks
```

### Cleanup
```bash
# Delete everything (WARNING: This will break GitHub Actions)
make delete
```

## ⚙️ Configuration

### Default Settings
- **Stack Name**: `gmv-github-actions-deployer`
- **AWS Region**: `ap-southeast-2`
- **S3 Bucket**: `gmv-test-deployment-bucket`
- **IAM User**: `gmv-github-actions-ci`

### Customization
Edit the Makefile variables at the top:
```makefile
STACK_NAME = your-custom-stack-name
AWS_REGION = your-region
PROJECT_NAME = your-project
S3_BUCKET = your-bucket-name
```

## 🔐 GitHub Secrets Setup

After running `make credentials`, add these secrets to your GitHub repository:

| Secret Name | Description |
|-------------|-------------|
| `AWS_ACCESS_KEY_ID` | Access key for GitHub Actions |
| `AWS_SECRET_ACCESS_KEY` | Secret access key for GitHub Actions |
| `LAMBDA_ARTIFACTS_BUCKET` | S3 bucket for Lambda deployment packages |

### Adding Secrets to GitHub:
1. Go to your repository on GitHub
2. Navigate to `Settings > Secrets and variables > Actions`
3. Click `New repository secret`
4. Add each secret from the `make credentials` output

## 🚨 Security Best Practices

### 1. **Credential Rotation**
```bash
# Rotate keys every 90 days
make rotate-keys
# Then update GitHub secrets with new values
make credentials
```

### 2. **Monitoring**
- Monitor AWS CloudTrail for API calls from this user
- Set up billing alerts for unusual costs
- Regular security audits

### 3. **Access Control**
- Only repository administrators should have access to secrets
- Use branch protection rules
- Require reviews for workflow changes

## 🔍 Troubleshooting

### Common Issues

1. **Deployment fails with permission error**:
   ```bash
   # Check your AWS credentials
   aws sts get-caller-identity
   ```

2. **Stack already exists**:
   ```bash
   # Update existing stack
   make update
   ```

3. **GitHub Actions still failing**:
   ```bash
   # Test the credentials
   make test-credentials
   
   # Verify permissions
   make verify
   ```

4. **Need to see what went wrong**:
   ```bash
   # Check CloudFormation events
   make events
   
   # Check stack status
   make status
   ```

### Debug Commands
```bash
# Validate template before deploying
make validate

# Lint template with cfn-lint (if installed)
make lint

# Check all outputs
make outputs
```

## 📊 Cost Information

This setup creates minimal resources:
- **IAM User**: No cost
- **IAM Policy**: No cost
- **S3 Bucket**: Storage costs only (~$0.023/GB/month)
- **Access Keys**: No cost

Estimated monthly cost: **< $1** (depends on S3 usage)

## 🔄 Automation Integration

This IAM user works with:
- ✅ **deploy-test-and-validate.yml** - Main CI/CD pipeline
- ✅ **build-lambdas.yml** - Lambda build and package
- ✅ **health-checks.yml** - Monitoring workflows

## 📞 Support

For issues:
1. Check the troubleshooting section above
2. Run `make help` for available commands
3. Verify AWS permissions and region settings
4. Check CloudFormation events with `make events`