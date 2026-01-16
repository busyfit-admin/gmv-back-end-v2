package Companylib

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
)

// TeamStatus represents the status of a team
type TeamStatus string

const (
	TeamStatusActive   TeamStatus = "ACTIVE"
	TeamStatusInactive TeamStatus = "INACTIVE"
)

// TeamMemberRole represents the role of a team member
type TeamMemberRole string

const (
	TeamMemberRoleAdmin  TeamMemberRole = "ADMIN"
	TeamMemberRoleMember TeamMemberRole = "MEMBER"
)

// TeamMetadata represents team information
type TeamMetadata struct {
	PK          string     `dynamodbav:"PK" json:"-"` // TEAM#uuid
	SK          string     `dynamodbav:"SK" json:"-"` // METADATA
	TeamId      string     `dynamodbav:"TeamId" json:"teamId"`
	TeamName    string     `dynamodbav:"TeamName" json:"teamName"`
	TeamDesc    string     `dynamodbav:"TeamDesc" json:"teamDesc"`
	Status      TeamStatus `dynamodbav:"Status" json:"status"`
	CreatedBy   string     `dynamodbav:"CreatedBy" json:"createdBy"`
	CreatedAt   string     `dynamodbav:"CreatedAt" json:"createdAt"`
	UpdatedAt   string     `dynamodbav:"UpdatedAt" json:"updatedAt"`
	MemberCount int        `dynamodbav:"MemberCount" json:"memberCount"`
}

// TeamMember represents a team member
type TeamMember struct {
	PK          string         `dynamodbav:"PK" json:"-"`     // TEAM#uuid
	SK          string         `dynamodbav:"SK" json:"-"`     // USER#username
	GSI1PK      string         `dynamodbav:"GSI1PK" json:"-"` // USER#username
	GSI1SK      string         `dynamodbav:"GSI1SK" json:"-"` // TEAM#uuid
	TeamId      string         `dynamodbav:"TeamId" json:"teamId"`
	UserName    string         `dynamodbav:"UserName" json:"userName"`
	DisplayName string         `dynamodbav:"DisplayName" json:"displayName"`
	Role        TeamMemberRole `dynamodbav:"Role" json:"role"`
	JoinedAt    string         `dynamodbav:"JoinedAt" json:"joinedAt"`
	IsActive    bool           `dynamodbav:"IsActive" json:"isActive"`
}

// UserTeamInfo represents team information from user's perspective
type UserTeamInfo struct {
	TeamId      string         `json:"teamId"`
	TeamName    string         `json:"teamName"`
	TeamDesc    string         `json:"teamDesc"`
	Role        TeamMemberRole `json:"role"`
	Status      TeamStatus     `json:"status"`
	MemberCount int            `json:"memberCount"`
	JoinedAt    string         `json:"joinedAt"`
	IsLoggedIn  bool           `json:"isLoggedIn"`
}

// CreateTeamInput represents input for creating a team
type CreateTeamInput struct {
	TeamName string `json:"teamName" validate:"required"`
	TeamDesc string `json:"teamDesc"`
	UserName string `json:"-"` // Set from auth context
}

// UpdateTeamInput represents input for updating team
type UpdateTeamInput struct {
	TeamId   string     `json:"teamId" validate:"required"`
	TeamName string     `json:"teamName"`
	TeamDesc string     `json:"teamDesc"`
	Status   TeamStatus `json:"status"`
}

// AddTeamMembersInput represents input for adding members to a team
type AddTeamMembersInput struct {
	TeamId    string   `json:"teamId" validate:"required"`
	UserNames []string `json:"userNames" validate:"required,min=1"`
}

// UpdateMemberRoleInput represents input for updating member role
type UpdateMemberRoleInput struct {
	TeamId   string         `json:"teamId" validate:"required"`
	UserName string         `json:"userName" validate:"required"`
	Role     TeamMemberRole `json:"role" validate:"required"`
}

// TeamsServiceV2 handles team operations
type TeamsServiceV2 struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger
	employeeSvc    *EmployeeService
	emailSvc       *EmailService

	TeamsTable string
}

// CreateTeamsServiceV2 creates a new teams service
func CreateTeamsServiceV2(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger, empSvc *EmployeeService, emailSvc *EmailService) *TeamsServiceV2 {
	return &TeamsServiceV2{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
		employeeSvc:    empSvc,
		emailSvc:       emailSvc,
	}
}

