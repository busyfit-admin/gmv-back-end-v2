module github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/admin-lib/tenant

go 1.21.4

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients => ../../clients

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils => ../../utils

require (
	github.com/aws/aws-sdk-go v1.49.16
	github.com/aws/aws-sdk-go-v2 v1.30.4
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.12.14
	github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider v1.32.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.26.8
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients v0.0.0-00010101000000-000000000000
	github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.9.0
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.5.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.16 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodbstreams v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.33.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.8.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.16.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.48.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.28.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sfn v1.26.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/verifiedpermissions v1.11.3 // indirect
	github.com/aws/smithy-go v1.20.4 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
