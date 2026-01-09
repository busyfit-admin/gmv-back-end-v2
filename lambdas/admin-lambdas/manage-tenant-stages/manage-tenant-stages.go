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
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"
	adminlib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/admin-lib"
)

type Service struct {
	ctx    context.Context
	logger *log.Logger

	subStageSVC adminlib.TenantStageService
}

func main() {
	ctx, root := xray.BeginSegment(context.TODO(), "manage-sub-domains")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("Cannot load config: %v\n", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)

	subStageSVC := adminlib.CreateTenantStageService(ctx, ddbclient, logger)
	subStageSVC.TenantStagesTable = os.Getenv("TENANT_STAGES_TABLE")
	subStageSVC.TenantStages_TenantIdIndex = os.Getenv("TENANT_STAGES_INDEX_TENANTID")

	svc := Service{
		ctx:    ctx,
		logger: logger,

		subStageSVC: *subStageSVC,
	}

	lambda.Start(svc.handleAPIRequests)
}

type FollowUpDetails struct {
	Comment   string `json:"comment"`
	CommentBy string `json:"commentBy"`
	Timestamp string `json:"Timestamp"`
}

type StageData struct {
	OverallStatus   string                 `json:"OverallStatus"`
	CommentsCount   int                    `json:"CommentsCount"`
	FollowUpDetails []adminlib.CommentData `json:"FollowUpDetails"`
}

type GetTenantStagesResponse struct {
	InitialOnboarding     StageData `json:"InitialOnboarding"`
	OnboardingDemo        StageData `json:"OnboardingDemo"`
	TrialSetup            StageData `json:"TrialSetup"`
	TrialInProg           StageData `json:"TrialInProg"`
	Trail_Discontinued    StageData `json:"Trail_Discontinued"`
	PreProvisioningChecks StageData `json:"PreProvisioningChecks"`
	Provisioning          StageData `json:"Provisioning"`
	Active                StageData `json:"Active"`
	Inactive              StageData `json:"Inactive"`
	Deactivated           StageData `json:"Deactivated"`
}

func (svc *Service) handleAPIRequests(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	svc.ctx = ctx

	switch request.HTTPMethod {

	case "GET":
		apiRes, err := svc.GETRequestHandler(request)
		return apiRes, err
	case "POST":
		apiRes, err := svc.POSTRequestHandler(request)
		return apiRes, err

	default:
		return events.APIGatewayProxyResponse{StatusCode: 500}, fmt.Errorf("HTTP Method Not Support for this endpoint")

	}
}

/*
expected Input in the body section when POST is received:

	{
			TenantId: "abc-123",
			StageName: "Active"
			StageId : "", // to be set internal to lambda

			StageStatus : "Assigned",

			Comment : "comment-abc",
			CommentBy : "Name of person logged in "

	}
*/
func (svc *Service) POSTRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var reqBody adminlib.PostReqNewStageData

	err := json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		svc.logger.Printf("Unable to Unmarshal the request body. Req Body string: %v", request.Body)
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	SetStageIdFromStageName(&reqBody)

	err = svc.subStageSVC.AddNewStageData(reqBody)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "tenantid,Tenantid,TenantId,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		}}, nil
}

/*
TenantId is passed in the request Headers from the frontEnd.
*/
func (svc *Service) GETRequestHandler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	// 1. Get TenantId from the Header
	TenantId := request.Headers["tenantid"]

	// 2. Get all Tenant related Stages from the tenant stage table

	stageData, err := svc.subStageSVC.GetAllTenantStages(TenantId)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	// Map the stage Data to api response struct
	respData := MapStageDataToResponse(stageData)

	// set default status if Stage not defined
	SetOverallStatus(&respData)

	jsonData, err := json.Marshal(respData)
	if err != nil {
		return events.APIGatewayProxyResponse{StatusCode: 500}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(jsonData),
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "*",
			"Access-Control-Allow-Headers": "tenantid,Tenantid,X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
		},
	}, nil
}

