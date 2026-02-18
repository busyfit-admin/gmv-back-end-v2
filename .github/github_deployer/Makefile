# Makefile for GitHub Actions IAM User Deployment

# Variables
STACK_NAME = gmv-github-actions-deployer
AWS_REGION = ap-southeast-2
PROJECT_NAME = gmv
S3_BUCKET = gmv-test-deployment-bucket

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[1;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

.PHONY: help deploy delete status outputs credentials validate lint clean

help: ## Show this help message
	@echo "$(BLUE)GitHub Actions IAM User Deployment$(NC)"
	@echo "====================================="
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Usage Examples:$(NC)"
	@echo "  make deploy          # Deploy the IAM user and get credentials"
	@echo "  make credentials     # Show credentials for GitHub secrets setup"
	@echo "  make delete          # Clean up all resources"

validate: ## Validate CloudFormation template
	@echo "$(BLUE)Validating CloudFormation template...$(NC)"
	@aws cloudformation validate-template \
		--template-body file://template.yaml \
		--region $(AWS_REGION) && \
	echo "$(GREEN)✓ Template is valid$(NC)" || \
	(echo "$(RED)✗ Template validation failed$(NC)" && exit 1)

lint: ## Lint CloudFormation template with cfn-lint
	@echo "$(BLUE)Linting CloudFormation template...$(NC)"
	@if command -v cfn-lint >/dev/null 2>&1; then \
		cfn-lint template.yaml && \
		echo "$(GREEN)✓ Template passes linting$(NC)"; \
	else \
		echo "$(YELLOW)⚠ cfn-lint not installed. Install with: pip install cfn-lint$(NC)"; \
	fi

deploy: validate ## Deploy the GitHub Actions IAM user and resources
	@echo "$(BLUE)Deploying GitHub Actions IAM User...$(NC)"
	@echo "Stack Name: $(STACK_NAME)"
	@echo "Region: $(AWS_REGION)"
	@echo "S3 Bucket: $(S3_BUCKET)"
	@echo ""
	
	@aws cloudformation deploy \
		--template-file template.yaml \
		--stack-name $(STACK_NAME) \
		--parameter-overrides \
			ProjectName=$(PROJECT_NAME) \
			S3DeploymentBucket=$(S3_BUCKET) \
		--capabilities CAPABILITY_NAMED_IAM \
		--region $(AWS_REGION) \
		--tags \
			Project=$(PROJECT_NAME) \
			Purpose="GitHub Actions CI/CD" \
			Environment="ci-cd" \
			CreatedBy="Makefile" && \
	echo "$(GREEN)✓ Deployment completed successfully$(NC)" || \
	(echo "$(RED)✗ Deployment failed$(NC)" && exit 1)
	
	@echo ""
	@echo "$(GREEN)Deployment completed! Use 'make credentials' to get GitHub secrets.$(NC)"

update: validate ## Update existing stack (same as deploy)
	@$(MAKE) deploy

status: ## Check deployment status
	@echo "$(BLUE)Checking stack status...$(NC)"
	@aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--region $(AWS_REGION) \
		--query 'Stacks[0].{Status:StackStatus,Created:CreationTime,Updated:LastUpdatedTime}' \
		--output table 2>/dev/null || \
	echo "$(YELLOW)Stack $(STACK_NAME) not found$(NC)"

outputs: ## Show all stack outputs
	@echo "$(BLUE)Stack Outputs:$(NC)"
	@aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME) \
		--region $(AWS_REGION) \
		--query 'Stacks[0].Outputs[*].{Key:OutputKey,Value:OutputValue,Description:Description}' \
		--output table 2>/dev/null || \
	echo "$(YELLOW)Stack $(STACK_NAME) not found$(NC)"

