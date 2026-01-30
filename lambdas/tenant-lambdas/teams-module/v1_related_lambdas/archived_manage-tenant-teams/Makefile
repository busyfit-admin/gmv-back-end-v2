test: 
	go mod tidy
	go vet
	env=0.6 go test -cover	

local:
	GOARCH=amd64 GOOS=linux go build -tags lambda.norpc -o bootstrap

build:
	# disabled for now

update:
	go get -u
	go mod tidy

.PHONY: test build update