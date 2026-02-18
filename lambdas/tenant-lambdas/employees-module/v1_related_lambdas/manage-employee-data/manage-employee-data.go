package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	employeeSvc companylib.EmployeeService
	cdnSvc      companylib.CDNService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("UsersAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-employee-data")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	dynamodbClient := dynamodb.NewFromConfig(cfg)
	secretsClient := secretsmanager.NewFromConfig(cfg)
	logger := log.New(os.Stdout, "", log.LstdFlags)

	employeeSvc := companylib.CreateEmployeeService(ctx, dynamodbClient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_ExternalId_Index = os.Getenv("EMPLOYEE_TABLE_EXTERNALID_INDEX")
	employeeSvc.EmployeeTable_EmailId_Index = os.Getenv("EMPLOYEE_TABLE_EMAILID_INDEX")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	// Here we are creating a CDN Service
	cdnSvc := companylib.CDNService{}
	err = cdnSvc.CreateCDNService(ctx, logger, secretsClient, os.Getenv("SECRETS_CND_PK_ARN"), os.Getenv("PUBLIC_KEY_ID"))
	if err != nil {
		log.Fatalf("Error creating CDN Service: %v\n", err)
	}
	cdnSvc.CDNDomain = os.Getenv("CDN_DOMAIN")

	svc := Service{
		ctx:         ctx,
		logger:      logger,
		employeeSvc: *employeeSvc,
		cdnSvc:      cdnSvc,
	}

	lambda.Start(svc.handleAPIRequests)

}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.ctx = ctx

	switch request.HTTPMethod {
	case "GET":
		return svc.GETRequestHandler(request)
	case "PATCH":
		return svc.PATCHRequestHandler(request)
	case "POST":
		return svc.POSTRequestHandler(request)
	default:
		svc.logger.Printf("entered Default section of the switch, Erroring by returning 500")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

const (
	GET_USER_BY_USERNAME            = "get-user-by-username"
	GET_USER_BY_USERNAME_BASIC_DATA = "get-user-by-username-basic-data"
	GET_USER_BY_EMAIL               = "get-user-by-email"
	GET_USER_BY_EXTID               = "get-user-by-ext-id"

	GET_ALL_USERS = "get-all-users"
)

func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user is Admin or User Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORUserManagementRole")
	if !isAuth || err != nil {
		svc.logger.Println("Error in Authorization:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	switch request.Headers["get_type"] {
	case GET_USER_BY_USERNAME:
		return svc.GetEmployeeByUserName(request)
	case GET_USER_BY_USERNAME_BASIC_DATA:
		return svc.GetEmployeeByUserNameBasicData(request)
	case GET_USER_BY_EMAIL:
		return svc.GetEmployeeByEmail(request)
	case GET_USER_BY_EXTID:
		return svc.GetEmployeeByExtId(request)
	case GET_ALL_USERS:
		return svc.GetAllUsers(request)
	default:
		svc.logger.Printf("Unknown Search Condition")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) GetEmployeeByUserName(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	user_id := request.Headers["user-id"]

	EmployeeData, err := svc.employeeSvc.GetEmployeeDataByUserName(user_id)
	if err != nil {
		svc.logger.Println("Error getting employee data by username:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	// get presigned URL for the image
	EmployeeData.ProfilePic = svc.cdnSvc.GetPreSignedCDN_URL_noError(EmployeeData.ProfilePic)

	// convert to Json String
	jsonData, err := json.Marshal(EmployeeData)
	if err != nil {
		svc.logger.Println("Error converting struct to JSON:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonData),
	}, nil
}
func (svc *Service) GetEmployeeByUserNameBasicData(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	user_id := request.Headers["user-id"]

	EmployeeData, err := svc.employeeSvc.GetEmployeeDataByUserNameBasicData(user_id)
	if err != nil {
		svc.logger.Println("Error getting employee data by username:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	// get presigned URL for the image
	EmployeeData.ProfilePic = svc.cdnSvc.GetPreSignedCDN_URL_noError(EmployeeData.ProfilePic)

	// convert to Json String
	jsonData, err := json.Marshal(EmployeeData)
	if err != nil {
		svc.logger.Println("Error converting struct to JSON:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonData),
	}, nil
}

func (svc *Service) GetEmployeeByEmail(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	user_id := request.Headers["user-id"]

	EmployeeData, err := svc.employeeSvc.GetEmployeeDataByEmail(user_id)
	if err != nil {
		svc.logger.Println("Error getting employee data by email:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	// convert to Json String
	jsonData, err := json.Marshal(EmployeeData)
	if err != nil {
		svc.logger.Println("Error converting struct to JSON:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonData),
	}, nil
}
func (svc *Service) GetEmployeeByExtId(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	user_id := request.Headers["user-id"]

	EmployeeData, err := svc.employeeSvc.GetEmployeeDataByExternalId(user_id)
	if err != nil {
		svc.logger.Println("Error getting employee data by external id:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	// convert to Json String
	jsonData, err := json.Marshal(EmployeeData)
	if err != nil {
		svc.logger.Println("Error converting struct to JSON:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonData),
	}, nil
}
func (svc *Service) GetAllUsers(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	EmployeeData, err := svc.employeeSvc.GetAllEmployeeData()
	if err != nil {
		svc.logger.Println("Error getting all employee data:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	for i := range EmployeeData {
		// get presigned URL for the image
		EmployeeData[i].ProfilePic = svc.cdnSvc.GetPreSignedCDN_URL_noError(EmployeeData[i].ProfilePic)
	}
	// convert to Json String
	jsonData, err := json.Marshal(EmployeeData)
	if err != nil {
		svc.logger.Println("Error converting struct to JSON:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(jsonData),
	}, nil
}

const (
	SEARCH_USRNM = "SEARCH_BY_USERNAME"
	SEARCH_EMAIL = "SEARCH_BY_EMAIL"
	SEARCH_EXTID = "SEARCH_BY_EXTERNAL_ID"
	SEARCH_ALLID = "SEARCH_ALL"
)

// Update Employee Roles
const (
	BASIC_UPDATE_USER = "basic-update-user-by-username"
	FULL_UPDATE_USER  = "full-update-user-by-username"

	UPDATE_BY_USERNAME   = "update-roles-by-username"
	UPDATE_BY_EMAILID    = "update-roles-by-emailid"
	UPDATE_BY_EXTERNALID = "update-roles-by-externalid"
)

type UpdateRolesRequestInput struct {
	UserName   string               `json:"UserName"`
	EmailId    string               `json:"EmailId"`
	ExternalId string               `json:"ExternalId"`
	RoleData   companylib.RoleNames `json:"RoleData"`
}

func (svc *Service) PATCHRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// 1) Authorization at User Level and Check if user is Admin or User Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORUserManagementRole")
	if !isAuth || err != nil {
		svc.logger.Println("Error in Authorization:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Map the event detail to the EmployeeDynamodbData struct
	var requestData UpdateRolesRequestInput
	if err := json.Unmarshal([]byte(request.Body), &requestData); err != nil {
		svc.logger.Println("Error unmarshaling employee data:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error unmarshaling employee data",
		}, err
	}

	switch request.Headers["patch_type"] {
	case BASIC_UPDATE_USER:
		return svc.basicUpdateUserByUserName(request)
	case FULL_UPDATE_USER:
		return svc.FullUpdateUserByUserName(request)
	case UPDATE_BY_USERNAME:
		return svc.updateByUserName(requestData.UserName, requestData.RoleData)
	case UPDATE_BY_EMAILID:
		return svc.updateByEmailId(requestData.EmailId, requestData.RoleData)
	case UPDATE_BY_EXTERNALID:
		return svc.updateByExternalId(requestData.ExternalId, requestData.RoleData)
	default:
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    RESP_HEADERS,
			Body:       "Invalid patch type",
		}, nil
	}
}

func (svc *Service) basicUpdateUserByUserName(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var requestData companylib.BasicEmployeeDetails
	if err := json.Unmarshal([]byte(request.Body), &requestData); err != nil {
		svc.logger.Println("Error unmarshal employee data:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       "Error unmarshal of employee data",
		}, err
	}

	err := svc.employeeSvc.UpdateEmployeeDetailsByUserName(requestData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       "Error unmarshal of employee data",
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

/*

ex:
	requestType: "PATCH"
	request headers:
		"patch_type" : "full-update-by-username"

	requestBody:
	{
		"UserName" : "abc",
		"IsActive": "Y"
		....
		"RoleData" : {
			"AdminRole": true,
			"User": true,
			....
		}
	}


*/

func (svc *Service) FullUpdateUserByUserName(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var requestData companylib.AllEmployeeDetails
	if err := json.Unmarshal([]byte(request.Body), &requestData); err != nil {
		svc.logger.Println("Error unmarshal employee data:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       "Error unmarshal of employee data",
		}, err
	}

	err := svc.employeeSvc.UpdateAllEmployeeDetailsByUserName(requestData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
			Body:       "Error unmarshal of employee data",
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

func (svc *Service) updateByUserName(userName string, roleData companylib.RoleNames) (events.APIGatewayProxyResponse, error) {

	err := svc.employeeSvc.UpdateRolesofEmployeeByUserName(userName, roleData)
	if err != nil {
		svc.logger.Println("Error in UpdateEmployeeRoles by userName:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    RESP_HEADERS,
			Body:       "Error updating employee roles by user name",
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
		Body:       "Updating role successful",
	}, err
}

func (svc *Service) updateByEmailId(emailId string, roleData companylib.RoleNames) (events.APIGatewayProxyResponse, error) {
	err := svc.employeeSvc.UpdateRolesofEmployeeByEmailId(emailId, roleData)
	if err != nil {
		svc.logger.Println("Error in UpdateEmployeeRoles by emailId:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    RESP_HEADERS,
			Body:       "Error updating employee roles by emailId",
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
		Body:       "Updating role successful",
	}, err
}

func (svc *Service) updateByExternalId(externalId string, roleData companylib.RoleNames) (events.APIGatewayProxyResponse, error) {
	err := svc.employeeSvc.UpdateRolesofEmployeeByExternalId(externalId, roleData)
	if err != nil {
		svc.logger.Println("Error in UpdateEmployeeRoles by externalId:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    RESP_HEADERS,
			Body:       "Error updating employee roles by externalId",
		}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
		Body:       "Updating role successful",
	}, err
}

// Create Employee Data:

func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user is Admin or User Manager
	_, isAuth, err := svc.employeeSvc.Authorizer(request, "AdminRoleORUserManagementRole")
	if !isAuth || err != nil {
		svc.logger.Println("Error in Authorization:", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	var userData companylib.EmployeeDynamodbData
	err = json.Unmarshal([]byte(request.Body), &userData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "error unmarshalling request body",
		}, nil
	}

	err = svc.employeeSvc.CreateEmployeeFromFrontend(userData)
	if err != nil {
		svc.logger.Println("Error converting all groups to JSON:", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Error creating employee",
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
	}, nil
}
