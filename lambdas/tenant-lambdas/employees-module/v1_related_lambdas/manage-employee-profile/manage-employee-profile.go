package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
	utils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	employeeSvc companylib.EmployeeService
	cdnSvc      companylib.CDNService
	contentSvc  companylib.TenantUploadContentService

	// Cards MetaData Service
	cardsMetaSvc  companylib.CompanyCardsMetadataService
	CardsMetaData map[string]companylib.CompanyCardsMetaDataTable
}

var RESP_HEADERS = companylib.GetHeadersForAPI("ProfileAPI")

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "manage-employee-profile")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	dynamodbClient := dynamodb.NewFromConfig(cfg)
	secretsClient := secretsmanager.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	logger := log.New(os.Stdout, "", log.LstdFlags)

	employeeSvc := companylib.CreateEmployeeService(ctx, dynamodbClient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	// Create Cards Meta Data Service
	cardsMetaSvc := companylib.CreateCompanyCardsMetadataService(ctx, logger, dynamodbClient, s3Client)
	cardsMetaSvc.CompanyCardsMetaDataTable = os.Getenv("CARDS_META_DATA_TABLE")
	cardsMetaSvc.BucketName = os.Getenv("BUCKET_NAME")

	// Tenant Upload Content Service
	contentSvc := companylib.CreateTenantUploadContentService(ctx, s3Client, logger)
	contentSvc.S3Bucket = os.Getenv("BUCKET_NAME")

	// Here we are creating a CDN Service
	cdnSvc := companylib.CDNService{}
	err = cdnSvc.CreateCDNService(ctx, logger, secretsClient, os.Getenv("SECRETS_CND_PK_ARN"), os.Getenv("PUBLIC_KEY_ID"))
	if err != nil {
		log.Fatalf("Error creating CDN Service: %v\n", err)
	}
	cdnSvc.CDNDomain = os.Getenv("CDN_DOMAIN")

	svc := Service{
		ctx:          ctx,
		logger:       logger,
		employeeSvc:  *employeeSvc,
		cdnSvc:       cdnSvc,
		cardsMetaSvc: *cardsMetaSvc,
		contentSvc:   *contentSvc,
	}

	// Assign all the Cards Meta Data to the Service
	err = svc.AssignCardsMetaData()
	if err != nil {
		svc.logger.Printf("Error Signing URLs for Cards Meta Data: %v\n", err)
		return // Exit the program
	}

	lambda.Start(svc.handleAPIRequests)
}

