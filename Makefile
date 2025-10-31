SUBDIRS := $(wildcard */.)
REVISION := $(shell git rev-parse --short HEAD)
ARTIFACT := artifact-$(REVISION).tar.bz2


test: $(SUBDIRS)

build: $(SUBDIRS)

update: $(SUBDIRS)

deploy: $(SUBDIRS)

local: $(SUBDIRS)

test-deployed-stack:
	make -C tests test-deployed-stack-admin
	make -C tests test-deployed-stack-tenant

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)

.PHONY: test build local deploy $(SUBDIRS)