module github.com/busyfit-admin/saas-integrated-apis/lambdas/tenant-lambdas/teams-module/list-user-teams

go 1.23

require (
	github.com/aws/aws-lambda-go v1.41.0
	github.com/aws/aws-sdk-go-v2/config v1.18.42
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.26.7
	github.com/aws/aws-xray-sdk-go v1.8.2
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib v0.0.0
)

require (
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/aws/aws-sdk-go v1.46.7 // indirect
	github.com/aws/aws-sdk-go-v2 v1.41.1 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.40 // indirect
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.12.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.43 // indirect
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
	github.com/aws/aws-sdk-go-v2/service/ses v1.34.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/sfn v1.26.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.14.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.17.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.22.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/verifiedpermissions v1.11.3 // indirect
	github.com/aws/smithy-go v1.24.0 // indirect
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients v0.0.0-00010101000000-000000000000 // indirect
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils v0.0.0-00010101000000-000000000000 // indirect
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
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

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib => ../../../lib/company-lib

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients => ../../../lib/clients

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils => ../../../lib/utils
