package common

// ==================== Post Types ====================

type PostType string

const (
	PostTypeUpdate    PostType = "update"
	PostTypeKudos     PostType = "kudos"
	PostTypeTask      PostType = "task"
	PostTypePoll      PostType = "poll"
	PostTypeChecklist PostType = "checklist"
	PostTypeEvent     PostType = "event"
)

// ==================== DDB Key Prefixes ====================

const (
	PrefixPost      = "POST#"
	PrefixTeam      = "TEAM#"
	PrefixComment   = "COMMENT#"
	SKMetadata      = "#METADATA"
	SKLikePrefix    = "LIKE#"
	SKCommentPrefix = "CMMNT#"
	SKVotePrefix    = "VOTE#"
	SKItemPrefix    = "ITEM#"
)

// ==================== Tag ====================

type Tag struct {
	Type  string `json:"type" dynamodbav:"type"` // "goal | skill | milestone"
	RefID string `json:"refId" dynamodbav:"refId"`
	Name  string `json:"name" dynamodbav:"name"`
}

// ==================== Kudos ====================

type KudosData struct {
	RecipientUserID string `json:"recipientUserId" dynamodbav:"recipientUserId"`
	RecipientName   string `json:"recipientName,omitempty" dynamodbav:"recipientName,omitempty"`
}

// ==================== Task ====================

type TaskData struct {
	TaskNumber     string  `json:"taskNumber,omitempty" dynamodbav:"taskNumber,omitempty"`
	Summary        string  `json:"taskSummary" dynamodbav:"taskSummary"`
	Description    string  `json:"taskDescription" dynamodbav:"taskDescription"`
	AssigneeUserID string  `json:"assigneeUserId" dynamodbav:"assigneeUserId"`
	AssigneeName   string  `json:"assigneeName,omitempty" dynamodbav:"assigneeName,omitempty"`
	DueDate        string  `json:"dueDate" dynamodbav:"dueDate"`
	Urgency        string  `json:"urgency" dynamodbav:"urgency"` // Low | Medium | High
	Status         string  `json:"status" dynamodbav:"status"`   // todo | in-progress | done
	TimeSpentHours float64 `json:"timeSpentHours" dynamodbav:"timeSpentHours"`
}

// ==================== Poll ====================

type PollOption struct {
	OptionID string `json:"optionId" dynamodbav:"optionId"`
	Text     string `json:"text" dynamodbav:"text"`
	Votes    int    `json:"votes,omitempty" dynamodbav:"-"`
}

type PollData struct {
	Question string       `json:"question" dynamodbav:"question"`
	Options  []PollOption `json:"options" dynamodbav:"options"`
}

// ==================== Checklist ====================

type ChecklistItem struct {
	ItemID    string `json:"itemId" dynamodbav:"itemId"`
	Text      string `json:"text" dynamodbav:"text"`
	Completed bool   `json:"completed" dynamodbav:"completed"`
	CreatedAt string `json:"createdAt,omitempty" dynamodbav:"createdAt,omitempty"`
}

type ChecklistData struct {
	Title              string `json:"title" dynamodbav:"title"`
	IsRecurring        bool   `json:"isRecurring" dynamodbav:"isRecurring"`
	RecurringFrequency string `json:"recurringFrequency,omitempty" dynamodbav:"recurringFrequency,omitempty"` // Daily | Weekly | Bi-Weekly | Monthly
}

// ==================== Event ====================

type EventData struct {
	Title     string `json:"title" dynamodbav:"title"`
	EventDate string `json:"eventDate" dynamodbav:"eventDate"`
	EventTime string `json:"eventTime" dynamodbav:"eventTime"`
	Location  string `json:"location,omitempty" dynamodbav:"location,omitempty"`
}

// ==================== PostData (type-specific map stored under "data" in DDB) ====================

type PostData struct {
	// Kudos fields
	KudosRecipientUserID string `dynamodbav:"kudosRecipientUserId,omitempty"`
	KudosRecipientName   string `dynamodbav:"kudosRecipientName,omitempty"`

	// Task fields
	TaskNumber     string  `dynamodbav:"taskNumber,omitempty"`
	TaskSummary    string  `dynamodbav:"taskSummary,omitempty"`
	TaskDesc       string  `dynamodbav:"taskDescription,omitempty"`
	AssigneeUserID string  `dynamodbav:"assigneeUserId,omitempty"`
	AssigneeName   string  `dynamodbav:"assigneeName,omitempty"`
	DueDate        string  `dynamodbav:"dueDate,omitempty"`
	Urgency        string  `dynamodbav:"urgency,omitempty"`
	TaskStatus     string  `dynamodbav:"taskStatus,omitempty"`
	TimeSpentHours float64 `dynamodbav:"timeSpentHours,omitempty"`

	// Poll fields
	PollQuestion string       `dynamodbav:"pollQuestion,omitempty"`
	PollOptions  []PollOption `dynamodbav:"pollOptions,omitempty"`

	// Checklist fields
	ChecklistTitle     string `dynamodbav:"checklistTitle,omitempty"`
	IsRecurring        bool   `dynamodbav:"isRecurring,omitempty"`
	RecurringFrequency string `dynamodbav:"recurringFrequency,omitempty"`

	// Event fields
	EventTitle string `dynamodbav:"eventTitle,omitempty"`
	EventDate  string `dynamodbav:"eventDate,omitempty"`
	EventTime  string `dynamodbav:"eventTime,omitempty"`
	Location   string `dynamodbav:"location,omitempty"`
}

