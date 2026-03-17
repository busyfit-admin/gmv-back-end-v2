package controllers

// ==================== DDB Key Helpers ====================

const (
	prefixUser = "USER#"
	prefixTeam = "TEAM#"

	skGoalPrefix           = "GOAL#"
	skTaskPrefix           = "TASK#"
	skMeetingPrefix        = "MEETING#"
	skAppreciationPrefix   = "APPR#"
	skFeedbackReqPrefix    = "FBREQ#"
	skCommentInfix         = "#CMMNT#"
	skManagerCommentPrefix = "MGRCMT#"
	skMemberReviewPrefix   = "REVIEW#MEMBER#"
)

// buildPK returns the per-user per-team partition key: USER#{userName}#TEAM#{teamID}
func buildPK(userName, teamID string) string {
	return prefixUser + userName + "#TEAM#" + teamID
}

// buildTeamPK returns the team-scoped partition key: TEAM#{teamID}
func buildTeamPK(teamID string) string {
	return prefixTeam + teamID
}

// ==================== Goal Types & Statuses ====================

type GoalType = string

const (
	GoalTypeIndividual GoalType = "individual"
	GoalTypeGrowth     GoalType = "growth"
	GoalTypeKPI        GoalType = "kpi"
	GoalTypeOKR        GoalType = "okr"
)

type GoalStatus = string

const (
	GoalStatusOnTrack   GoalStatus = "on-track"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusAhead     GoalStatus = "ahead"
	GoalStatusAtRisk    GoalStatus = "at-risk"
	GoalStatusBehind    GoalStatus = "behind"
)

// ==================== Task Types ====================

type TaskStatus = string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in-progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusClosed     TaskStatus = "closed"
)

// ==================== Meeting Statuses ====================

type MeetingStatus = string

const (
	MeetingStatusScheduled MeetingStatus = "scheduled"
	MeetingStatusCompleted MeetingStatus = "completed"
)

// ==================== DDB Record Types ====================

// GoalRecord is a single user-team goal stored in the performance hub table.
// PK=USER#{userName}#TEAM#{teamId}  SK=GOAL#{goalId}
type GoalRecord struct {
	PK          string `dynamodbav:"PK"`
	SK          string `dynamodbav:"SK"`
	GoalID      string `dynamodbav:"goalId"`
	UserName    string `dynamodbav:"userName"`
	Title       string `dynamodbav:"title"`
	Type        string `dynamodbav:"type"`
	Progress    int    `dynamodbav:"progress"`
	DueDate     string `dynamodbav:"dueDate"`
	Status      string `dynamodbav:"status"`
	Description string `dynamodbav:"description,omitempty"`
	OrgGoalID   string `dynamodbav:"orgGoalId,omitempty"`
	CreatedAt   string `dynamodbav:"createdAt"`
	UpdatedAt   string `dynamodbav:"updatedAt"`
}

// LinkedTaskRecord is a task optionally linked to a goal.
// PK=USER#{userName}#TEAM#{teamId}  SK=TASK#{taskId}
type LinkedTaskRecord struct {
	PK          string   `dynamodbav:"PK"`
	SK          string   `dynamodbav:"SK"`
	TaskID      string   `dynamodbav:"taskId"`
	TaskNumber  int      `dynamodbav:"taskNumber"`
	GoalID      string   `dynamodbav:"goalId,omitempty"`
	UserName    string   `dynamodbav:"userName"`
	Title       string   `dynamodbav:"title"`
	Description string   `dynamodbav:"description,omitempty"`
	Priority    string   `dynamodbav:"priority,omitempty"`
	Status      string   `dynamodbav:"status"`
	Done        bool     `dynamodbav:"done"`
	Tags        []string `dynamodbav:"tags,omitempty"`
	TimeHours   float64  `dynamodbav:"timeHours,omitempty"`
	TimeDays    float64  `dynamodbav:"timeDays,omitempty"`
	DueDate     string   `dynamodbav:"dueDate,omitempty"`
	CreatedAt   string   `dynamodbav:"createdAt"`
	UpdatedAt   string   `dynamodbav:"updatedAt,omitempty"`
}

// GoalCommentRecord is a comment on a goal (member or manager authored).
// PK=USER#{userName}#TEAM#{teamId}  SK=GOAL#{goalId}#CMMNT#{commentId}
type GoalCommentRecord struct {
	PK             string `dynamodbav:"PK"`
	SK             string `dynamodbav:"SK"`
	CommentID      string `dynamodbav:"commentId"`
	GoalID         string `dynamodbav:"goalId"`
	UserName       string `dynamodbav:"userName"`
	AuthorUserName string `dynamodbav:"authorUserName"`
	Author         string `dynamodbav:"author"`
	Initials       string `dynamodbav:"initials"`
	Role           string `dynamodbav:"role"` // "member" | "manager"
	Text           string `dynamodbav:"text"`
	Date           string `dynamodbav:"date"`
	CreatedAt      string `dynamodbav:"createdAt"`
}

// MeetingRecord is a 1-on-1 meeting record.
// PK=USER#{userName}#TEAM#{teamId}  SK=MEETING#{meetingId}
type MeetingRecord struct {
	PK          string   `dynamodbav:"PK"`
	SK          string   `dynamodbav:"SK"`
	MeetingID   string   `dynamodbav:"meetingId"`
	UserName    string   `dynamodbav:"userName"`
	Date        string   `dynamodbav:"date"`
	Status      string   `dynamodbav:"status"`
	ManagerName string   `dynamodbav:"managerName,omitempty"`
	ManagerRole string   `dynamodbav:"managerRole,omitempty"`
	Summary     string   `dynamodbav:"summary,omitempty"`
	Tags        []string `dynamodbav:"tags,omitempty"`
	ActionItems []string `dynamodbav:"actionItems,omitempty"`
	CreatedAt   string   `dynamodbav:"createdAt"`
}

