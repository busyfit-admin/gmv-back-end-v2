package Companylib

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

// DDB Table to store Certificates Types
type TenantCertificates struct {
	CertificateId   string `json:"CertificateId" dynamodbav:"CertificateId"` // PK
	CertificateName string `json:"CertificateName" dynamodbav:"CertificateName"`
	CertificateDesc string `json:"CertificateDesc" dynamodbav:"CertificateDesc"`
	Criteria        string `json:"Criteria" dynamodbav:"Criteria"`   // e.g., "Anniversary", "Skills"
	Threshold       int    `json:"Threshold" dynamodbav:"Threshold"` // The threshold to meet for certificate assignment
	IsActive        string `json:"IsActive" dynamodbav:"IsActive"`   // "Active", "Draft", "InActive"

	CertificateImage string `json:"CertificateImage" dynamodbav:"CertificateImage"`

	LastModifiedDate string `json:"LastModifiedDate" dynamodbav:"LastModifiedDate"`
	CertificateMode  string `json:"CertificateMode" dynamodbav:"CertificateMode"` // "Automatic", "Manual"

	// Top level Filter - TBD - enabled for later
	// Level_TeamType_TeamId string `json:"TeamType_TeamId" dynamodbav:"TeamType_TeamId"` // Index_Filter_Certificates: Range Key
	/*
		Ex:
		Employee Level certificates :
		  "Level_TeamType_TeamId" : "Employee"

		Team level certificates:  // TeamType#TeamID
			"Level_TeamType_TeamId" : "TEAM-Sales#TeamId" ( all criteria that is applicable at the teamId Level)

			At Team Type Level:
			"Level_TeamType_TeamId" : "TEAM-Ops" ( all criteria that is applicable at the TeamType Ops Level)

			At Team Level:
			"Level_TeamType_TeamId" : "Team" ( all criteria that is applicable at all team types)
	*/
}

type TenantCertificatesService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	TenantCertificatesTable       string
	CertificatesTransferLogsTable string
	CertificateLogsIndex string
}

func CreateTenantCertificatesService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *TenantCertificatesService {
	return &TenantCertificatesService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

type AllCertificates struct {
	Active []TenantCertificates `json:"Active"`
	Draft  []TenantCertificates `json:"Draft"`
}

func (svc *TenantCertificatesService) GetAllTenantCertificates() (AllCertificates, error) {

	var allTenantCertificatesData AllCertificates

	// 1. Get Active Tenant Certificates
	queryGetActiveTenantCertificates := "SELECT CertificateId, CertificateName, CertificateDesc, CertificateImage, Criteria, Threshold, IsActive, LastModifiedDate, Mode FROM \"" + svc.TenantCertificatesTable + "\" WHERE IsActive = 'Active'"

	activeCertificatesData, err := svc.GetCertificatesData(queryGetActiveTenantCertificates)
	if err != nil {
		return AllCertificates{}, err
	}

	allTenantCertificatesData.Active = activeCertificatesData

	// 2. Get InActive Tenant Certificates
	queryGetInactiveTenantTeams := "SELECT CertificateId, CertificateName, CertificateDesc, CertificateImage, Criteria, Threshold, IsActive, LastModifiedDate, Mode FROM \"" + svc.TenantCertificatesTable + "\" WHERE IsActive = 'Inactive'"

	inactiveCertificateData, err := svc.GetCertificatesData(queryGetInactiveTenantTeams)
	if err != nil {
		return AllCertificates{}, err
	}
	allTenantCertificatesData.Draft = inactiveCertificateData

	return allTenantCertificatesData, nil
}

func (svc *TenantCertificatesService) GetCertificatesData(stmt string) ([]TenantCertificates, error) {

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(stmt),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []TenantCertificates{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for Team Data Query %s", stmt)
		return []TenantCertificates{}, nil
	}

	allCertificatesData := []TenantCertificates{}
	for _, stageItem := range output.Items {
		certificateData := TenantCertificates{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &certificateData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal Team data. Failed with  Error : %v", err)
			return []TenantCertificates{}, err
		}

		// Append data to the overall rule data
		allCertificatesData = append(allCertificatesData, certificateData)
	}

	return allCertificatesData, nil
}

func (svc *TenantCertificatesService) GetCertificate(certificateId string) (TenantCertificates, error) {
	output, err := svc.dynamodbClient.GetItem(svc.ctx, &dynamodb.GetItemInput{
		TableName: aws.String(svc.TenantCertificatesTable),
		Key: map[string]types.AttributeValue{
			"CertificateId": &types.AttributeValueMemberS{Value: certificateId},
		},
	})
	if err != nil {
		svc.logger.Printf("Failed to get the certificate with certificateID: %v, error :%v", certificateId, err)
		return TenantCertificates{}, err
	}

	certificateData := TenantCertificates{}
	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &certificateData)
	if err != nil {
		svc.logger.Printf("Failed to Unmarshal the ddb output: %v, error :%v", certificateId, err)
		return TenantCertificates{}, err
	}

	return certificateData, nil
}

