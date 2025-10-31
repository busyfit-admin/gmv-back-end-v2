package Companylib

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	dynamodb_attributevalue "github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	dynamodb_types "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	utils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

// Define constants for team types
const (
	TEAM_TYPE_SOFTWARE_DEV       = "SoftwareDev"
	TEAM_TYPE_MARKETING          = "Marketing"
	TEAM_TYPE_DESIGN             = "Design"
	TEAM_TYPE_SALES              = "Sales"
	TEAM_TYPE_OPERATIONS         = "Operations"
	TEAM_TYPE_PROJECT_MANAGEMENT = "ProjectManagement"
	TEAM_TYPE_SERVICE_MANAGEMENT = "ServiceManagement"
	TEAM_TYPE_HUMAN_RESOURCES    = "HumanResources"
	TEAM_TYPE_FINANCE            = "Finance"
	TEAM_TYPE_GENERAL            = "General"
)

type TenantTeamsTable struct {
	EntityId        string `dynamodbav:"EntityId"`
	RelatedEntityId string `dynamodbav:"RelatedEntityId"`
	TeamName        string `dynamodbav:"TeamName"`
	TeamDesc        string `dynamodbav:"TeamDesc"`
	IsActive        string `dynamodbav:"IsActive"`
}

type CreateTenantTeamsInput struct {
	TeamName string `dynamodbav:"TeamName"`
	TeamDesc string `dynamodbav:"TeamDesc"`
	IsActive string `dynamodbav:"IsActive"`
	MngrId   string `dynamodbav:"MngrId"`
}

const DEFAULT_TEAM_ID = "TEAM-DEFAULT"
const NON_DEFAULT_TEAM_ID = "TEAM-NONDEFAULT"

type TeamUserorMngr struct {
	EntityId string `dynamodbav:"EntityId"`
}

type UserList struct {
	EntityId    string `dynamodbav:"EntityId"`
	DisplayName string `dynamodbav:"DisplayName"`
	ProfilePic  string `dynamodbav:"ProfilePic"`
	Designation string `dynamodbav:"Designation"`
}

type TeamUserorMngrList struct {
	Members []TeamUserorMngr `json:"Members"`
}

type AddDeleteUserInput struct {
	RelatedEntityId string     `dynamodbav:"RelatedEntityId"`
	Users           []UserList `json:"Users"`
	TeamName        string     `dynamodbav:"TeamName"`
	TeamDesc        string     `dynamodbav:"TeamDesc"`
	IsActive        string     `dynamodbav:"IsActive"`
}

type TenantTeamsService struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger

	TenantTeamsTable      string
	TenantTeams_TeamIndex string
	OutBound_Integration  string
}

func CreateTenantTeamsService(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger) *TenantTeamsService {
	return &TenantTeamsService{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
	}
}

type TenantTeams struct {
	Active []TenantTeamsTable `json:"Active"`
	Draft  []TenantTeamsTable `json:"Draft"`
}

func (svc *TenantTeamsService) GetAllTenantTeams() (TenantTeams, error) {

	var allTenantTeamsData TenantTeams

	// 1. Get Active Tenant Teams
	queryGetActiveTenantTeams := "SELECT EntityId, RelatedEntityId, TeamName, TeamDesc, IsActive FROM \"" + svc.TenantTeamsTable + "\" WHERE RelatedEntityId = 'TEAM-DEFAULT' AND IsActive = 'Active'"

	activeTenantTeamsData, err := svc.GetTeamsData(queryGetActiveTenantTeams)
	if err != nil {
		return TenantTeams{}, err
	}
	// svc.logger.Printf("Active Tenant Teams Data : %v", activeTenantTeamsData)
	allTenantTeamsData.Active = activeTenantTeamsData

	// 2. Get InActive Tenant Teams
	queryGetInactiveTenantTeams := "SELECT EntityId, RelatedEntityId, TeamName, TeamDesc, IsActive FROM \"" + svc.TenantTeamsTable + "\" WHERE RelatedEntityId = 'TEAM-DEFAULT' AND IsActive = 'Inactive'"

	inactiveTenantTeamsData, err := svc.GetTeamsData(queryGetInactiveTenantTeams)
	if err != nil {
		return TenantTeams{}, err
	}
	allTenantTeamsData.Draft = inactiveTenantTeamsData

	return allTenantTeamsData, nil
}

