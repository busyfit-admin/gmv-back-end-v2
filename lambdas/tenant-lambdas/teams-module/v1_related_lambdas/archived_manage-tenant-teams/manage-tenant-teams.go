package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	"github.com/aws/aws-lambda-go/lambda"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	teamsSVC companylib.TenantTeamsService
	empSVC   companylib.EmployeeService
	cdnSvc   companylib.CDNService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("TeamsAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-tenant-teams")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	secretsClient := secretsmanager.NewFromConfig(cfg)

	teamssvc := companylib.CreateTenantTeamsService(ctx, ddbclient, logger)
	teamssvc.TenantTeamsTable = os.Getenv("TENANT_TEAMS_TABLE")
	teamssvc.TenantTeams_TeamIndex = os.Getenv("TENANT_TEAM_INDEX_TEAMS")

	empSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	empSvc.EmployeeTable = os.Getenv("EMPLOYEES_TABLE")
	empSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEES_TABLE_COGNITO_ID_INDEX")

	cdnSvc := companylib.CDNService{}
	err = cdnSvc.CreateCDNService(ctx, logger, secretsClient, os.Getenv("SECRETS_CND_PK_ARN"), os.Getenv("PUBLIC_KEY_ID"))
	if err != nil {
		log.Fatalf("Error creating CDN Service: %v\n", err)
	}
	cdnSvc.CDNDomain = os.Getenv("CDN_DOMAIN")

	svc := Service{
		ctx:      ctx,
		logger:   logger,
		teamsSVC: *teamssvc,
		empSVC:   *empSvc,
		cdnSvc:   cdnSvc,
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
	GET_ALL_TEAMS_DATA  = "get-all-teams"
	GET_TEAM_DATA       = "get-team"
	GET_TEAM_USERS_DATA = "get-team-users"
	GET_TEAM_MNGRS_DATA = "get-team-managers"
	GET_USER_TEAMS_DATA = "get-user-teams"
	GET_MNGR_TEAMS_DATA = "get-manager-teams"
	CREATE_TEAMS        = "create-team"
	ADD_USERS           = "add-users"
	DELETE_USERS        = "delete-users"
)

func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	getType := request.Headers["get_type"]
	switch getType {
	case GET_ALL_TEAMS_DATA:
		return svc.GetAllTeams(request)
	case GET_TEAM_DATA:
		return svc.GetTeam(request)
	case GET_TEAM_USERS_DATA:
		return svc.GetAllUsersInTeam(request)
	case GET_TEAM_MNGRS_DATA:
		return svc.GetAllManagersOfTeams(request)
	case GET_USER_TEAMS_DATA:
		return svc.GetTeamsOfOneUser(request)
	case GET_MNGR_TEAMS_DATA:
		return svc.GetTeamsOfOneManager(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	postType := request.Headers["post_type"]
	switch postType {
	case CREATE_TEAMS:
		return svc.CreateTeams(request)
	case ADD_USERS:
		return svc.AddUsers(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) DELETERequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	deleteType := request.Headers["delete_type"]
	switch deleteType {
	case DELETE_USERS:
		return svc.DeleteUsers(request)
	default:
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) PATCHRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var updateTeamData companylib.TenantTeamsTable
	err := json.Unmarshal([]byte(request.Body), &updateTeamData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	err = svc.teamsSVC.UpdateTenantTeam(updateTeamData)
	if err != nil {
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

func (svc *Service) GetAllTeams(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allTeams, err := svc.teamsSVC.GetAllTenantTeams()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allTeams)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}
func (svc *Service) GetTeam(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	teamId := request.Headers["team-id"]
	teamDef := request.Headers["related-id"]

	TeamData, err := svc.teamsSVC.GetTeamDetails(teamId, teamDef)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(TeamData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func (svc *Service) GetTeamsOfOneUser(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Authorizer for Employee Data
	authData, status, err := svc.empSVC.Authorizer(request, companylib.QUERY_NULL)
	if !status || err != nil {
		svc.logger.Printf("Error getting Employee Authorization Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	// Get the teams of the user
	userId := "USER-" + authData.Username
	userTeams, err := svc.teamsSVC.GetTeams(userId)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(userTeams)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func (svc *Service) GetTeamsOfOneManager(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	mngrId := request.Headers["manager-id"]
	userTeams, err := svc.teamsSVC.GetTeams(mngrId)

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(userTeams)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func (svc *Service) GetAllManagersOfTeams(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	teamId := request.Headers["team-id"]
	userTeams, err := svc.teamsSVC.GetTeamMngrs(teamId)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	// Convert the userList to the UserDetails
	userDetailsList, err := svc.empSVC.GetUserDetailsFromList(userTeams)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	for i := range userDetailsList {
		// get presigned URL for the image
		userDetailsList[i].ProfilePic = svc.cdnSvc.GetPreSignedCDN_URL_noError(userDetailsList[i].ProfilePic)
		if err != nil {
			svc.logger.Println("Error getting presigned URL:", err)
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
			}, nil
		}
	}

	responseBody, err := json.Marshal(userDetailsList)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func (svc *Service) GetAllUsersInTeam(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	teamId := request.Headers["team-id"]
	userTeams, err := svc.teamsSVC.GetTeamUsers(teamId)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// svc.logger.Printf("userTeams : %v", userTeams)
	// Convert the userList to the UserDetails
	userDetailsList, err := svc.empSVC.GetUserDetailsFromList(userTeams)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	for i := range userDetailsList {
		// get presigned URL for the image
		userDetailsList[i].ProfilePic = svc.cdnSvc.GetPreSignedCDN_URL_noError(userDetailsList[i].ProfilePic)
		if err != nil {
			svc.logger.Println("Error getting presigned URL:", err)
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
			}, nil
		}
	}

	responseBody, err := json.Marshal(userDetailsList)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(responseBody),
	}, nil
}

func (svc *Service) CreateTeams(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var createTeamData companylib.CreateTenantTeamsInput
	err := json.Unmarshal([]byte(request.Body), &createTeamData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Add the prefix to the Manager ID
	createTeamData.MngrId = "MNGR-" + createTeamData.MngrId

	EntityId, err := svc.teamsSVC.CreateTenantTeams(createTeamData)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    RESP_HEADERS,
		Body:       EntityId,
	}, nil
}

func (svc *Service) AddUsers(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var addUserData companylib.AddDeleteUserInput
	err := json.Unmarshal([]byte(request.Body), &addUserData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	svc.logger.Printf("manage Data : %v", addUserData)

	err = svc.teamsSVC.AddDeleteUsersToTeam(addUserData, "POST")
	if err != nil {
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

func (svc *Service) DeleteUsers(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var deleteUserData companylib.AddDeleteUserInput
	err := json.Unmarshal([]byte(request.Body), &deleteUserData)
	if err != nil {
		svc.logger.Printf("error unmarshal the input: %v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	err = svc.teamsSVC.AddDeleteUsersToTeam(deleteUserData, "DELETE")
	if err != nil {
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
