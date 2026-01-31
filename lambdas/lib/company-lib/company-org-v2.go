package Companylib

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"

	awsclients "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/clients"
	utils "github.com/busyfit-admin/saas-integrated-apis/lambdas/lib/utils"
)

// BillingMode represents the billing mode of an organization
type BillingMode string

const (
	BillingModeFree BillingMode = "FREE"
	BillingModePaid BillingMode = "PAID"
)

// SubscriptionType represents the subscription type
type SubscriptionType string

const (
	SubscriptionTypeTrial        SubscriptionType = "TRIAL"
	SubscriptionTypeSubscription SubscriptionType = "SUBSCRIPTION"
)

// BillingPlan represents the billing frequency
type BillingPlan string

const (
	BillingPlanMonthly BillingPlan = "MONTHLY"
	BillingPlanYearly  BillingPlan = "YEARLY"
)

// OrgBillingStatus represents the billing status
type OrgBillingStatus string

const (
	OrgBillingStatusActive    OrgBillingStatus = "ACTIVE"
	OrgBillingStatusSuspended OrgBillingStatus = "SUSPENDED"
	OrgBillingStatusTrial     OrgBillingStatus = "TRIAL"
	OrgBillingStatusOverdue   OrgBillingStatus = "OVERDUE"
)

// OrgAdminRole represents the role of an organization admin
type OrgAdminRole string

const (
	OrgAdminRoleOwner       OrgAdminRole = "OWNER"        // Full org control
	OrgAdminRoleAdmin       OrgAdminRole = "ADMIN"        // Can manage org settings
	OrgAdminRoleBillingOnly OrgAdminRole = "BILLING_ONLY" // Can only manage billing
)

// SubscriptionPlan represents available subscription plans
type SubscriptionPlan struct {
	PlanID          string   `json:"planId"`
	PlanName        string   `json:"planName"`
	PlanDescription string   `json:"planDescription"`
	MaxTeams        int      `json:"maxTeams"`
	MaxMembers      int      `json:"maxMembers"`
	MonthlyPrice    float64  `json:"monthlyPrice"`
	YearlyPrice     float64  `json:"yearlyPrice"`
	Features        []string `json:"features"`
}

// Organization represents the enhanced organization structure
type Organization struct {
	// Composite key structure
	PK     string `dynamodbav:"PK" json:"-"`     // ORG#{organizationId}
	SK     string `dynamodbav:"SK" json:"-"`     // METADATA
	GSI1PK string `dynamodbav:"GSI1PK" json:"-"` // For GSI queries
	GSI1SK string `dynamodbav:"GSI1SK" json:"-"` // For GSI queries

	// Primary key - matches CloudFormation template
	OrganizationId string `dynamodbav:"OrganizationId" json:"organizationId"`

	// Basic organization info
	OrgName string `dynamodbav:"OrgName" json:"orgName"`
	OrgDesc string `dynamodbav:"OrgDesc" json:"orgDesc"`

	// Client and industry details
	ClientName   string `dynamodbav:"ClientName" json:"clientName"`
	Industry     string `dynamodbav:"Industry" json:"industry"`
	CompanySize  string `dynamodbav:"CompanySize" json:"companySize"` // e.g., "1-10", "11-50", "51-200", etc.
	Website      string `dynamodbav:"Website" json:"website"`
	ContactEmail string `dynamodbav:"ContactEmail" json:"contactEmail"`
	ContactPhone string `dynamodbav:"ContactPhone" json:"contactPhone"`
	Address      string `dynamodbav:"Address" json:"address"`
	City         string `dynamodbav:"City" json:"city"`
	State        string `dynamodbav:"State" json:"state"`
	Country      string `dynamodbav:"Country" json:"country"`
	ZipCode      string `dynamodbav:"ZipCode" json:"zipCode"`
	TaxID        string `dynamodbav:"TaxID" json:"taxId"`

	// Billing and subscription info
	BillingMode      BillingMode      `dynamodbav:"BillingMode" json:"billingMode"`
	SubscriptionType SubscriptionType `dynamodbav:"SubscriptionType" json:"subscriptionType"`
	BillingPlan      BillingPlan      `dynamodbav:"BillingPlan" json:"billingPlan"`
	OrgBillingStatus OrgBillingStatus `dynamodbav:"OrgBillingStatus" json:"orgBillingStatus"`
	CurrentPlanID    string           `dynamodbav:"CurrentPlanID" json:"currentPlanId"`
	PlanType         string           `dynamodbav:"PlanType" json:"planType"` // For GSI

	// Usage and limits
	CurrentTeamCount  int `dynamodbav:"CurrentTeamCount" json:"currentTeamCount"`
	MaxTeamsAllowed   int `dynamodbav:"MaxTeamsAllowed" json:"maxTeamsAllowed"`
	MaxMembersAllowed int `dynamodbav:"MaxMembersAllowed" json:"maxMembersAllowed"`

	// Promo and billing details
	AppliedPromoCode     string  `dynamodbav:"AppliedPromoCode" json:"appliedPromoCode"`
	PromoDiscountPercent float64 `dynamodbav:"PromoDiscountPercent" json:"promoDiscountPercent"`
	PromoValidUntil      string  `dynamodbav:"PromoValidUntil" json:"promoValidUntil"`

	// Trial info
	TrialStartDate string `dynamodbav:"TrialStartDate" json:"trialStartDate"`
	TrialEndDate   string `dynamodbav:"TrialEndDate" json:"trialEndDate"`

	// Billing dates
	BillingStartDate string `dynamodbav:"BillingStartDate" json:"billingStartDate"`
	NextBillingDate  string `dynamodbav:"NextBillingDate" json:"nextBillingDate"`
	LastPaymentDate  string `dynamodbav:"LastPaymentDate" json:"lastPaymentDate"`

	// Admin users
	AdminUsers []OrgAdmin `dynamodbav:"AdminUsers" json:"adminUsers"`

	// Timestamps
	CreatedAt       string `dynamodbav:"CreatedAt" json:"createdAt"`
	UpdatedAt       string `dynamodbav:"UpdatedAt" json:"updatedAt"`
	CreatorUserName string `dynamodbav:"CreatorUserName" json:"creatorUserName"`
}

