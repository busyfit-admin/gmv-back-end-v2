package common

// ==================== DDB Key Prefixes ====================

const (
	PrefixUser = "USER#"

	SKGoalPrefix         = "GOAL#"
	SKMeetingPrefix      = "MEETING#"
	SKAppreciationPrefix = "APPR#"
	SKFeedbackReqPrefix  = "FBREQ#"
	SKTaskInfix          = "#TASK#"
	SKCommentInfix       = "#CMMNT#"
)

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

// GoalRecord — PK=USER#{userName} SK=GOAL#{goalId}
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
	CreatedAt   string `dynamodbav:"createdAt"`
	UpdatedAt   string `dynamodbav:"updatedAt"`
}

// LinkedTaskRecord — PK=USER#{userName} SK=GOAL#{goalId}#TASK#{taskId}
type LinkedTaskRecord struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	TaskID    string `dynamodbav:"taskId"`
	GoalID    string `dynamodbav:"goalId"`
	UserName  string `dynamodbav:"userName"`
	Title     string `dynamodbav:"title"`
	Done      bool   `dynamodbav:"done"`
	CreatedAt string `dynamodbav:"createdAt"`
}

// GoalCommentRecord — PK=USER#{userName} SK=GOAL#{goalId}#CMMNT#{commentId}
type GoalCommentRecord struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	CommentID string `dynamodbav:"commentId"`
	GoalID    string `dynamodbav:"goalId"`
	UserName  string `dynamodbav:"userName"`
	Author    string `dynamodbav:"author"`
	Initials  string `dynamodbav:"initials"`
	Text      string `dynamodbav:"text"`
	Date      string `dynamodbav:"date"`
	CreatedAt string `dynamodbav:"createdAt"`
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
}

type UpdateGoalRequest struct {
	Progress    *int         `json:"progress,omitempty"`
	LinkedTasks []LinkedTask `json:"linkedTasks,omitempty"`
	Status      string       `json:"status,omitempty"`
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

// ==================== Response Envelope ====================

type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error *ErrBody    `json:"error,omitempty"`
}

type ErrBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
