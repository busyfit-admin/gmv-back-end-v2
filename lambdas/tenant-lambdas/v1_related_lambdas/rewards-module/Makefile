SUBDIRS := $(wildcard */.)

test: $(SUBDIRS)

build: 
	# ingore build at this level

update: $(SUBDIRS)

deploy: $(SUBDIRS)

local: $(SUBDIRS)

$(SUBDIRS):
	$(MAKE) -C $@ $(MAKECMDGOALS)

.PHONY: test build local deploy $(SUBDIRS)