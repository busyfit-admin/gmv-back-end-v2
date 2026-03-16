package common

// ==================== DDB Key Prefixes ====================

const (
	PrefixUser = "USER#"
	PrefixTeam = "TEAM#"

	SKGoalPrefix           = "GOAL#"
	SKMeetingPrefix        = "MEETING#"
	SKAppreciationPrefix   = "APPR#"
	SKFeedbackReqPrefix    = "FBREQ#"
	SKTaskPrefix           = "TASK#"
	SKCommentInfix         = "#CMMNT#"
	SKManagerCommentPrefix = "MGRCMT#"
	SKMemberReviewPrefix   = "REVIEW#MEMBER#"
)

// buildTeamPK constructs the DynamoDB partition key for team-scoped data.
// Format: TEAM#{teamID}
func buildTeamPK(teamID string) string {
	return PrefixTeam + teamID
}

// buildPK constructs the DynamoDB partition key scoped per-user per-team.
// Format: USER#{userName}#TEAM#{teamID}
func buildPK(userName, teamID string) string {
	return PrefixUser + userName + "#TEAM#" + teamID
}

// ==================== Goal Types & Statuses ====================

type GoalType string

const (
	GoalTypeIndividual GoalType = "individual"
	GoalTypeGrowth     GoalType = "growth"
	GoalTypeKPI        GoalType = "kpi"
	GoalTypeOKR        GoalType = "okr"
)

type GoalStatus string

const (
	GoalStatusOnTrack   GoalStatus = "on-track"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusAhead     GoalStatus = "ahead"
	GoalStatusAtRisk    GoalStatus = "at-risk"
	GoalStatusBehind    GoalStatus = "behind"
)

// ==================== Task Status & Priority ====================

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in-progress"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusClosed     TaskStatus = "closed"
)

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityUrgent TaskPriority = "urgent"
)

// ==================== Meeting Status ====================

type MeetingStatus string

const (
	MeetingStatusScheduled MeetingStatus = "scheduled"
	MeetingStatusCompleted MeetingStatus = "completed"
)

// ==================== Feedback Category ====================

type FeedbackCategory string

const (
	FeedbackCategoryTechnical     FeedbackCategory = "technical"
	FeedbackCategoryLeadership    FeedbackCategory = "leadership"
	FeedbackCategoryCommunication FeedbackCategory = "communication"
	FeedbackCategoryCollaboration FeedbackCategory = "collaboration"
)

// ==================== DDB Records ====================

// GoalRecord — PK=USER#{userName}#TEAM#{teamId} SK=GOAL#{goalId}
// OrgGoalID is an optional reference to an org-level goal (from OrgPerformanceTable).
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

// LinkedTaskRecord — PK=USER#{userName}#TEAM#{teamId} SK=TASK#{taskId}
// TaskID is a human-readable team-scoped reference in the format TASK-{N}, starting at TASK-101.
// GoalID is optional — empty string means the task is not linked to any goal.
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

// GoalCommentRecord — PK=USER#{userName}#TEAM#{teamId} SK=GOAL#{goalId}#CMMNT#{commentId}
// Role distinguishes who wrote the comment: "member" (the goal owner) or "manager" (reviewer).
// AuthorUserName is the actual commenter's username; UserName is the goal owner's username.
type GoalCommentRecord struct {
	PK             string `dynamodbav:"PK"`
	SK             string `dynamodbav:"SK"`
	CommentID      string `dynamodbav:"commentId"`
	GoalID         string `dynamodbav:"goalId"`
	UserName       string `dynamodbav:"userName"`
	AuthorUserName string `dynamodbav:"authorUserName"`
	Author         string `dynamodbav:"author"`
	Initials       string `dynamodbav:"initials"`
	Role           string `dynamodbav:"role"`
	Text           string `dynamodbav:"text"`
	Date           string `dynamodbav:"date"`
	CreatedAt      string `dynamodbav:"createdAt"`
}

// MeetingRecord — PK=USER#{userName} SK=MEETING#{meetingId}
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

// AppreciationRecord — PK=USER#{userName} SK=APPR#{appreciationId}
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

