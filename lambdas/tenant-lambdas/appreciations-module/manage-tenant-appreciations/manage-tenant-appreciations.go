package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

type SQSClient interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type Service struct {
	ctx                       context.Context
	logger                    *log.Logger
	sqsClient                 SQSClient
	REWARDS_TRANSFER_SQS_NAME string

	apprSvc    companylib.TenantEngagementService
	cdnSvc     companylib.CDNService
	empSvc     companylib.EmployeeService
	teamSvc    companylib.TenantTeamsService
	contentSvc companylib.TenantUploadContentService
}

var RESP_HEADERS = companylib.GetHeadersForAPI("EngagementsAPI")

func main() {

	ctx, root := xray.BeginSegment(context.TODO(), "manage-tenant-appreciations")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	// All Clients
	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	secretsClient := secretsmanager.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	sqsClient := sqs.NewFromConfig(cfg)

	// Create Engagement Service
	engagementSsvc := companylib.CreateEngagementService(ctx, ddbclient, logger)
	engagementSsvc.TenantSkillsTable = os.Getenv("TENANT_SKILLS_TABLE")
	engagementSsvc.TenantValuesTable = os.Getenv("TENANT_VALUES_TABLE")
	engagementSsvc.TenantMilestonesTable = os.Getenv("TENANT_MILESTONES_TABLE")
	engagementSsvc.TenantMetricsTable = os.Getenv("TENANT_METRICS_TABLE")
	engagementSsvc.TenantEngagementTable = os.Getenv("TENANT_ENGAGEMENT_TABLE")
	engagementSsvc.TenantEngagementEntityIdIndex = os.Getenv("TENANT_ENGAGEMENT_ENTITY_ID_INDEX")
	engagementSsvc.TenantEngagementEntityIdTimestampIndex = os.Getenv("TENANT_ENGAGEMENT_TIMESTAMP_INDEX")

	// Create CDN Service
	cdnSvc := companylib.CDNService{}
	err = cdnSvc.CreateCDNService(ctx, logger, secretsClient, os.Getenv("SECRETS_CND_PK_ARN"), os.Getenv("PUBLIC_KEY_ID"))
	if err != nil {
		log.Fatalf("Error creating CDN Service: %v\n", err)
	}
	cdnSvc.CDNDomain = os.Getenv("CDN_DOMAIN")

	// Employee Service
	employeeSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	employeeSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	employeeSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")
	employeeSvc.EmployeeGroupsTable = os.Getenv("EMPLOYEE_GROUPS_TABLE")

	// Team Service
	teamssvc := companylib.CreateTenantTeamsService(ctx, ddbclient, logger)
	teamssvc.TenantTeamsTable = os.Getenv("TENANT_TEAMS_TABLE")

	// Content Service
	contentSvc := companylib.CreateTenantUploadContentService(ctx, s3Client, logger)
	contentSvc.S3Bucket = os.Getenv("S3_BUCKET")

	svc := Service{
		ctx:    ctx,
		logger: logger,

		apprSvc:    *engagementSsvc,
		cdnSvc:     cdnSvc,
		empSvc:     *employeeSvc,
		teamSvc:    *teamssvc,
		contentSvc: *contentSvc,

		sqsClient:                 sqsClient,
		REWARDS_TRANSFER_SQS_NAME: os.Getenv("REWARD_TRANSFER_SQS_QUEUE"),
	}

	lambda.Start(svc.handleAPIRequests)
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
	GET_ALL_SKILLS_DATA      = "get-all-skills"
	GET_ALL_VALUES_DATA      = "get-all-values"
	GET_ALL_MILESTONES_DATA  = "get-all-milestones"
	GET_ALL_METRICS_DATA     = "get-all-metrics"
	GET_ALL_APP_OBJECTS_DATA = "get-all-app-objects"

	GET_TEAM_FEED_ENTITY_DATA = "get-team-feed-entity"
	GET_GRPS_FEED_ENTITY_DATA = "get-groups-feed-entity"

	GET_EVENTS_FEED_ENTITY = "get-events-feed-entity"

	GET_USER_PROFILE_APPRECIATIONS        = "get-user-appreciations"
	GET_COUNT_OF_EVENTS_AND_APPRECIATIONS = "get-count-of-events-and-appreciations"
)

