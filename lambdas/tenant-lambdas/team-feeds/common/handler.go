package common

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// ==================== Entry Point ====================

func (svc *Service) HandleWithGroup(request events.APIGatewayProxyRequest, routeGroup string) (events.APIGatewayProxyResponse, error) {
	svc.logger.Printf("Received request: %s %s (group=%s)", request.HTTPMethod, request.Path, routeGroup)

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

	switch routeGroup {
	case RouteGroupPosts:
		return svc.handlePosts(request, parts, userName, cognitoID)
	case RouteGroupLikes:
		return svc.handleLikes(request, parts, userName, cognitoID)
	case RouteGroupComments:
		return svc.handleComments(request, parts, userName, cognitoID)
	case RouteGroupPoll:
		return svc.handlePoll(request, parts, userName, cognitoID)
	case RouteGroupChecklist:
		return svc.handleChecklist(request, parts, userName, cognitoID)
	case RouteGroupTask:
		return svc.handleTask(request, parts, userName, cognitoID)
	default:
		return svc.errResp(http.StatusNotFound, "NOT_FOUND", "Route not found")
	}
}

// ==================== Response Helpers ====================

func (svc *Service) okResp(data interface{}, meta *MetaResponse) (events.APIGatewayProxyResponse, error) {
	return svc.statusResp(http.StatusOK, data, meta)
}

func (svc *Service) createdResp(data interface{}) (events.APIGatewayProxyResponse, error) {
	return svc.statusResp(http.StatusCreated, data, nil)
}

func (svc *Service) noContentResp() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent, Headers: RESP_HEADERS, Body: ""}, nil
}

func (svc *Service) statusResp(status int, data interface{}, meta *MetaResponse) (events.APIGatewayProxyResponse, error) {
	envelope := APIResponse{Data: data, Meta: meta, Error: nil}
	body, err := json.Marshal(envelope)
	if err != nil {
		return svc.errResp(http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to serialise response")
	}
	return events.APIGatewayProxyResponse{StatusCode: status, Headers: RESP_HEADERS, Body: string(body)}, nil
}

func (svc *Service) errResp(statusCode int, code, message string) (events.APIGatewayProxyResponse, error) {
	envelope := APIResponse{
		Data: nil, Meta: nil,
		Error: &ErrorResponse{Code: code, Message: message},
	}
	body, _ := json.Marshal(envelope)
	return events.APIGatewayProxyResponse{StatusCode: statusCode, Headers: RESP_HEADERS, Body: string(body)}, nil
}

// ==================== Path Helpers ====================

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	raw := strings.Split(trimmed, "/")
	decoded := make([]string, 0, len(raw))
	for _, part := range raw {
		if v, err := url.PathUnescape(part); err == nil {
			decoded = append(decoded, v)
		} else {
			decoded = append(decoded, part)
		}
	}
	return decoded
}

// ==================== Request Helpers ====================

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

func queryInt(params map[string]string, key string, def int) int {
	if v := params[key]; v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func queryString(params map[string]string, key string) string {
	return strings.TrimSpace(params[key])
}

// ==================== Authorization Helpers ====================

func (svc *Service) ensureTeamMember(teamID, userName string) error {
	member, err := svc.teamsSVC.GetTeamMemberDetails(teamID, userName)
	if err != nil || member == nil || !member.IsActive {
		return fmt.Errorf("user is not a member of this team")
	}
	return nil
}

func (svc *Service) ensureTeamAdminOrAuthor(teamID, userName, authorUserName string) error {
	if userName == authorUserName {
		return nil
	}
	isAdmin, err := svc.teamsSVC.IsTeamAdmin(teamID, userName)
	if err != nil || !isAdmin {
		return fmt.Errorf("access denied: not the author or a team admin")
	}
	return nil
}