// OrgAdmin represents an organization administrator (stored in AdminUsers array)
type OrgAdmin struct {
	UserName    string       `dynamodbav:"UserName" json:"userName"`
	DisplayName string       `dynamodbav:"DisplayName" json:"displayName"`
	Role        OrgAdminRole `dynamodbav:"Role" json:"role"`
	AddedAt     string       `dynamodbav:"AddedAt" json:"addedAt"`
	IsActive    bool         `dynamodbav:"IsActive" json:"isActive"`
}

// OrgUser represents a user-organization relationship stored as separate table items
type OrgUser struct {
	// Composite key structure
	PK     string `dynamodbav:"PK" json:"-"`     // ORG#{organizationId}
	SK     string `dynamodbav:"SK" json:"-"`     // USER#{username}
	GSI1PK string `dynamodbav:"GSI1PK" json:"-"` // USER#{username}
	GSI1SK string `dynamodbav:"GSI1SK" json:"-"` // ORG#{organizationId}

	// User information
	OrganizationId string       `dynamodbav:"OrganizationId" json:"organizationId"`
	UserName       string       `dynamodbav:"UserName" json:"userName"`
	DisplayName    string       `dynamodbav:"DisplayName" json:"displayName"`
	Role           OrgAdminRole `dynamodbav:"Role" json:"role"`
	JoinedAt       string       `dynamodbav:"JoinedAt" json:"joinedAt"`
	IsActive       bool         `dynamodbav:"IsActive" json:"isActive"`
	UpdatedAt      string       `dynamodbav:"UpdatedAt" json:"updatedAt"`
}

// OrgMember represents a simplified view of organization members
type OrgMember struct {
	UserName    string       `json:"userName"`
	DisplayName string       `json:"displayName"`
	Role        OrgAdminRole `json:"role"`
	JoinedAt    string       `json:"joinedAt"`
	IsActive    bool         `json:"isActive"`
}

// PromoCode represents a promotional code
type PromoCode struct {
	// Primary key - matches CloudFormation template
	PromoCode       string   `dynamodbav:"PromoCode" json:"promoCode"`
	DiscountPercent float64  `dynamodbav:"DiscountPercent" json:"discountPercent"`
	DiscountAmount  float64  `dynamodbav:"DiscountAmount" json:"discountAmount"` // Fixed amount discount
	ValidFrom       string   `dynamodbav:"ValidFrom" json:"validFrom"`
	ValidUntil      string   `dynamodbav:"ValidUntil" json:"validUntil"`
	MaxUsages       int      `dynamodbav:"MaxUsages" json:"maxUsages"` // 0 means unlimited
	CurrentUsages   int      `dynamodbav:"CurrentUsages" json:"currentUsages"`
	FreeTrialDays   int      `dynamodbav:"FreeTrialDays" json:"freeTrialDays"`     // 0 means no trial extension
	ApplicablePlans []string `dynamodbav:"ApplicablePlans" json:"applicablePlans"` // Empty means all plans
	// For GSI - matches CloudFormation template
	IsActive  string `dynamodbav:"IsActive" json:"isActive"` // Changed to string for GSI
	CreatedAt string `dynamodbav:"CreatedAt" json:"createdAt"`
	UpdatedAt string `dynamodbav:"UpdatedAt" json:"updatedAt"`
}

// Input structs
type CreateOrganizationInput struct {
	OrgName         string `json:"orgName" validate:"required"`
	OrgDesc         string `json:"orgDesc"`
	ClientName      string `json:"clientName"`
	Industry        string `json:"industry"`
	CompanySize     string `json:"companySize"`
	Website         string `json:"website"`
	ContactEmail    string `json:"contactEmail"`
	ContactPhone    string `json:"contactPhone"`
	Address         string `json:"address"`
	City            string `json:"city"`
	State           string `json:"state"`
	Country         string `json:"country"`
	ZipCode         string `json:"zipCode"`
	TaxID           string `json:"taxId"`
	CreatorUserName string `json:"-"` // Set from auth context
}

type UpdateOrganizationInput struct {
	OrganizationId string `json:"organizationId" validate:"required"`
	OrgName        string `json:"orgName"`
	OrgDesc        string `json:"orgDesc"`
	ClientName     string `json:"clientName"`
	Industry       string `json:"industry"`
	CompanySize    string `json:"companySize"`
	Website        string `json:"website"`
	ContactEmail   string `json:"contactEmail"`
	ContactPhone   string `json:"contactPhone"`
	Address        string `json:"address"`
	City           string `json:"city"`
	State          string `json:"state"`
	Country        string `json:"country"`
	ZipCode        string `json:"zipCode"`
	TaxID          string `json:"taxId"`
}

type UpdateSubscriptionInput struct {
	OrganizationId string      `json:"organizationId" validate:"required"`
	PlanID         string      `json:"planId" validate:"required"`
	BillingPlan    BillingPlan `json:"billingPlan" validate:"required"`
}

type ApplyPromoCodeInput struct {
	OrganizationId string `json:"organizationId" validate:"required"`
	PromoCode      string `json:"promoCode" validate:"required"`
}

type CreatePromoCodeInput struct {
	PromoCode       string   `json:"promoCode" validate:"required"`
	DiscountPercent float64  `json:"discountPercent"`
	DiscountAmount  float64  `json:"discountAmount"`
	ValidFrom       string   `json:"validFrom" validate:"required"`
	ValidUntil      string   `json:"validUntil" validate:"required"`
	MaxUsages       int      `json:"maxUsages"`
	FreeTrialDays   int      `json:"freeTrialDays"`
	ApplicablePlans []string `json:"applicablePlans"`
}

// OrgServiceV2 handles organization operations
type OrgServiceV2 struct {
	ctx            context.Context
	dynamodbClient awsclients.DynamodbClient
	logger         *log.Logger
	employeeSvc    *EmployeeService
	emailSvc       *EmailService

	OrganizationTable string
	PromoCodesTable   string
}

// CreateOrgServiceV2 creates a new organization service
func CreateOrgServiceV2(ctx context.Context, ddbClient awsclients.DynamodbClient, logger *log.Logger, empSvc *EmployeeService, emailSvc *EmailService) *OrgServiceV2 {
	return &OrgServiceV2{
		ctx:            ctx,
		dynamodbClient: ddbClient,
		logger:         logger,
		employeeSvc:    empSvc,
		emailSvc:       emailSvc,
	}
}

