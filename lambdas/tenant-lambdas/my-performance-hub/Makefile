SUBDIRS := $(filter-out common/., $(wildcard */.))

test: $(SUBDIRS)

build: $(SUBDIRS)

update: $(SUBDIRS)

deploy: $(SUBDIRS)

local: $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)

.PHONY: test build local deploy $(SUBDIRS)