// Create and Update Certificates
func (svc *TenantCertificatesService) UpdateCertificateData(certificateData TenantCertificates) error {

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantCertificatesTable),
		Key: map[string]types.AttributeValue{
			"CertificateId": &types.AttributeValueMemberS{Value: certificateData.CertificateId},
		},
		UpdateExpression: aws.String("SET CertificateName = :CertificateName, CertificateDesc = :CertificateDesc, Criteria = :Criteria, Threshold = :Threshold, IsActive = :IsActive, LastModifiedDate = :LastModifiedDate, CertificateMode = :CertificateMode"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":CertificateName":  &types.AttributeValueMemberS{Value: certificateData.CertificateName},
			":CertificateDesc":  &types.AttributeValueMemberS{Value: certificateData.CertificateDesc},
			":Criteria":         &types.AttributeValueMemberS{Value: certificateData.Criteria},
			":Threshold":        &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", certificateData.Threshold)},
			":IsActive":         &types.AttributeValueMemberS{Value: certificateData.IsActive},
			":LastModifiedDate": &types.AttributeValueMemberS{Value: certificateData.LastModifiedDate},
			":CertificateMode":  &types.AttributeValueMemberS{Value: certificateData.CertificateMode},
		},
		ReturnValues: types.ReturnValueAllNew,
	})
	if err != nil {
		return fmt.Errorf("failed to put item in ddb table: %v", err)
	}

	return nil
}

func (svc *TenantCertificatesService) CreateCertificateData(certificateData TenantCertificates) error {

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.TenantCertificatesTable),
		Item: map[string]types.AttributeValue{
			"CertificateId":    &types.AttributeValueMemberS{Value: certificateData.CertificateId},
			"CertificateName":  &types.AttributeValueMemberS{Value: certificateData.CertificateName},
			"CertificateDesc":  &types.AttributeValueMemberS{Value: certificateData.CertificateDesc},
			"CertificateImage": &types.AttributeValueMemberS{Value: certificateData.CertificateImage},
			"Criteria":         &types.AttributeValueMemberS{Value: certificateData.Criteria},
			"Threshold":        &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", certificateData.Threshold)},
			"IsActive":         &types.AttributeValueMemberS{Value: certificateData.IsActive},
			"LastModifiedDate": &types.AttributeValueMemberS{Value: certificateData.LastModifiedDate},
			"CertificateMode":  &types.AttributeValueMemberS{Value: certificateData.CertificateMode},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to put item in ddb table: %v", err)
	}

	return nil
}

// Delete Certificates

func (svc *TenantCertificatesService) DeleteCertificateData(certificateId string) error {
	_, err := svc.dynamodbClient.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{

		TableName: aws.String(svc.TenantCertificatesTable),
		Key: map[string]types.AttributeValue{
			"CertificateId": &types.AttributeValueMemberS{Value: certificateId},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to delete item in ddb table: %v", err)
	}
	return nil
}

type CertificateLogsData struct {
	CertificateTransferId string
	SourceUserName        string
	DestinationUserName   string

	CertificateId   string
	CertificateName string
	Criteria        string
	Message         string

	CertificateTransferStatus  string
	CertificateTransferLogTime string
	CertificateTransferError   string
}

func (svc *TenantCertificatesService) GetCertificateLogs(SourceUserName string) ([]CertificateLogsData, error) {

	stmt := "SELECT CertificateTransferId, SourceUserName, DestinationUserName, CertificateId, CertificateName, Criteria, Message, CertificateTransferStatus, CertificateTransferLogTime, CertificateTransferError FROM \"" + svc.CertificatesTransferLogsTable + "\".\"" + svc.CertificateLogsIndex + "\" WHERE SourceUserName = ? ORDER BY CertificateTransferLogTime DESC"

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(stmt),
		Parameters:     []dynamodb_types.AttributeValue{&dynamodb_types.AttributeValueMemberS{Value: SourceUserName}},
		Limit:          aws.Int32(30),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []CertificateLogsData{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for Team Data Query %s", stmt)
		return []CertificateLogsData{}, nil
	}

	allCertificatesData := []CertificateLogsData{}
	for _, stageItem := range output.Items {
		certificateData := CertificateLogsData{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &certificateData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal Team data. Failed with  Error : %v", err)
			return []CertificateLogsData{}, err
		}

		// Append data to the overall rule data
		allCertificatesData = append(allCertificatesData, certificateData)
	}

	return allCertificatesData, nil

}