// AppreciationRecord is a recognition received by an employee.
// PK=USER#{userName}#TEAM#{teamId}  SK=APPR#{appreciationId}
type AppreciationRecord struct {
	PK             string `dynamodbav:"PK"`
	SK             string `dynamodbav:"SK"`
	AppreciationID string `dynamodbav:"appreciationId"`
	UserName       string `dynamodbav:"userName"`
	From           string `dynamodbav:"from"`
	FromInitials   string `dynamodbav:"fromInitials"`
	FromRole       string `dynamodbav:"fromRole,omitempty"`
	Message        string `dynamodbav:"message"`
	Skill          string `dynamodbav:"skill,omitempty"`
	Date           string `dynamodbav:"date"`
	Category       string `dynamodbav:"category,omitempty"`
	CreatedAt      string `dynamodbav:"createdAt"`
}

// FeedbackRequestRecord is a feedback request sent by an employee to a colleague.
// PK=USER#{userName}#TEAM#{teamId}  SK=FBREQ#{requestId}
type FeedbackRequestRecord struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	RequestID string `dynamodbav:"requestId"`
	UserName  string `dynamodbav:"userName"` // sender
	To        string `dynamodbav:"to"`       // recipient userName
	Message   string `dynamodbav:"message"`
	Date      string `dynamodbav:"date"`
	Status    string `dynamodbav:"status"` // "pending" | "completed"
	CreatedAt string `dynamodbav:"createdAt"`
}

// ManagerCommentRecord is a manager-authored comment on a team member.
// PK=USER#{memberUserName}#TEAM#{teamID}  SK=MGRCMT#{commentId}
type ManagerCommentRecord struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	CommentID string `dynamodbav:"commentId"`
	TeamID    string `dynamodbav:"teamId"`
	MemberID  string `dynamodbav:"memberId"`
	Author    string `dynamodbav:"author"`
	Initials  string `dynamodbav:"initials"`
	Text      string `dynamodbav:"text"`
	Type      string `dynamodbav:"type"` // "feedback" | "coaching" | "general"
	Date      string `dynamodbav:"date"`
	CreatedAt string `dynamodbav:"createdAt"`
}

// TeamMemberReviewRecord tracks per-member review lifecycle state within a team.
// PK=TEAM#{teamID}  SK=REVIEW#MEMBER#{memberUserName}
type TeamMemberReviewRecord struct {
	PK                    string  `dynamodbav:"PK"`
	SK                    string  `dynamodbav:"SK"`
	TeamID                string  `dynamodbav:"teamId"`
	MemberUserName        string  `dynamodbav:"memberUserName"`
	OverallRating         float64 `dynamodbav:"overallRating"`
	LastReviewDate        string  `dynamodbav:"lastReviewDate,omitempty"`
	IsPendingReview       bool    `dynamodbav:"isPendingReview"`
	HasUserUpdatedReviews bool    `dynamodbav:"hasUserUpdatedReviews"`
	UpdatedAt             string  `dynamodbav:"updatedAt"`
}

// ==================== Filter Types ====================

// GoalFilters contains optional filters for listing goals.
type GoalFilters struct {
	// Type filters by goal type: "okr" | "kpi" | "individual" | "growth". Empty = no filter.
	Type string
	// Status filters by goal status: "on-track" | "completed" | "ahead" | "at-risk" | "behind". Empty = no filter.
	Status string
}

// TaskFilters contains optional filters for listing tasks.
type TaskFilters struct {
	// GoalID filters by linked goal UUID. Use "none" to return only tasks with no linked goal. Empty = no filter.
	GoalID string
	// Status filters by task status: "todo" | "in-progress" | "done" | "closed". Empty = no filter.
	Status string
	// Done filters by completion flag. nil = no filter.
	Done *bool
}

// ==================== Composite Result Types ====================

// TeamMemberWithReview represents a team member enhanced with their review record.
type TeamMemberWithReview struct {
	UserName              string  `json:"id"`
	DisplayName           string  `json:"name"`
	Role                  string  `json:"role"`
	OverallRating         float64 `json:"overallRating"`
	LastReviewDate        string  `json:"lastReviewDate,omitempty"`
	IsPendingReview       bool    `json:"isPendingReview"`
	HasUserUpdatedReviews bool    `json:"hasUserUpdatedReviews"`
}

// MemberGoalsResult holds a team member's goals split by type.
type MemberGoalsResult struct {
	OKRs []GoalRecord `json:"okrs"`
	KPIs []GoalRecord `json:"kpis"`
}

// MemberPerformanceSummary is the full picture of a team member for a manager.
type MemberPerformanceSummary struct {
	MemberUserName  string                 `json:"memberId"`
	DisplayName     string                 `json:"name"`
	Role            string                 `json:"role"`
	OverallRating   float64                `json:"overallRating"`
	LastReviewDate  string                 `json:"lastReviewDate,omitempty"`
	IsPendingReview bool                   `json:"isPendingReview"`
	Goals           MemberGoalsResult      `json:"goals"`
	Meetings        []MeetingRecord        `json:"meetings"`
	Appreciations   []AppreciationRecord   `json:"appreciations"`
	Comments        []ManagerCommentRecord `json:"comments"`
}
