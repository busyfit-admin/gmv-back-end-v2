package lib

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"

	"time"

	// "github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	// "github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go-v2/service/verifiedpermissions"
)

type VerifiedPermissionsClient interface {
	IsAuthorized(ctx context.Context, params *verifiedpermissions.IsAuthorizedInput, optFns ...func(*verifiedpermissions.Options)) (*verifiedpermissions.IsAuthorizedOutput, error)
}

type MockVerifiedPermissionsClient struct {
	IsAuthInputs []verifiedpermissions.IsAuthorizedInput
	IsAuthOutput []verifiedpermissions.IsAuthorizedOutput
	IsAuthErrors []error
}

func (client *MockVerifiedPermissionsClient) IsAuthorized(ctx context.Context, params *verifiedpermissions.IsAuthorizedInput, optFns ...func(*verifiedpermissions.Options)) (*verifiedpermissions.IsAuthorizedOutput, error) {
	client.IsAuthInputs = append(client.IsAuthInputs, *params)
	index := len(client.IsAuthInputs) - 1

	return &client.IsAuthOutput[index], client.IsAuthErrors[index]
}

type DynamodbClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error)
	ExecuteStatement(ctx context.Context, params *dynamodb.ExecuteStatementInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ExecuteStatementOutput, error)
	TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error)
}

type MockDynamodbClient struct {
	QueryInputs  []dynamodb.QueryInput
	QueryOutputs []dynamodb.QueryOutput
	QueryErrors  []error

	GetItemInputs  []dynamodb.GetItemInput
	GetItemOutputs []dynamodb.GetItemOutput
	GetItemErrors  []error

	UpdateItemInputs  []dynamodb.UpdateItemInput
	UpdateItemOutputs []dynamodb.UpdateItemOutput
	UpdateItemErrors  []error

	DeleteItemInputs  []dynamodb.DeleteItemInput
	DeleteItemOutputs []dynamodb.DeleteItemOutput
	DeleteItemErrors  []error

	PutItemInputs  []dynamodb.PutItemInput
	PutItemOutputs []dynamodb.PutItemOutput
	PutItemErrors  []error

	ScanInputs  []dynamodb.ScanInput
	ScanOutputs []dynamodb.ScanOutput
	ScanErrors  []error

	BatchWriteItemsInputs []dynamodb.BatchWriteItemInput
	BatchWriteItemOutputs []dynamodb.BatchWriteItemOutput
	BatchErrors           []error

	ExecuteStatementInputs  []dynamodb.ExecuteStatementInput
	ExecuteStatementOutputs []dynamodb.ExecuteStatementOutput
	ExecuteStatementErrors  []error

	TransactWriteItemsInputs []dynamodb.TransactWriteItemsInput
	TransactWriteItemsOutput []dynamodb.TransactWriteItemsOutput
	TransactWriteItemsErrors []error
}

func (client *MockDynamodbClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	client.QueryInputs = append(client.QueryInputs, *params)
	index := len(client.QueryInputs) - 1

	return &client.QueryOutputs[index], client.QueryErrors[index]
}

func (client *MockDynamodbClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	client.GetItemInputs = append(client.GetItemInputs, *params)
	index := len(client.GetItemInputs) - 1

	return &client.GetItemOutputs[index], client.GetItemErrors[index]
}
func (client *MockDynamodbClient) Scan(ctx context.Context, params *dynamodb.ScanInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	client.ScanInputs = append(client.ScanInputs, *params)

	index := len(client.ScanInputs) - 1

	return &client.ScanOutputs[index], client.ScanErrors[index]
}

func (client *MockDynamodbClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	client.UpdateItemInputs = append(client.UpdateItemInputs, *params)

	index := len(client.UpdateItemInputs) - 1

	return &client.UpdateItemOutputs[index], client.UpdateItemErrors[index]
}

func (client *MockDynamodbClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	client.DeleteItemInputs = append(client.DeleteItemInputs, *params)
	index := len(client.DeleteItemInputs) - 1

	return &client.DeleteItemOutputs[index], client.DeleteItemErrors[index]
}