// CreateTeam creates a new team with the creator as admin
func (svc *TeamsServiceV2) CreateTeam(input CreateTeamInput) (*TeamMetadata, error) {
	// Generate team ID
	teamId := fmt.Sprintf("TEAM#%s", uuid.New().String())
	now := time.Now().UTC().Format(time.RFC3339)

	// Create team metadata
	teamMetadata := TeamMetadata{
		PK:          teamId,
		SK:          "METADATA",
		TeamId:      teamId,
		TeamName:    input.TeamName,
		TeamDesc:    input.TeamDesc,
		Status:      TeamStatusActive,
		CreatedBy:   input.UserName,
		CreatedAt:   now,
		UpdatedAt:   now,
		MemberCount: 1, // Creator is the first member
	}

	// Create team member entry for creator (as admin)
	teamMember := TeamMember{
		PK:          teamId,
		SK:          fmt.Sprintf("USER#%s", input.UserName),
		GSI1PK:      fmt.Sprintf("USER#%s", input.UserName),
		GSI1SK:      teamId,
		TeamId:      teamId,
		UserName:    input.UserName,
		DisplayName: input.UserName, // Will be updated if we fetch from employee table
		Role:        TeamMemberRoleAdmin,
		JoinedAt:    now,
		IsActive:    true,
	}

	// Fetch display name from employee service if available
	if svc.employeeSvc != nil {
		employee, err := svc.employeeSvc.GetEmployeeDataByUserName(input.UserName)
		if err == nil && employee.DisplayName != "" {
			teamMember.DisplayName = employee.DisplayName
		}
	}

	// Marshal items
	metadataItem, err := attributevalue.MarshalMap(teamMetadata)
	if err != nil {
		svc.logger.Printf("Failed to marshal team metadata: %v", err)
		return nil, fmt.Errorf("failed to marshal team metadata: %w", err)
	}

	memberItem, err := attributevalue.MarshalMap(teamMember)
	if err != nil {
		svc.logger.Printf("Failed to marshal team member: %v", err)
		return nil, fmt.Errorf("failed to marshal team member: %w", err)
	}

	// Use TransactWriteItems to ensure atomicity
	transactItems := []types.TransactWriteItem{
		{
			Put: &types.Put{
				TableName:           aws.String(svc.TeamsTable),
				Item:                metadataItem,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			},
		},
		{
			Put: &types.Put{
				TableName: aws.String(svc.TeamsTable),
				Item:      memberItem,
			},
		},
	}

	_, err = svc.dynamodbClient.TransactWriteItems(svc.ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})

	if err != nil {
		svc.logger.Printf("Failed to create team: %v", err)
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	svc.logger.Printf("Successfully created team: %s", teamId)
	return &teamMetadata, nil
}

// SetCurrentTeam updates the user's current team preference
func (svc *TeamsServiceV2) SetCurrentTeam(userName string, teamId string) error {
	if svc.employeeSvc == nil {
		return fmt.Errorf("employee service not initialized")
	}

	// Verify user is a member of the team
	memberInput := &dynamodb.GetItemInput{
		TableName: aws.String(svc.TeamsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: teamId},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userName)},
		},
	}

	memberResult, err := svc.dynamodbClient.GetItem(svc.ctx, memberInput)
	if err != nil {
		svc.logger.Printf("Failed to verify team membership: %v", err)
		return fmt.Errorf("failed to verify team membership: %w", err)
	}

	if memberResult.Item == nil {
		return fmt.Errorf("user is not a member of team %s", teamId)
	}

	var member TeamMember
	err = attributevalue.UnmarshalMap(memberResult.Item, &member)
	if err != nil {
		return fmt.Errorf("failed to unmarshal member: %w", err)
	}

	if !member.IsActive {
		return fmt.Errorf("user membership is inactive")
	}

	// Update employee's current team
	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.employeeSvc.EmployeeTable),
		Key: map[string]types.AttributeValue{
			"UserName": &types.AttributeValueMemberS{Value: userName},
		},
		UpdateExpression: aws.String("SET CurrentTeamId = :teamId"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":teamId": &types.AttributeValueMemberS{Value: teamId},
		},
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, updateInput)
	if err != nil {
		svc.logger.Printf("Failed to update current team: %v", err)
		return fmt.Errorf("failed to update current team: %w", err)
	}

	svc.logger.Printf("Successfully set current team %s for user %s", teamId, userName)
	return nil
}