// GetAvailableSubscriptionPlans returns all available subscription plans
func (svc *OrgServiceV2) GetAvailableSubscriptionPlans() []SubscriptionPlan {
	return []SubscriptionPlan{
		{
			PlanID:          "starter",
			PlanName:        "Starter Plan",
			PlanDescription: "Perfect for small teams just getting started",
			MaxTeams:        5,
			MaxMembers:      25,
			MonthlyPrice:    29.99,
			YearlyPrice:     299.99,
			Features:        []string{"Basic team management", "Email support", "5 teams", "25 members"},
		},
		{
			PlanID:          "professional",
			PlanName:        "Professional Plan",
			PlanDescription: "Great for growing organizations",
			MaxTeams:        25,
			MaxMembers:      150,
			MonthlyPrice:    79.99,
			YearlyPrice:     799.99,
			Features:        []string{"Advanced team management", "Priority support", "25 teams", "150 members", "Analytics dashboard"},
		},
		{
			PlanID:          "enterprise",
			PlanName:        "Enterprise Plan",
			PlanDescription: "For large organizations with advanced needs",
			MaxTeams:        -1, // Unlimited
			MaxMembers:      -1, // Unlimited
			MonthlyPrice:    199.99,
			YearlyPrice:     1999.99,
			Features:        []string{"Unlimited teams", "Unlimited members", "24/7 support", "Custom integrations", "Advanced analytics"},
		},
	}
}

// GetSubscriptionPlanByID returns a specific subscription plan
func (svc *OrgServiceV2) GetSubscriptionPlanByID(planID string) (*SubscriptionPlan, error) {
	plans := svc.GetAvailableSubscriptionPlans()
	for _, plan := range plans {
		if plan.PlanID == planID {
			return &plan, nil
		}
	}
	return nil, fmt.Errorf("subscription plan not found: %s", planID)
}

// CreateOrganization creates a new organization with the creator as owner
func (svc *OrgServiceV2) CreateOrganization(input CreateOrganizationInput) (*Organization, error) {
	// Generate organization ID
	orgId := fmt.Sprintf("ORG#%s", uuid.New().String())
	now := time.Now().UTC().Format(time.RFC3339)

	// Set default trial period (30 days)
	trialEndDate := time.Now().UTC().AddDate(0, 0, 30).Format(time.RFC3339)

	// Create organization with starter plan as default
	starterPlan, err := svc.GetSubscriptionPlanByID("starter")
	if err != nil {
		return nil, err
	}

	organization := Organization{
		// Composite key structure
		PK:     fmt.Sprintf("ORG#%s", orgId),
		SK:     "METADATA",
		GSI1PK: "", // Not used for organization metadata
		GSI1SK: "", // Not used for organization metadata

		OrganizationId: orgId,
		OrgName:        input.OrgName,
		OrgDesc:        input.OrgDesc,

		// Client details
		ClientName:   input.ClientName,
		Industry:     input.Industry,
		CompanySize:  input.CompanySize,
		Website:      input.Website,
		ContactEmail: input.ContactEmail,
		ContactPhone: input.ContactPhone,
		Address:      input.Address,
		City:         input.City,
		State:        input.State,
		Country:      input.Country,
		ZipCode:      input.ZipCode,
		TaxID:        input.TaxID,

		// Default billing settings
		BillingMode:      BillingModeFree,
		SubscriptionType: SubscriptionTypeTrial,
		BillingPlan:      BillingPlanMonthly,
		OrgBillingStatus: OrgBillingStatusTrial,
		CurrentPlanID:    starterPlan.PlanID,
		PlanType:         starterPlan.PlanID, // For GSI

		// Usage limits from starter plan
		CurrentTeamCount:  0,
		MaxTeamsAllowed:   starterPlan.MaxTeams,
		MaxMembersAllowed: starterPlan.MaxMembers,

		// Trial settings
		TrialStartDate: now,
		TrialEndDate:   trialEndDate,

		// Timestamps
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Create organization admin entry for creator
	orgAdmin := OrgAdmin{
		UserName:    input.CreatorUserName,
		DisplayName: input.CreatorUserName,
		Role:        OrgAdminRoleOwner,
		AddedAt:     now,
		IsActive:    true,
	}

	// Fetch display name from employee service if available
	if svc.employeeSvc != nil {
		employee, err := svc.employeeSvc.GetEmployeeDataByUserName(input.CreatorUserName)
		if err == nil && employee.DisplayName != "" {
			orgAdmin.DisplayName = employee.DisplayName
		}
	}

	// Add admin to organization
	organization.AdminUsers = []OrgAdmin{orgAdmin}
	organization.CreatorUserName = input.CreatorUserName

	// Marshal organization (includes embedded admin)
	orgItem, err := attributevalue.MarshalMap(organization)
	if err != nil {
		svc.logger.Printf("Failed to marshal organization: %v", err)
		return nil, fmt.Errorf("failed to marshal organization: %w", err)
	}

	// Log the organization item
	utils.LogAsJSON(svc.logger, "Organization Item", orgItem)

	svc.logger.Printf("Creating organization with ID: %s", orgId)

	// Use PutItem for organization (admins will be managed separately)
	_, err = svc.dynamodbClient.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(svc.OrganizationTable),
		Item:                orgItem,
		ConditionExpression: aws.String("attribute_not_exists(OrganizationId)"),
	})

	if err != nil {
		svc.logger.Printf("Failed to create organization: %v", err)
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	svc.logger.Printf("Successfully created organization with admin: %s", orgId)
	return &organization, nil
}

// GetOrganization retrieves organization details
func (svc *OrgServiceV2) GetOrganization(organizationId string) (*Organization, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", organizationId)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
	}

	result, err := svc.dynamodbClient.GetItem(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to get organization: %v", err)
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("organization not found: %s", organizationId)
	}

	var org Organization
	err = attributevalue.UnmarshalMap(result.Item, &org)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal organization: %v", err)
		return nil, fmt.Errorf("failed to unmarshal organization: %w", err)
	}

	return &org, nil
}

// IsOrgAdmin checks if a user is an admin of an organization
func (svc *OrgServiceV2) IsOrgAdmin(organizationId string, userName string) (bool, error) {
	svc.logger.Printf("Checking if user %s is admin of organization %s", userName, organizationId)

	// Get organization details
	org, err := svc.GetOrganization(organizationId)
	if err != nil {
		return false, err
	}

	// Check if user is in the AdminUsers list
	for _, admin := range org.AdminUsers {
		if admin.UserName == userName && admin.IsActive {
			return true, nil
		}
	}

	return false, nil
}