// Function to sign the URLS of the Cards Meta Data
func (svc *Service) AssignCardsMetaData() error {

	allCardsMetaData, err := svc.cardsMetaSvc.GetAllCardsMetaData()
	if err != nil {
		svc.logger.Printf("Error getting all Cards Meta Data: %v\n", err)
		return err
	}

	for metadataKey, cardMetaData := range allCardsMetaData.AllCardsTemplates {
		// Sign the URLS
		for cardKey, card := range cardMetaData.CardImages {
			allCardsMetaData.AllCardsTemplates[metadataKey].CardImages[cardKey] = svc.cdnSvc.GetPreSignedCDN_URL_noError(card)
		}
	}

	svc.CardsMetaData = allCardsMetaData.AllCardsTemplates

	return nil
}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.ctx = ctx

	switch request.HTTPMethod {
	case "GET":
		return svc.GETRequestHandler(request)
	case "POST":
		return svc.POSTRequestHandler(request)
	case "DELETE":
		return svc.DELETERequestHandler(request)
	default:
		svc.logger.Printf("entered Default section of the switch, Erroring by returning 500")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

const (
	GET_DASHBOARD_PROFILE  = "get-dashboard-profile"
	GET_REWARDS_PROFILE    = "get-rewards-profile"
	GET_TEAMS_PROFILE      = "get-teams-profile"
	GET_SEND_KUDOS_PROFILE = "get-send-kudos-profile"

	GET_PROFILE_DATA      = "get-profile-edit-data"
	GET_USER_CERTIFICATES = "get-user-certificates"

	PATCH_PROFILE_DATA = "patch-profile-data"
)

func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	switch request.Headers["get_type"] {
	case GET_DASHBOARD_PROFILE:
		return svc.GetDashboardProfile(request)
	case GET_REWARDS_PROFILE:
		return svc.GetRewardsProfile(request)
	case GET_USER_CERTIFICATES:
		return svc.GetUserProfileCertificates(request)
	case GET_SEND_KUDOS_PROFILE:
		return svc.GetSendKudosProfile(request)

	default:
		svc.logger.Printf("Unknown Search Condition")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

type DashboardProfileData struct {
	DisplayName string          `json:"DisplayName"`
	Designation string          `json:"Designation"`
	ProfilePic  string          `json:"ProfilePic"`
	EmailID     string          `json:"EmailId"`
	Location    string          `json:"Location"`
	PhoneNumber string          `json:"PhoneNumber"`
	Roles       map[string]bool `json:"RolesData"`

	Certificates map[string]companylib.EmployeeCertificates `json:"Certificates"`

	// Dashboard Specific Data
	TotalRewardPoints int `json:"TotalRewardPoints"`
	TotalCertificates int `json:"TotalCertificates"`
}

func (svc *Service) GetDashboardProfile(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)
	employeeData, err := svc.employeeSvc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	profilePic := svc.cdnSvc.GetPreSignedCDN_URL_noError(employeeData.ProfilePic)

	for _, cert := range employeeData.CertificatesData {
		cert.CertificatesImg = svc.cdnSvc.GetPreSignedCDN_URL_noError(cert.CertificatesImg)
	}

	dashboardData := DashboardProfileData{
		DisplayName:       employeeData.DisplayName,
		Designation:       employeeData.Designation,
		ProfilePic:        profilePic,
		EmailID:           employeeData.EmailID,
		Location:          employeeData.Location,
		PhoneNumber:       employeeData.PhoneNumber,
		Certificates:      employeeData.CertificatesData,
		Roles:             employeeData.RolesData,
		TotalRewardPoints: GetTotalRewards(employeeData),
		TotalCertificates: GetTotalCertificates(employeeData),
	}

	apiRes, err := json.Marshal(dashboardData)
	if err != nil {
		svc.logger.Printf("Error marshalling Dashboard Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(apiRes),
	}, nil
}

func GetTotalRewards(data companylib.EmployeeDynamodbData) int {
	// Rewards data in DDB : RewardsData map[string]EmployeeRewards `json:"RewardsData" dynamodbav:"RewardsData"`

	totalRewards := 0
	for _, reward := range data.RewardsData {
		totalRewards += reward.RewardPoints
	}
	return totalRewards
}
func GetTotalCertificates(data companylib.EmployeeDynamodbData) int {
	// Certificates Data : CertificatesData map[string]EmployeeCertificates `json:"CertificatesData" dynamodbav:"CertificatesData"`
	return len(data.CertificatesData)
}

// Cards Meta Data Signed
type RewardsProfileData struct {
	TotalRewardPoints     int    `json:"TotalRewardPoints"`
	RewardExpiryStatement string `json:"RewardExpiryStatement"`

	// Rewards Data
	RewardsData map[string]companylib.EmployeeRewards `json:"RewardsData"`

	// Redeemed Cards
	RedeemedCards map[string]RedeemedCardsFullData `json:"RedeemedCards"` // map of CardID to Redeemed Data
}

type RedeemedCardsFullData struct {
	CardMetaData companylib.CompanyCardsMetaDataTable `json:"CardMetaData"`
	RedeemedData companylib.RewardCards               `json:"RedeemedData"`
}

func (svc *Service) GetRewardsProfile(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)
	empData, err := svc.employeeSvc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Create RewardsProfileData
	rewardsProfileData := RewardsProfileData{
		TotalRewardPoints:     GetTotalRewards(empData),
		RewardExpiryStatement: svc.GetExpiryStatement(empData),
		RewardsData:           empData.RewardsData,
		RedeemedCards:         svc.GetCardsFullData(empData),
	}

	apiRes, err := json.Marshal(rewardsProfileData)
	if err != nil {
		svc.logger.Printf("Error marshalling Rewards Profile Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(apiRes),
	}, nil
}

func (svc *Service) GetCardsFullData(empData companylib.EmployeeDynamodbData) map[string]RedeemedCardsFullData {
	// Get the Redeemed Cards Data
	// Redeemed Data : RedeemedData map[string]RewardCards `json:"RedeemedData" dynamodbav:"RedeemedData"`
	// Card Meta Data : CardMetaData map[string]CompanyCardsMetaDataTable `json:"CardMetaData" dynamodbav:"CardMetaData"`

	redeemedCardsData := make(map[string]RedeemedCardsFullData)

	for cardNumber, redeemedData := range empData.RedeemedCards {
		redeemedCardsData[cardNumber] = RedeemedCardsFullData{
			CardMetaData: svc.CardsMetaData[redeemedData.CardId],
			RedeemedData: redeemedData,
		}
	}

	return redeemedCardsData
}

// Get Expiry Statement notifications for the Cards
func (svc *Service) GetExpiryStatement(empData companylib.EmployeeDynamodbData) string {
	// Get the Current Date in format YYYY-MM-DD
	currentDate := utils.GenerateDate()

	totalExpiringRewards := 0
	// Get the Expiry Date for each rewards
	for _, reward := range empData.RewardsData {
		// Get the Expiry Date
		expiryDate := reward.RewardsExpiryDate
		// Calculate the difference between current date and ExpiryDate of the format("YYYY-MM-DD")
		// Parse the date strings into time.Time objects
		layout := "2006-01-02" // Layout for parsing dates in YYYY-MM-DD format
		parsedDate1, err := time.Parse(layout, currentDate)
		if err != nil {
			return ""
		}
		parsedDate2, err := time.Parse(layout, expiryDate)
		if err != nil {
			return ""
		}
		duration := parsedDate2.Sub(parsedDate1)
		// Get the difference in days
		days := int(duration.Hours() / 24)

		if days <= 30 {
			totalExpiringRewards = totalExpiringRewards + reward.RewardPoints
		}
	}
	if totalExpiringRewards == 0 {
		return ""
	}
	// Calculate the difference
	// Return the Statement
	return fmt.Sprintf("You have %d rewards expiring in the next 30 days", totalExpiringRewards)
}

type ProfileCertificateData struct {
	CertificatesId  string `json:"CertificatesId"`
	CertificatesImg string `json:"CertificatesImg"`
	Title           string `json:"Title"`
	DateAwarded     string `json:"DateAwarded"`
}

func (svc *Service) GetUserProfileCertificates(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)
	employeeData, err := svc.employeeSvc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Printf("Employee Data: %v\n", employeeData)
	svc.logger.Printf("Employee Certificates Data: %v\n", employeeData.CertificatesData)

	profileCertificates := make([]ProfileCertificateData, 0)
	for _, cert := range employeeData.CertificatesData {
		svc.logger.Printf("Certificate: %v\n", cert)
		profileCertificates = append(profileCertificates, ProfileCertificateData{
			CertificatesId:  cert.CertificatesId,
			CertificatesImg: svc.cdnSvc.GetPreSignedCDN_URL_noError(cert.CertificatesImg),
			Title:           cert.Title,
			DateAwarded:     cert.DateAwarded,
		})
	}

	respBody, err := json.Marshal(profileCertificates)
	if err != nil {
		svc.logger.Printf("Error marshalling Profile Certificates Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(respBody),
	}, nil
}

// Get Send Kudos Profile
type SendKudosProfileData struct {
	DisplayName string                                `json:"DisplayName"`
	Designation string                                `json:"Designation"`
	ProfilePic  string                                `json:"ProfilePic"`
	RewardsData map[string]companylib.EmployeeRewards `json:"RewardsData"`
	TotalPoints int                                   `json:"TotalPoints"`
}

func (svc *Service) GetSendKudosProfile(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)
	employeeData, err := svc.employeeSvc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	profilePic := svc.cdnSvc.GetPreSignedCDN_URL_noError(employeeData.ProfilePic)

	sendKudosProfileData := SendKudosProfileData{
		DisplayName: employeeData.DisplayName,
		Designation: employeeData.Designation,
		ProfilePic:  profilePic,
		RewardsData: employeeData.RewardsData,
		TotalPoints: GetTotalTransferableRewards(employeeData),
	}

	apiRes, err := json.Marshal(sendKudosProfileData)
	if err != nil {
		svc.logger.Printf("Error marshalling Send Kudos Profile Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(apiRes),
	}, nil
}

func GetTotalTransferableRewards(data companylib.EmployeeDynamodbData) int {
	// Rewards data in DDB : RewardsData map[string]EmployeeRewards `json:"RewardsData" dynamodbav:"RewardsData"`

	totalRewards := 0
	for _, reward := range data.RewardsData {
		totalRewards += reward.TransferablePoints
	}
	return totalRewards
}

const (
	UPLOAD_PROFILE_PIC = "upload-profile-picture"
)

// POST Request Handler
func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.Headers["post_type"] {
	case UPLOAD_PROFILE_PIC:
		return svc.UploadProfilePic(request)
	default:
		svc.logger.Printf("Unknown Search Condition")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

type ProfilePicData struct {
	ProfilePic string `json:"ProfilePic"` // base64 encoded image
}

func (svc *Service) UploadProfilePic(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)
	employeeData, err := svc.employeeSvc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Get the Base64 Encoded Profile Pic
	var profilePic ProfilePicData
	err = json.Unmarshal([]byte(request.Body), &profilePic)
	if err != nil {
		svc.logger.Printf("Error un-marshaling Profile Pic Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	UploadKey := fmt.Sprintf("users/%s/profilepic/profile.png", employeeData.UserName)

	svc.logger.Printf("Profile Pic being uploaded to : %v\n", UploadKey)

	// Upload the Profile Pic to S3
	err = svc.contentSvc.UploadContentToS3_Base64Content(UploadKey, profilePic.ProfilePic)
	if err != nil {
		svc.logger.Printf("Error uploading Profile Pic: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Update the Profile Pic in the Employee Data
	err = svc.employeeSvc.UpdateEmployeeProfilePicByUserName(employeeData.UserName, UploadKey)
	if err != nil {
		svc.logger.Printf("Error updating Profile Pic in Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

const (
	DELETE_PROFILE_PIC = "delete-profile-picture"
)

// Delete Request Handler
func (svc *Service) DELETERequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.Headers["delete_type"] {
	case DELETE_PROFILE_PIC:
		return svc.DeleteProfilePic(request)
	default:
		svc.logger.Printf("Unknown Search Condition")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) DeleteProfilePic(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)
	employeeData, err := svc.employeeSvc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Delete the Profile Pic
	DeleteKey := fmt.Sprintf("users/%s/profilepic/profile.png", employeeData.UserName)
	err = svc.contentSvc.DeleteContentFromS3(DeleteKey)
	if err != nil {
		svc.logger.Printf("Error deleting Profile Pic: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Update the Profile Pic in the Employee Data with empty string
	err = svc.employeeSvc.UpdateEmployeeProfilePicByUserName(employeeData.UserName, "")
	if err != nil {
		svc.logger.Printf("Error updating Profile Pic in Employee Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil

}
