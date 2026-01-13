.PHONY: build clean

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap manage-employee-profile-v2.go
	zip -j bootstrap.zip bootstrap
	rm bootstrap

clean:
	rm -f bootstrap bootstrap.zip