// UpdateOrganization updates organization details (only org admins)
func (svc *OrgServiceV2) UpdateOrganization(input UpdateOrganizationInput, requestingUser string) error {
	// Verify requesting user is org admin
	isAdmin, err := svc.IsOrgAdmin(input.OrganizationId, requestingUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of organization %s", requestingUser, input.OrganizationId)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Build update expression dynamically
	updateExpressions := []string{"UpdatedAt = :updatedAt"}
	expressionAttributeValues := map[string]types.AttributeValue{
		":updatedAt": &types.AttributeValueMemberS{Value: now},
	}

	if input.OrgName != "" {
		updateExpressions = append(updateExpressions, "OrgName = :orgName")
		expressionAttributeValues[":orgName"] = &types.AttributeValueMemberS{Value: input.OrgName}
	}
	if input.OrgDesc != "" {
		updateExpressions = append(updateExpressions, "OrgDesc = :orgDesc")
		expressionAttributeValues[":orgDesc"] = &types.AttributeValueMemberS{Value: input.OrgDesc}
	}
	if input.ClientName != "" {
		updateExpressions = append(updateExpressions, "ClientName = :clientName")
		expressionAttributeValues[":clientName"] = &types.AttributeValueMemberS{Value: input.ClientName}
	}
	if input.Industry != "" {
		updateExpressions = append(updateExpressions, "Industry = :industry")
		expressionAttributeValues[":industry"] = &types.AttributeValueMemberS{Value: input.Industry}
	}
	if input.CompanySize != "" {
		updateExpressions = append(updateExpressions, "CompanySize = :companySize")
		expressionAttributeValues[":companySize"] = &types.AttributeValueMemberS{Value: input.CompanySize}
	}
	if input.Website != "" {
		updateExpressions = append(updateExpressions, "Website = :website")
		expressionAttributeValues[":website"] = &types.AttributeValueMemberS{Value: input.Website}
	}
	if input.ContactEmail != "" {
		updateExpressions = append(updateExpressions, "ContactEmail = :contactEmail")
		expressionAttributeValues[":contactEmail"] = &types.AttributeValueMemberS{Value: input.ContactEmail}
	}
	if input.ContactPhone != "" {
		updateExpressions = append(updateExpressions, "ContactPhone = :contactPhone")
		expressionAttributeValues[":contactPhone"] = &types.AttributeValueMemberS{Value: input.ContactPhone}
	}
	if input.Address != "" {
		updateExpressions = append(updateExpressions, "Address = :address")
		expressionAttributeValues[":address"] = &types.AttributeValueMemberS{Value: input.Address}
	}
	if input.City != "" {
		updateExpressions = append(updateExpressions, "City = :city")
		expressionAttributeValues[":city"] = &types.AttributeValueMemberS{Value: input.City}
	}
	if input.State != "" {
		updateExpressions = append(updateExpressions, "#state = :state")
		expressionAttributeValues[":state"] = &types.AttributeValueMemberS{Value: input.State}
	}
	if input.Country != "" {
		updateExpressions = append(updateExpressions, "Country = :country")
		expressionAttributeValues[":country"] = &types.AttributeValueMemberS{Value: input.Country}
	}
	if input.ZipCode != "" {
		updateExpressions = append(updateExpressions, "ZipCode = :zipCode")
		expressionAttributeValues[":zipCode"] = &types.AttributeValueMemberS{Value: input.ZipCode}
	}
	if input.TaxID != "" {
		updateExpressions = append(updateExpressions, "TaxID = :taxId")
		expressionAttributeValues[":taxId"] = &types.AttributeValueMemberS{Value: input.TaxID}
	}

	if len(updateExpressions) == 1 { // Only UpdatedAt
		return fmt.Errorf("no fields to update")
	}

	updateExpression := "SET " + fmt.Sprintf("%v", updateExpressions)[1:len(fmt.Sprintf("%v", updateExpressions))-1]
	updateExpression = updateExpression[1 : len(updateExpression)-1] // Remove brackets

	expressionAttributeNames := map[string]string{}
	if input.State != "" {
		expressionAttributeNames["#state"] = "State" // State is a reserved word
	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", input.OrganizationId)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeValues: expressionAttributeValues,
		ConditionExpression:       aws.String("attribute_exists(PK) AND attribute_exists(SK)"),
	}

	if len(expressionAttributeNames) > 0 {
		updateInput.ExpressionAttributeNames = expressionAttributeNames
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, updateInput)
	if err != nil {
		svc.logger.Printf("Failed to update organization: %v", err)
		return fmt.Errorf("failed to update organization: %w", err)
	}

	svc.logger.Printf("Successfully updated organization: %s", input.OrganizationId)
	return nil
}

// UpdateSubscription updates organization subscription plan (only org admins)
func (svc *OrgServiceV2) UpdateSubscription(input UpdateSubscriptionInput, requestingUser string) error {
	// Verify requesting user is org admin
	isAdmin, err := svc.IsOrgAdmin(input.OrganizationId, requestingUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of organization %s", requestingUser, input.OrganizationId)
	}

	// Validate plan exists
	plan, err := svc.GetSubscriptionPlanByID(input.PlanID)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Calculate next billing date based on billing plan
	var nextBillingDate string
	if input.BillingPlan == BillingPlanMonthly {
		nextBillingDate = time.Now().UTC().AddDate(0, 1, 0).Format(time.RFC3339)
	} else {
		nextBillingDate = time.Now().UTC().AddDate(1, 0, 0).Format(time.RFC3339)
	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", input.OrganizationId)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET CurrentPlanID = :planId, BillingPlan = :billingPlan, MaxTeamsAllowed = :maxTeams, MaxMembersAllowed = :maxMembers, NextBillingDate = :nextBillingDate, BillingMode = :billingMode, SubscriptionType = :subscriptionType, OrgBillingStatus = :billingStatus, PlanType = :planType, UpdatedAt = :updatedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":planId":           &types.AttributeValueMemberS{Value: input.PlanID},
			":billingPlan":      &types.AttributeValueMemberS{Value: string(input.BillingPlan)},
			":maxTeams":         &types.AttributeValueMemberN{Value: strconv.Itoa(plan.MaxTeams)},
			":maxMembers":       &types.AttributeValueMemberN{Value: strconv.Itoa(plan.MaxMembers)},
			":nextBillingDate":  &types.AttributeValueMemberS{Value: nextBillingDate},
			":billingMode":      &types.AttributeValueMemberS{Value: string(BillingModePaid)},
			":subscriptionType": &types.AttributeValueMemberS{Value: string(SubscriptionTypeSubscription)},
			":billingStatus":    &types.AttributeValueMemberS{Value: string(OrgBillingStatusActive)},
			":planType":         &types.AttributeValueMemberS{Value: input.PlanID}, // For GSI
			":updatedAt":        &types.AttributeValueMemberS{Value: now},
		},
		ConditionExpression: aws.String("attribute_exists(OrganizationId)"),
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, updateInput)
	if err != nil {
		svc.logger.Printf("Failed to update subscription: %v", err)
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	svc.logger.Printf("Successfully updated subscription for organization %s to plan %s", input.OrganizationId, input.PlanID)
	return nil
}

// CanCreateTeam checks if organization can create more teams based on their plan
func (svc *OrgServiceV2) CanCreateTeam(orgId string) (bool, error) {
	org, err := svc.GetOrganization(orgId)
	if err != nil {
		return false, err
	}

	// Unlimited teams (enterprise plan)
	if org.MaxTeamsAllowed == -1 {
		return true, nil
	}

	return org.CurrentTeamCount < org.MaxTeamsAllowed, nil
}

// IncrementTeamCount increments the current team count for an organization
func (svc *OrgServiceV2) IncrementTeamCount(orgId string) error {
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", orgId)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET CurrentTeamCount = CurrentTeamCount + :inc, UpdatedAt = :updatedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":inc":       &types.AttributeValueMemberN{Value: "1"},
			":updatedAt": &types.AttributeValueMemberS{Value: time.Now().UTC().Format(time.RFC3339)},
		},
		ConditionExpression: aws.String("attribute_exists(OrganizationId)"),
	}

	_, err := svc.dynamodbClient.UpdateItem(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to increment team count: %v", err)
		return fmt.Errorf("failed to increment team count: %w", err)
	}

	return nil
}

