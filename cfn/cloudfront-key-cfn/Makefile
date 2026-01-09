.PHONY: deploy delete validate build-layer

STACK_NAME ?= cloudfront-public-key-stack
ENVIRONMENT ?= dev
AWS_REGION ?= ap-south-1

build-layer:
	@echo "Building Lambda layer with cryptography..."
	./build-layer.sh $(ENVIRONMENT) $(AWS_REGION)

deploy: build-layer
	@echo "Deploying CloudFront Public Key Stack..."
	aws cloudformation deploy \
		--template-file template.yaml \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--parameter-overrides Environment=$(ENVIRONMENT) \
		--capabilities CAPABILITY_NAMED_IAM \
		--region $(AWS_REGION)
	@echo "Deployment complete!"
	@echo "Fetching outputs..."
	@aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--region $(AWS_REGION) \
		--query 'Stacks[0].Outputs' \
		--output table

delete:
	@echo "Deleting CloudFront Public Key Stack..."
	aws cloudformation delete-stack \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--region $(AWS_REGION)
	@echo "Stack deletion initiated. Waiting for completion..."
	aws cloudformation wait stack-delete-complete \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--region $(AWS_REGION)
	@echo "Stack deleted successfully!"

validate:
	@echo "Validating CloudFormation template..."
	aws cloudformation validate-template \
		--template-body file://template.yaml \
		--region $(AWS_REGION)
	@echo "Template is valid!"

outputs:
	@echo "Stack Outputs:"
	@aws cloudformation describe-stacks \
		--stack-name $(STACK_NAME)-$(ENVIRONMENT) \
		--region $(AWS_REGION) \
		--query 'Stacks[0].Outputs' \
		--output table

help:
	@echo "Available commands:"
	@echo "  make deploy ENVIRONMENT=dev    - Deploy the CloudFront key stack"
	@echo "  make delete ENVIRONMENT=dev    - Delete the CloudFront key stack"
	@echo "  make validate                  - Validate the CloudFormation template"
	@echo "  make outputs ENVIRONMENT=dev   - Display stack outputs"
	@echo ""
	@echo "Environment options: dev, test, prod"