func (s *TenantTeamsService) GetTeamDetails(teamId string, relatedId string) (TenantTeamsTable, error) {

	getItemInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: teamId},
			"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: relatedId},
		},
		TableName:      aws.String(s.TenantTeamsTable),
		ConsistentRead: aws.Bool(true),
	}

	output, err := s.dynamodbClient.GetItem(s.ctx, &getItemInput)
	if err != nil {
		s.logger.Printf("Get TenantTeams Failed with error :%v", err)
		return TenantTeamsTable{}, err
	}
	teamData := TenantTeamsTable{}

	err = dynamodb_attributevalue.UnmarshalMap(output.Item, &teamData)
	if err != nil {
		s.logger.Printf("Get TenantTeams Unmarshal failed with error :%v", err)
		return TenantTeamsTable{}, err
	}

	return teamData, nil
}

func (svc *TenantTeamsService) GetTeams(id string) (TenantTeams, error) {

	var allTeamsData TenantTeams

	// 1. Get Active Tenant Teams
	queryGetUsersActiveTeams := "SELECT EntityId, RelatedEntityId, TeamName, TeamDesc, IsActive FROM \"" + svc.TenantTeamsTable + "\" WHERE EntityId = '" + id + "' AND IsActive = 'Active'"

	activeTeamsData, err := svc.GetTeamsData(queryGetUsersActiveTeams)
	if err != nil {
		return TenantTeams{}, err
	}
	// svc.logger.Printf("Active Tenant Teams Data : %v", activeTenantTeamsData)
	allTeamsData.Active = activeTeamsData

	// 2. Get InActive Tenant Teams
	queryGetInactiveTeams := "SELECT EntityId, RelatedEntityId, TeamName, TeamDesc, IsActive FROM \"" + svc.TenantTeamsTable + "\" WHERE EntityId = '" + id + "' AND IsActive = 'Inactive'"

	inactiveTeamsData, err := svc.GetTeamsData(queryGetInactiveTeams)
	if err != nil {
		return TenantTeams{}, err
	}
	allTeamsData.Draft = inactiveTeamsData

	svc.logger.Printf("Active user Teams Data : %v", allTeamsData.Active)
	svc.logger.Printf("Inactive user Teams Data : %v", allTeamsData.Draft)

	return allTeamsData, nil
}

func (svc *TenantTeamsService) GetTeamUsers(id string) (TeamUserorMngrList, error) {

	var userData TeamUserorMngrList

	// 1. Get users in a Team
	queryGetUsers := "SELECT EntityId FROM \"" + svc.TenantTeamsTable + "\".\"" + svc.TenantTeams_TeamIndex + "\" WHERE RelatedEntityId = '" + id + "' AND begins_with(\"EntityId\", 'USER')"

	svc.logger.Printf("Query to get users in a team : %s", queryGetUsers)
	data, err := svc.GetMemberData(queryGetUsers)
	if err != nil {
		return TeamUserorMngrList{}, err
	}
	userData.Members = data
	// svc.logger.Printf("Data : %v", userData.Members)

	return userData, nil
}

func (svc *TenantTeamsService) GetTeamMngrs(id string) (TeamUserorMngrList, error) {

	var mngrData TeamUserorMngrList

	// 1. Get manager in a Teams
	queryGetMngrs := "SELECT EntityId FROM \"" + svc.TenantTeamsTable + "\".\"" + svc.TenantTeams_TeamIndex + "\" WHERE RelatedEntityId = '" + id + "' AND begins_with(\"EntityId\", 'MNGR')"

	data, err := svc.GetMemberData(queryGetMngrs)
	if err != nil {
		return TeamUserorMngrList{}, err
	}
	mngrData.Members = data
	svc.logger.Printf("Active Tenant Teams Data : %v", mngrData)

	return mngrData, nil
}

type UsersDetailsList struct {
	UserName    string `json:"UserName"`
	DisplayName string `json:"DisplayName"`
	Designation string `json:"Designation"`
	ProfilePic  string `json:"ProfilePic"`
}

