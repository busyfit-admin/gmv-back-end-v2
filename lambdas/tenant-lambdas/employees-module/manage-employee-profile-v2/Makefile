.PHONY: build clean

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap manage-employee-profile-v2.go

clean:
	rm -f bootstrap bootstrap.zip
