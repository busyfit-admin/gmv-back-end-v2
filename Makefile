SUBDIRS := $(wildcard */.)
REVISION := $(shell git rev-parse --short HEAD)
ARTIFACT := artifact-$(REVISION).tar.bz2


test: $(SUBDIRS)

build: $(SUBDIRS)

build-tenant-lambdas:
	$(MAKE) -C lambdas/tenant-lambdas

tidy-v2:
	$(MAKE) -C lambdas/tenant-lambdas/org-module tidy

build-v2: 
	$(MAKE) -C lambdas/tenant-lambdas/teams-module build
	$(MAKE) -C lambdas/tenant-lambdas/org-module build

update: $(SUBDIRS)

deploy: $(SUBDIRS)

deploy-tenant-cfn: 
	$(MAKE) -C cfn/tenant-cfn deploy

local: $(SUBDIRS)

test-deployed-stack:
	make -C tests test-deployed-stack-admin
	make -C tests test-deployed-stack-tenant

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)

.PHONY: test build local deploy $(SUBDIRS)