/*
Takes in the input :

	type TeamUserorMngrList struct {
	    Members []TeamUserorMngr `json:"Members"`
	}
	type TeamUserorMngr struct {
	EntityId string `dynamodbav:"EntityId"`
	}

Gets the Employee Details of the list and returns in struct: UsersDetailsList
*/
func (svc *EmployeeService) GetUserDetailsFromList(teamList TeamUserorMngrList) ([]UsersDetailsList, error) {

	var userDetailsList []UsersDetailsList

	// Loop through each member of the input teamList
	for _, member := range teamList.Members {
		// Define the key to retrieve employee details
		key := map[string]types.AttributeValue{
			"UserName": &types.AttributeValueMemberS{Value: RemovePrefix(member.EntityId)},
		}

		svc.logger.Printf("Key : %v", key)
		// Prepare the GetItemInput with a ProjectionExpression to get specific attributes
		input := &dynamodb.GetItemInput{
			TableName: aws.String(svc.EmployeeTable),
			Key:       key,
			// Specify the attributes you want to fetch from DynamoDB
			ProjectionExpression: aws.String("UserName, DisplayName, Designation, ProfilePic"),
		}

		svc.logger.Printf("Input : %v", input)

		// Call DynamoDB to get the employee data with only the specified attributes
		result, err := svc.dynamodbClient.GetItem(svc.ctx, input)
		if err != nil {
			svc.logger.Printf("Failed to get employee details for EntityId: %s, error: %v", member.EntityId, err)
			return nil, err
		}

		// If no data is found, skip to the next member
		if result.Item == nil {
			continue
		}

		// Unmarshal the DynamoDB result into the EmployeeDynamodbData struct
		var employeeData EmployeeDynamodbData
		err = attributevalue.UnmarshalMap(result.Item, &employeeData)
		if err != nil {
			svc.logger.Printf("Failed to unmarshal employee data for EntityId: %s, error: %v", member.EntityId, err)
			return nil, err
		}

		// Map the data to UsersDetailsList (assuming UsersDetailsList struct exists and corresponds to the fields in EmployeeDynamodbData)
		userDetails := UsersDetailsList{
			UserName:    employeeData.UserName,
			DisplayName: employeeData.DisplayName,
			Designation: employeeData.Designation,
			ProfilePic:  employeeData.ProfilePic,
		}

		// Append the user details to the list
		userDetailsList = append(userDetailsList, userDetails)
	}

	// Return the list of user details
	return userDetailsList, nil
}

// RemovePrefix removes "USER-" or "MNGR-" from the EntityId and returns the cleaned EntityId.
// If neither prefix is found, it returns as is.
func RemovePrefix(entityId string) string {
	// Check if the entityId starts with "USER-" or "MNGR-"
	if strings.HasPrefix(entityId, "USER-") {
		return strings.TrimPrefix(entityId, "USER-")
	} else if strings.HasPrefix(entityId, "MNGR-") {
		return strings.TrimPrefix(entityId, "MNGR-")
	}
	// If no valid prefix is found, return as it is
	return entityId
}

func (svc *TenantTeamsService) GetTeamsData(stmt string) ([]TenantTeamsTable, error) {

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(stmt),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []TenantTeamsTable{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for Team Data Query %s", stmt)
		return []TenantTeamsTable{}, nil
	}

	allTeamData := []TenantTeamsTable{}
	for _, stageItem := range output.Items {
		teamData := TenantTeamsTable{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &teamData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal Team data. Failed with  Error : %v", err)
			return []TenantTeamsTable{}, err
		}

		// Append data to the overall rule data
		allTeamData = append(allTeamData, teamData)
	}

	return allTeamData, nil
}

func (svc *TenantTeamsService) GetMemberData(stmt string) ([]TeamUserorMngr, error) {

	output, err := svc.dynamodbClient.ExecuteStatement(svc.ctx, &dynamodb.ExecuteStatementInput{
		Statement:      aws.String(stmt),
		ConsistentRead: aws.Bool(false),
	})

	if err != nil {
		svc.logger.Printf("Failed to run the query on DDB table and failed with error : %v", err)
		return []TeamUserorMngr{}, err
	}

	if len(output.Items) == 0 {
		svc.logger.Printf("No Items found for Team Data Query %s", stmt)
		return []TeamUserorMngr{}, nil
	}

	allTeamData := []TeamUserorMngr{}
	for _, stageItem := range output.Items {
		teamData := TeamUserorMngr{}
		err = dynamodb_attributevalue.UnmarshalMap(stageItem, &teamData)
		if err != nil {
			svc.logger.Printf("Couldn't unmarshal Team data. Failed with  Error : %v", err)
			return []TeamUserorMngr{}, err
		}

		// Append data to the overall rule data
		allTeamData = append(allTeamData, teamData)
	}

	return allTeamData, nil
}

