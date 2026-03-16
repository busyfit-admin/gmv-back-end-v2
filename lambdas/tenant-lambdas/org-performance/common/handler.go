package common

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-xray-sdk-go/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/xray"

	companylib "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/company-lib"
)

type Service struct {
	ctx          context.Context
	logger       *log.Logger
	orgSVC       *companylib.OrgServiceV2
	empSVC       *companylib.EmployeeService
	perfSVC      *companylib.PerformanceService
	ddb          *dynamodb.Client
	perfHubTable string
}

const (
	RouteGroupAll    = ""
	RouteGroupCycles = "cycles"
	RouteGroupKPIs   = "kpis"
	RouteGroupOKRs   = "okrs"
	RouteGroupGoals  = "goals"
)

var RESP_HEADERS = companylib.GetHeadersForAPI("OrganizationAPI")

func NewService() (*Service, error) {
	ctx, root := xray.BeginSegment(context.TODO(), "manage-org-performance")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load config: %w", err)
	}

	awsv2.AWSV2Instrumentor(&cfg.APIOptions)

	logger := log.New(os.Stdout, "", log.LstdFlags)
	ddbclient := dynamodb.NewFromConfig(cfg)
	sesClient := ses.NewFromConfig(cfg)

	empSvc := companylib.CreateEmployeeService(ctx, ddbclient, nil, logger)
	empSvc.EmployeeTable = os.Getenv("EMPLOYEE_TABLE")
	empSvc.EmployeeTable_CognitoId_Index = os.Getenv("EMPLOYEE_TABLE_COGNITO_ID_INDEX")

	emailSvc := companylib.CreateEmailService(ctx, sesClient, logger)
	orgSvc := companylib.CreateOrgServiceV2(ctx, ddbclient, logger, empSvc, emailSvc)
	orgSvc.OrganizationTable = os.Getenv("ORGANIZATION_TABLE")
	orgSvc.PromoCodesTable = os.Getenv("PROMO_CODES_TABLE")

	perfSvc := companylib.CreatePerformanceService(ctx, ddbclient, logger)
	perfSvc.OrgPerformanceTable = os.Getenv("ORG_PERFORMANCE_TABLE")
	perfSvc.OrganizationTable = os.Getenv("ORGANIZATION_TABLE")

	svc := &Service{
		ctx:          ctx,
		logger:       logger,
		orgSVC:       orgSvc,
		empSVC:       empSvc,
		perfSVC:      perfSvc,
		ddb:          ddbclient,
		perfHubTable: os.Getenv("PERF_HUB_TABLE"),
	}

	return svc, nil
}

func (svc *Service) Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return svc.HandleWithGroup(request, RouteGroupAll)
}

