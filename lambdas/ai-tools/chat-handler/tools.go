package main

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	bedrockdoc "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	ctrl "github.com/busyfit-admin/saas-integrated-apis/lambdas/ai-tools/controllers"
)

// toolExecutorFn is the signature every tool executor must satisfy.
type toolExecutorFn func(input map[string]interface{}, svc *ctrl.Service, chatCtx ChatContext) (interface{}, error)

// toolRegistry maps Bedrock tool names to their Go executor functions.
var toolRegistry = map[string]toolExecutorFn{
	// Employee
	"get_employee_information": execGetEmployeeInformation,
	"get_all_employees":        execGetAllEmployees,
	"find_employee_by_email":   execFindEmployeeByEmail,
	"get_all_employee_groups":  execGetAllEmployeeGroups,
	// Team
	"get_team_information":      execGetTeamInformation,
	"get_all_org_teams":         execGetAllOrgTeams,
	"get_team_members":          execGetTeamMembers,
	"get_team_member_directory": execGetTeamMemberDirectory,
	"get_user_teams":            execGetUserTeams,
	"is_team_admin":             execIsTeamAdmin,
	// Org
	"get_org_info":   execGetOrgInfo,
	"get_org_admins": execGetOrgAdmins,
	"get_org_users":  execGetOrgUsers,
	"is_org_admin":   execIsOrgAdmin,
	// Performance cycles
	"get_performance_cycles":        execGetPerformanceCycles,
	"get_performance_cycle_details": execGetPerformanceCycleDetails,
	"get_cycle_analytics":           execGetCycleAnalytics,
	"get_all_quarters":              execGetAllQuarters,
	"get_quarter_details":           execGetQuarterDetails,
	"get_quarter_analytics":         execGetQuarterAnalytics,
	"get_quarter_meeting_notes":     execGetQuarterMeetingNotes,
	// KPIs / OKRs
	"get_all_kpis":   execGetAllKPIs,
	"get_kpi_detail": execGetKPIDetail,
	"get_all_okrs":   execGetAllOKRs,
	"get_okr_detail": execGetOKRDetail,
	// Org goals
	"get_org_goal_detail":         execGetOrgGoalDetail,
	"get_team_org_goals":          execGetTeamOrgGoals,
	"get_org_goal_sub_items":      execGetOrgGoalSubItems,
	"get_user_goals_for_org_goal": execGetUserGoalsForOrgGoal,
	"get_goal_ladder_up":          execGetGoalLadderUp,
	"get_goal_value_history":      execGetGoalValueHistory,
	"get_goal_tasks":              execGetGoalTasks,
	"get_goal_tagged_teams":       execGetGoalTaggedTeams,
	// User goals
	"get_my_goals":          execGetMyGoals,
	"get_my_goal":           execGetMyGoal,
	"get_goal_linked_tasks": execGetGoalLinkedTasks,
	"get_goal_comments":     execGetGoalComments,
	// User tasks
	"get_all_tasks": execGetAllTasks,
	"get_task":      execGetTask,
	// User meetings
	"get_my_meetings": execGetMyMeetings,
	"get_meeting":     execGetMeeting,
	// User appreciations/feedback
	"get_my_appreciations":     execGetMyAppreciations,
	"get_my_feedback_requests": execGetMyFeedbackRequests,
	// Manager — team member performance
	"get_team_performance_members":   execGetTeamPerformanceMembers,
	"get_member_goals":               execGetMemberGoals,
	"get_member_tasks":               execGetMemberTasks,
	"get_member_meetings":            execGetMemberMeetings,
	"get_member_appreciations":       execGetMemberAppreciations,
	"get_member_manager_comments":    execGetMemberManagerComments,
	"get_member_performance_summary": execGetMemberPerformanceSummary,
}

// executeToolCall dispatches a tool call from Bedrock to the correct executor and
// returns a JSON-encoded string suitable for use as a Bedrock ToolResultBlock text.
func executeToolCall(toolName string, inputDoc bedrockdoc.Interface, svc *ctrl.Service, chatCtx ChatContext) (string, error) {
	executor, ok := toolRegistry[toolName]
	if !ok {
		return fmt.Sprintf(`{"error":"unknown tool: %s"}`, toolName), nil
	}
	var input map[string]interface{}
	if err := inputDoc.UnmarshalSmithyDocument(&input); err != nil {
		return fmt.Sprintf(`{"error":"failed to parse tool input: %v"}`, err), nil
	}
	result, err := executor(input, svc, chatCtx)
	if err != nil {
		return jsonStr(map[string]interface{}{"error": err.Error()}), nil
	}
	return jsonStr(result), nil
}