credentials: ## Display GitHub secrets setup instructions
	@echo "$(BLUE)GitHub Secrets Configuration$(NC)"
	@echo "=============================="
	@echo ""
	
	@if aws cloudformation describe-stacks --stack-name $(STACK_NAME) --region $(AWS_REGION) >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Stack $(STACK_NAME) found. Getting credentials...$(NC)"; \
		echo ""; \
		AWS_ACCESS_KEY_ID=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`AccessKeyId`].OutputValue' \
			--output text); \
		AWS_SECRET_ACCESS_KEY=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`SecretAccessKey`].OutputValue' \
			--output text); \
		S3_BUCKET_NAME=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`S3BucketName`].OutputValue' \
			--output text); \
		IAM_USER=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`IAMUserName`].OutputValue' \
			--output text); \
		echo "$(YELLOW)Add these secrets to your GitHub repository:$(NC)"; \
		echo "$(YELLOW)Repository Settings > Secrets and variables > Actions > New repository secret$(NC)"; \
		echo ""; \
		echo "$(GREEN)Secret Name:$(NC) AWS_ACCESS_KEY_ID"; \
		echo "$(GREEN)Secret Value:$(NC) $$AWS_ACCESS_KEY_ID"; \
		echo ""; \
		echo "$(GREEN)Secret Name:$(NC) AWS_SECRET_ACCESS_KEY"; \
		echo "$(GREEN)Secret Value:$(NC) $$AWS_SECRET_ACCESS_KEY"; \
		echo ""; \
		echo "$(GREEN)Secret Name:$(NC) LAMBDA_ARTIFACTS_BUCKET"; \
		echo "$(GREEN)Secret Value:$(NC) $$S3_BUCKET_NAME"; \
		echo ""; \
		echo "$(BLUE)Created Resources:$(NC)"; \
		echo "- IAM User: $$IAM_USER"; \
		echo "- S3 Bucket: $$S3_BUCKET_NAME"; \
		echo "- Region: $(AWS_REGION)"; \
		echo ""; \
		echo "$(YELLOW)⚠️  Important Security Notes:$(NC)"; \
		echo "- Store these credentials securely"; \
		echo "- These credentials have limited permissions"; \
		echo "- Rotate credentials every 90 days"; \
		echo "- Monitor usage in CloudTrail"; \
	else \
		echo "$(RED)✗ Stack $(STACK_NAME) not found. Run 'make deploy' first.$(NC)"; \
		exit 1; \
	fi

verify: ## Verify the IAM user has correct permissions
	@echo "$(BLUE)Verifying IAM user permissions...$(NC)"
	@if aws cloudformation describe-stacks --stack-name $(STACK_NAME) --region $(AWS_REGION) >/dev/null 2>&1; then \
		IAM_USER=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`IAMUserName`].OutputValue' \
			--output text); \
		echo "Checking permissions for user: $$IAM_USER"; \
		aws iam list-attached-user-policies --user-name $$IAM_USER --output table; \
		echo ""; \
		echo "$(GREEN)✓ Verification completed$(NC)"; \
	else \
		echo "$(RED)✗ Stack not found$(NC)"; \
		exit 1; \
	fi

test-credentials: ## Test the generated credentials work
	@echo "$(BLUE)Testing GitHub Actions credentials...$(NC)"
	@if aws cloudformation describe-stacks --stack-name $(STACK_NAME) --region $(AWS_REGION) >/dev/null 2>&1; then \
		AWS_ACCESS_KEY_ID=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`AccessKeyId`].OutputValue' \
			--output text); \
		AWS_SECRET_ACCESS_KEY=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`SecretAccessKey`].OutputValue' \
			--output text); \
		echo "Testing credentials..."; \
		AWS_ACCESS_KEY_ID=$$AWS_ACCESS_KEY_ID AWS_SECRET_ACCESS_KEY=$$AWS_SECRET_ACCESS_KEY \
		aws sts get-caller-identity --region $(AWS_REGION) && \
		echo "$(GREEN)✓ Credentials are working$(NC)" || \
		echo "$(RED)✗ Credentials test failed$(NC)"; \
	else \
		echo "$(RED)✗ Stack not found$(NC)"; \
		exit 1; \
	fi

