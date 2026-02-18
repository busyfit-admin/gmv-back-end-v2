.PHONY: build clean tidy

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap list-user-organizations.go

clean:
	rm -f bootstrap

tidy:
	go mod tidy		
