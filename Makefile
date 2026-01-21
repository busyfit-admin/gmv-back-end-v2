SUBDIRS := $(wildcard */.)
REVISION := $(shell git rev-parse --short HEAD)
ARTIFACT := artifact-$(REVISION).tar.bz2


test: $(SUBDIRS)

build: $(SUBDIRS)

build-teams-module: 
	$(MAKE) -C lambdas/tenant-lambdas/teams-module build

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