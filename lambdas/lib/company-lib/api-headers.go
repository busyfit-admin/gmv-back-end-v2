package Companylib

import (
	"strings"
)

// Define common headers for all APIs
var commonHeaders = []string{
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"X-Amz-Date",
	"X-Api-Key",
	"X-Amz-Security-Token",
	"Content-Type",
	"Authorization",

	"get_type",
	"post_type",
	"patch_type",
	"delete_type",
}

// Define specific headers for each API
var apiHeaders = map[string][]string{
	"ProfileAPI": {
		"get-dashboard-profile",
		"get-rewards-profile",
		"get-teams-profile",
		"upload-profile-picture",
		"get-profile-edit-data",
		"patch-profile-data",
		"get-user-certificates",
	},
	"ProfileV2API": {
		"user-id",
		"profile-data",
		"content-type",
		"accept",
	},
	"UsersAPI": {
		"user-id",
		"get-user-by-username",
		"get-user-by-email",
		"get-user-by-ext-id",

		"basic-update-user-by-username",
		"full-update-user-by-username",

		"update-roles-by-username",
		"update-roles-by-emailid",
		"update-roles-by-external-id",
	},
	"TeamsAPI": {
		"get-all-teams",
		"get-team",
		"get-team-users",
		"get-team-managers",
		"get-user-teams",
		"get-manager-teams",
		"create-team",
		"add-users",
		"delete-users",

		"team-id",
		"related-id",
		"user-id",
		"manager-id",
	},
	"AppreciationsAPI": {

		"get-all-skills",
		"get-all-values",
		"get-all-milestones",
		"get-all-metrics",
		"get-all-appreciation-entity",
		"get-appreciation-entity",
		"post-skills",
		"post-values",
		"post-milestones",
		"post-metrics",
		"post-appreciation-entity",
		"post-engagement-send-kudos",

		"delete-skills",
		"delete-values",
		"delete-milestones",
		"delete-metrics",

		"entity-id",
		"last-evaluated-key",
		"appreciation-id",
		"entity-id",
		"file-data",

		"get-count-of-events-and-appreciations",
	},
	"CertificatesAPI": {
		"certificate-id",
		"get-all-certificates",
		"get-certificate",
		"create-certificates",
		"update-certificates",
		"delete-certificates",
		"transfer-certificate",
	},
	"RewardsAPI": {
		"TrackingId",
		"CardId",
		"rule-id",
		"reward_type_patch",
		"reward_unit_patch",
		"reward_rule_patch",
		"create-card-template",
		"card-checkout",
		"get-card-template",
		"get-all-card-template",
		"get-all-cards",
		"redeem-card",
		"get-cards-order-history",
		"get_rule_settings",
		"get_reward_rules",
		"get_reward_logs",
		"get_reward_admin_points",
		"get_users_reward_logs",
		"get_reward_admin_logs",
	},
	"SurveysAPI": {
		"get-survey-questions",
		"get-survey-responses",
		"submit-survey-response",
	},
}

// Function to initialize headers for a specific API
func GetHeadersForAPI(apiName string) map[string]string {
	headers := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "*",
	}

	// Combine common headers with specific headers for the API
	if specificHeaders, exists := apiHeaders[apiName]; exists {
		allHeaders := append(commonHeaders, specificHeaders...)
		headers["Access-Control-Allow-Headers"] = strings.Join(allHeaders, ",")
	} else {
		// Default to common headers if no specific headers exist for the API
		headers["Access-Control-Allow-Headers"] = strings.Join(commonHeaders, ",")
	}

	return headers
}

func ExampleGet() {
	// Example: Retrieve headers for the UserAPI
	userAPIHeaders := GetHeadersForAPI("UserAPI")
	productAPIHeaders := GetHeadersForAPI("ProductAPI")

	_ = userAPIHeaders
	_ = productAPIHeaders
}