// FeedbackRequestRecord — PK=USER#{userName} SK=FBREQ#{requestId}
type FeedbackRequestRecord struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	RequestID string `dynamodbav:"requestId"`
	UserName  string `dynamodbav:"userName"` // the requester (sender)
	To        string `dynamodbav:"to"`       // toUsername
	Message   string `dynamodbav:"message"`
	Date      string `dynamodbav:"date"`
	Status    string `dynamodbav:"status"` // pending | completed
	CreatedAt string `dynamodbav:"createdAt"`
}

// ==================== Request Bodies ====================

type CreateGoalRequest struct {
	Title       string `json:"title"`
	Type        string `json:"type"`
	DueDate     string `json:"dueDate"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	OrgGoalID   string `json:"orgGoalId,omitempty"`
}

type UpdateGoalRequest struct {
	Progress    *int         `json:"progress,omitempty"`
	LinkedTasks []LinkedTask `json:"linkedTasks,omitempty"`
	Status      string       `json:"status,omitempty"`
	DueDate     string       `json:"dueDate,omitempty"`
	OrgGoalID   *string      `json:"orgGoalId"`
}

type LinkedTask struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type AddGoalCommentRequest struct {
	Text string `json:"text"`
}

type AddLinkedTaskRequest struct {
	Title string `json:"title"`
}

// CreateTaskRequest — for POST /v2/users/me/tasks (standalone or linked)
type CreateTaskRequest struct {
	Title       string   `json:"title"`
	GoalID      string   `json:"goalId,omitempty"`
	Description string   `json:"description,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	TimeHours   float64  `json:"timeHours,omitempty"`
	TimeDays    float64  `json:"timeDays,omitempty"`
	DueDate     string   `json:"dueDate,omitempty"`
}

// UpdateTaskRequest — for PATCH /v2/users/me/tasks/{taskId}
// GoalID: nil = don't change; pointer to "" = unlink from goal; pointer to UUID = relink.
// Tags: nil = don't change; pointer to [] = clear tags; pointer to ["a","b"] = replace tags.
type UpdateTaskRequest struct {
	Done        *bool     `json:"done,omitempty"`
	Title       string    `json:"title,omitempty"`
	GoalID      *string   `json:"goalId"`
	Description string    `json:"description,omitempty"`
	Priority    string    `json:"priority,omitempty"`
	Status      string    `json:"status,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
	TimeHours   *float64  `json:"timeHours,omitempty"`
	TimeDays    *float64  `json:"timeDays,omitempty"`
	DueDate     string    `json:"dueDate,omitempty"`
}

type ToggleTaskRequest struct {
	Done bool `json:"done"`
}

type CreateMeetingRequest struct {
	Date        string   `json:"date"`
	Summary     string   `json:"summary,omitempty"`
	ActionItems []string `json:"actionItems,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	ManagerName string   `json:"managerName,omitempty"`
	ManagerRole string   `json:"managerRole,omitempty"`
}

type SendFeedbackRequestBody struct {
	ToUsername string `json:"toUsername"`
	Message    string `json:"message"`
}

// ==================== Team Performance (Manager View) Records ====================

// ManagerCommentRecord — PK=USER#{memberUserName}#TEAM#{teamID} SK=MGRCMT#{commentId}
type ManagerCommentRecord struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	CommentID string `dynamodbav:"commentId"`
	TeamID    string `dynamodbav:"teamId"`
	MemberID  string `dynamodbav:"memberId"` // member's userName
	Author    string `dynamodbav:"author"`
	Initials  string `dynamodbav:"initials"`
	Text      string `dynamodbav:"text"`
	Type      string `dynamodbav:"type"` // feedback | coaching | general
	Date      string `dynamodbav:"date"`
	CreatedAt string `dynamodbav:"createdAt"`
}

// TeamMemberReviewRecord — PK=TEAM#{teamID} SK=REVIEW#MEMBER#{memberUserName}
// Tracks per-member review lifecycle state visible to the manager.
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

// ==================== Team Performance Request Bodies ====================

// AddManagerCommentRequest — for POST /v2/teams/{teamId}/members/{memberId}/comments
type AddManagerCommentRequest struct {
	Text string `json:"text"`
	Type string `json:"type"` // feedback | coaching | general
}

// ==================== Response Envelope ====================

type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error *ErrBody    `json:"error,omitempty"`
}

type ErrBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