// buildToolList returns the complete list of Bedrock-compatible tool definitions.
func buildToolList() []bedrocktypes.Tool {
	defs := []struct {
		name, desc string
		schema     map[string]interface{}
	}{
		// ----- Employee -----
		{
			"get_employee_information",
			"Retrieve the full profile of an employee by their username, including name, email, role, team memberships and metadata.",
			obj(prop("userName", str("The employee's unique username (e.g. john.doe)")), req("userName")),
		},
		{
			"get_all_employees",
			"List all employees in an organisation.",
			obj(prop("orgId", str("Organisation ID to list employees for")), req("orgId")),
		},
		{
			"find_employee_by_email",
			"Look up an employee by their email address.",
			obj(prop("email", str("Employee email address")), req("email")),
		},
		{
			"get_all_employee_groups",
			"Return the Cognito group memberships for an employee.",
			obj(prop("userName", str("Employee username")), req("userName")),
		},
		// ----- Team -----
		{
			"get_team_information",
			"Get team metadata (name, description, member count, status) for a teamId.",
			obj(prop("teamId", str("Team ID")), req("teamId")),
		},
		{
			"get_all_org_teams",
			"List every team that belongs to an organisation.",
			obj(prop("orgId", str("Organisation ID")), req("orgId")),
		},
		{
			"get_team_members",
			"Return the full member list for a team, including roles and join dates.",
			obj(prop("teamId", str("Team ID")), req("teamId")),
		},
		{
			"get_team_member_directory",
			"Return a simplified directory for a team (userName, displayName, initials, role).",
			obj(prop("teamId", str("Team ID")), req("teamId")),
		},
		{
			"get_user_teams",
			"Return the list of teams a user belongs to.",
			obj(
				prop("userName", str("Employee username")),
				prop("cognitoId", str("Cognito sub (optional if userName is known)")),
				req("userName"),
			),
		},
		{
			"is_team_admin",
			"Check whether a specific user is a team admin.",
			obj(prop("teamId", str("Team ID")), prop("userName", str("Employee username")), req("teamId", "userName")),
		},
		// ----- Org -----
		{
			"get_org_info",
			"Return organisation details (name, settings, subscription).",
			obj(prop("orgId", str("Organisation ID")), req("orgId")),
		},
		{
			"get_org_admins",
			"List all admin users for an organisation.",
			obj(prop("orgId", str("Organisation ID")), req("orgId")),
		},
		{
			"get_org_users",
			"List all users who belong to an organisation.",
			obj(prop("orgId", str("Organisation ID")), req("orgId")),
		},
		{
			"is_org_admin",
			"Check whether a specific user is an org admin.",
			obj(prop("orgId", str("Organisation ID")), prop("userName", str("Employee username")), req("orgId", "userName")),
		},
		// ----- Performance cycles -----
		{
			"get_performance_cycles",
			"List performance cycles for an organisation, with optional status filter.",
			obj(
				prop("orgId", str("Organisation ID")),
				prop("status", str("Optional filter: active | archived | completed")),
				prop("includeQuarters", boolean("Include quarters in response")),
				prop("includeKPIs", boolean("Include KPIs in response")),
				prop("includeOKRs", boolean("Include OKRs in response")),
				req("orgId"),
			),
		},
		{
			"get_performance_cycle_details",
			"Get detailed information about a single performance cycle.",
			obj(
				prop("cycleId", str("Performance cycle ID")),
				prop("includeQuarters", boolean("")),
				prop("includeKPIs", boolean("")),
				prop("includeOKRs", boolean("")),
				prop("includeAnalytics", boolean("")),
				req("cycleId"),
			),
		},
		{
			"get_cycle_analytics",
			"Return aggregated analytics (completion rates, distributions) for a cycle.",
			obj(prop("cycleId", str("Performance cycle ID")), req("cycleId")),
		},
		{
			"get_all_quarters",
			"List all quarters within a performance cycle.",
			obj(prop("cycleId", str("Performance cycle ID")), req("cycleId")),
		},
		{
			"get_quarter_details",
			"Get details for a single quarter including goals and notes.",
			obj(
				prop("quarterId", str("Quarter ID")),
				prop("includeKPIs", boolean("")),
				prop("includeOKRs", boolean("")),
				req("quarterId"),
			),
		},
		{
			"get_quarter_analytics",
			"Return analytics summary for a quarter.",
			obj(prop("quarterId", str("Quarter ID")), req("quarterId")),
		},
		{
			"get_quarter_meeting_notes",
			"Return meeting notes recorded within a quarter.",
			obj(prop("quarterId", str("Quarter ID")), req("quarterId")),
		},
		// ----- KPIs -----
		{
			"get_all_kpis",
			"List KPIs for an organisation. Filters by status, cycle, or whether sub-KPIs should be expanded.",
			obj(
				prop("orgId", str("Organisation ID")),
				prop("status", str("Optional filter: active | completed | draft")),
				prop("cycleId", str("Optional cycle ID filter")),
				prop("includeSubKPIs", boolean("")),
				req("orgId"),
			),
		},
		{
			"get_kpi_detail",
			"Get full detail for a single KPI, optionally including sub-KPIs and value history.",
			obj(
				prop("kpiId", str("KPI ID")),
				prop("includeSubKPIs", boolean("")),
				prop("includeValueHistory", boolean("")),
				req("kpiId"),
			),
		},
		// ----- OKRs -----
		{
			"get_all_okrs",
			"List OKRs for an organisation with optional filters.",
			obj(
				prop("orgId", str("Organisation ID")),
				prop("status", str("Optional status filter")),
				prop("cycleId", str("Optional cycle ID filter")),
				prop("includeKeyResults", boolean("")),
				req("orgId"),
			),
		},
		{
			"get_okr_detail",
			"Get full detail for a single OKR.",
			obj(
				prop("okrId", str("OKR ID")),
				prop("includeKeyResults", boolean("")),
				prop("includeProgressHistory", boolean("")),
				req("okrId"),
			),
		},
		// ----- Org goals -----
		{
			"get_org_goal_detail",
			"Get comprehensive detail for an org-level goal including history, teams, and sub-items.",
			obj(
				prop("goalId", str("Goal ID")),
				prop("includeValueHistory", boolean("")),
				prop("includeTaggedTeams", boolean("")),
				prop("includeSubItems", boolean("")),
				prop("includeLadderUp", boolean("")),
				req("goalId"),
			),
		},
		{
			"get_team_org_goals",
			"Return org-level goals assigned to a specific team.",
			obj(
				prop("teamId", str("Team ID")),
				prop("orgId", str("Organisation ID")),
				prop("goalType", str("Optional type filter: kpi | okr | objective")),
				req("teamId", "orgId"),
			),
		},
		{
			"get_org_goal_sub_items",
			"List sub-items (milestones / key results) that belong to an org goal.",
			obj(prop("goalId", str("Goal ID")), req("goalId")),
		},
		{
			"get_user_goals_for_org_goal",
			"Find all user-level goals that are linked to a given org goal, with linked tasks and a status summary.",
			obj(
				prop("orgGoalId", str("Org-level Goal ID")),
				prop("statusFilter", str("Optional: on-track | ahead | at-risk | behind | completed")),
				req("orgGoalId"),
			),
		},
		{
			"get_goal_ladder_up",
			"Return ladder-up requests for an org goal.",
			obj(
				prop("goalId", str("Goal ID")),
				prop("status", str("Optional filter: pending | approved | rejected")),
				req("goalId"),
			),
		},
		{
			"get_goal_value_history",
			"Return value-update history for an org goal within an optional date range.",
			obj(
				prop("goalId", str("Goal ID")),
				prop("startDate", str("Optional ISO-8601 start date")),
				prop("endDate", str("Optional ISO-8601 end date")),
				req("goalId"),
			),
		},
		{
			"get_goal_tasks",
			"Return tasks assigned to an org goal, optionally filtered by user and/or status.",
			obj(
				prop("goalId", str("Goal ID")),
				prop("userName", str("Optional: limit to one user's tasks")),
				prop("status", str("Optional status filter")),
				req("goalId"),
			),
		},
		{
			"get_goal_tagged_teams",
			"Return the list of teams tagged to an org goal.",
			obj(prop("goalId", str("Goal ID")), req("goalId")),
		},
		// ----- User goals -----
		{
			"get_my_goals",
			"Return goals for a user/team combination. Defaults to the calling user if userName is omitted.",
			obj(
				prop("userName", str("Username; defaults to caller if omitted")),
				prop("teamId", str("Team ID; defaults to caller's team if omitted")),
				prop("type", str("Optional goal type filter")),
				prop("status", str("Optional status filter")),
			),
		},
		{
			"get_my_goal",
			"Return detailed information about a single user goal.",
			obj(
				prop("goalId", str("Goal ID")),
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
				req("goalId"),
			),
		},
		{
			"get_goal_linked_tasks",
			"Return all tasks linked to a specific user goal.",
			obj(
				prop("goalId", str("Goal ID")),
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
				req("goalId"),
			),
		},
		{
			"get_goal_comments",
			"Return comments on a user goal.",
			obj(
				prop("goalId", str("Goal ID")),
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
				req("goalId"),
			),
		},
		// ----- User tasks -----
		{
			"get_all_tasks",
			"List tasks for a user, with optional filters by goal, status, and done flag.",
			obj(
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
				prop("goalId", str("Optional: limit to tasks for one goal")),
				prop("status", str("Optional status filter")),
				prop("done", boolean("Optional done filter")),
			),
		},
		{
			"get_task",
			"Return details for a single task.",
			obj(
				prop("taskId", str("Task ID")),
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
				req("taskId"),
			),
		},
		// ----- User meetings -----
		{
			"get_my_meetings",
			"Return 1-on-1 meeting records for a user.",
			obj(
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
				prop("status", str("Optional status filter: scheduled | completed | cancelled")),
			),
		},
		{
			"get_meeting",
			"Return details for a single meeting.",
			obj(
				prop("meetingId", str("Meeting ID")),
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
				req("meetingId"),
			),
		},
		// ----- Appreciations / feedback -----
		{
			"get_my_appreciations",
			"Return appreciation records received or given by a user.",
			obj(
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
			),
		},
		{
			"get_my_feedback_requests",
			"Return feedback requests for a user.",
			obj(
				prop("userName", str("Username; defaults to caller")),
				prop("teamId", str("Team ID; defaults to caller's team")),
				prop("status", str("Optional status filter")),
			),
		},
		// ----- Manager — team member performance -----
		{
			"get_team_performance_members",
			"Return all team members with their performance summary (for manager/admin view).",
			obj(prop("teamId", str("Team ID")), req("teamId")),
		},
		{
			"get_member_goals",
			"Return goals for a specific team member (manager/admin use).",
			obj(prop("teamId", str("Team ID")), prop("memberId", str("Target member username")), req("teamId", "memberId")),
		},
		{
			"get_member_tasks",
			"Return tasks for a specific team member.",
			obj(
				prop("teamId", str("Team ID")),
				prop("memberId", str("Target member username")),
				prop("status", str("Optional status filter")),
				req("teamId", "memberId"),
			),
		},
		{
			"get_member_meetings",
			"Return 1-on-1 meetings for a team member.",
			obj(prop("teamId", str("Team ID")), prop("memberId", str("Target member username")), req("teamId", "memberId")),
		},
		{
			"get_member_appreciations",
			"Return appreciations for a team member.",
			obj(prop("teamId", str("Team ID")), prop("memberId", str("Target member username")), req("teamId", "memberId")),
		},
		{
			"get_member_manager_comments",
			"Return manager review comments for a team member.",
			obj(prop("teamId", str("Team ID")), prop("memberId", str("Target member username")), req("teamId", "memberId")),
		},
		{
			"get_member_performance_summary",
			"Return a full performance summary for a team member, including goals, tasks, meetings, and appreciations.",
			obj(prop("teamId", str("Team ID")), prop("memberId", str("Target member username")), req("teamId", "memberId")),
		},
	}

	tools := make([]bedrocktypes.Tool, 0, len(defs))
	for _, d := range defs {
		desc := d.desc
		tools = append(tools, &bedrocktypes.ToolMemberToolSpec{
			Value: bedrocktypes.ToolSpecification{
				Name:        aws.String(d.name),
				Description: aws.String(desc),
				InputSchema: &bedrocktypes.ToolInputSchemaMemberJson{
					Value: bedrockdoc.NewLazyDocument(d.schema),
				},
			},
		})
	}
	return tools
}