// GetCurrentTeam retrieves the user's current team preference
func (svc *TeamsServiceV2) GetCurrentTeam(userName string) (string, error) {
	if svc.employeeSvc == nil {
		svc.logger.Printf("Employee service not initialized, cannot get current team")
		return "", nil
	}

	employee, err := svc.employeeSvc.GetEmployeeDataByUserName(userName)
	if err != nil {
		svc.logger.Printf("Failed to get employee data for user %s: %v", userName, err)
		return "", nil // Return empty string if employee not found
	}

	svc.logger.Printf("Current team for user %s: %s", userName, employee.CurrentTeamId)
	return employee.CurrentTeamId, nil
}

// GetUserTeams retrieves all teams for a user
func (svc *TeamsServiceV2) GetUserTeams(userName string) ([]UserTeamInfo, error) {
	// Get user's current team preference
	currentTeamId, _ := svc.GetCurrentTeam(userName)

	// Query GSI1 to get all teams for the user
	input := &dynamodb.QueryInput{
		TableName:              aws.String(svc.TeamsTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :userKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":userKey": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userName)},
		},
	}

	result, err := svc.dynamodbClient.Query(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to query user teams: %v", err)
		return nil, fmt.Errorf("failed to query user teams: %w", err)
	}

	if len(result.Items) == 0 {
		return []UserTeamInfo{}, nil
	}

	// Unmarshal team members
	var teamMembers []TeamMember
	err = attributevalue.UnmarshalListOfMaps(result.Items, &teamMembers)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal team members: %v", err)
		return nil, fmt.Errorf("failed to unmarshal team members: %w", err)
	}

	// Get metadata for each team
	userTeams := make([]UserTeamInfo, 0, len(teamMembers))
	for _, member := range teamMembers {
		if !member.IsActive {
			continue // Skip inactive memberships
		}

		metadata, err := svc.GetTeamMetadata(member.TeamId)
		if err != nil {
			svc.logger.Printf("Failed to get team metadata for %s: %v", member.TeamId, err)
			continue
		}

		userTeams = append(userTeams, UserTeamInfo{
			TeamId:      metadata.TeamId,
			TeamName:    metadata.TeamName,
			TeamDesc:    metadata.TeamDesc,
			Role:        member.Role,
			Status:      metadata.Status,
			MemberCount: metadata.MemberCount,
			JoinedAt:    member.JoinedAt,
			IsLoggedIn:  metadata.TeamId == currentTeamId,
		})
	}

	// If no current team is set and user has teams, automatically set the first one
	if currentTeamId == "" && len(userTeams) > 0 {
		firstTeamId := userTeams[0].TeamId
		svc.logger.Printf("No current team set for user %s, auto-setting to first team: %s", userName, firstTeamId)

		// Attempt to set the first team as current (ignore error if it fails)
		if err := svc.SetCurrentTeam(userName, firstTeamId); err == nil {
			// Mark the first team as logged in
			userTeams[0].IsLoggedIn = true
			svc.logger.Printf("Successfully auto-set current team to %s for user %s", firstTeamId, userName)
		} else {
			svc.logger.Printf("Failed to auto-set current team: %v", err)
		}
	}

	return userTeams, nil
}

// GetTeamMetadata retrieves team metadata
func (svc *TeamsServiceV2) GetTeamMetadata(teamId string) (*TeamMetadata, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(svc.TeamsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: teamId},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	}

	result, err := svc.dynamodbClient.GetItem(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to get team metadata: %v", err)
		return nil, fmt.Errorf("failed to get team metadata: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("team not found: %s", teamId)
	}

	var metadata TeamMetadata
	err = attributevalue.UnmarshalMap(result.Item, &metadata)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal team metadata: %v", err)
		return nil, fmt.Errorf("failed to unmarshal team metadata: %w", err)
	}

	return &metadata, nil
}