// Map all stage data to response struct
func MapStageDataToResponse(allStageData []adminlib.TenantStages) GetTenantStagesResponse {

	var apiResp GetTenantStagesResponse
	for _, stageData := range allStageData {

		switch stageData.StageId {

		case adminlib.INITIAL_ONBOARDING_STAGE_ID:
			apiResp.InitialOnboarding = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}

		case adminlib.ONBOARDING_DEMO_STAGE_ID:
			apiResp.OnboardingDemo = StageData{
				OverallStatus:   stageData.StageStatus,
				FollowUpDetails: stageData.StageComments,
			}
		case adminlib.TRIAL_SETUP_STAGE_ID:
			apiResp.TrialSetup = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}
		case adminlib.TRIAL_IN_PROG_STAGE_ID:
			apiResp.TrialInProg = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}
		case adminlib.TRAIL_DISCONTINUED_STAGE_ID:
			apiResp.Trail_Discontinued = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}
		case adminlib.PRE_PROVISIONING_CHECKS_STAGE_ID:
			apiResp.PreProvisioningChecks = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}
		case adminlib.PROVISIONING_STAGE_ID:
			apiResp.Provisioning = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}
		case adminlib.ACTIVE_STAGE_ID:
			apiResp.Active = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}
		case adminlib.INACTIVE_STAGE_ID:
			apiResp.Inactive = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}
		case adminlib.DEACTIVATED_STAGE_ID:
			apiResp.Deactivated = StageData{
				OverallStatus:   stageData.StageStatus,
				CommentsCount:   stageData.CommentsCount,
				FollowUpDetails: stageData.StageComments,
			}
		}

	}

	return apiResp
}

func SetStageIdFromStageName(stageData *adminlib.PostReqNewStageData) {

	switch stageData.StageName {
	case adminlib.INITIAL_ONBOARDING:
		stageData.StageId = adminlib.INITIAL_ONBOARDING_STAGE_ID
	case adminlib.ONBOARDING_DEMO:
		stageData.StageId = adminlib.ONBOARDING_DEMO_STAGE_ID
	case adminlib.TRIAL_SETUP:
		stageData.StageId = adminlib.TRIAL_SETUP_STAGE_ID
	case adminlib.TRIAL_IN_PROG:
		stageData.StageId = adminlib.TRIAL_IN_PROG_STAGE_ID
	case adminlib.TRAIL_DISCONTINUED:
		stageData.StageId = adminlib.TRAIL_DISCONTINUED_STAGE_ID
	case adminlib.PRE_PROVISIONING_CHECKS:
		stageData.StageId = adminlib.PRE_PROVISIONING_CHECKS_STAGE_ID
	case adminlib.PROVISIONING:
		stageData.StageId = adminlib.PROVISIONING_STAGE_ID
	case adminlib.ACTIVE:
		stageData.StageId = adminlib.ACTIVE_STAGE_ID
	case adminlib.INACTIVE:
		stageData.StageId = adminlib.INACTIVE_STAGE_ID
	case adminlib.DEACTIVATED:
		stageData.StageId = adminlib.DEACTIVATED_STAGE_ID
	}

}

// Set Default status if stage data not found
func SetOverallStatus(response *GetTenantStagesResponse) {
	// Define a helper function to check and set OverallStatus
	checkAndSetStatus := func(stage *StageData) {
		if stage.OverallStatus == "" {
			stage.OverallStatus = adminlib.UNDEFINED
			stage.FollowUpDetails = []adminlib.CommentData{}
		}
	}

	// Check and set OverallStatus for each stage
	checkAndSetStatus(&response.InitialOnboarding)
	checkAndSetStatus(&response.OnboardingDemo)
	checkAndSetStatus(&response.TrialSetup)
	checkAndSetStatus(&response.TrialInProg)
	checkAndSetStatus(&response.Trail_Discontinued)
	checkAndSetStatus(&response.PreProvisioningChecks)
	checkAndSetStatus(&response.Provisioning)
	checkAndSetStatus(&response.Active)
	checkAndSetStatus(&response.Inactive)
	checkAndSetStatus(&response.Deactivated)
}