// ==================== Author ====================

type AuthorInfo struct {
	UserID     string  `json:"userId"`
	Name       string  `json:"name"`
	Role       string  `json:"role,omitempty"`
	ProfilePic *string `json:"profilePic"`
}

// ==================== Post DDB Record ====================

type PostRecord struct {
	PK     string `dynamodbav:"PK"`
	SK     string `dynamodbav:"SK"`
	GSI1PK string `dynamodbav:"GSI1PK"`
	GSI1SK string `dynamodbav:"GSI1SK"`
	PostID string `dynamodbav:"postId"`
	TeamID string `dynamodbav:"teamId"`
	Type   string `dynamodbav:"type"`

	AuthorUserID     string `dynamodbav:"authorUserId"`
	AuthorName       string `dynamodbav:"authorName"`
	AuthorRole       string `dynamodbav:"authorRole,omitempty"`
	AuthorProfilePic string `dynamodbav:"authorProfilePic,omitempty"`

	Content string `dynamodbav:"content,omitempty"`
	Tags    []Tag  `dynamodbav:"tags,omitempty"`

	LikeCount    int `dynamodbav:"likeCount"`
	CommentCount int `dynamodbav:"commentCount"`

	CreatedAt string `dynamodbav:"createdAt"`
	UpdatedAt string `dynamodbav:"updatedAt"`

	// Type-specific data stored as a DDB Map under the "data" field
	Data PostData `dynamodbav:"data,omitempty"`
}

// ==================== Comment DDB Record ====================

type CommentRecord struct {
	PK              string `dynamodbav:"PK"`
	SK              string `dynamodbav:"SK"`
	GSI1PK          string `dynamodbav:"GSI1PK"`
	GSI1SK          string `dynamodbav:"GSI1SK"`
	CommentID       string `dynamodbav:"commentId"`
	PostID          string `dynamodbav:"postId"`
	AuthorUserID    string `dynamodbav:"authorUserId"`
	AuthorName      string `dynamodbav:"authorName"`
	Content         string `dynamodbav:"content"`
	ParentCommentID string `dynamodbav:"parentCommentId,omitempty"`
	LikeCount       int    `dynamodbav:"likeCount"`
	CreatedAt       string `dynamodbav:"createdAt"`
	UpdatedAt       string `dynamodbav:"updatedAt"`
}

// ==================== Like DDB Record ====================

type LikeRecord struct {
	PK      string `dynamodbav:"PK"`
	SK      string `dynamodbav:"SK"`
	UserID  string `dynamodbav:"userId"`
	LikedAt string `dynamodbav:"likedAt"`
}

// ==================== Vote DDB Record ====================

type VoteRecord struct {
	PK       string `dynamodbav:"PK"`
	SK       string `dynamodbav:"SK"`
	UserID   string `dynamodbav:"userId"`
	OptionID string `dynamodbav:"optionId"`
	VotedAt  string `dynamodbav:"votedAt"`
}

// ==================== Checklist Item DDB Record ====================

type ChecklistItemRecord struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	ItemID    string `dynamodbav:"itemId"`
	PostID    string `dynamodbav:"postId"`
	Text      string `dynamodbav:"text"`
	Completed bool   `dynamodbav:"completed"`
	CreatedAt string `dynamodbav:"createdAt"`
}

// ==================== Request Bodies ====================

type CreatePostRequest struct {
	Type    PostType `json:"type"`
	Content string   `json:"content,omitempty"`
	Tags    []Tag    `json:"tags,omitempty"`

	// kudos
	RecipientUserID string `json:"recipientUserId,omitempty"`

	// task
	TaskSummary     string `json:"taskSummary,omitempty"`
	TaskDescription string `json:"taskDescription,omitempty"`
	AssigneeUserID  string `json:"assigneeUserId,omitempty"`
	DueDate         string `json:"dueDate,omitempty"`
	Urgency         string `json:"urgency,omitempty"`

	// poll
	Question string       `json:"question,omitempty"`
	Options  []PollOption `json:"options,omitempty"`

	// checklist
	Title              string          `json:"title,omitempty"`
	Items              []ChecklistItem `json:"items,omitempty"`
	IsRecurring        bool            `json:"isRecurring,omitempty"`
	RecurringFrequency string          `json:"recurringFrequency,omitempty"`

	// event
	EventDate string `json:"eventDate,omitempty"`
	EventTime string `json:"eventTime,omitempty"`
	Location  string `json:"location,omitempty"`
}

type AddCommentRequest struct {
	Content         string `json:"content"`
	ParentCommentID string `json:"parentCommentId,omitempty"`
}

type EditCommentRequest struct {
	Content string `json:"content"`
}

type CastVoteRequest struct {
	OptionID string `json:"optionId"`
}

type ToggleChecklistItemRequest struct {
	Completed bool `json:"completed"`
}

type AddChecklistItemRequest struct {
	Text string `json:"text"`
}

type UpdateTaskStatusRequest struct {
	Status string `json:"status"` // todo | in-progress | done
}

type LogTimeRequest struct {
	Hours float64 `json:"hours"`
}

// ==================== Response Envelope ====================

type MetaResponse struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIResponse struct {
	Data  interface{}    `json:"data"`
	Meta  *MetaResponse  `json:"meta"`
	Error *ErrorResponse `json:"error"`
}