func (svc *TenantTeamsService) CreateTenantTeams(createTeamData CreateTenantTeamsInput) (string, error) {

	EntityId := "TEAMID-" + utils.GenerateRandomString(8)

	teamItem := map[string]dynamodb_types.AttributeValue{
		"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: EntityId},
		"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: NON_DEFAULT_TEAM_ID},
		"TeamName":        &dynamodb_types.AttributeValueMemberS{Value: createTeamData.TeamName},
		"TeamDesc":        &dynamodb_types.AttributeValueMemberS{Value: createTeamData.TeamDesc},
		"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: createTeamData.IsActive},
	}

	teamPutItemInput := dynamodb.PutItemInput{
		Item:      teamItem,
		TableName: aws.String(svc.TenantTeamsTable),
	}

	_, err := svc.dynamodbClient.PutItem(svc.ctx, &teamPutItemInput)
	if err != nil {
		svc.logger.Printf("Tenant Team PutItem failed with error :%v", err)
		return "", err
	}
	svc.logger.Print("Tenant Team Put item success")

	mngrEntry := map[string]dynamodb_types.AttributeValue{
		"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: createTeamData.MngrId},
		"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: EntityId},
		"TeamName":        &dynamodb_types.AttributeValueMemberS{Value: createTeamData.TeamName},
		"TeamDesc":        &dynamodb_types.AttributeValueMemberS{Value: createTeamData.TeamDesc},
		"IsActive":        &dynamodb_types.AttributeValueMemberS{Value: createTeamData.IsActive},
	}

	mngrPutItemInput := dynamodb.PutItemInput{
		Item:      mngrEntry,
		TableName: aws.String(svc.TenantTeamsTable),
	}

	output, err := svc.dynamodbClient.PutItem(svc.ctx, &mngrPutItemInput)
	if err != nil {
		svc.logger.Printf("Tenant Team PutItem failed with error :%v", err)
		return "", err
	}
	svc.logger.Print("Tenant Team Put item success ", output)

	return EntityId, nil
}

func (svc *TenantTeamsService) UpdateTenantTeam(updateTeamData TenantTeamsTable) error {

	userData, err := svc.GetTeamUsers(updateTeamData.EntityId)
	if err != nil {
		return err
	}
	mngrData, err := svc.GetTeamMngrs(updateTeamData.EntityId)
	if err != nil {
		return err
	}

	updateItemInput := dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TenantTeamsTable),
		Key: map[string]dynamodb_types.AttributeValue{
			"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.EntityId},
			"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.RelatedEntityId},
		},

		UpdateExpression: aws.String("SET TeamName = :TeamName, TeamDesc = :TeamDesc, IsActive = :IsActive"),
		ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
			":TeamName": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.TeamName},
			":TeamDesc": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.TeamDesc},
			":IsActive": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.IsActive},
		},
		ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
	if err != nil {
		return err
	}

	for _, data := range userData.Members {
		updateItemInput := dynamodb.UpdateItemInput{
			TableName: aws.String(svc.TenantTeamsTable),
			Key: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: data.EntityId},
				"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.EntityId},
			},

			UpdateExpression: aws.String("SET TeamName = :TeamName, TeamDesc = :TeamDesc, IsActive = :IsActive"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":TeamName": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.TeamName},
				":TeamDesc": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.TeamDesc},
				":IsActive": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.IsActive},
			},
			ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
		}

		_, err = svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
		if err != nil {
			return err
		}
	}

	for _, data := range mngrData.Members {
		updateItemInput := dynamodb.UpdateItemInput{
			TableName: aws.String(svc.TenantTeamsTable),
			Key: map[string]dynamodb_types.AttributeValue{
				"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: data.EntityId},
				"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.EntityId},
			},

			UpdateExpression: aws.String("SET TeamName = :TeamName, TeamDesc = :TeamDesc, IsActive = :IsActive"),
			ExpressionAttributeValues: map[string]dynamodb_types.AttributeValue{
				":TeamName": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.TeamName},
				":TeamDesc": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.TeamDesc},
				":IsActive": &dynamodb_types.AttributeValueMemberS{Value: updateTeamData.IsActive},
			},
			ReturnValues: dynamodb_types.ReturnValueUpdatedNew,
		}

		_, err = svc.dynamodbClient.UpdateItem(svc.ctx, &updateItemInput)
		if err != nil {
			return err
		}
	}

	return nil
}

