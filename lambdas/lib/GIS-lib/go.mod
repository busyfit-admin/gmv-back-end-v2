module github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/GIS-lib

go 1.21.7

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients => ../clients

replace github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils => ../utils

require (
	github.com/aws/aws-sdk-go v1.55.1
	github.com/crolly/dyngeo v0.0.0-20190527163316-50ec62c8839f
	github.com/gofrs/uuid v4.4.0+incompatible
)

require (
	github.com/golang/geo v0.0.0-20230421003525-6adc56603217 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
)
