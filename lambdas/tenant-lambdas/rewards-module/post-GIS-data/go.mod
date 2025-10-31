module github.com/busyfit-admin/saas-integrated-apis/lambdas/tenant-lambdas/rewards-module/post-GIS-data

go 1.21.7

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients => ../../../lib/clients

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/GIS-lib => ../../../lib/GIS-lib

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils => ../../../lib/utils

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go v1.55.1
	github.com/aws/aws-sdk-go-v2/config v1.27.27
	github.com/aws/aws-xray-sdk-go v1.8.4
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/GIS-lib v0.0.0-00010101000000-000000000000
)

require (
	github.com/andybalholm/brotli v1.0.6 // indirect
	github.com/aws/aws-sdk-go-v2 v1.30.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.27 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.15 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.3 // indirect
	github.com/aws/smithy-go v1.20.3 // indirect
	github.com/crolly/dyngeo v0.0.0-20190527163316-50ec62c8839f // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/golang/geo v0.0.0-20230421003525-6adc56603217 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.50.0 // indirect
	golang.org/x/net v0.23.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231106174013-bbf56f31fb17 // indirect
	google.golang.org/grpc v1.59.0 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