func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	getType := request.Headers["get_type"]

	svc.logger.Println("Got", getType)
	switch getType {
	// Get all the Skills, Values, Milestones, Metrics and Appreciations Data Objects
	case GET_ALL_SKILLS_DATA:
		return svc.GetAllSkills(request)
	case GET_ALL_VALUES_DATA:
		return svc.GetAllValues(request)
	case GET_ALL_MILESTONES_DATA:
		return svc.GetAllMilestones(request)
	case GET_ALL_METRICS_DATA:
		return svc.GetAllMetrics(request)
	case GET_COUNT_OF_EVENTS_AND_APPRECIATIONS: // get count of events and appreciations
		return svc.GetCountOfEventsAndAppreciations(request)
	case GET_ALL_APP_OBJECTS_DATA:
		return svc.GetAllAppObjectsData(request) // Get all Appreciations Data Objects
	// Feeds for Teams and Groups
	case GET_TEAM_FEED_ENTITY_DATA: // Get feed lists for a team
		return svc.GetAllFeedForTeams(request)
	case GET_GRPS_FEED_ENTITY_DATA: // Get feed lists for a group
		return svc.GetAllFeedForGrps(request)
	// Events for Teams
	case GET_EVENTS_FEED_ENTITY: // Get events feed for a team
		return svc.GetAllEventsForTeams(request)
	// User Profile Appreciations
	case GET_USER_PROFILE_APPRECIATIONS:
		return svc.GetUserProfileAppreciations(request)
	default:
		svc.logger.Printf("Incorrect Get Header type")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

func (svc *Service) GetAllSkills(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allSkills, err := svc.apprSvc.GetAllSkills()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allSkills)
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
func (svc *Service) GetAllValues(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allValues, err := svc.apprSvc.GetAllValues()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allValues)
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
func (svc *Service) GetAllMilestones(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allMilestones, err := svc.apprSvc.GetAllMilestones()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allMilestones)
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
func (svc *Service) GetAllMetrics(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allMetrics, err := svc.apprSvc.GetAllMetrics()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allMetrics)
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

func (svc *Service) GetCountOfEventsAndAppreciations(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Println("Got GetCountOfEventsAndAppreciations")
	entityId := request.Headers["entity-id"]
	t, err := svc.apprSvc.CountOfEventsAndAppreciations(entityId)
	if err != nil {
		svc.logger.Printf("Error getting Count of Events and Appreciations: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	res, err1 := json.Marshal(t)
	if err1 != nil {
		svc.logger.Printf("Error marshalling Count of Events and Appreciations: %v\n", err1)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Println(string(res))
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
		Body:       string(res),
	}, nil
}

func (svc *Service) GetAllAppObjectsData(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	allAppObjects, err := svc.apprSvc.GetAllAppreciationsObjects()

	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	responseBody, err := json.Marshal(allAppObjects)
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

func (svc *Service) GetAllFeedForTeams(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	entityId := request.Headers["entity-id"]
	lastEvaluatedKey := request.Headers["last-evaluated-key"]

	// 1) Authorization at User Level and Check if user has the teams access
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		svc.logger.Printf("Error Authorizing the request: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	isAuth, err = svc.teamSvc.AuthorizerTeams(entityId, authData.Username)
	if !isAuth || err != nil {
		svc.logger.Printf("Error Authorizing the Team access request: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	//	2) Get Engagement Data
	allEngagements, err := svc.apprSvc.GetAllEngagementsFeed(entityId, lastEvaluatedKey, "TEAMFEED")
	if err != nil {
		svc.logger.Printf("Error getting Engagement Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// 3) Sign the contents of the response and assign to all engagements
	allEngagements.EngagementFeed = svc.cdnSvc.SignContentInAllEngagements(allEngagements.EngagementFeed)

	responseBody, err := json.Marshal(allEngagements)
	if err != nil {
		svc.logger.Printf("Error marshalling Engagement Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

func (svc *Service) GetUserProfileAppreciations(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	entityId := request.Headers["entity-id"]
	lastEvaluatedKey := request.Headers["last-evaluated-key"]

	svc.logger.Printf("Entity ID : %s\n", entityId)

	// 1) Authorization at User Level and Check if user has the teams access
	_, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	//	2) Get Engagement Data
	allAppreciation, err := svc.apprSvc.GetUserProfileAppreciations(entityId, lastEvaluatedKey)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// 3) Sign the contents of the response and assign to all engagements
	allAppreciation.EngagementFeed = svc.cdnSvc.SignContentInAllEngagements(allAppreciation.EngagementFeed)

	responseBody, err := json.Marshal(allAppreciation)
	if err != nil {
		svc.logger.Printf("Error marshalling Engagement Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

func (svc *Service) GetAllFeedForGrps(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	lastEvaluatedKey := request.Headers["last-evaluated-key"]

	// 1) Authorization at User Level and Check if user has the teams access
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Print("here", authData)

	// Add "Everyone" group to the list of groups
	authData.Groups = append(authData.Groups, "Everyone")

	// 2) Get Feed data for groups of the user
	feedData, err := svc.apprSvc.GetAllFeedGroups(authData.Groups, lastEvaluatedKey)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// 3) Sign the contents of the response
	feedData.EngagementFeed = svc.cdnSvc.SignContentInAllEngagements(feedData.EngagementFeed)

	responseBody, err := json.Marshal(feedData)
	if err != nil {
		svc.logger.Printf("Error marshalling Engagement Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}
func (svc *Service) GetAllEventsForTeams(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	entityId := request.Headers["entity-id"]
	lastEvaluatedKey := request.Headers["last-evaluated-key"]

	// 1) Authorization at User Level and Check if user has the teams access
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	isAuth, err = svc.teamSvc.AuthorizerTeams(entityId, authData.Username)
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// 2) Get Engagement Data
	allEngagements, err := svc.apprSvc.GetAllEngagementEvents(entityId, lastEvaluatedKey)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// 3) Sign the contents of the response and assign to all engagements
	allEngagements.EngagementFeed = svc.cdnSvc.SignContentInAllEngagements(allEngagements.EngagementFeed)

	responseBody, err := json.Marshal(allEngagements)
	if err != nil {
		svc.logger.Printf("Error marshalling Engagement Data: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       string(responseBody),
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

const (
	POST_SKILLS_DATA             = "post-skills"
	POST_VALUES_DATA             = "post-values"
	POST_MILESTONES_DATA         = "post-milestones"
	POST_METRICS_DATA            = "post-metrics"
	POST_ENGAGEMENT_FEED_TEAMS   = "post-engagement-feed-teams"
	POST_ENGAGEMENT_FEED_GROUP   = "post-engagement-feed-group"
	POST_ENGAGEMENT_FEED_LIKES   = "post-engagement-feed-likes"
	POST_ENGAGEMENT_EVENTS_TEAMS = "post-engagement-events-teams"
	POST_ENGAGEMENT_EVENTS_RSVP  = "post-engagement-events-rsvp"

	POST_ENGAGEMENT_SEND_KUDOS = "post-engagement-send-kudos"
)

func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	postType := request.Headers["post_type"]
	svc.logger.Printf("Post Type: %s\n", postType)
	switch postType {
	case POST_SKILLS_DATA:
		return svc.PostSkills(request)
	case POST_VALUES_DATA:
		return svc.PostValues(request)
	case POST_MILESTONES_DATA:
		return svc.PostMilestones(request)
	case POST_METRICS_DATA:
		return svc.PostMetrics(request)
	case POST_ENGAGEMENT_FEED_TEAMS:
		return svc.PostEngagementsFeedForTeams(request)
	case POST_ENGAGEMENT_FEED_GROUP:
		return svc.PostEngagementsFeedForGroup(request)
	case POST_ENGAGEMENT_EVENTS_TEAMS:
		return svc.PostEngagementsForEvents(request)
	case POST_ENGAGEMENT_EVENTS_RSVP:
		return svc.PostEngagementEventRSVP(request)
	case POST_ENGAGEMENT_FEED_LIKES:
		return svc.PostEngagementFeedsLikes(request)
	case POST_ENGAGEMENT_SEND_KUDOS:
		return svc.PostEngagementSendKudos(request)
	default:
		svc.logger.Printf("Incorrect Get Header type")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

type PostSkillInput struct {
	SkillName string `json:"skillName"`
	SkillDesc string `json:"skillDesc"`
}

func (svc *Service) PostSkills(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user has the teams access
	_, isAuth, err := svc.empSvc.Authorizer(request, "AdminRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Body:       "User doesn't have admin access",
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	var inputBody []PostSkillInput
	err = json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	var ddbInput []companylib.TenantSkillsTable

	for _, item := range inputBody {
		ddbInput = append(ddbInput, companylib.TenantSkillsTable{
			SkillId:   "SKILL-" + utils.GenerateRandomString(6),
			SkillName: item.SkillName,
			SkillDesc: item.SkillDesc,
		})
	}

	err = svc.apprSvc.PutSkillsToDynamoDB(ddbInput, "POST")
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

type PostValueInput struct {
	ValueName string `json:"valueName"`
	ValueDesc string `json:"valueDesc"`
}

func (svc *Service) PostValues(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user has the teams access
	_, isAuth, err := svc.empSvc.Authorizer(request, "AdminRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	var inputBody []PostValueInput
	err = json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	var ddbInput []companylib.TenantValuesTable

	for _, item := range inputBody {
		ddbInput = append(ddbInput, companylib.TenantValuesTable{
			ValueId:   "VALUE-" + utils.GenerateRandomString(6),
			ValueName: item.ValueName,
			ValueDesc: item.ValueDesc,
		})
	}

	err = svc.apprSvc.PutValuesToDynamoDB(ddbInput, "POST")
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

type PostMilestoneInput struct {
	MileStoneName string `json:"milestoneName"`
	MileStoneDesc string `json:"milestoneDesc"`
}

func (svc *Service) PostMilestones(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user has the teams access
	_, isAuth, err := svc.empSvc.Authorizer(request, "AdminRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	var inputBody []PostMilestoneInput
	err = json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	var ddbInput []companylib.TenantMilestonesTable

	for _, item := range inputBody {
		ddbInput = append(ddbInput, companylib.TenantMilestonesTable{
			MilestoneId:   "MILESTONE-" + utils.GenerateRandomString(6),
			MilestoneName: item.MileStoneName,
			MilestoneDesc: item.MileStoneDesc,
		})
	}

	err = svc.apprSvc.PutMilestonesToDynamoDB(ddbInput, "POST")
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

type PostMetricInput struct {
	MetricName string `json:"metricName"`
	MetricDesc string `json:"metricDesc"`
}

func (svc *Service) PostMetrics(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1) Authorization at User Level and Check if user has the teams access
	_, isAuth, err := svc.empSvc.Authorizer(request, "AdminRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	var inputBody []PostMetricInput
	err = json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	var ddbInput []companylib.TenantMetricsTable

	for _, item := range inputBody {
		ddbInput = append(ddbInput, companylib.TenantMetricsTable{
			MetricId:   "METRIC-" + utils.GenerateRandomString(6),
			MetricName: item.MetricName,
			MetricDesc: item.MetricDesc,
		})
	}

	err = svc.apprSvc.PutMetricsToDynamoDB(ddbInput, "POST")
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

// --------- Post Appreciations feed to Teams and Groups ------------

// 1. PostEngagementsForEntityInput is the input structure for the Post Appreciations in the Teams
type PostEngagementsForEntityInput struct {
	EntityId string `json:"EntityId"` // To Teams ID or User ID ( user ID functionality is not yet enabled for MVP)

	Skill     []string `json:"Skill"`     // skill Name to be entered
	Value     []string `json:"Value"`     // Value Name to be entered
	Milestone []string `json:"Milestone"` // Milestone Name to be entered
	Metrics   []string `json:"Metrics"`   // Metric Name to be entered

	Message string `json:"Message"` // Message of either engagement/ Milestone

	Images []string `json:"Images"` // Base64 encoded image data

	TransferrablePoints map[string]int `json:"TransferrablePoints"` // Transferrable Points to be entered
	TaggedUsers         []string       `json:"TaggedUsers"`         // Stores all the tagged users
}

// 2. PostEngagementsForEntityOutput is the output structure for the Post Appreciations in the Teams
type PostEngagementsForEntityOutput struct {
	EngagementId string `json:"EngagementId"`
}

func (svc *Service) PostEngagementsFeedForTeams(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get the Input Body
	var inputBody PostEngagementsForEntityInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	// 1) Authorization at User Level and Check if user has the teams access
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	isAuth, err = svc.teamSvc.AuthorizerTeams(inputBody.EntityId, authData.Username)
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Create Engagement ID
	appId := "TEAMFEED-" + utils.GenerateRandomString(12)

	// 2) Upload Images to S3 and get the images keys
	imageMap, imageKeys := generateImageMap(inputBody.Images, appId)

	// 3) Upload the images to S3
	err = svc.contentSvc.UploadMultipleContentsToS3_Base64Content(imageMap)
	if err != nil {
		svc.logger.Printf("Error uploading images to S3: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// 4) Create Engagement in DynamoDB for the tagger
	err = svc.apprSvc.CreateEngagement(companylib.TenantEngagementTable{
		EngagementId: appId,
		EntityId:     inputBody.EntityId,
		ProvidedBy:   authData.Username,
		ProvidedByContent: companylib.EntityData{
			DisplayName: authData.DisplayName,
			ProfilePic:  companylib.GetDefaultThumbnailPicPath(authData.Username),
			Designation: authData.Designation,
		},
		Message:     inputBody.Message,
		Skill:       inputBody.Skill,
		Value:       inputBody.Value,
		Milestone:   inputBody.Milestone,
		Metrics:     inputBody.Metrics,
		Likes:       map[string]companylib.EntityData{},
		Images:      imageKeys,
		TaggedUsers: inputBody.TaggedUsers, // Store the list of tagged users
		Timestamp:   utils.GenerateTimestamp(),
	})
	if err != nil {
		svc.logger.Printf("Unable to create team feed and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// 5) Create Engagement for each Tagged User
	for _, taggedUser := range inputBody.TaggedUsers {
		taggedAppId := "USER-" + utils.GenerateRandomString(12)

		taggedEngagement := companylib.TenantEngagementTable{
			EngagementId: taggedAppId,
			EntityId:     taggedUser,
			ProvidedBy:   authData.Username,
			ProvidedByContent: companylib.EntityData{
				DisplayName: authData.DisplayName,
				ProfilePic:  companylib.GetDefaultThumbnailPicPath(authData.Username),
				Designation: authData.Designation,
			},
			Message:     inputBody.Message,
			Skill:       inputBody.Skill,
			Value:       inputBody.Value,
			Milestone:   inputBody.Milestone,
			Metrics:     inputBody.Metrics,
			Likes:       map[string]companylib.EntityData{},
			Images:      imageKeys,
			TaggedUsers: []string{authData.Username}, // Store only the tagger's information
			Timestamp:   utils.GenerateTimestamp(),
		}

		err = svc.apprSvc.CreateEngagement(taggedEngagement)
		if err != nil {
			svc.logger.Printf("Failed to create engagement for tagged user %s: %v", taggedUser, err)
			continue
		}
	}

	resp := PostEngagementsForEntityOutput{
		EngagementId: appId,
	}

	responseBody, err := json.Marshal(resp)
	if err != nil {
		svc.logger.Printf("Error marshalling Engagement Data: %v\n", err)
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

func (svc *Service) PostEngagementsFeedForGroup(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get the Input Body
	var inputBody PostEngagementsForEntityInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	// 1) Authorization at User Level and Check if user has access to the Group
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		svc.logger.Printf("Unable to Authorize the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	svc.logger.Printf("Auth Data: %v\n", authData)

	isPartOfGroup := svc.empSvc.AuthorizerGroups(authData.Groups, inputBody.EntityId)

	if !isPartOfGroup {
		svc.logger.Printf("User is not part of the group %s", inputBody.EntityId)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Create Engagement ID
	appId := "GRPFEED-" + utils.GenerateRandomString(12)

	// 2) Upload Images to S3 and get the image keys
	imageMap, imageKeys := generateImageMap(inputBody.Images, appId)

	// 3) Upload the images to S3
	err = svc.contentSvc.UploadMultipleContentsToS3_Base64Content(imageMap)
	if err != nil {
		svc.logger.Printf("Error uploading images to S3: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// 4) Create Engagement in DynamoDB for the tagger
	err = svc.apprSvc.CreateEngagement(companylib.TenantEngagementTable{
		EngagementId: appId,
		EntityId:     inputBody.EntityId,
		ProvidedBy:   authData.Username,
		ProvidedByContent: companylib.EntityData{
			DisplayName: authData.DisplayName,
			ProfilePic:  companylib.GetDefaultThumbnailPicPath(authData.Username),
			Designation: authData.Designation,
		},
		Message:     inputBody.Message,
		Skill:       inputBody.Skill,
		Value:       inputBody.Value,
		Milestone:   inputBody.Milestone,
		Metrics:     inputBody.Metrics,
		Images:      imageKeys,
		TaggedUsers: inputBody.TaggedUsers, // Store the list of tagged users
		Timestamp:   utils.GenerateTimestamp(),
	})
	if err != nil {
		svc.logger.Printf("Unable to Create a Group Feed and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// 5) Create Engagement for each Tagged User
	for _, taggedUser := range inputBody.TaggedUsers {
		taggedAppId := "USER-" + utils.GenerateRandomString(12)
		svc.logger.Printf("INFO: Creating engagement entry for tagged user: %s", taggedUser)

		taggedEngagement := companylib.TenantEngagementTable{
			EngagementId: taggedAppId,
			EntityId:     taggedUser,
			ProvidedBy:   authData.Username,
			ProvidedByContent: companylib.EntityData{
				DisplayName: authData.DisplayName,
				ProfilePic:  companylib.GetDefaultThumbnailPicPath(authData.Username),
				Designation: authData.Designation,
			},
			Message:   inputBody.Message,
			Skill:     inputBody.Skill,
			Value:     inputBody.Value,
			Milestone: inputBody.Milestone,
			Metrics:   inputBody.Metrics,
			Likes:     map[string]companylib.EntityData{},
			Images:    imageKeys,
			Timestamp: utils.GenerateTimestamp(),
		}

		err = svc.apprSvc.CreateEngagement(taggedEngagement)
		if err != nil {
			svc.logger.Printf("ERROR: Failed to create engagement for tagged user %s: %v", taggedUser, err)
			continue
		}
		svc.logger.Printf("INFO: Successfully created engagement entry for tagged user %s with ID: %s", taggedUser, taggedAppId)
	}

	resp := PostEngagementsForEntityOutput{
		EngagementId: appId,
	}

	responseBody, err := json.Marshal(resp)
	if err != nil {
		svc.logger.Printf("Error marshalling Engagement Data: %v\n", err)
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

// Helper functions for PostEngagementsFeedForTeams and PostEngagementsFeedForGroup

// Generate random string for Image Key along with the engagement ID
func generateImageKey(engagementId string) string {
	return fmt.Sprintf("engagements/%s/images/img-%s", engagementId, utils.GenerateRandomString(8))
}

// Generate map[string]string (ie map[image_key]base64_data from input []string ( ie []base64_data ), image key is generated using the generateImageKey function
func generateImageMap(images []string, engagementId string) (map[string]string, []string) {
	imageMap := make(map[string]string)
	var imageKeys []string
	for _, image := range images {
		imageKey := generateImageKey(engagementId)
		imageMap[imageKey] = image
		imageKeys = append(imageKeys, imageKey)
	}
	// Return the image map and the keys
	return imageMap, imageKeys

}

type PostEngagementFeedsLikesInput struct {
	GroupId  string `json:"GroupId"`  // The Group ID where the user is providing the Likes.User needs to be part of the group.
	EntityId string `json:"EntityId"` // The Teams ID where the user is providing the Likes.User needs to be part of the team.

	EngagementId string `json:"EngagementId"`
	IsLike       bool   `json:"IsLike"`
}

// PostEngagementFeedsLikes is the function to post the Likes in the Team Feeds
func (svc *Service) PostEngagementFeedsLikes(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Get the Input Body
	var inputBody PostEngagementFeedsLikesInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	var EntityId string

	// 1) Authorization at User Level and Check if user has the teams access
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	if inputBody.EntityId == "" {
		// Likes for Groups Feed
		isAuth = svc.empSvc.AuthorizerGroups(authData.Groups, inputBody.GroupId)
		EntityId = inputBody.GroupId
		if !isAuth {
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
			}, nil
		}
	} else {
		// Likes for Teams Feed
		isAuth, err = svc.teamSvc.AuthorizerTeams(inputBody.EntityId, authData.Username)
		EntityId = inputBody.EntityId
		if !isAuth || err != nil {
			return events.APIGatewayProxyResponse{
				Headers:    RESP_HEADERS,
				StatusCode: 500,
			}, nil
		}
	}

	// 2) Update Engagement in DynamoDB
	err = svc.apprSvc.UpdateEngagementFeedLikes(inputBody.EngagementId, EntityId, authData.Username, companylib.EntityData{
		DisplayName: authData.DisplayName,
		ProfilePic:  companylib.GetDefaultThumbnailPicPath(authData.Username),
		Designation: authData.Designation,

		IsLike: inputBody.IsLike,
	})
	if err != nil {
		svc.logger.Printf("Unable to Update the Likes and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

// PostEngagementEventTeamsInput is the input structure for the Post Events in the Teams
type PostEngagementEventTeamsInput struct {
	EntityId string `json:"entityId"` // To Teams ID or User ID ( user ID functionality is not yet enabled for MVP)

	Skill     []string `json:"skill"`     // skill Name to be entered
	Value     []string `json:"value"`     // Value Name to be entered
	Milestone []string `json:"Milestone"` // Milestone Name to be entered
	Metrics   []string `json:"metrics"`   // Metric Name to be entered

	EventTitle string `json:"EventTitle"` // Event Title
	EventDesc  string `json:"EventDesc"`  // Message of either engagement/ Milestone

	FromDateTime string `json:"fromDateTime"` // From Date Time
	ToDateTime   string `json:"toDateTime"`   // To Date Time

	Location    string `json:"location"`    // Location of the event
	MeetingLink string `json:"meetingLink"` // Meeting Link of the event
}

// PostEngagementEventTeamsOutput is the output structure for the Post Events in the Teams
type PostEngagementEventTeamsOutput struct {
	EngagementId string `json:"engagementId"`
}

// PostEngagementsForEvents is the function to post the events in the Teams
func (svc *Service) PostEngagementsForEvents(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Get the Input Body
	var inputBody PostEngagementEventTeamsInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	// 1) Authorization at User Level and Check if user has the teams access
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	isAuth, err = svc.teamSvc.AuthorizerTeams(inputBody.EntityId, authData.Username)
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Create Engagement ID
	appId := "TEAMEVENT-" + utils.GenerateRandomString(12)

	// 2) Create Engagement in DynamoDB
	err = svc.apprSvc.CreateEngagement(companylib.TenantEngagementTable{
		EngagementId: appId,

		EntityId:   inputBody.EntityId,
		ProvidedBy: authData.Username,
		ProvidedByContent: companylib.EntityData{
			DisplayName: authData.DisplayName,
			ProfilePic:  companylib.GetDefaultThumbnailPicPath(authData.Username),
			Designation: authData.Designation,
		},
		Skill:     inputBody.Skill,
		Value:     inputBody.Value,
		Milestone: inputBody.Milestone,
		Metrics:   inputBody.Metrics,

		EventTitle:   inputBody.EventTitle,
		EventDesc:    inputBody.EventDesc,
		FromDateTime: inputBody.FromDateTime,
		ToDateTime:   inputBody.ToDateTime,
		Location:     inputBody.Location,
		MeetingLink:  inputBody.MeetingLink,

		Timestamp: utils.GenerateTimestamp(),
	})
	if err != nil {
		svc.logger.Printf("Unable to Create an Team Feed and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// 3) Return the Engagement ID
	resp := PostEngagementsForEntityOutput{
		EngagementId: appId,
	}

	// 4) Marshal the response
	responseBody, err := json.Marshal(resp)
	if err != nil {
		svc.logger.Printf("Error marshalling Engagement Data: %v\n", err)
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

// PostEngagementEventRSVPInput is the input structure for the Post RSVP in the Events
type PostEngagementEventRSVPInput struct {
	EntityId     string `json:"entityId"` // The Teams ID where the user is providing the RSVP.User needs to be part of the team.
	EngagementId string `json:"engagementId"`
	IsRSVP       bool   `json:"isRSVP"`
}

// PostEngagementEventRSVP is the function to post the RSVP in the Events
func (svc *Service) PostEngagementEventRSVP(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Get the Input Body
	var inputBody PostEngagementEventRSVPInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// 1) Authorization at User Level and Check if user has the teams access
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
	isAuth, err = svc.teamSvc.AuthorizerTeams(inputBody.EntityId, authData.Username)
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// 2) Update Engagement in DynamoDB
	err = svc.apprSvc.UpdateEngagementRSVP(inputBody.EngagementId, inputBody.EntityId, authData.Username, companylib.EntityData{
		DisplayName: authData.DisplayName,
		ProfilePic:  companylib.GetDefaultThumbnailPicPath(authData.Username),
		Designation: authData.Designation,

		IsRSVP: inputBody.IsRSVP,
	})
	if err != nil {
		svc.logger.Printf("Unable to Update the Events RSVP and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

func (svc *Service) PostEngagementSendKudos(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// Get the Input Body
	var inputBody PostEngagementsForEntityInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	// 1) Authorization at User Level and Check if user has to the Group
	authData, isAuth, err := svc.empSvc.Authorizer(request, "")
	if !isAuth || err != nil {
		svc.logger.Printf("Unable to Authorize the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	// Create Engagement ID
	appId := "USER-" + utils.GenerateRandomString(12)

	// 2) Upload Images to S3 and get the images keys
	imageMap, imageKeys := generateImageMap(inputBody.Images, appId)

	// 3) Upload the images to S3
	err = svc.contentSvc.UploadMultipleContentsToS3_Base64Content(imageMap)
	if err != nil {
		svc.logger.Printf("Error uploading images to S3: %v\n", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// 4) Create Engagement in DynamoDB
	err = svc.apprSvc.CreateEngagement(companylib.TenantEngagementTable{
		EngagementId: appId,

		EntityId:   inputBody.EntityId,
		ProvidedBy: authData.Username,
		ProvidedByContent: companylib.EntityData{
			DisplayName: authData.DisplayName,
			ProfilePic:  companylib.GetDefaultThumbnailPicPath(authData.Username),
			Designation: authData.Designation,
		},

		Message: inputBody.Message,

		Skill:     inputBody.Skill,
		Value:     inputBody.Value,
		Milestone: inputBody.Milestone,
		Metrics:   inputBody.Metrics,

		Images: imageKeys,

		TransferredPoints: inputBody.TransferrablePoints,

		Timestamp: utils.GenerateTimestamp(),
	})
	if err != nil {
		svc.logger.Printf("Unable to Create an Group Feed and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	// Send the message to the SQS queue to transfer the points. Which will invoke the step function to transfer the points.
	err = svc.SendKudosToSQS(inputBody.TransferrablePoints, companylib.RewardsTransferInput{
		TxId:                appId,
		SourceUserName:      authData.Username,
		DestinationUserName: inputBody.EntityId,
	})
	if err != nil {
		svc.logger.Printf("Unable to send the Kudos to SQS and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}

	resp := PostEngagementsForEntityOutput{
		EngagementId: appId,
	}

	responseBody, err := json.Marshal(resp)
	if err != nil {
		svc.logger.Printf("Error marshalling Engagement Data: %v\n", err)
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

// SendKudosToSQS is the function to send the Kudos to the SQS
func (svc *Service) SendKudosToSQS(txPoints map[string]int, rwdTxInput companylib.RewardsTransferInput) error {

	// For each element in txPoints, create a RewardsTransferInput and send it to the SQS
	for key, value := range txPoints {

		rwdTxSQSInput := companylib.RewardsTransferInput{
			TxId:                rwdTxInput.TxId + "-" + key,
			TxType:              companylib.TxType_TX_RP_USERS,
			SourceUserName:      rwdTxInput.SourceUserName,
			DestinationUserName: rwdTxInput.DestinationUserName,
			TransferPoints:      int32(value),
			RewardType:          key,
		}
		// Struct to String
		rwdTxInputBytes, _ := json.Marshal(rwdTxSQSInput)

		// Send the message to the SQS queue to transfer the points.
		_, err := svc.sqsClient.SendMessage(svc.ctx, &sqs.SendMessageInput{
			MessageBody: aws.String(string(rwdTxInputBytes)),
			QueueUrl:    aws.String(svc.REWARDS_TRANSFER_SQS_NAME),

			MessageDeduplicationId: aws.String(rwdTxInput.TxId),
			MessageGroupId:         aws.String(rwdTxInput.SourceUserName),
		})

		if err != nil {
			svc.logger.Printf("Error sending message to SQS: %v\n", err)
			return err
		}
	}

	return nil
}

const (
	DELETE_SKILLS_DATA     = "delete-skills"
	DELETE_VALUES_DATA     = "delete-values"
	DELETE_MILESTONES_DATA = "delete-milestones"
	DELETE_METRICS_DATA    = "delete-metrics"
)

func (svc *Service) DELETERequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	deleteType := request.Headers["delete_type"]

	// 1) Authorization at User Level and Check if user has the teams access
	_, isAuth, err := svc.empSvc.Authorizer(request, "AdminRole")
	if !isAuth || err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}

	switch deleteType {
	case DELETE_SKILLS_DATA:
		return svc.DeleteSkills(request)
	case DELETE_VALUES_DATA:
		return svc.DeleteValues(request)
	case DELETE_MILESTONES_DATA:
		return svc.DeleteMilestone(request)
	case DELETE_METRICS_DATA:
		return svc.DeleteMetrics(request)
	default:
		svc.logger.Printf("Incorrect Get Header type")
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, nil
	}
}

type DeleteSkillInput struct {
	SkillId string `json:"skillId"`
}

func (svc *Service) DeleteSkills(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var inputBody []DeleteSkillInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	var ddbInput []companylib.TenantSkillsTable

	for _, item := range inputBody {
		ddbInput = append(ddbInput, companylib.TenantSkillsTable{
			SkillId: item.SkillId,
		})
	}

	err = svc.apprSvc.PutSkillsToDynamoDB(ddbInput, "DELETE")
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

type DeleteValueInput struct {
	ValueId string `json:"valueId"`
}

func (svc *Service) DeleteValues(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var inputBody []DeleteValueInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	var ddbInput []companylib.TenantValuesTable

	for _, item := range inputBody {
		ddbInput = append(ddbInput, companylib.TenantValuesTable{
			ValueId: item.ValueId,
		})
	}

	err = svc.apprSvc.PutValuesToDynamoDB(ddbInput, "DELETE")
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

type DeleteMilestoneInput struct {
	MilestoneId string `json:"milestoneId"`
}

func (svc *Service) DeleteMilestone(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var inputBody []DeleteMilestoneInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	var ddbInput []companylib.TenantMilestonesTable

	for _, item := range inputBody {
		ddbInput = append(ddbInput, companylib.TenantMilestonesTable{
			MilestoneId: item.MilestoneId,
		})
	}

	err = svc.apprSvc.PutMilestonesToDynamoDB(ddbInput, "DELETE")
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}

type DeleteMetricsInput struct {
	MetricId string `json:"metricId"`
}

func (svc *Service) DeleteMetrics(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var inputBody []DeleteMetricsInput
	err := json.Unmarshal([]byte(request.Body), &inputBody)
	if err != nil {
		svc.logger.Printf("Unable to UnMarshall the request and failed with error :%v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
		}, err
	}

	var ddbInput []companylib.TenantMetricsTable

	for _, item := range inputBody {
		ddbInput = append(ddbInput, companylib.TenantMetricsTable{
			MetricId: item.MetricId,
		})
	}

	err = svc.apprSvc.PutMetricsToDynamoDB(ddbInput, "DELETE")
	if err != nil {
		return events.APIGatewayProxyResponse{
			Headers:    RESP_HEADERS,
			StatusCode: 500,
		}, err
	}
	return events.APIGatewayProxyResponse{
		Headers:    RESP_HEADERS,
		StatusCode: 200,
	}, nil
}