// CreatePromoCode creates a new promotional code (system admin function)
func (svc *OrgServiceV2) CreatePromoCode(input CreatePromoCodeInput) (*PromoCode, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	promoCode := PromoCode{
		PromoCode:       input.PromoCode,
		DiscountPercent: input.DiscountPercent,
		DiscountAmount:  input.DiscountAmount,
		ValidFrom:       input.ValidFrom,
		ValidUntil:      input.ValidUntil,
		MaxUsages:       input.MaxUsages,
		CurrentUsages:   0,
		FreeTrialDays:   input.FreeTrialDays,
		ApplicablePlans: input.ApplicablePlans,
		IsActive:        "true", // String for GSI
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	item, err := attributevalue.MarshalMap(promoCode)
	if err != nil {
		svc.logger.Printf("Failed to marshal promo code: %v", err)
		return nil, fmt.Errorf("failed to marshal promo code: %w", err)
	}

	putInput := &dynamodb.PutItemInput{
		TableName:           aws.String(svc.PromoCodesTable),
		Item:                item,
		ConditionExpression: aws.String("attribute_not_exists(PromoCode)"),
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, putInput)
	if err != nil {
		svc.logger.Printf("Failed to create promo code: %v", err)
		return nil, fmt.Errorf("failed to create promo code: %w", err)
	}

	svc.logger.Printf("Successfully created promo code: %s", input.PromoCode)
	return &promoCode, nil
}

// GetPromoCode retrieves a promotional code
func (svc *OrgServiceV2) GetPromoCode(promoCode string) (*PromoCode, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(svc.PromoCodesTable),
		Key: map[string]types.AttributeValue{
			"PromoCode": &types.AttributeValueMemberS{Value: promoCode},
		},
	}

	result, err := svc.dynamodbClient.GetItem(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to get promo code: %v", err)
		return nil, fmt.Errorf("failed to get promo code: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("promo code not found: %s", promoCode)
	}

	var promo PromoCode
	err = attributevalue.UnmarshalMap(result.Item, &promo)
	if err != nil {
		svc.logger.Printf("Failed to unmarshal promo code: %v", err)
		return nil, fmt.Errorf("failed to unmarshal promo code: %w", err)
	}

	return &promo, nil
}

// ApplyPromoCode applies a promotional code to an organization (only org admins)
func (svc *OrgServiceV2) ApplyPromoCode(input ApplyPromoCodeInput, requestingUser string) error {
	// Verify requesting user is org admin
	isAdmin, err := svc.IsOrgAdmin(input.OrganizationId, requestingUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of organization %s", requestingUser, input.OrganizationId)
	}

	// Get and validate promo code
	promoCode, err := svc.GetPromoCode(input.PromoCode)
	if err != nil {
		return err
	}

	// Validate promo code
	now := time.Now().UTC()
	validFrom, err := time.Parse(time.RFC3339, promoCode.ValidFrom)
	if err != nil {
		return fmt.Errorf("invalid promo code valid from date: %w", err)
	}
	validUntil, err := time.Parse(time.RFC3339, promoCode.ValidUntil)
	if err != nil {
		return fmt.Errorf("invalid promo code valid until date: %w", err)
	}

	if promoCode.IsActive != "true" {
		return fmt.Errorf("promo code is not active")
	}
	if now.Before(validFrom) {
		return fmt.Errorf("promo code is not yet valid")
	}
	if now.After(validUntil) {
		return fmt.Errorf("promo code has expired")
	}
	if promoCode.MaxUsages > 0 && promoCode.CurrentUsages >= promoCode.MaxUsages {
		return fmt.Errorf("promo code usage limit exceeded")
	}

	// Get organization details to check plan compatibility
	org, err := svc.GetOrganization(input.OrganizationId)
	if err != nil {
		return err
	}

	// Check if promo code is applicable to current plan
	if len(promoCode.ApplicablePlans) > 0 {
		planApplicable := false
		for _, applicablePlan := range promoCode.ApplicablePlans {
			if applicablePlan == org.CurrentPlanID {
				planApplicable = true
				break
			}
		}
		if !planApplicable {
			return fmt.Errorf("promo code is not applicable to current plan")
		}
	}

	nowStr := now.Format(time.RFC3339)
	promoValidUntil := validUntil.Format(time.RFC3339)

	// Apply trial extension if applicable
	var trialEndDate string
	if promoCode.FreeTrialDays > 0 {
		if org.OrgBillingStatus == OrgBillingStatusTrial {
			// Extend existing trial
			currentTrialEnd, err := time.Parse(time.RFC3339, org.TrialEndDate)
			if err == nil {
				newTrialEnd := currentTrialEnd.AddDate(0, 0, promoCode.FreeTrialDays)
				trialEndDate = newTrialEnd.Format(time.RFC3339)
			}
		} else {
			// Start new trial
			newTrialEnd := now.AddDate(0, 0, promoCode.FreeTrialDays)
			trialEndDate = newTrialEnd.Format(time.RFC3339)
		}
	}

	// Prepare transaction items
	transactItems := []types.TransactWriteItem{
		// Update organization with promo code
		{
			Update: &types.Update{
				TableName: aws.String(svc.OrganizationTable),
				Key: map[string]types.AttributeValue{
					"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", input.OrganizationId)},
					"SK": &types.AttributeValueMemberS{Value: "METADATA"},
				},
				UpdateExpression: aws.String("SET AppliedPromoCode = :promoCode, PromoDiscountPercent = :discountPercent, PromoValidUntil = :promoValidUntil, UpdatedAt = :updatedAt"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":promoCode":       &types.AttributeValueMemberS{Value: input.PromoCode},
					":discountPercent": &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", promoCode.DiscountPercent)},
					":promoValidUntil": &types.AttributeValueMemberS{Value: promoValidUntil},
					":updatedAt":       &types.AttributeValueMemberS{Value: nowStr},
				},
			},
		},
		// Increment promo code usage
		{
			Update: &types.Update{
				TableName: aws.String(svc.PromoCodesTable),
				Key: map[string]types.AttributeValue{
					"PromoCode": &types.AttributeValueMemberS{Value: input.PromoCode},
				},
				UpdateExpression: aws.String("SET CurrentUsages = CurrentUsages + :inc, UpdatedAt = :updatedAt"),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":inc":       &types.AttributeValueMemberN{Value: "1"},
					":updatedAt": &types.AttributeValueMemberS{Value: nowStr},
				},
			},
		},
	}

	// Add trial extension if applicable
	if trialEndDate != "" {
		transactItems[0].Update.UpdateExpression = aws.String("SET AppliedPromoCode = :promoCode, PromoDiscountPercent = :discountPercent, PromoValidUntil = :promoValidUntil, TrialEndDate = :trialEndDate, OrgBillingStatus = :trialStatus, UpdatedAt = :updatedAt")
		transactItems[0].Update.ExpressionAttributeValues[":trialEndDate"] = &types.AttributeValueMemberS{Value: trialEndDate}
		transactItems[0].Update.ExpressionAttributeValues[":trialStatus"] = &types.AttributeValueMemberS{Value: string(OrgBillingStatusTrial)}
	}

	_, err = svc.dynamodbClient.TransactWriteItems(svc.ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: transactItems,
	})

	if err != nil {
		svc.logger.Printf("Failed to apply promo code: %v", err)
		return fmt.Errorf("failed to apply promo code: %w", err)
	}

	svc.logger.Printf("Successfully applied promo code %s to organization %s", input.PromoCode, input.OrganizationId)
	return nil
}

