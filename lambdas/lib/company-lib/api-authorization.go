package Companylib

import (
	"fmt"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

const (
	QUERY_NULL = ""
	AND        = "AND"
	OR         = "OR"

	ADMINRole          = "AdminRole"
	UserManagementRole = "UserManagementRole"
	AnalyticsRole      = "AnalyticsRole"
	RewardsManagerRole = "RewardsManagerRole"
	TeamsManagerRole   = "TeamsManagerRole"
)

type AuthData struct {
	Username    string          `json:"username"`
	DisplayName string          `json:"displayName"`
	Designation string          `json:"designation"`
	Roles       map[string]bool `json:"roles"`
	Groups      []string        `json:"groups"` // Key: Group IDs as Names of Groups
}

// Authorizer handles authorization based on query strings and roles.
func (svc *EmployeeService) Authorizer(request events.APIGatewayProxyRequest, query string) (AuthData, bool, error) {
	var EmployeeData EmployeeDynamodbData
	var err error

	// Handle case where query is empty.
	if query == QUERY_NULL {
		employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)
		EmployeeData, err = svc.GetEmployeeDataByCognitoId(employeeCognitoId)
		if err != nil {
			svc.logger.Printf("Error getting Employee Data: %v\n", err)
			return AuthData{}, false, err
		}
		return AuthData{
			Username:    EmployeeData.UserName,
			DisplayName: EmployeeData.DisplayName,
			Designation: EmployeeData.Designation,
			Roles:       EmployeeData.RolesData,
			Groups:      utils.ConvertToArray(EmployeeData.TopLevelGroupName, "|"), // Convert the string to an array
		}, true, nil
	}

	// Parse the query string.
	parts := splitQuery(query)
	if len(parts) == 0 {
		svc.logger.Printf("Invalid query string: %s\n", query)
		return AuthData{}, false, fmt.Errorf("invalid query string")
	}

	// Fetch employee data.
	employeeCognitoId := request.RequestContext.Authorizer["claims"].(map[string]interface{})["cognito:username"].(string)
	EmployeeData, err = svc.GetEmployeeDataByCognitoId(employeeCognitoId)
	if err != nil {
		svc.logger.Printf("Error getting Employee Data: %v\n", err)
		return AuthData{}, false, err
	}

	// Evaluate the roles against the parsed query.
	isAuthorized := evaluateRoles(EmployeeData.RolesData, parts)
	if !isAuthorized {
		return AuthData{}, false, nil
	}

	return AuthData{
		Username:    EmployeeData.UserName,
		DisplayName: EmployeeData.DisplayName,
		Designation: EmployeeData.Designation,
		Roles:       EmployeeData.RolesData,
	}, true, nil
}

// splitQuery breaks a query string into its components.
func splitQuery(query string) []string {
	query = strings.ReplaceAll(query, AND, "|AND|")
	query = strings.ReplaceAll(query, OR, "|OR|")
	return strings.Split(query, "|")
}

// evaluateRoles checks the roles based on AND/OR conditions.
func evaluateRoles(rolesData map[string]bool, queryParts []string) bool {
	if len(queryParts) == 0 {
		return false
	}

	var result bool
	var operation string
	for _, part := range queryParts {
		part = strings.TrimSpace(part)
		if part == AND || part == OR {
			operation = part
			continue
		}

		roleValue := rolesData[part]
		if operation == "" {
			result = roleValue
		} else if operation == AND {
			result = result && roleValue
		} else if operation == OR {
			result = result || roleValue
		}
	}
	return result
}

func (svc *TenantTeamsService) AuthorizerTeams(teamsId string, userId string) (bool, error) {
	// Fetch team data.
	ddbInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: "USER-" + userId},
			"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: teamsId},
		},
		TableName:      aws.String(svc.TenantTeamsTable),
		ConsistentRead: aws.Bool(true),
	}

	result, err := svc.dynamodbClient.GetItem(svc.ctx, &ddbInput)
	if err != nil {
		svc.logger.Printf("Error getting team data: %v\n", err)
		return false, err
	}
	if result.Item == nil {
		return false, nil
	}

	return true, nil
}

func (svc *EmployeeService) AuthorizerGroups(groups []string, groupId string) bool {
	if groupId == "Everyone" {
		return true
	}
	for _, group := range groups {
		if group == groupId {
			return true
		}
	}
	return false
}
