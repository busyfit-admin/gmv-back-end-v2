.PHONY: build clean tidy

build:
	$(MAKE) -C create-organization build
	$(MAKE) -C manage-organization build
	$(MAKE) -C manage-subscription build
	$(MAKE) -C manage-promo-codes build
	$(MAKE) -C list-user-organizations build

clean:
	$(MAKE) -C create-organization clean
	$(MAKE) -C manage-organization clean
	$(MAKE) -C manage-subscription clean
	$(MAKE) -C manage-promo-codes clean
	$(MAKE) -C list-user-organizations clean

tidy:
	$(MAKE) -C create-organization tidy
	$(MAKE) -C manage-organization tidy
	$(MAKE) -C manage-subscription tidy
	$(MAKE) -C manage-promo-codes tidy
	$(MAKE) -C list-user-organizations tidy
		


