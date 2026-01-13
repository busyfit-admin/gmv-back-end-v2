AWS_REGION=ap-southeast-2
ENVIRONMENT?=dev
TEMPLATE := template.yaml
STACK_NAME := tenant-portal-apis-$(ENVIRONMENT)
CLOUDFRONT_KEY_STACK := cloudfront-public-key-stack-$(ENVIRONMENT)

#Data Stack
DATA_STACK=A

# CloudFront Keys - will be fetched from cloudfront-key stack if it exists
CLOUDFRONT_PUBLIC_KEY_ID ?= $(shell aws cloudformation describe-stacks --stack-name $(CLOUDFRONT_KEY_STACK) --region ap-southeast-2 --query 'Stacks[0].Outputs[?OutputKey==`PublicKeyId`].OutputValue' --output text 2>/dev/null || echo "")
CLOUDFRONT_PRIVATE_KEY_SECRET_ARN ?= $(shell aws cloudformation describe-stacks --stack-name $(CLOUDFRONT_KEY_STACK) --region ap-southeast-2 --query 'Stacks[0].Outputs[?OutputKey==`PrivateKeySecretArn`].OutputValue' --output text 2>/dev/null || echo "")


#UserPool Setting

ifeq ($(ENVIRONMENT),dev)
	COGNITO_USERPOOL := arn:aws:cognito-idp:ap-south-1:231252353945:userpool/ap-south-1_dev
else
# checkout for valid COGNITO_USERPOOL
	COGNITO_USERPOOL := arn:aws:cognito-idp:ap-south-1:231252353945:userpool/ap-south-1_i4x6Lpum5
endif


# stack tag values
DEPARTMENT := "CLOUD DEVOPS"
PROJECT := "AdminV1"
PROVISIONER := CloudFormation

test:
	cfn-lint $(TEMPLATE) --ignore-checks E1019 E1011

local:

build:

deploy:

	$(eval GITSHA := $(shell git rev-parse --short=7 HEAD ) )
	$(eval AWS_ACCOUNT := $(shell aws sts get-caller-identity --query Account --output text ) )
	$(eval CFN_BUCKET := artifactory-$(AWS_REGION)-$(AWS_ACCOUNT))
	$(eval CLOUDFRONT_PARAMS := $(if $(CLOUDFRONT_PUBLIC_KEY_ID),CloudFrontPublicKeyIdParam=$(CLOUDFRONT_PUBLIC_KEY_ID),))
	$(eval CLOUDFRONT_SECRET_PARAMS := $(if $(CLOUDFRONT_PRIVATE_KEY_SECRET_ARN),CloudFrontPrivateKeySecretArn=$(CLOUDFRONT_PRIVATE_KEY_SECRET_ARN),))
	@echo "Deploying to environment $(ENVIRONMENT) from bucket $(CFN_BUCKET)"
	@echo "CloudFront Public Key ID: $(CLOUDFRONT_PUBLIC_KEY_ID)"
	@echo "CloudFront Private Key Secret ARN: $(CLOUDFRONT_PRIVATE_KEY_SECRET_ARN)"
	aws cloudformation package \
		--template-file $(TEMPLATE) \
		--s3-bucket $(CFN_BUCKET) \
		--s3-prefix $(STACK_NAME) \
		--output-template-file rendered.yml
	aws cloudformation deploy \
		--template-file rendered.yml \
		--stack-name $(STACK_NAME) \
		--s3-bucket $(CFN_BUCKET) \
		--capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND CAPABILITY_NAMED_IAM \
		--parameter-overrides Environment=$(ENVIRONMENT) \
			$$( cat params-$(ENVIRONMENT).txt 2>/dev/null ) \
			GitSha=${GITSHA}\
			DataStack=${DATA_STACK}\
			$(CLOUDFRONT_PARAMS) \
			$(CLOUDFRONT_SECRET_PARAMS) \
		--no-fail-on-empty-changeset \
		--tags app:governance:department=$(DEPARTMENT) \
			app:governance:project=$(PROJECT) \
			app:governance:environment=$(ENVIRONMENT) \
			app:governance:provisioner=$(PROVISIONER)

update:

undeploy:
	echo Undeploying from $(ENVIRONMENT)
	aws cloudformation delete-stack \
		--stack-name $(STACK_NAME)
	aws cloudformation wait stack-delete-complete \
		--stack-name $(STACK_NAME)

.PHONY: test build update deploy local