module github.com/busyfit-admin/saas-integreted-apis/lambdas/tenant-lambdas/rewards-module/reward-transfer-logs

go 1.22

toolchain go1.23.9

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients => ../../../lib/clients

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib => ../../../lib/company-lib

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils => ../../../lib/utils

require (
	github.com/aws/aws-lambda-go v1.49.0
	github.com/aws/aws-sdk-go-v2/config v1.29.14
	github.com/aws/aws-xray-sdk-go v1.8.5
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib v0.0.0-00010101000000-000000000000
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/aws/aws-sdk-go v1.47.9 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.12.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.32.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.26.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.18.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.33.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.8.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.16.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.48.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.28.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sfn v1.26.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/verifiedpermissions v1.11.3 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients v0.0.0-00010101000000-000000000000 // indirect
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils v0.0.0-00010101000000-000000000000 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.17.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.52.0 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sys v0.21.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/grpc v1.64.1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)
