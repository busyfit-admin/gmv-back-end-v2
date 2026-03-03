package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// ==================== Entry Point ====================

func (svc *Service) Handle(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Request: %s %s", request.HTTPMethod, request.Path)

	if request.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Headers: RESP_HEADERS, Body: ""}, nil
	}

	cognitoID, err := svc.getCognitoIDFromRequest(request)
	if err != nil {
		return svc.errResp(http.StatusUnauthorized, "UNAUTHORIZED", "Authorization token is missing or invalid")
	}

	employee, err := svc.empSVC.GetEmployeeDataByCognitoId(cognitoID)
	if err != nil {
		return svc.errResp(http.StatusUnauthorized, "UNAUTHORIZED", "User not found")
	}
	userName := employee.EmailID

	parts := splitPath(request.Path)
	if len(parts) < 2 || parts[0] != "v2" {
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Route not found")
	}

	// /v2/users/me/...
	if len(parts) >= 4 && parts[1] == "users" && parts[2] == "me" {
		return svc.handleMe(request, parts, userName, employee.FirstName+" "+employee.LastName)
	}

	// /v2/teams/{teamId}/members/directory
	if len(parts) == 5 && parts[1] == "teams" && parts[3] == "members" && parts[4] == "directory" {
		teamID := parts[2]
		if request.HTTPMethod == "GET" {
			return svc.getTeamMemberDirectory(teamID, userName)
		}
	}

	return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Route not found")
}

// handleMe dispatches /v2/users/me/{resource}/... routes.
func (svc *Service) handleMe(request events.APIGatewayProxyRequest, parts []string, userName, displayName string) (events.APIGatewayProxyResponse, error) {
	// All /v2/users/me/... endpoints are team-scoped.
	teamID := strings.TrimSpace(request.QueryStringParameters["teamId"])
	if teamID == "" {
		return svc.errResp(http.StatusBadRequest, "VALIDATION_ERROR", "teamId query parameter is required")
	}

	// parts: [v2, users, me, resource, ...]
	resource := parts[3]

	switch resource {
	case "goals":
		return svc.handleGoals(request, parts, userName, displayName, teamID)
	case "meetings":
		return svc.handleMeetings(request, parts, userName, teamID)
	case "appreciations":
		return svc.handleAppreciations(request, parts, userName, teamID)
	case "feedback-requests":
		return svc.handleFeedbackRequests(request, parts, userName, displayName, teamID)
	case "tasks":
		return svc.handleTasks(request, parts, userName, teamID)
	}

	return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Route not found")
}

// ==================== Response Helpers ====================

func (svc *Service) okResp(data interface{}) (events.APIGatewayProxyResponse, error) {
	return svc.statusResp(http.StatusOK, data)
}

func (svc *Service) createdResp(data interface{}) (events.APIGatewayProxyResponse, error) {
	return svc.statusResp(http.StatusCreated, data)
}

func (svc *Service) noContentResp() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
}

func (svc *Service) statusResp(status int, data interface{}) (events.APIGatewayProxyResponse, error) {
	body, err := json.Marshal(APIResponse{Data: data})
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to serialise response")
	}
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: RESP_HEADERS, Body: string(body)}, nil
}

func (svc *Service) errResp(statusCode int, code, message string) (events.APIGatewayProxyResponse, error) {
	body, _ := json.Marshal(APIResponse{Error: &ErrBody{Code: code, Message: message}})
	return events.APIGatewayProxyResponse{StatusCode: statusCode, Headers: RESP_HEADERS, Body: string(body)}, nil
}

// ==================== Path / Request Helpers ====================

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	raw := strings.Split(trimmed, "/")
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if v, err := url.PathUnescape(p); err == nil {
			out = append(out, v)
		} else {
			out = append(out, p)
		}
	}
	return out
}

func (svc *Service) getCognitoIDFromRequest(request events.APIGatewayProxyRequest) (string, error) {
	if claims, ok := request.RequestContext.Authorizer["claims"].(map[string]interface{}); ok {
		if sub, ok := claims["sub"].(string); ok && sub != "" {
			return sub, nil
		}
	}
	if id := request.Headers["X-Cognito-Id"]; id != "" {
		return id, nil
	}
	return "", fmt.Errorf("cognito ID not found")
}

func parseBody[T any](body string) (*T, error) {
	if strings.TrimSpace(body) == "" {
		return new(T), nil
	}
	var out T
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func queryString(params map[string]string, key string) string {
	return strings.TrimSpace(params[key])
}

// initials returns up to two uppercase initials from a display name.
func initials(name string) string {
	parts := strings.Fields(name)
	out := ""
	for _, p := range parts {
		if len(p) > 0 {
			out += strings.ToUpper(string(p[0]))
		}
		if len(out) == 2 {
			break
		}
	}
	return out
}