// UpdateTeamStatus updates the team status (activate/deactivate)
func (svc *TeamsServiceV2) UpdateTeamStatus(teamId string, status TeamStatus, userName string) error {
	// Verify user is admin
	isAdmin, err := svc.IsTeamAdmin(teamId, userName)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of team %s", userName, teamId)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TeamsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: teamId},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET #status = :status, UpdatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#status": "Status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":status":    &types.AttributeValueMemberS{Value: string(status)},
			":updatedAt": &types.AttributeValueMemberS{Value: now},
		},
		ConditionExpression: aws.String("attribute_exists(PK)"),
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to update team status: %v", err)
		return fmt.Errorf("failed to update team status: %w", err)
	}

	svc.logger.Printf("Successfully updated team %s status to %s", teamId, status)
	return nil
}

// GetTeamMemberDetails retrieves details of a team member
func (svc *TeamsServiceV2) GetTeamMemberDetails(teamId string, userName string) (*TeamMember, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(svc.TeamsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: teamId},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userName)},
		},
	}

	result, err := svc.dynamodbClient.GetItem(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to get team member details: %v", err)
		return nil, fmt.Errorf("failed to get team member details: %w", err)
	}

	if result.Item == nil {
		return nil, nil // Member not found
	}

	var member TeamMember
	err = attributevalue.UnmarshalMap(result.Item, &member)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal team member details: %v", err)
		return nil, fmt.Errorf("failed to unmarshal team member details: %w", err)
	}

	return &member, nil
}

// IsTeamAdmin checks if a user is an admin of a team
func (svc *TeamsServiceV2) IsTeamAdmin(teamId string, userName string) (bool, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(svc.TeamsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: teamId},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userName)},
		},
	}

	result, err := svc.dynamodbClient.GetItem(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to get team member: %v", err)
		return false, fmt.Errorf("failed to get team member: %w", err)
	}

	if result.Item == nil {
		return false, nil
	}

	var member TeamMember
	err = attributevalue.UnmarshalMap(result.Item, &member)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal team member: %v", err)
		return false, fmt.Errorf("failed to unmarshal team member: %w", err)
	}

	return member.Role == TeamMemberRoleAdmin && member.IsActive, nil
}

// AddTeamMembers adds members to a team
func (svc *TeamsServiceV2) AddTeamMembers(input AddTeamMembersInput, requestingUser string) error {
	// Verify requesting user is admin
	isAdmin, err := svc.IsTeamAdmin(input.TeamId, requestingUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of team %s", requestingUser, input.TeamId)
	}

	// Verify team exists and is active
	metadata, err := svc.GetTeamMetadata(input.TeamId)
	if err != nil {
		return err
	}
	if metadata.Status != TeamStatusActive {
		return fmt.Errorf("cannot add members to inactive team")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	transactItems := make([]types.TransactWriteItem, 0, len(input.UserNames)+1)
	successfullyAddedMembers := make([]string, 0, len(input.UserNames)) // Track successful additions for email

	// Create member entries
	for _, userName := range input.UserNames {
		displayName := userName
		var employeeEmail string

		// Fetch display name and email from employee service if available
		if svc.employeeSvc != nil {
			employee, err := svc.employeeSvc.GetEmployeeDataByUserName(userName)
			if err == nil {
				if employee.DisplayName != "" {
					displayName = employee.DisplayName
				}
				employeeEmail = employee.EmailID
			}
		}

		teamMember := TeamMember{
			PK:          input.TeamId,
			SK:          fmt.Sprintf("USER#%s", userName),
			GSI1PK:      fmt.Sprintf("USER#%s", userName),
			GSI1SK:      input.TeamId,
			TeamId:      input.TeamId,
			UserName:    userName,
			DisplayName: displayName,
			Role:        TeamMemberRoleMember,
			JoinedAt:    now,
			IsActive:    true,
		}

		memberItem, err := attributevalue.MarshalMap(teamMember)
		if err != nil {
			svc.logger.Printf("Failed to marshal team member: %v", err)
			return fmt.Errorf("failed to marshal team member: %w", err)
		}

		transactItems = append(transactItems, types.TransactWriteItem{
			Put: &types.Put{
				TableName:           aws.String(svc.TeamsTable),
				Item:                memberItem,
				ConditionExpression: aws.String("attribute_not_exists(PK)"),
			},
		})

		// Store member info for email sending (only if we have email)
		if employeeEmail != "" {
			successfullyAddedMembers = append(successfullyAddedMembers, userName)
		}
	}

	// Update member count
	transactItems = append(transactItems, types.TransactWriteItem{
		Update: &types.Update{
			TableName: aws.String(svc.TeamsTable),
			Key: map[string]types.AttributeValue{
				"PK": &types.AttributeValueMemberS{Value: input.TeamId},
				"SK": &types.AttributeValueMemberS{Value: "METADATA"},
			},
			UpdateExpression: aws.String("SET MemberCount = MemberCount + :increment, UpdatedAt = :updatedAt"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":increment": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", len(input.UserNames))},
				":updatedAt": &types.AttributeValueMemberS{Value: now},
			},
		},
	})

	_, err = svc.dynamodbClient.TransactWriteItems(svc.ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})

	if err != nil {
		svc.logger.Printf("Failed to add team members: %v", err)
		return fmt.Errorf("failed to add team members: %w", err)
	}

	svc.logger.Printf("Successfully added %d members to team %s", len(input.UserNames), input.TeamId)

	// Send invitation emails to successfully added members
	if svc.emailSvc != nil && len(successfullyAddedMembers) > 0 {
		go svc.sendTeamInvitationEmails(metadata, successfullyAddedMembers, requestingUser)
	}

	return nil
}

