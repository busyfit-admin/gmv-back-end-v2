module github.com/busyfit-admin/saas-integrated-apis/lambdas/tenant-lambdas/employees-module/update-employee-groups

go 1.21.4

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients => ../../../lib/clients

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib => ../../../lib/company-lib

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils => ../../../lib/utils

require (
	github.com/aws/aws-lambda-go v1.42.0
	github.com/aws/aws-sdk-go-v2/config v1.26.1
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.26.7
	github.com/aws/aws-xray-sdk-go v1.8.3
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib v0.0.0-00010101000000-000000000000
)

require (
	github.com/andybalholm/brotli v1.0.6 // indirect
	github.com/aws/aws-sdk-go v1.47.9 // indirect
	github.com/aws/aws-sdk-go-v2 v1.30.4 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.12 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.12.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.32.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.18.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.33.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.8.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.16.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.48.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.28.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sfn v1.26.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/verifiedpermissions v1.11.3 // indirect
	github.com/aws/smithy-go v1.20.4 // indirect
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients v0.0.0-00010101000000-000000000000 // indirect
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils v0.0.0-00010101000000-000000000000 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.17.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.8.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.50.0 // indirect
	golang.org/x/net v0.18.0 // indirect
	golang.org/x/sys v0.14.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20231106174013-bbf56f31fb17 // indirect
	google.golang.org/grpc v1.59.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