// AddOrgAdmin adds a new admin to an organization (only org owners can do this)
// For this release, users can only be part of one organization
func (svc *OrgServiceV2) AddOrgAdmin(organizationId string, newAdminUserName string, role OrgAdminRole, requestingUser string) error {
	// Check if user can join this organization (single org constraint)
	if err := svc.CheckUserCanJoinOrganization(newAdminUserName, organizationId); err != nil {
		return err
	}

	// Verify requesting user is org admin
	isAdmin, err := svc.IsOrgAdmin(organizationId, requestingUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of organization %s", requestingUser, organizationId)
	}

	// Get organization to check current admins and verify requesting user role
	org, err := svc.GetOrganization(organizationId)
	if err != nil {
		return err
	}

	// Check if requesting user is owner (only owners can add admins)
	isOwner := false
	for _, admin := range org.AdminUsers {
		if admin.UserName == requestingUser && admin.IsActive && admin.Role == OrgAdminRoleOwner {
			isOwner = true
			break
		}
	}

	if !isOwner {
		return fmt.Errorf("only organization owners can add new admins")
	}

	// Check if user is already an admin
	for _, admin := range org.AdminUsers {
		if admin.UserName == newAdminUserName {
			if admin.IsActive {
				return fmt.Errorf("user %s is already an admin of organization %s", newAdminUserName, organizationId)
			} else {
				// Reactivate existing admin
				return svc.reactivateAdmin(organizationId, newAdminUserName, role)
			}
		}
	}

	// Create new admin
	now := time.Now().UTC().Format(time.RFC3339)
	newAdmin := OrgAdmin{
		UserName:    newAdminUserName,
		DisplayName: newAdminUserName,
		Role:        role,
		AddedAt:     now,
		IsActive:    true,
	}

	// Fetch display name from employee service if available
	if svc.employeeSvc != nil {
		employee, err := svc.employeeSvc.GetEmployeeDataByUserName(newAdminUserName)
		if err == nil && employee.DisplayName != "" {
			newAdmin.DisplayName = employee.DisplayName
		}
	}

	// Add admin to the list
	updatedAdmins := append(org.AdminUsers, newAdmin)

	// Marshal the updated admins list
	adminsList, err := attributevalue.MarshalList(updatedAdmins)
	if err != nil {
		return fmt.Errorf("failed to marshal admins list: %w", err)
	}

	// Update organization with new admin
	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", organizationId)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET AdminUsers = :adminUsers, UpdatedAt = :updatedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":adminUsers": &types.AttributeValueMemberL{Value: adminsList},
			":updatedAt":  &types.AttributeValueMemberS{Value: now},
		},
		ConditionExpression: aws.String("attribute_exists(OrganizationId)"),
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, updateInput)
	if err != nil {
		svc.logger.Printf("Failed to add org admin: %v", err)
		return fmt.Errorf("failed to add org admin: %w", err)
	}

	svc.logger.Printf("Successfully added admin %s to organization %s", newAdminUserName, organizationId)
	return nil
}