func (svc *Service) HandleWithGroup(request events.APIGatewayProxyRequest, routeGroup string) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Received request: %s %s", request.HTTPMethod, request.Path)

	if request.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Headers: RESP_HEADERS, Body: ""}, nil
	}

	cognitoId, err := svc.getCognitoIdFromRequest(request)
	if err != nil {
		return svc.errorResponse(http.StatusUnauthorized, "Unauthorized", err)
	}

	employee, err := svc.empSVC.GetEmployeeDataByCognitoId(cognitoId)
	if err != nil {
		return svc.errorResponse(http.StatusUnauthorized, "User not found", err)
	}
	userName := employee.EmailID

	parts := splitPath(request.Path)
	if len(parts) < 2 || parts[0] != "v2" {
		return svc.errorResponse(http.StatusNotFound, "Route not found", nil)
	}
	if !isPathHandledByGroup(parts, routeGroup) {
		return svc.errorResponse(http.StatusNotFound, "Route not found", nil)
	}

	if len(parts) == 4 && parts[1] == "organizations" && parts[3] == "performance-cycles" {
		orgID := parts[2]
		if err := svc.ensureOrgAdmin(orgID, userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			includeQuarters := queryBool(request.QueryStringParameters, "includeQuarters", false)
			includeKPIs := queryBool(request.QueryStringParameters, "includeKPIs", false)
			includeOKRs := queryBool(request.QueryStringParameters, "includeOKRs", false)
			options := getListOptions(request.QueryStringParameters)
			filters := map[string]string{
				"status":     request.QueryStringParameters["status"],
				"fiscalYear": request.QueryStringParameters["fiscalYear"],
			}
			res, err := svc.perfSVC.ListPerformanceCycles(orgID, filters, options, includeQuarters, includeKPIs, includeOKRs)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list performance cycles", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.CreatePerformanceCycle(orgID, input)
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to create performance cycle", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	if len(parts) == 3 && parts[1] == "performance-cycles" {
		cycleID := parts[2]
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.GetPerformanceCycleDetails(
				cycleID,
				queryBool(request.QueryStringParameters, "includeQuarters", true),
				queryBool(request.QueryStringParameters, "includeKPIs", true),
				queryBool(request.QueryStringParameters, "includeOKRs", true),
				queryBool(request.QueryStringParameters, "includeAnalytics", false),
			)
			if err != nil {
				return svc.errorResponse(http.StatusNotFound, "Performance cycle not found", err)
			}
			if err := svc.ensureOrgAdmin(toString(res["organizationId"]), userName); err != nil {
				return svc.errorResponse(http.StatusForbidden, "Access denied", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "PATCH":
			base, err := svc.perfSVC.GetPerformanceCycleDetails(cycleID, false, false, false, false)
			if err != nil {
				return svc.errorResponse(http.StatusNotFound, "Performance cycle not found", err)
			}
			if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
				return svc.errorResponse(http.StatusForbidden, "Access denied", err)
			}
			patch, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.UpdatePerformanceCycle(cycleID, patch)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to update performance cycle", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "DELETE":
			base, err := svc.perfSVC.GetPerformanceCycleDetails(cycleID, false, false, false, false)
			if err != nil {
				return svc.errorResponse(http.StatusNotFound, "Performance cycle not found", err)
			}
			if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
				return svc.errorResponse(http.StatusForbidden, "Access denied", err)
			}
			if err := svc.perfSVC.DeletePerformanceCycle(cycleID); err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to delete performance cycle", err)
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
		}
	}

	if len(parts) == 4 && parts[1] == "performance-cycles" && parts[3] == "quarters" {
		cycleID := parts[2]
		cycle, err := svc.perfSVC.GetPerformanceCycleDetails(cycleID, false, false, false, false)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Performance cycle not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(cycle["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.ListQuarters(cycleID)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list quarters", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.CreateQuarter(cycleID, input)
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to create quarter", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	if len(parts) == 3 && parts[1] == "quarters" {
		quarterID := parts[2]
		quarter, err := svc.perfSVC.GetQuarterDetails(quarterID, false, false, false, false)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Quarter not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(quarter["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.GetQuarterDetails(
				quarterID,
				queryBool(request.QueryStringParameters, "includeKPIs", false),
				queryBool(request.QueryStringParameters, "includeOKRs", false),
				queryBool(request.QueryStringParameters, "includeMeetingNotes", false),
				queryBool(request.QueryStringParameters, "includePendingReviews", false),
			)
			if err != nil {
				return svc.errorResponse(http.StatusNotFound, "Quarter not found", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "PATCH":
			patch, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.UpdateQuarter(quarterID, patch)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to update quarter", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "DELETE":
			if err := svc.perfSVC.DeleteQuarter(quarterID); err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to delete quarter", err)
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
		}
	}

	if len(parts) == 2 && parts[1] == "kpis" {
		orgID := svc.getOrgIDFromHeaders(request)
		if orgID == "" {
			return svc.errorResponse(http.StatusBadRequest, "Organization-Id header is required", nil)
		}
		if err := svc.ensureOrgAdmin(orgID, userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			filters := map[string]string{
				"cycleId":     request.QueryStringParameters["cycleId"],
				"quarterId":   request.QueryStringParameters["quarterId"],
				"department":  request.QueryStringParameters["department"],
				"owner":       request.QueryStringParameters["owner"],
				"status":      request.QueryStringParameters["status"],
				"parentKpiId": request.QueryStringParameters["parentKpiId"],
			}
			res, err := svc.perfSVC.ListKPIs(orgID, filters, getListOptions(request.QueryStringParameters), queryBool(request.QueryStringParameters, "includeSubKPIs", false))
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list KPIs", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.CreateKPI(input, "")
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to create KPI", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	if len(parts) == 3 && parts[1] == "kpis" {
		kpiID := parts[2]
		kpi, err := svc.perfSVC.GetKPIDetails(kpiID, false, false)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "KPI not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(kpi["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.GetKPIDetails(
				kpiID,
				queryBool(request.QueryStringParameters, "includeSubKPIs", false),
				queryBool(request.QueryStringParameters, "includeValueHistory", false),
			)
			if err != nil {
				return svc.errorResponse(http.StatusNotFound, "KPI not found", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "PATCH":
			patch, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.UpdateKPI(kpiID, patch)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to update KPI", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "DELETE":
			deleteSubs := queryBool(request.QueryStringParameters, "deleteSubKPIs", false)
			if err := svc.perfSVC.DeleteKPI(kpiID, deleteSubs); err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to delete KPI", err)
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
		}
	}

	if len(parts) == 4 && parts[1] == "kpis" && parts[3] == "sub-kpis" && request.HTTPMethod == "POST" {
		parentKPIID := parts[2]
		parent, err := svc.perfSVC.GetKPIDetails(parentKPIID, false, false)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Parent KPI not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(parent["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		input, err := parseBody(request.Body)
		if err != nil {
			return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
		}
		res, err := svc.perfSVC.CreateKPI(input, parentKPIID)
		if err != nil {
			return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to create sub-KPI", err)
		}
		return svc.successResponse(http.StatusCreated, res)
	}

	if len(parts) == 4 && parts[1] == "kpis" && parts[3] == "values" && request.HTTPMethod == "POST" {
		kpiID := parts[2]
		kpi, err := svc.perfSVC.GetKPIDetails(kpiID, false, false)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "KPI not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(kpi["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		input, err := parseBody(request.Body)
		if err != nil {
			return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
		}
		res, err := svc.perfSVC.AddKPIValue(kpiID, input, userName)
		if err != nil {
			return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to add KPI value", err)
		}
		return svc.successResponse(http.StatusCreated, res)
	}

	if len(parts) == 2 && parts[1] == "okrs" {
		orgID := svc.getOrgIDFromHeaders(request)
		if orgID == "" {
			return svc.errorResponse(http.StatusBadRequest, "Organization-Id header is required", nil)
		}
		if err := svc.ensureOrgAdmin(orgID, userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			filters := map[string]string{
				"cycleId":   request.QueryStringParameters["cycleId"],
				"quarterId": request.QueryStringParameters["quarterId"],
				"owner":     request.QueryStringParameters["owner"],
				"status":    request.QueryStringParameters["status"],
			}
			res, err := svc.perfSVC.ListOKRs(orgID, filters, getListOptions(request.QueryStringParameters), queryBool(request.QueryStringParameters, "includeKeyResults", false))
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list OKRs", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.CreateOKR(input)
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to create OKR", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	if len(parts) == 3 && parts[1] == "okrs" {
		okrID := parts[2]
		okr, err := svc.perfSVC.GetOKRDetails(okrID, false, false)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "OKR not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(okr["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.GetOKRDetails(
				okrID,
				queryBool(request.QueryStringParameters, "includeKeyResults", true),
				queryBool(request.QueryStringParameters, "includeProgressHistory", false),
			)
			if err != nil {
				return svc.errorResponse(http.StatusNotFound, "OKR not found", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "PATCH":
			patch, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.UpdateOKR(okrID, patch)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to update OKR", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "DELETE":
			if err := svc.perfSVC.DeleteOKR(okrID); err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to delete OKR", err)
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
		}
	}

	if len(parts) == 3 && parts[1] == "key-results" && request.HTTPMethod == "PATCH" {
		keyResultID := parts[2]
		patch, err := parseBody(request.Body)
		if err != nil {
			return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
		}
		res, err := svc.perfSVC.UpdateKeyResult(keyResultID, patch)
		if err != nil {
			return svc.errorResponse(http.StatusInternalServerError, "Failed to update key result", err)
		}
		if err := svc.ensureOrgAdmin(toString(res["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		return svc.successResponse(http.StatusOK, res)
	}

	if len(parts) == 4 && parts[1] == "quarters" && parts[3] == "meeting-notes" {
		quarterID := parts[2]
		quarter, err := svc.perfSVC.GetQuarterDetails(quarterID, false, false, false, false)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Quarter not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(quarter["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.ListMeetingNotes(quarterID, request.QueryStringParameters["sortBy"], request.QueryStringParameters["order"])
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list meeting notes", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.CreateMeetingNote(quarterID, input)
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to create meeting note", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	if len(parts) == 3 && parts[1] == "meeting-notes" {
		noteID := parts[2]
		switch request.HTTPMethod {
		case "PATCH":
			patch, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.UpdateMeetingNote(noteID, patch)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to update meeting note", err)
			}
			if err := svc.ensureOrgAdmin(toString(res["organizationId"]), userName); err != nil {
				return svc.errorResponse(http.StatusForbidden, "Access denied", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "DELETE":
			if err := svc.perfSVC.DeleteMeetingNote(noteID); err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to delete meeting note", err)
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
		}
	}

	if len(parts) == 4 && parts[1] == "performance-cycles" && parts[3] == "analytics" && request.HTTPMethod == "GET" {
		cycleID := parts[2]
		res, err := svc.perfSVC.GetCycleAnalytics(cycleID)
		if err != nil {
			return svc.errorResponse(http.StatusInternalServerError, "Failed to get cycle analytics", err)
		}
		if err := svc.ensureOrgAdmin(toString(res["organizationId"]), userName); err != nil {
			// fallback for analytics payload if organizationId is nested
			summaryOrg := svc.getOrgIDFromHeaders(request)
			if summaryOrg == "" || svc.ensureOrgAdmin(summaryOrg, userName) != nil {
				return svc.errorResponse(http.StatusForbidden, "Access denied", err)
			}
		}
		return svc.successResponse(http.StatusOK, res)
	}

	if len(parts) == 4 && parts[1] == "quarters" && parts[3] == "analytics" && request.HTTPMethod == "GET" {
		quarterID := parts[2]
		quarter, err := svc.perfSVC.GetQuarterDetails(quarterID, false, false, false, false)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Quarter not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(quarter["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		res, err := svc.perfSVC.GetQuarterAnalytics(quarterID)
		if err != nil {
			return svc.errorResponse(http.StatusInternalServerError, "Failed to get quarter analytics", err)
		}
		return svc.successResponse(http.StatusOK, res)
	}

	if len(parts) == 3 && parts[1] == "goals" {
		goalID := parts[2]
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.GetGoalDetails(
				goalID,
				queryBool(request.QueryStringParameters, "includeValueHistory", false),
				queryBool(request.QueryStringParameters, "includeTaggedTeams", false),
				queryBool(request.QueryStringParameters, "includeSubItems", false),
				queryBool(request.QueryStringParameters, "includeLadderUp", false),
				queryBool(request.QueryStringParameters, "includePrivateTasks", false),
				userName,
			)
			if err != nil {
				return svc.errorResponse(http.StatusNotFound, "Goal not found", err)
			}
			if err := svc.ensureOrgAdmin(toString(res["organizationId"]), userName); err != nil {
				return svc.errorResponse(http.StatusForbidden, "Access denied", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "PATCH":
			base, err := svc.perfSVC.GetGoalDetails(goalID, false, false, false, false, false, userName)
			if err != nil {
				return svc.errorResponse(http.StatusNotFound, "Goal not found", err)
			}
			if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
				return svc.errorResponse(http.StatusForbidden, "Access denied", err)
			}
			patch, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.UpdateGoal(goalID, patch)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to update goal", err)
			}
			return svc.successResponse(http.StatusOK, res)
		}
	}

	if len(parts) == 4 && parts[1] == "goals" && parts[3] == "value-history" {
		goalID := parts[2]
		base, err := svc.perfSVC.GetGoalDetails(goalID, false, false, false, false, false, userName)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Goal not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			filters := map[string]string{
				"startDate": request.QueryStringParameters["startDate"],
				"endDate":   request.QueryStringParameters["endDate"],
			}
			res, err := svc.perfSVC.GetGoalValueHistory(goalID, filters, getListOptions(request.QueryStringParameters))
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list value history", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.AddGoalValueEntry(goalID, input, userName)
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to add value entry", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	if len(parts) == 4 && parts[1] == "goals" && parts[3] == "teams" {
		goalID := parts[2]
		base, err := svc.perfSVC.GetGoalDetails(goalID, false, false, false, false, false, userName)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Goal not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.GetGoalTeams(goalID)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list tagged teams", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.TagTeamToGoal(goalID, input, userName)
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to tag team", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	if len(parts) == 5 && parts[1] == "goals" && parts[3] == "teams" && request.HTTPMethod == "DELETE" {
		goalID := parts[2]
		teamID := parts[4]
		base, err := svc.perfSVC.GetGoalDetails(goalID, false, false, false, false, false, userName)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Goal not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		if err := svc.perfSVC.RemoveGoalTeam(goalID, teamID); err != nil {
			return svc.errorResponse(http.StatusInternalServerError, "Failed to remove team tag", err)
		}
		return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
	}

	if len(parts) == 4 && parts[1] == "goals" && parts[3] == "sub-items" {
		goalID := parts[2]
		base, err := svc.perfSVC.GetGoalDetails(goalID, false, false, false, false, false, userName)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Goal not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.GetGoalSubItems(goalID)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list sub-items", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.AddGoalSubItem(goalID, input)
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to add sub-item", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	if len(parts) == 3 && parts[1] == "sub-items" {
		subItemID := parts[2]
		switch request.HTTPMethod {
		case "PATCH":
			patch, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.UpdateSubItem(subItemID, patch)
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to update sub-item", err)
			}
			if err := svc.ensureOrgAdmin(toString(res["organizationId"]), userName); err != nil {
				return svc.errorResponse(http.StatusForbidden, "Access denied", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "DELETE":
			if err := svc.perfSVC.DeleteSubItem(subItemID); err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to delete sub-item", err)
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
		}
	}

	if len(parts) == 4 && parts[1] == "goals" && parts[3] == "ladder-up" && request.HTTPMethod == "GET" {
		goalID := parts[2]
		base, err := svc.perfSVC.GetGoalDetails(goalID, false, false, false, false, false, userName)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Goal not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		res, err := svc.perfSVC.GetGoalLadderUp(goalID, request.QueryStringParameters["status"])
		if err != nil {
			return svc.errorResponse(http.StatusInternalServerError, "Failed to list ladder-up items", err)
		}
		return svc.successResponse(http.StatusOK, res)
	}

	if len(parts) == 4 && parts[1] == "ladder-up" && request.HTTPMethod == "PATCH" {
		ladderID := parts[2]
		action := parts[3]
		patch, err := parseBody(request.Body)
		if err != nil {
			return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
		}
		var res map[string]interface{}
		switch action {
		case "approve":
			res, err = svc.perfSVC.ApproveLadderUp(ladderID, patch)
		case "reject":
			res, err = svc.perfSVC.RejectLadderUp(ladderID, patch)
		default:
			return svc.errorResponse(http.StatusNotFound, "Route not found", nil)
		}
		if err != nil {
			return svc.errorResponse(http.StatusInternalServerError, "Failed to update ladder-up item", err)
		}
		if err := svc.ensureOrgAdmin(toString(res["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		return svc.successResponse(http.StatusOK, res)
	}

	if len(parts) == 4 && parts[1] == "goals" && parts[3] == "tasks" {
		goalID := parts[2]
		switch request.HTTPMethod {
		case "GET":
			res, err := svc.perfSVC.GetGoalTasks(goalID, userName, map[string]string{"status": request.QueryStringParameters["status"]}, getListOptions(request.QueryStringParameters))
			if err != nil {
				return svc.errorResponse(http.StatusInternalServerError, "Failed to list tasks", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "POST":
			input, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.CreateGoalTask(goalID, userName, input)
			if err != nil {
				return svc.errorResponse(http.StatusUnprocessableEntity, "Failed to create task", err)
			}
			return svc.successResponse(http.StatusCreated, res)
		}
	}

	// GET /v2/teams/{teamId}/goals — list all OKRs & KPIs tagged to a team.
	// Requires the Organization-Id header; caller must be an org admin.
	// Query params: type ("kpi"|"okr"), cycleId, status, page, pageSize, sortBy, order.
	if len(parts) == 4 && parts[1] == "teams" && parts[3] == "goals" && request.HTTPMethod == "GET" {
		teamID := parts[2]
		orgID := svc.getOrgIDFromHeaders(request)
		if err := svc.ensureOrgAdmin(orgID, userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		goalType := request.QueryStringParameters["type"]
		filters := map[string]string{
			"cycleId": request.QueryStringParameters["cycleId"],
			"status":  request.QueryStringParameters["status"],
		}
		res, err := svc.perfSVC.GetTeamGoals(teamID, orgID, goalType, filters, getListOptions(request.QueryStringParameters))
		if err != nil {
			return svc.errorResponse(http.StatusInternalServerError, "Failed to list team goals", err)
		}
		return svc.successResponse(http.StatusOK, res)
	}

	if len(parts) == 5 && parts[1] == "goals" && parts[3] == "tasks" {
		goalID := parts[2]
		taskID := parts[4]
		switch request.HTTPMethod {
		case "PATCH":
			patch, err := parseBody(request.Body)
			if err != nil {
				return svc.errorResponse(http.StatusBadRequest, "Invalid request body", err)
			}
			res, err := svc.perfSVC.UpdateGoalTask(goalID, taskID, userName, patch)
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "forbidden") {
					return svc.errorResponse(http.StatusForbidden, "Access denied", err)
				}
				return svc.errorResponse(http.StatusInternalServerError, "Failed to update task", err)
			}
			return svc.successResponse(http.StatusOK, res)
		case "DELETE":
			if err := svc.perfSVC.DeleteGoalTask(goalID, taskID, userName); err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "forbidden") {
					return svc.errorResponse(http.StatusForbidden, "Access denied", err)
				}
				return svc.errorResponse(http.StatusInternalServerError, "Failed to delete task", err)
			}
			return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
		}
	}

	// GET /v2/goals/{goalId}/user-goals — list all user-level individual goals linked to this org goal (OKR/KPI).
	// Returns each goal's progress/status and a rolled-up summary count per status.
	if len(parts) == 4 && parts[1] == "goals" && parts[3] == "user-goals" && request.HTTPMethod == "GET" {
		goalID := parts[2]
		base, err := svc.perfSVC.GetGoalDetails(goalID, false, false, false, false, false, userName)
		if err != nil {
			return svc.errorResponse(http.StatusNotFound, "Goal not found", err)
		}
		if err := svc.ensureOrgAdmin(toString(base["organizationId"]), userName); err != nil {
			return svc.errorResponse(http.StatusForbidden, "Access denied", err)
		}
		statusFilter := request.QueryStringParameters["status"]
		res, err := svc.listUserGoalsForOrgGoal(goalID, statusFilter)
		if err != nil {
			return svc.errorResponse(http.StatusInternalServerError, "Failed to list user goals", err)
		}
		return svc.successResponse(http.StatusOK, res)
	}

	return svc.errorResponse(http.StatusMethodNotAllowed, "Method not allowed", nil)
}

// userGoalItem is a minimal projection of a GoalRecord from UserPerformanceHubTable.
type userGoalItem struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	GoalID    string `dynamodbav:"goalId"`
	OrgGoalID string `dynamodbav:"orgGoalId"`
	UserName  string `dynamodbav:"userName"`
	Title     string `dynamodbav:"title"`
	Type      string `dynamodbav:"type"`
	Progress  int    `dynamodbav:"progress"`
	Status    string `dynamodbav:"status"`
	DueDate   string `dynamodbav:"dueDate"`
	UpdatedAt string `dynamodbav:"updatedAt"`
}

// userTaskItem is a minimal projection of a LinkedTaskRecord from UserPerformanceHubTable.
type userTaskItem struct {
	SK          string  `dynamodbav:"SK"`
	TaskID      string  `dynamodbav:"taskId"`
	TaskNumber  int     `dynamodbav:"taskNumber"`
	GoalID      string  `dynamodbav:"goalId"`
	Title       string  `dynamodbav:"title"`
	Description string  `dynamodbav:"description"`
	Priority    string  `dynamodbav:"priority"`
	Status      string  `dynamodbav:"status"`
	Done        bool    `dynamodbav:"done"`
	DueDate     string  `dynamodbav:"dueDate"`
	TimeHours   float64 `dynamodbav:"timeHours"`
	TimeDays    float64 `dynamodbav:"timeDays"`
	UpdatedAt   string  `dynamodbav:"updatedAt"`
}

// fetchTasksForGoal queries the base table for all TASK# items on the given PK that
// belong to the specified goalID. Returns a slice ready for JSON serialisation.
func (svc *Service) fetchTasksForGoal(pk, goalID string) ([]map[string]interface{}, error) {
	out, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :prefix)"),
		FilterExpression:       aws.String("goalId = :goalId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":     &types.AttributeValueMemberS{Value: pk},
			":prefix": &types.AttributeValueMemberS{Value: "TASK#"},
			":goalId": &types.AttributeValueMemberS{Value: goalID},
		},
	})
	if err != nil {
		return nil, err
	}

	tasks := make([]map[string]interface{}, 0, len(out.Items))
	for _, item := range out.Items {
		var t userTaskItem
		if err := attributevalue.UnmarshalMap(item, &t); err != nil {
			continue
		}
		taskEntry := map[string]interface{}{
			"id":       t.TaskID,
			"taskId":   t.TaskID,
			"title":    t.Title,
			"status":   t.Status,
			"done":     t.Done,
			"priority": t.Priority,
			"dueDate":  t.DueDate,
		}
		if t.TaskNumber > 0 {
			taskEntry["taskNumber"] = t.TaskNumber
		}
		if t.Description != "" {
			taskEntry["description"] = t.Description
		}
		if t.TimeHours > 0 {
			taskEntry["timeHours"] = t.TimeHours
		}
		if t.TimeDays > 0 {
			taskEntry["timeDays"] = t.TimeDays
		}
		if t.UpdatedAt != "" {
			taskEntry["updatedAt"] = t.UpdatedAt
		}
		tasks = append(tasks, taskEntry)
	}
	return tasks, nil
}

// listUserGoalsForOrgGoal queries the OrgGoalIdIndex GSI on UserPerformanceHubTable
// to find all user-level goals linked to the given org goal ID.
// It returns the full list (including linked tasks per goal) plus a rolled-up status summary.
func (svc *Service) listUserGoalsForOrgGoal(orgGoalID, statusFilter string) (map[string]interface{}, error) {
	if svc.perfHubTable == "" {
		return nil, fmt.Errorf("PERF_HUB_TABLE is not configured")
	}

	out, err := svc.ddb.Query(svc.ctx, &dynamodb.QueryInput{
		TableName:              aws.String(svc.perfHubTable),
		IndexName:              aws.String("OrgGoalIdIndex"),
		KeyConditionExpression: aws.String("orgGoalId = :orgGoalId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":orgGoalId": &types.AttributeValueMemberS{Value: orgGoalID},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("GSI query failed: %w", err)
	}

	summary := map[string]int{
		"total":     0,
		"onTrack":   0,
		"ahead":     0,
		"atRisk":    0,
		"behind":    0,
		"completed": 0,
	}

	goals := make([]map[string]interface{}, 0, len(out.Items))
	for _, item := range out.Items {
		var g userGoalItem
		if err := attributevalue.UnmarshalMap(item, &g); err != nil {
			svc.logger.Printf("warn: failed to unmarshal user goal item: %v", err)
			continue
		}
		// Only include items that are goal records (SK starts with GOAL# but not a comment)
		if !strings.HasPrefix(g.SK, "GOAL#") || strings.Contains(g.SK, "#CMMNT#") {
			continue
		}
		// Optional status filter
		if statusFilter != "" && !strings.EqualFold(g.Status, statusFilter) {
			continue
		}
		// Parse teamId from PK: USER#{userName}#TEAM#{teamId}
		teamID := ""
		if idx := strings.LastIndex(g.PK, "#TEAM#"); idx != -1 {
			teamID = g.PK[idx+6:]
		}
		// Fetch tasks linked to this goal
		tasks, err := svc.fetchTasksForGoal(g.PK, g.GoalID)
		if err != nil {
			svc.logger.Printf("warn: failed to fetch tasks for goal %s: %v", g.GoalID, err)
			tasks = []map[string]interface{}{}
		}
		// Build summary counts
		summary["total"]++
		switch strings.ToLower(g.Status) {
		case "on-track":
			summary["onTrack"]++
		case "ahead":
			summary["ahead"]++
		case "at-risk":
			summary["atRisk"]++
		case "behind":
			summary["behind"]++
		case "completed":
			summary["completed"]++
		}
		goals = append(goals, map[string]interface{}{
			"goalId":      g.GoalID,
			"userName":    g.UserName,
			"teamId":      teamID,
			"title":       g.Title,
			"type":        g.Type,
			"progress":    g.Progress,
			"status":      g.Status,
			"dueDate":     g.DueDate,
			"updatedAt":   g.UpdatedAt,
			"linkedTasks": tasks,
		})
	}

	return map[string]interface{}{
		"orgGoalId": orgGoalID,
		"userGoals": goals,
		"summary":   summary,
	}, nil
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	parts := strings.Split(trimmed, "/")
	decoded := make([]string, 0, len(parts))
	for _, part := range parts {
		value, err := url.PathUnescape(part)
		if err != nil {
			decoded = append(decoded, part)
			continue
		}
		decoded = append(decoded, value)
	}
	return decoded
}

func isPathHandledByGroup(parts []string, routeGroup string) bool {
	if routeGroup == RouteGroupAll {
		return true
	}
	if len(parts) < 2 || parts[0] != "v2" {
		return false
	}
	resource := parts[1]
	switch routeGroup {
	case RouteGroupCycles:
		if resource == "organizations" {
			return len(parts) >= 4 && parts[3] == "performance-cycles"
		}
		return resource == "performance-cycles" || resource == "quarters" || resource == "meeting-notes"
	case RouteGroupKPIs:
		return resource == "kpis"
	case RouteGroupOKRs:
		return resource == "okrs" || resource == "key-results"
	case RouteGroupGoals:
		if resource == "teams" {
			// /v2/teams/{teamId}/goals
			return len(parts) == 4 && parts[3] == "goals"
		}
		return resource == "goals" || resource == "sub-items" || resource == "ladder-up"
	default:
		return false
	}
}

func queryBool(query map[string]string, key string, defaultValue bool) bool {
	raw := strings.TrimSpace(strings.ToLower(query[key]))
	if raw == "" {
		return defaultValue
	}
	return raw == "true" || raw == "1" || raw == "yes"
}

func getListOptions(query map[string]string) companylib.ListQueryOptions {
	page, _ := strconv.Atoi(query["page"])
	pageSize, _ := strconv.Atoi(query["pageSize"])
	return companylib.ListQueryOptions{
		Page:     page,
		PageSize: pageSize,
		SortBy:   query["sortBy"],
		Order:    query["order"],
	}
}

func parseBody(body string) (map[string]interface{}, error) {
	if strings.TrimSpace(body) == "" {
		return map[string]interface{}{}, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (svc *Service) getOrgIDFromHeaders(request events.APIGatewayProxyRequest) string {
	orgID := request.Headers["organization-id"]
	if orgID == "" {
		orgID = request.Headers["Organization-Id"]
	}
	return orgID
}

func (svc *Service) ensureOrgAdmin(orgID string, userName string) error {
	if orgID == "" {
		return fmt.Errorf("organization ID is required")
	}
	isAdmin, err := svc.orgSVC.IsOrgAdmin(orgID, userName)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user is not an organization admin")
	}
	return nil
}

func (svc *Service) getCognitoIdFromRequest(request events.APIGatewayProxyRequest) (string, error) {
	if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		if sub, ok := claims["sub"].(string); ok && sub != "" {
			return sub, nil
		}
	}
	if cognitoID := request.Headers["X-Cognito-Id"]; cognitoID != "" {
		return cognitoID, nil
	}
	return "", fmt.Errorf("cognito ID not found in request")
}

func (svc *Service) successResponse(statusCode int, payload interface{}) (events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return svc.errorResponse(http.StatusInternalServerError, "Failed to marshal response", err)
	}
	return events.APIGatewayProxyResponse{StatusCode: statusCode, Headers: RESP_HEADERS, Body: string(body)}, nil
}

func (svc *Service) errorResponse(statusCode int, message string, err error) (events.APIGatewayProxyResponse, error) {
	errorMsg := message
	if err != nil {
		errorMsg = fmt.Sprintf("%s: %v", message, err)
	}
	body, _ := json.Marshal(map[string]string{"error": message, "message": errorMsg})
	return events.APIGatewayProxyResponse{StatusCode: statusCode, Headers: RESP_HEADERS, Body: string(body)}, nil
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