// UpdateMemberRole updates a team member's role
func (svc *TeamsServiceV2) UpdateMemberRole(input UpdateMemberRoleInput, requestingUser string) error {
	// Verify requesting user is admin
	isAdmin, err := svc.IsTeamAdmin(input.TeamId, requestingUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of team %s", requestingUser, input.TeamId)
	}

	// Prevent self-demotion if user is the only admin
	if requestingUser == input.UserName && input.Role == TeamMemberRoleMember {
		adminCount, err := svc.GetAdminCount(input.TeamId)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return fmt.Errorf("cannot demote the last admin of the team")
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.TeamsTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: input.TeamId},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", input.UserName)},
		},
		UpdateExpression: aws.String("SET #role = :role, UpdatedAt = :updatedAt"),
		ExpressionAttributeNames: map[string]string{
			"#role": "Role",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":role":      &types.AttributeValueMemberS{Value: string(input.Role)},
			":updatedAt": &types.AttributeValueMemberS{Value: now},
			":true":      &types.AttributeValueMemberBOOL{Value: true},
		},
		ConditionExpression: aws.String("attribute_exists(PK) AND IsActive = :true"),
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, updateInput)
	if err != nil {
		svc.logger.Printf("Failed to update member role: %v", err)
		return fmt.Errorf("failed to update member role: %w", err)
	}

	svc.logger.Printf("Successfully updated role for user %s in team %s to %s", input.UserName, input.TeamId, input.Role)
	return nil
}

// GetAdminCount returns the number of active admins in a team
func (svc *TeamsServiceV2) GetAdminCount(teamId string) (int, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(svc.TeamsTable),
		KeyConditionExpression: aws.String("PK = :teamId AND begins_with(SK, :userPrefix)"),
		FilterExpression:       aws.String("#role = :adminRole AND IsActive = :true"),
		ExpressionAttributeNames: map[string]string{
			"#role": "Role",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":teamId":     &types.AttributeValueMemberS{Value: teamId},
			":userPrefix": &types.AttributeValueMemberS{Value: "USER#"},
			":adminRole":  &types.AttributeValueMemberS{Value: string(TeamMemberRoleAdmin)},
			":true":       &types.AttributeValueMemberBOOL{Value: true},
		},
		Select: types.SelectCount,
	}

	result, err := svc.dynamodbClient.Query(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to get admin count: %v", err)
		return 0, fmt.Errorf("failed to get admin count: %w", err)
	}

	return int(result.Count), nil
}

// GetTeamMembers retrieves all members of a team
func (svc *TeamsServiceV2) GetTeamMembers(teamId string) ([]TeamMember, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(svc.TeamsTable),
		KeyConditionExpression: aws.String("PK = :teamId AND begins_with(SK, :userPrefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":teamId":     &types.AttributeValueMemberS{Value: teamId},
			":userPrefix": &types.AttributeValueMemberS{Value: "USER#"},
		},
	}

	result, err := svc.dynamodbClient.Query(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to query team members: %v", err)
		return nil, fmt.Errorf("failed to query team members: %w", err)
	}

	var members []TeamMember
	err = attributevalue.UnmarshalListOfMaps(result.Items, &members)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal team members: %v", err)
		return nil, fmt.Errorf("failed to unmarshal team members: %w", err)
	}

	return members, nil
}