rotate-keys: ## Rotate access keys (creates new, deletes old)
	@echo "$(BLUE)Rotating access keys...$(NC)"
	@echo "$(YELLOW)⚠️  This will invalidate current GitHub secrets!$(NC)"
	@if ! command -v jq >/dev/null 2>&1; then \
		echo "$(RED)✗ jq is required for key rotation. Install with:$(NC)"; \
		echo "  Ubuntu/Debian: sudo apt install jq"; \
		echo "  macOS: brew install jq"; \
		exit 1; \
	fi
	@read -p "Are you sure? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	
	@if aws cloudformation describe-stacks --stack-name $(STACK_NAME) --region $(AWS_REGION) >/dev/null 2>&1; then \
		IAM_USER=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`IAMUserName`].OutputValue' \
			--output text); \
		echo "$(BLUE)Rotating keys for user: $$IAM_USER$(NC)"; \
		echo ""; \
		OLD_KEY_ID=$$(aws cloudformation describe-stacks \
			--stack-name $(STACK_NAME) \
			--region $(AWS_REGION) \
			--query 'Stacks[0].Outputs[?OutputKey==`AccessKeyId`].OutputValue' \
			--output text); \
		echo "$(YELLOW)Current Key ID: $$OLD_KEY_ID$(NC)"; \
		echo ""; \
		echo "$(BLUE)Creating new access key...$(NC)"; \
		NEW_KEY_OUTPUT=$$(aws iam create-access-key --user-name $$IAM_USER --output json); \
		NEW_KEY_ID=$$(echo $$NEW_KEY_OUTPUT | jq -r '.AccessKey.AccessKeyId'); \
		NEW_SECRET_KEY=$$(echo $$NEW_KEY_OUTPUT | jq -r '.AccessKey.SecretAccessKey'); \
		echo "$(GREEN)✓ New Key ID: $$NEW_KEY_ID$(NC)"; \
		echo ""; \
		echo "$(BLUE)Waiting for key propagation...$(NC)"; \
		sleep 15; \
		echo "$(BLUE)Testing new credentials...$(NC)"; \
		if AWS_ACCESS_KEY_ID=$$NEW_KEY_ID AWS_SECRET_ACCESS_KEY=$$NEW_SECRET_KEY aws sts get-caller-identity --region $(AWS_REGION) >/dev/null 2>&1; then \
			echo "$(GREEN)✓ New credentials are working$(NC)"; \
			echo ""; \
			echo "$(BLUE)Deleting old access key...$(NC)"; \
			aws iam delete-access-key --user-name $$IAM_USER --access-key-id $$OLD_KEY_ID && \
			echo "$(GREEN)✓ Old key deleted successfully$(NC)"; \
			echo ""; \
			echo "$(YELLOW)New GitHub Secrets:$(NC)"; \
			echo "$(GREEN)AWS_ACCESS_KEY_ID:$(NC) $$NEW_KEY_ID"; \
			echo "$(GREEN)AWS_SECRET_ACCESS_KEY:$(NC) $$NEW_SECRET_KEY"; \
			echo ""; \
			echo "$(YELLOW)⚠️  Update your GitHub repository secrets immediately!$(NC)"; \
			echo "$(YELLOW)The old credentials are now invalid.$(NC)"; \
		else \
			echo "$(RED)✗ New credentials failed test. Debugging...$(NC)"; \
			echo "$(YELLOW)Testing with verbose output:$(NC)"; \
			AWS_ACCESS_KEY_ID=$$NEW_KEY_ID AWS_SECRET_ACCESS_KEY=$$NEW_SECRET_KEY aws sts get-caller-identity --region $(AWS_REGION) 2>&1 || true; \
			echo ""; \
			echo "$(YELLOW)Checking IAM user permissions...$(NC)"; \
			aws iam list-attached-user-policies --user-name $$IAM_USER --output table 2>/dev/null || true; \
			echo ""; \
			echo "$(BLUE)Cleaning up failed key...$(NC)"; \
			aws iam delete-access-key --user-name $$IAM_USER --access-key-id $$NEW_KEY_ID; \
			echo "$(RED)Key rotation failed. Old key is still active.$(NC)"; \
			echo "$(YELLOW)Trying to get current credentials instead...$(NC)"; \
			$(MAKE) credentials; \
		fi; \
	else \
		echo "$(RED)✗ Stack not found$(NC)"; \
		exit 1; \
	fi

events: ## Show recent CloudFormation events
	@echo "$(BLUE)Recent CloudFormation Events:$(NC)"
	@aws cloudformation describe-stack-events \
		--stack-name $(STACK_NAME) \
		--region $(AWS_REGION) \
		--max-items 10 \
		--query 'StackEvents[*].{Time:Timestamp,Status:ResourceStatus,Type:ResourceType,Reason:ResourceStatusReason}' \
		--output table 2>/dev/null || \
	echo "$(YELLOW)Stack $(STACK_NAME) not found$(NC)"

delete: ## Delete the CloudFormation stack and all resources
	@echo "$(RED)⚠️  This will delete the IAM user and invalidate GitHub secrets!$(NC)"
	@read -p "Are you sure you want to delete stack $(STACK_NAME)? (y/N): " confirm && [ "$$confirm" = "y" ] || exit 1
	
	@echo "$(BLUE)Deleting CloudFormation stack...$(NC)"
	@aws cloudformation delete-stack \
		--stack-name $(STACK_NAME) \
		--region $(AWS_REGION) && \
	echo "$(YELLOW)Deletion initiated. Waiting for completion...$(NC)" && \
	aws cloudformation wait stack-delete-complete \
		--stack-name $(STACK_NAME) \
		--region $(AWS_REGION) && \
	echo "$(GREEN)✓ Stack deleted successfully$(NC)" || \
	echo "$(RED)✗ Stack deletion failed or timed out$(NC)"

clean: delete ## Alias for delete

list-stacks: ## List all CloudFormation stacks
	@echo "$(BLUE)CloudFormation Stacks:$(NC)"
	@aws cloudformation list-stacks \
		--region $(AWS_REGION) \
		--stack-status-filter CREATE_COMPLETE UPDATE_COMPLETE \
		--query 'StackSummaries[*].{Name:StackName,Status:StackStatus,Created:CreationTime}' \
		--output table

# Set default target
.DEFAULT_GOAL := help