// RemoveOrgAdmin removes an admin from an organization (only org owners can do this)
func (svc *OrgServiceV2) RemoveOrgAdmin(organizationId string, adminUserName string, requestingUser string) error {
	// Verify requesting user is org admin
	isAdmin, err := svc.IsOrgAdmin(organizationId, requestingUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of organization %s", requestingUser, organizationId)
	}

	// Get organization to check current admins
	org, err := svc.GetOrganization(organizationId)
	if err != nil {
		return err
	}

	// Check if requesting user is owner
	isOwner := false
	for _, admin := range org.AdminUsers {
		if admin.UserName == requestingUser && admin.IsActive && admin.Role == OrgAdminRoleOwner {
			isOwner = true
			break
		}
	}

	if !isOwner {
		return fmt.Errorf("only organization owners can remove admins")
	}

	// Can't remove self if they're the only owner
	if requestingUser == adminUserName {
		ownerCount := 0
		for _, admin := range org.AdminUsers {
			if admin.IsActive && admin.Role == OrgAdminRoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return fmt.Errorf("cannot remove the only owner of the organization")
		}
	}

	// Update the admin list by deactivating the admin
	var updatedAdmins []OrgAdmin
	adminFound := false
	now := time.Now().UTC().Format(time.RFC3339)

	for _, admin := range org.AdminUsers {
		if admin.UserName == adminUserName {
			adminFound = true
			admin.IsActive = false
			admin.AddedAt = now // Update timestamp
		}
		updatedAdmins = append(updatedAdmins, admin)
	}

	if !adminFound {
		return fmt.Errorf("admin %s not found in organization %s", adminUserName, organizationId)
	}

	// Marshal the updated admins list
	adminsList, err := attributevalue.MarshalList(updatedAdmins)
	if err != nil {
		return fmt.Errorf("failed to marshal admins list: %w", err)
	}

	// Update organization
	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Key: map[string]types.AttributeValue{
			"OrganizationId": &types.AttributeValueMemberS{Value: organizationId},
		},
		UpdateExpression: aws.String("SET AdminUsers = :adminUsers, UpdatedAt = :updatedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":adminUsers": &types.AttributeValueMemberL{Value: adminsList},
			":updatedAt":  &types.AttributeValueMemberS{Value: now},
		},
		ConditionExpression: aws.String("attribute_exists(OrganizationId)"),
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, updateInput)
	if err != nil {
		svc.logger.Printf("Failed to remove org admin: %v", err)
		return fmt.Errorf("failed to remove org admin: %w", err)
	}

	svc.logger.Printf("Successfully removed admin %s from organization %s", adminUserName, organizationId)
	return nil
}

// GetOrgAdmins returns all active admins for an organization
func (svc *OrgServiceV2) GetOrgAdmins(organizationId string, requestingUser string) ([]OrgAdmin, error) {
	// Verify requesting user is org admin
	isAdmin, err := svc.IsOrgAdmin(organizationId, requestingUser)
	if err != nil {
		return nil, err
	}
	if !isAdmin {
		return nil, fmt.Errorf("user %s is not an admin of organization %s", requestingUser, organizationId)
	}

	// Get organization
	org, err := svc.GetOrganization(organizationId)
	if err != nil {
		return nil, err
	}

	// Return only active admins
	var activeAdmins []OrgAdmin
	for _, admin := range org.AdminUsers {
		if admin.IsActive {
			activeAdmins = append(activeAdmins, admin)
		}
	}

	return activeAdmins, nil
}

// reactivateAdmin is a helper function to reactivate an existing inactive admin
func (svc *OrgServiceV2) reactivateAdmin(organizationId string, adminUserName string, newRole OrgAdminRole) error {
	org, err := svc.GetOrganization(organizationId)
	if err != nil {
		return err
	}

	var updatedAdmins []OrgAdmin
	now := time.Now().UTC().Format(time.RFC3339)

	for _, admin := range org.AdminUsers {
		if admin.UserName == adminUserName {
			admin.IsActive = true
			admin.Role = newRole
			admin.AddedAt = now
		}
		updatedAdmins = append(updatedAdmins, admin)
	}

	// Marshal and update
	adminsList, err := attributevalue.MarshalList(updatedAdmins)
	if err != nil {
		return fmt.Errorf("failed to marshal admins list: %w", err)
	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", organizationId)},
			"SK": &types.AttributeValueMemberS{Value: "METADATA"},
		},
		UpdateExpression: aws.String("SET AdminUsers = :adminUsers, UpdatedAt = :updatedAt"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":adminUsers": &types.AttributeValueMemberL{Value: adminsList},
			":updatedAt":  &types.AttributeValueMemberS{Value: now},
		},
	}

	_, err = svc.dynamodbClient.UpdateItem(svc.ctx, updateInput)
	if err != nil {
		return fmt.Errorf("failed to reactivate admin: %w", err)
	}

	return nil
}

// AddOrgUser adds a user to an organization (separate from admin users in metadata)
// For this release, users can only be part of one organization
func (svc *OrgServiceV2) AddOrgUser(organizationId, userName, displayName string, role OrgAdminRole) error {
	// Check if user can join this organization (single org constraint)
	if err := svc.CheckUserCanJoinOrganization(userName, organizationId); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	orgUser := OrgUser{
		PK:             fmt.Sprintf("ORG#%s", organizationId),
		SK:             fmt.Sprintf("USER#%s", userName),
		GSI1PK:         fmt.Sprintf("USER#%s", userName),
		GSI1SK:         fmt.Sprintf("ORG#%s", organizationId),
		OrganizationId: organizationId,
		UserName:       userName,
		DisplayName:    displayName,
		Role:           role,
		JoinedAt:       now,
		IsActive:       true,
		UpdatedAt:      now,
	}

	// Get display name from employee service if available
	if svc.employeeSvc != nil && displayName == "" {
		employee, err := svc.employeeSvc.GetEmployeeDataByUserName(userName)
		if err == nil && employee.DisplayName != "" {
			orgUser.DisplayName = employee.DisplayName
		}
	}

	userItem, err := attributevalue.MarshalMap(orgUser)
	if err != nil {
		return fmt.Errorf("failed to marshal org user: %w", err)
	}

	_, err = svc.dynamodbClient.PutItem(svc.ctx, &dynamodb.PutItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Item:      userItem,
	})

	if err != nil {
		return fmt.Errorf("failed to add org user: %w", err)
	}

	return nil
}