// sendTeamInvitationEmails sends invitation emails to new team members
func (svc *TeamsServiceV2) sendTeamInvitationEmails(teamMetadata *TeamMetadata, memberUserNames []string, invitedBy string) {
	defer func() {
		if r := recover(); r != nil {
			svc.logger.Printf("Panic in sendTeamInvitationEmails: %v", r)
		}
	}()

	// Get inviter's display name
	inviterDisplayName := invitedBy
	if svc.employeeSvc != nil {
		if inviterData, err := svc.employeeSvc.GetEmployeeDataByUserName(invitedBy); err == nil && inviterData.DisplayName != "" {
			inviterDisplayName = inviterData.DisplayName
		}
	}

	for _, userName := range memberUserNames {
		if svc.employeeSvc == nil {
			continue
		}

		// Get member's email and display name
		memberData, err := svc.employeeSvc.GetEmployeeDataByUserName(userName)
		if err != nil {
			svc.logger.Printf("Failed to get employee data for %s: %v", userName, err)
			continue
		}

		if memberData.EmailID == "" {
			svc.logger.Printf("No email found for user %s, skipping invitation email", userName)
			continue
		}

		// Create email content
		subject := fmt.Sprintf("Welcome to Team: %s - Gomovo Hub", teamMetadata.TeamName)

		htmlBody := fmt.Sprintf(`
			<html>
			<body>
				<div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
					<div style="text-align: center; margin-bottom: 30px;">
						<h1 style="color: #2c5aa0; margin-bottom: 10px;">Welcome to Gomovo Hub!</h1>
					</div>
					
					<div style="background-color: #f8f9fa; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
						<h2 style="color: #333; margin-top: 0;">You've been added to a team!</h2>
						<p style="font-size: 16px; color: #666; line-height: 1.5;">
							Hi <strong>%s</strong>,
						</p>
						<p style="font-size: 16px; color: #666; line-height: 1.5;">
							<strong>%s</strong> has added you to the team <strong>"%s"</strong> in Gomovo Hub.
						</p>
						
						<div style="background-color: white; padding: 15px; border-radius: 5px; margin: 20px 0;">
							<h3 style="color: #2c5aa0; margin-top: 0;">Team Details:</h3>
							<p><strong>Team Name:</strong> %s</p>
							<p><strong>Team Description:</strong> %s</p>
							<p><strong>Added by:</strong> %s</p>
						</div>
					</div>
					
					<div style="text-align: center; margin: 30px 0;">
						<a href="https://gomovo.com/login" style="display: inline-block; background-color: #2c5aa0; color: white; padding: 12px 30px; text-decoration: none; border-radius: 5px; font-weight: bold;">
							Access Gomovo Hub
						</a>
					</div>
					
					<div style="border-top: 1px solid #eee; padding-top: 20px; text-align: center; color: #888; font-size: 14px;">
						<p>If you don't have an account yet, you can sign up using this email address.</p>
						<p>Need help? Contact our support team.</p>
					</div>
				</div>
			</body>
			</html>
		`, memberData.DisplayName, inviterDisplayName, teamMetadata.TeamName, teamMetadata.TeamName, teamMetadata.TeamDesc, inviterDisplayName)

		textBody := fmt.Sprintf(`
Welcome to Gomovo Hub!

Hi %s,

%s has added you to the team "%s" in Gomovo Hub.

Team Details:
- Team Name: %s
- Team Description: %s  
- Added by: %s

You can access Gomovo Hub at: https://gomovo.com/login

If you don't have an account yet, you can sign up using this email address.

Need help? Contact our support team.
		`, memberData.DisplayName, inviterDisplayName, teamMetadata.TeamName, teamMetadata.TeamName, teamMetadata.TeamDesc, inviterDisplayName)

		// Send email using email service
		emailInput := EmailInput{
			ToEmails:  []string{memberData.EmailID},
			Subject:   subject,
			HtmlBody:  htmlBody,
			TextBody:  textBody,
			FromEmail: "noreply@gomovo.com", // Configure as needed
			FromName:  "Gomovo Hub",
		}

		err = svc.emailSvc.SendEmail(emailInput)
		if err != nil {
			svc.logger.Printf("Failed to send team invitation email to %s: %v", memberData.EmailID, err)
		} else {
			svc.logger.Printf("Successfully sent team invitation email to %s for team %s", memberData.EmailID, teamMetadata.TeamName)
		}
	}
}