func (svc *TenantTeamsService) AddDeleteUsersToTeam(addUserData AddDeleteUserInput, reqType string) error {

	svc.logger.Printf("Add User Data : %v", addUserData)

	var writeRequests []dynamodb_types.WriteRequest

	for _, user := range addUserData.Users {
		av, err := dynamodb_attributevalue.MarshalMap(TenantTeamsTable{
			EntityId:        user.EntityId,
			RelatedEntityId: addUserData.RelatedEntityId,
			TeamName:        addUserData.TeamName,
			TeamDesc:        addUserData.TeamDesc,
			IsActive:        addUserData.IsActive,
		})
		svc.logger.Printf("Add/Delete User Data : %v", av)
		if err != nil {
			svc.logger.Printf("Marshal failed with error :%v", err)
			return fmt.Errorf("failed to marshal add users data: %v", err)
		}
		if reqType == "POST" {
			writeRequests = append(writeRequests, dynamodb_types.WriteRequest{
				PutRequest: &dynamodb_types.PutRequest{
					Item: av,
				},
			})
		} else if reqType == "DELETE" {
			writeRequests = append(writeRequests, dynamodb_types.WriteRequest{
				DeleteRequest: &dynamodb_types.DeleteRequest{
					Key: map[string]dynamodb_types.AttributeValue{
						"EntityId":        &dynamodb_types.AttributeValueMemberS{Value: user.EntityId},
						"RelatedEntityId": &dynamodb_types.AttributeValueMemberS{Value: addUserData.RelatedEntityId},
					},
				},
			})
		}

	}

	svc.logger.Printf("Add/Delete User Data 2 : %v", writeRequests)

	batchWriteInput := dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]dynamodb_types.WriteRequest{
			svc.TenantTeamsTable: writeRequests,
		},
	}

	svc.logger.Printf("Batch Put/Delete op started for users %v", batchWriteInput)
	_, err := svc.dynamodbClient.BatchWriteItem(svc.ctx, &batchWriteInput)
	if err != nil {
		svc.logger.Printf("Tenant Team BatchWriteItem failed with error :%v", err)
		return fmt.Errorf("failed to write batch to DynamoDB: %v", err)
	}

	svc.logger.Printf("Batch Put/Delete op completed for skills")

	return nil
}

type OutBoundWebHooks struct {
	SlackWebHookUrl string `dynamodbav:"SlackWebHookUrl"`
	TeamsWebHookUrl string `dynamodbav:"TeamsWebHookUrl"`
	Username        string `dynamodbav:"Username"`
	Channel         string `dynamodbav:"Channel"`
}

func (svc *TenantTeamsService) GetAllTenantWebHooks(TeamId string) (OutBoundWebHooks, error) {
	var webHookURL OutBoundWebHooks

	getItemInput := dynamodb.GetItemInput{
		Key: map[string]dynamodb_types.AttributeValue{
			"TeamId": &dynamodb_types.AttributeValueMemberS{Value: TeamId},
		},
		TableName:      aws.String(svc.OutBound_Integration),
		ConsistentRead: aws.Bool(true),
	}

	output, err := svc.dynamodbClient.GetItem(svc.ctx, &getItemInput)
	if err != nil {
		svc.logger.Println("Error getting item from DynamoDB:", err)
		return OutBoundWebHooks{}, err
	}

	err = attributevalue.UnmarshalMap(output.Item, &webHookURL)
	if err != nil {
		svc.logger.Println("Error unmarshalling item:", err)
		svc.logger.Println("DynamoDB Output Item:", output.Item)
		return OutBoundWebHooks{}, err
	}

	return webHookURL, nil
}
