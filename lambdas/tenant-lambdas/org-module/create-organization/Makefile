.PHONY: build clean tidy

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap create-organization.go

clean:
	rm -f bootstrap

tidy:
	go mod tidy	