func (client *MockDynamodbClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	client.PutItemInputs = append(client.PutItemInputs, *params)
	index := len(client.PutItemInputs) - 1

	return &client.PutItemOutputs[index], client.PutItemErrors[index]
}

func (client *MockDynamodbClient) BatchWriteItem(ctx context.Context, params *dynamodb.BatchWriteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.BatchWriteItemOutput, error) {
	client.BatchWriteItemsInputs = append(client.BatchWriteItemsInputs, *params)
	index := len(client.BatchWriteItemsInputs) - 1

	return &client.BatchWriteItemOutputs[index], client.BatchErrors[index]
}

func (client *MockDynamodbClient) ExecuteStatement(ctx context.Context, params *dynamodb.ExecuteStatementInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ExecuteStatementOutput, error) {
	client.ExecuteStatementInputs = append(client.ExecuteStatementInputs, *params)
	index := len(client.ExecuteStatementInputs) - 1

	return &client.ExecuteStatementOutputs[index], client.ExecuteStatementErrors[index]
}

func (client *MockDynamodbClient) TransactWriteItems(ctx context.Context, params *dynamodb.TransactWriteItemsInput, optFns ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {

	client.TransactWriteItemsInputs = append(client.TransactWriteItemsInputs, *params)
	index := len(client.TransactWriteItemsInputs) - 1

	return &client.TransactWriteItemsOutput[index], client.TransactWriteItemsErrors[index]
}

// Step Function Client

type StepFunctionClient interface {
	StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error)
}

type MockStepFunctionClient struct {
	StartExecutionInput  []sfn.StartExecutionInput
	StartExecutionOutput []sfn.StartExecutionOutput
	Errors               []error
}

func (client *MockStepFunctionClient) StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error) {
	client.StartExecutionInput = append(client.StartExecutionInput, *params)
	index := len(client.StartExecutionInput) - 1

	return &client.StartExecutionOutput[index], client.Errors[index]
}

// Cognito Clients

type CognitoClient interface {
	AdminGetUser(ctx context.Context, params *cognitoidentityprovider.AdminGetUserInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminGetUserOutput, error)
	AdminCreateUser(ctx context.Context, params *cognitoidentityprovider.AdminCreateUserInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminCreateUserOutput, error)
	AdminDeleteUser(ctx context.Context, params *cognitoidentityprovider.AdminDeleteUserInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminDeleteUserOutput, error)
}

type MockCognitoClient struct {
	// Get user
	AdminGetUserInput  []cognitoidentityprovider.AdminGetUserInput
	AdminGetUserOutput []cognitoidentityprovider.AdminGetUserOutput
	AdminGetUserError  []error

	// Delete user
	AdminDeleteUserInput  []cognitoidentityprovider.AdminDeleteUserInput
	AdminDeleteUserOutput []cognitoidentityprovider.AdminDeleteUserOutput
	AdminDeleteUserError  []error

	// Create user
	AdminCreateUserInput  []cognitoidentityprovider.AdminCreateUserInput
	AdminCreateUserOutput []cognitoidentityprovider.AdminCreateUserOutput
	AdminCreateUserError  []error
}

func (client *MockCognitoClient) AdminGetUser(ctx context.Context, params *cognitoidentityprovider.AdminGetUserInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminGetUserOutput, error) {

	client.AdminGetUserInput = append(client.AdminGetUserInput, *params)
	index := len(client.AdminGetUserInput) - 1

	return &client.AdminGetUserOutput[index], client.AdminGetUserError[index]
}

func (client *MockCognitoClient) AdminCreateUser(ctx context.Context, params *cognitoidentityprovider.AdminCreateUserInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminCreateUserOutput, error) {

	client.AdminCreateUserInput = append(client.AdminCreateUserInput, *params)
	index := len(client.AdminCreateUserInput) - 1

	return &client.AdminCreateUserOutput[index], client.AdminCreateUserError[index]
}

func (client *MockCognitoClient) AdminDeleteUser(ctx context.Context, params *cognitoidentityprovider.AdminDeleteUserInput, optFns ...func(*cognitoidentityprovider.Options)) (*cognitoidentityprovider.AdminDeleteUserOutput, error) {
	client.AdminDeleteUserInput = append(client.AdminDeleteUserInput, *params)
	index := len(client.AdminDeleteUserInput) - 1

	return &client.AdminDeleteUserOutput[index], client.AdminDeleteUserError[index]
}

