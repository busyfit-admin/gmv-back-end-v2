.PHONY: build clean

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap manage-team-operations.go

clean:
	rm -f bootstrap
