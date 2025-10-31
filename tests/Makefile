ENVIRONMENT ?= test
TEMPLATE := template.yml
ADMIN_STACK_NAME := admin-portal-apis-$(ENVIRONMENT)
TENANT_STACK_NAME := tenant-portal-apis-$(ENVIRONMENT)
USER_STACK_NAME := user-portal-apis-$(ENVIRONMENT)

#Data Stack
DATA_STACK_NAME := saas-data-stack-zone-a-test

build:

update:

test:
	@echo "only run with a deployed stack"

# Ensure the AWS_REGION in aws configure is set to the required region. use "aws configure" command on cmd
test-deployed-stack-admin:
	@echo "Running Behave Tests! for $(ADMIN_STACK_NAME) with Data Stack $(DATA_STACK_NAME)"

	behave -D stack_name=$(ADMIN_STACK_NAME) -D data_stack_name=$(DATA_STACK_NAME) --tags=end2endAdmin


test-deployed-stack-tenant:
#Currently Data stack for Tenant is Tenant stack itself, for future improvements, we'll need to move to seperate Tenant Data stack
	@echo "Running Behave Tests! for $(TENANT_STACK_NAME) with Data Stack $(TENANT_STACK_NAME)"

	behave -D stack_name=$(TENANT_STACK_NAME) -D data_stack_name=$(TENANT_STACK_NAME) --tags=end2endTenant


test-deployed-stack-admin-subdomains:
	@echo "Running Behave Tests! for $(ADMIN_STACK_NAME) with Data Stack $(DATA_STACK_NAME)"

	behave -D stack_name=$(ADMIN_STACK_NAME) -D data_stack_name=$(DATA_STACK_NAME) --tags=end2endTenantSubDomain