// RemoveOrgUser removes a user from an organization
func (svc *OrgServiceV2) RemoveOrgUser(organizationId, userName string) error {
	_, err := svc.dynamodbClient.DeleteItem(svc.ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(svc.OrganizationTable),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", organizationId)},
			"SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userName)},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to remove org user: %w", err)
	}

	return nil
}

// GetOrgUsers retrieves all users for an organization
func (svc *OrgServiceV2) GetOrgUsers(organizationId string) ([]OrgMember, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(svc.OrganizationTable),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":pk":        &types.AttributeValueMemberS{Value: fmt.Sprintf("ORG#%s", organizationId)},
			":sk_prefix": &types.AttributeValueMemberS{Value: "USER#"},
		},
	}

	result, err := svc.dynamodbClient.Query(svc.ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query org users: %w", err)
	}

	var orgUsers []OrgUser
	err = attributevalue.UnmarshalListOfMaps(result.Items, &orgUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal org users: %w", err)
	}

	// Convert to OrgMember format
	members := make([]OrgMember, len(orgUsers))
	for i, user := range orgUsers {
		members[i] = OrgMember{
			UserName:    user.UserName,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			JoinedAt:    user.JoinedAt,
			IsActive:    user.IsActive,
		}
	}

	return members, nil
}

// GetUserOrganizations retrieves all organizations for a user
func (svc *OrgServiceV2) GetUserOrganizations(userName string) ([]Organization, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(svc.OrganizationTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :gsi1pk AND begins_with(GSI1SK, :gsi1sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk":        &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userName)},
			":gsi1sk_prefix": &types.AttributeValueMemberS{Value: "ORG#"},
		},
	}

	result, err := svc.dynamodbClient.Query(svc.ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query user organizations: %w", err)
	}

	var orgUsers []OrgUser
	err = attributevalue.UnmarshalListOfMaps(result.Items, &orgUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal org users: %w", err)
	}

	// Get organization details for each organization
	var organizations []Organization
	for _, orgUser := range orgUsers {
		if orgUser.IsActive {
			org, err := svc.GetOrganization(orgUser.OrganizationId)
			if err == nil {
				organizations = append(organizations, *org)
			}
		}
	}

	return organizations, nil
}

// CheckUserCanJoinOrganization validates if a user can join an organization
// For this release, users can only be part of one organization
func (svc *OrgServiceV2) CheckUserCanJoinOrganization(userName, organizationId string) error {
	existingOrgs, err := svc.GetUserOrganizations(userName)
	if err != nil {
		return fmt.Errorf("failed to check existing user organizations: %w", err)
	}

	// If user is already in an organization, check if it's the same one
	if len(existingOrgs) > 0 {
		for _, org := range existingOrgs {
			if org.OrganizationId != organizationId {
				return fmt.Errorf("user %s is already a member of organization %s (%s). Users can only belong to one organization", userName, org.OrganizationId, org.OrgName)
			}
		}
	}

	return nil
}

// GetUserOrganization returns the organization that a user belongs to
// For this release, users can only be part of one organization
func (svc *OrgServiceV2) GetUserOrganization(userIdentifier string) (*Organization, error) {
	orgs, err := svc.GetUserOrganizations(userIdentifier)
	if err != nil {
		return nil, err
	}

	if len(orgs) == 0 {
		return nil, fmt.Errorf("no organization found for user")
	}

	// For this release, enforce single organization membership
	if len(orgs) > 1 {
		svc.logger.Printf("Warning: User %s found in multiple organizations (%d), returning first one", userIdentifier, len(orgs))
	}

	// Return the first organization
	return &orgs[0], nil
}

// TransferUserToOrganization moves a user from their current organization to a new one
// This is an admin operation that bypasses the single organization constraint
func (svc *OrgServiceV2) TransferUserToOrganization(userName, fromOrgId, toOrgId string, newRole OrgAdminRole, requestingUser string) error {
	// Verify requesting user is admin of the target organization
	isAdmin, err := svc.IsOrgAdmin(toOrgId, requestingUser)
	if err != nil {
		return err
	}
	if !isAdmin {
		return fmt.Errorf("user %s is not an admin of target organization %s", requestingUser, toOrgId)
	}

	// Remove user from current organization if specified
	if fromOrgId != "" {
		if err := svc.RemoveOrgUser(fromOrgId, userName); err != nil {
			return fmt.Errorf("failed to remove user from current organization: %w", err)
		}
	} else {
		// Remove from all current organizations
		existingOrgs, err := svc.GetUserOrganizations(userName)
		if err != nil {
			return fmt.Errorf("failed to get user's current organizations: %w", err)
		}

		for _, org := range existingOrgs {
			if err := svc.RemoveOrgUser(org.OrganizationId, userName); err != nil {
				svc.logger.Printf("Warning: failed to remove user %s from organization %s: %v", userName, org.OrganizationId, err)
			}
		}
	}

	// Add user to new organization (this will now succeed since they're no longer in another org)
	return svc.AddOrgUser(toOrgId, userName, "", newRole)
}

// GetUserOrganizationMembership returns information about user's organization membership
func (svc *OrgServiceV2) GetUserOrganizationMembership(userName string) (*OrgUser, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String(svc.OrganizationTable),
		IndexName:              aws.String("GSI1"),
		KeyConditionExpression: aws.String("GSI1PK = :gsi1pk AND begins_with(GSI1SK, :gsi1sk_prefix)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":gsi1pk":        &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", userName)},
			":gsi1sk_prefix": &types.AttributeValueMemberS{Value: "ORG#"},
		},
		Limit: aws.Int32(1), // We only expect one organization per user
	}

	result, err := svc.dynamodbClient.Query(svc.ctx, queryInput)
	if err != nil {
		return nil, fmt.Errorf("failed to query user organization membership: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("user %s is not a member of any organization", userName)
	}

	if len(result.Items) > 1 {
		svc.logger.Printf("Warning: User %s found in multiple organizations (%d), returning first one", userName, len(result.Items))
	}

	var orgUser OrgUser
	err = attributevalue.UnmarshalMap(result.Items[0], &orgUser)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal org user: %w", err)
	}

	return &orgUser, nil
}