// S3 clients

// S3Client interface
type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
}

type PresignClient interface {
	PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// MockS3Client struct
type MockS3Client struct {
	GetObjectInputs  []s3.GetObjectInput
	GetObjectOutputs []s3.GetObjectOutput
	GetObjectErrors  []error

	PutObjectInputs  []s3.PutObjectInput
	PutObjectOutputs []s3.PutObjectOutput
	PutObjectErrors  []error

	DeleteObjectInputs  []s3.DeleteObjectInput
	DeleteObjectOutputs []s3.DeleteObjectOutput
	DeleteObjectErrors  []error

	CopyObjectInputs  []s3.CopyObjectInput
	CopyObjectOutputs []s3.CopyObjectOutput
	CopyObjectErrors  []error

	PresignGetObjectInputs  []s3.GetObjectInput
	PresignGetObjectOutputs []v4.PresignedHTTPRequest
	PresignGetObjectErrors  []error
}

// PresignGetObject method for MockS3Client
func (client *MockS3Client) PresignGetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	client.PresignGetObjectInputs = append(client.PresignGetObjectInputs, *params)
	index := len(client.PresignGetObjectInputs) - 1
	return &client.PresignGetObjectOutputs[index], client.PresignGetObjectErrors[index]
}

// GetObject method for MockS3Client
func (client *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	client.GetObjectInputs = append(client.GetObjectInputs, *params)
	index := len(client.GetObjectInputs) - 1
	return &client.GetObjectOutputs[index], client.GetObjectErrors[index]
}

// PutObject method for MockS3Client
func (client *MockS3Client) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	client.PutObjectInputs = append(client.PutObjectInputs, *params)
	index := len(client.PutObjectInputs) - 1
	return &client.PutObjectOutputs[index], client.PutObjectErrors[index]
}

// DeleteObject method for MockS3Client
func (client *MockS3Client) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	client.DeleteObjectInputs = append(client.DeleteObjectInputs, *params)
	index := len(client.DeleteObjectInputs) - 1
	return &client.DeleteObjectOutputs[index], client.DeleteObjectErrors[index]
}

// CopyObject method for MockS3Client
func (client *MockS3Client) CopyObject(ctx context.Context, params *s3.CopyObjectInput, optFns ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	client.CopyObjectInputs = append(client.CopyObjectInputs, *params)
	index := len(client.CopyObjectInputs) - 1
	return &client.CopyObjectOutputs[index], client.CopyObjectErrors[index]
}

// Cognito Pre signer Function Clients
type CognitoSigner interface {
	Sign(url string, expires time.Time) (string, error)
}

type MockCognitoSigner struct {
	urls    []string
	expires []time.Time

	signedUrl []string
	error     []error
}

func (client *MockCognitoSigner) Sign(url string, expires time.Time) (string, error) {

	client.urls = append(client.urls, url)
	client.expires = append(client.expires, expires)

	index := len(client.urls) - 1

	return client.signedUrl[index], client.error[index]
}

// Secrets Manager Clients

type SecretManagerClient interface {
	GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type MockSecretManager struct {
	GetSecretValueInput  secretsmanager.GetSecretValueInput
	GetSecretValueOutput secretsmanager.GetSecretValueOutput
	GetSecretValueError  error
}

func (client *MockSecretManager) GetSecretValue(ctx context.Context, params *secretsmanager.GetSecretValueInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	client.GetSecretValueInput = *params

	return &client.GetSecretValueOutput, client.GetSecretValueError
}

// EVB Clients

type EventBridgeClient interface {
	PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type MockEventBridgeClient struct {
	PutEventsInput  []eventbridge.PutEventsInput
	PutEventsOutput []eventbridge.PutEventsOutput
	PutEventsError  []error
}

func (client *MockEventBridgeClient) PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error) {
	client.PutEventsInput = append(client.PutEventsInput, *params)

	index := len(client.PutEventsInput) - 1

	return &client.PutEventsOutput[index], client.PutEventsError[index]
}

type CloudfrontClient interface {
	Sign(url string, expires time.Time) (string, error)
}
