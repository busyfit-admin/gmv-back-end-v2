module github.com/busyfit-admin/saas-integreted-apis/lambdas/tenant-lambdas/rewards-module/start-reward-rule-transaction

go 1.21.4

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients => ../../../lib/clients

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib => ../../../lib/company-lib

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils => ../../../lib/utils

require (
	github.com/aws/aws-lambda-go v1.41.0
	github.com/aws/aws-sdk-go-v2/config v1.19.1
	github.com/aws/aws-xray-sdk-go v1.8.2
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/aws/aws-sdk-go v1.46.7 // indirect
	github.com/aws/aws-sdk-go-v2 v1.26.1 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.43 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.13 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.45 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.15.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.23.2 // indirect
	github.com/aws/smithy-go v1.20.2 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.15.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.34.0 // indirect
	golang.org/x/net v0.7.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	golang.org/x/text v0.7.0 // indirect
	google.golang.org/genproto v0.0.0-20210114201628-6edceaf6022f // indirect
	google.golang.org/grpc v1.35.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)