// ==================== JSON Schema helpers ====================

func obj(parts ...map[string]interface{}) map[string]interface{} {
	properties := map[string]interface{}{}
	required := []interface{}{}
	for _, p := range parts {
		for k, v := range p {
			if k == "__required__" {
				if reqs, ok := v.([]interface{}); ok {
					required = append(required, reqs...)
				}
			} else {
				properties[k] = v
			}
		}
	}
	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func prop(name string, def map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{name: def}
}

func str(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "string", "description": desc}
}

func boolean(desc string) map[string]interface{} {
	return map[string]interface{}{"type": "boolean", "description": desc}
}

func req(names ...string) map[string]interface{} {
	reqs := make([]interface{}, len(names))
	for i, n := range names {
		reqs[i] = n
	}
	return map[string]interface{}{"__required__": reqs}
}

// ==================== Helper utilities ====================

func getStr(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func jsonStr(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"marshal failure: %v"}`, err)
	}
	return string(b)
}

// withDefault returns s if non-empty, otherwise returns fallback.
func withDefault(s, fallback string) string {
	if s != "" {
		return s
	}
	return fallback
}

// ==================== Tool executors ====================

func execGetEmployeeInformation(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetEmployeeBasicData(getStr(in, "userName"))
}

func execGetAllEmployees(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetAllEmployees()
}

func execFindEmployeeByEmail(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.FindEmployeeByEmail(getStr(in, "email"))
}

func execGetAllEmployeeGroups(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetAllEmployeeGroups()
}

func execGetTeamInformation(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.TeamInformation(getStr(in, "teamId"))
}

func execGetAllOrgTeams(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetAllOrgTeams(getStr(in, "orgId"))
}

func execGetTeamMembers(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetTeamMembers(getStr(in, "teamId"))
}

func execGetTeamMemberDirectory(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetTeamMemberDirectory(getStr(in, "teamId"))
}

func execGetUserTeams(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetUserTeams(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "cognitoId"), ctx.CallerCognitoID),
	)
}

func execIsTeamAdmin(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.IsTeamAdmin(getStr(in, "teamId"), getStr(in, "userName"))
}

func execGetOrgInfo(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetOrgInfo(withDefault(getStr(in, "orgId"), ctx.CallerOrgID))
}

func execGetOrgAdmins(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetOrgAdmins(withDefault(getStr(in, "orgId"), ctx.CallerOrgID), ctx.CallerUserName)
}

func execGetOrgUsers(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetOrgUsers(withDefault(getStr(in, "orgId"), ctx.CallerOrgID))
}

func execIsOrgAdmin(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.IsOrgAdmin(
		withDefault(getStr(in, "orgId"), ctx.CallerOrgID),
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
	)
}

func execGetPerformanceCycles(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetPerformanceCycles(
		withDefault(getStr(in, "orgId"), ctx.CallerOrgID),
		ctrl.PerformanceCycleFilters{Status: getStr(in, "status")},
		getBool(in, "includeQuarters"),
		getBool(in, "includeKPIs"),
		getBool(in, "includeOKRs"),
	)
}

func execGetPerformanceCycleDetails(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetPerformanceCycleDetails(
		getStr(in, "cycleId"),
		getBool(in, "includeQuarters"),
		getBool(in, "includeKPIs"),
		getBool(in, "includeOKRs"),
		getBool(in, "includeAnalytics"),
	)
}

func execGetCycleAnalytics(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetCycleAnalytics(getStr(in, "cycleId"))
}

func execGetAllQuarters(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetAllQuarters(getStr(in, "cycleId"))
}

func execGetQuarterDetails(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetQuarterDetails(
		getStr(in, "quarterId"),
		getBool(in, "includeKPIs"),
		getBool(in, "includeOKRs"),
		false, // includeMeetingNotes
		false, // includePendingReviews
	)
}

func execGetQuarterAnalytics(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetQuarterAnalytics(getStr(in, "quarterId"))
}

func execGetQuarterMeetingNotes(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetQuarterMeetingNotes(getStr(in, "quarterId"), "", "")
}

func execGetAllKPIs(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetAllKPIs(
		withDefault(getStr(in, "orgId"), ctx.CallerOrgID),
		ctrl.KPIFilters{
			Status:  getStr(in, "status"),
			CycleID: getStr(in, "cycleId"),
		},
		getBool(in, "includeSubKPIs"),
	)
}

func execGetKPIDetail(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetKPIDetail(
		getStr(in, "kpiId"),
		getBool(in, "includeSubKPIs"),
		getBool(in, "includeValueHistory"),
	)
}

func execGetAllOKRs(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetAllOKRs(
		withDefault(getStr(in, "orgId"), ctx.CallerOrgID),
		ctrl.OKRFilters{
			Status:  getStr(in, "status"),
			CycleID: getStr(in, "cycleId"),
		},
		getBool(in, "includeKeyResults"),
	)
}

func execGetOKRDetail(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetOKRDetail(
		getStr(in, "okrId"),
		getBool(in, "includeKeyResults"),
		getBool(in, "includeProgressHistory"),
	)
}

func execGetOrgGoalDetail(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetOrgGoalDetail(
		getStr(in, "goalId"),
		getBool(in, "includeValueHistory"),
		getBool(in, "includeTaggedTeams"),
		getBool(in, "includeSubItems"),
		getBool(in, "includeLadderUp"),
		ctx.CallerUserName,
	)
}

func execGetTeamOrgGoals(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetTeamOrgGoals(
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		withDefault(getStr(in, "orgId"), ctx.CallerOrgID),
		getStr(in, "goalType"),
	)
}

func execGetOrgGoalSubItems(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetOrgGoalSubItems(getStr(in, "goalId"))
}

func execGetUserGoalsForOrgGoal(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetUserGoalsForOrgGoal(getStr(in, "orgGoalId"), getStr(in, "statusFilter"))
}

func execGetGoalLadderUp(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetGoalLadderUp(getStr(in, "goalId"), getStr(in, "status"))
}

func execGetGoalValueHistory(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetGoalValueHistory(getStr(in, "goalId"), getStr(in, "startDate"), getStr(in, "endDate"))
}

func execGetGoalTasks(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetOrgGoalTasks(getStr(in, "goalId"), getStr(in, "userName"), getStr(in, "status"))
}

func execGetGoalTaggedTeams(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetGoalTaggedTeams(getStr(in, "goalId"))
}

func execGetMyGoals(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMyGoals(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		ctrl.GoalFilters{Type: getStr(in, "type"), Status: getStr(in, "status")},
	)
}

func execGetMyGoal(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMyGoal(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		getStr(in, "goalId"),
	)
}

func execGetGoalLinkedTasks(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetGoalLinkedTasks(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		getStr(in, "goalId"),
	)
}

func execGetGoalComments(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetGoalComments(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		getStr(in, "goalId"),
	)
}

func execGetAllTasks(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	done := (*bool)(nil)
	if v, ok := in["done"]; ok {
		if b, ok := v.(bool); ok {
			done = &b
		}
	}
	return svc.GetAllTasks(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		ctrl.TaskFilters{
			GoalID: getStr(in, "goalId"),
			Status: getStr(in, "status"),
			Done:   done,
		},
	)
}

func execGetTask(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetTask(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		getStr(in, "taskId"),
	)
}

func execGetMyMeetings(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMyMeetings(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		getStr(in, "status"),
	)
}

func execGetMeeting(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMeeting(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		getStr(in, "meetingId"),
	)
}

func execGetMyAppreciations(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMyAppreciations(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
	)
}

func execGetMyFeedbackRequests(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMyFeedbackRequests(
		withDefault(getStr(in, "userName"), ctx.CallerUserName),
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		getStr(in, "status"),
	)
}

func execGetTeamPerformanceMembers(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetTeamPerformanceMembers(withDefault(getStr(in, "teamId"), ctx.CallerTeamID))
}

func execGetMemberGoals(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMemberGoals(
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		withDefault(getStr(in, "memberId"), ctx.TargetUserID),
	)
}

func execGetMemberTasks(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMemberTasks(
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		withDefault(getStr(in, "memberId"), ctx.TargetUserID),
		ctrl.TaskFilters{Status: getStr(in, "status")},
	)
}

func execGetMemberMeetings(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMemberMeetings(
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		withDefault(getStr(in, "memberId"), ctx.TargetUserID),
	)
}

func execGetMemberAppreciations(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMemberAppreciations(
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		withDefault(getStr(in, "memberId"), ctx.TargetUserID),
	)
}

func execGetMemberManagerComments(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMemberManagerComments(
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		withDefault(getStr(in, "memberId"), ctx.TargetUserID),
	)
}

func execGetMemberPerformanceSummary(in map[string]interface{}, svc *ctrl.Service, ctx ChatContext) (interface{}, error) {
	return svc.GetMemberPerformanceSummary(
		withDefault(getStr(in, "teamId"), ctx.CallerTeamID),
		withDefault(getStr(in, "memberId"), ctx.TargetUserID),
	)
}
