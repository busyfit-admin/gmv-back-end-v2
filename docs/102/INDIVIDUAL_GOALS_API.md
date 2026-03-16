# Individual Goals — API Reference

All endpoints require a Cognito JWT in the `Authorization` header.  
Pass `Organization-Id` as a header to identify the organisation.  
All responses are wrapped in `{ "data": { ... } }` on success or `{ "error": { "code": "...", "message": "..." } }` on failure.  
Dates are `YYYY-MM-DD`, timestamps are ISO 8601 (UTC).

> **Lambda** — routed through `ManageUserPerformanceLambda` (user endpoints) and `ManageTeamPerformanceLambda` (manager endpoints).  
> **Table** — `PerfHubTable`, single-table DynamoDB design.  
> **Key pattern** — `PK = USER#{userName}#TEAM#{teamId}`, `SK = GOAL#{goalId}`.  
> **Team scoping** — all `/v2/users/me/...` endpoints require a `teamId` query parameter.

---

## Summary Table

| # | Method | Endpoint | Caller | Purpose |
|---|--------|----------|--------|---------|
| 1 | GET | `/v2/users/me/goals` | Member | List all goals |
| 2 | POST | `/v2/users/me/goals` | Member | Create a new individual goal |
| 3 | PATCH | `/v2/users/me/goals/{goalId}` | Member | Update progress, status, dueDate, orgGoalId |
| 4 | POST | `/v2/users/me/goals/{goalId}/comments` | Member | Add a comment on own goal |
| 5 | POST | `/v2/teams/{teamId}/members/{username}/goals/{goalId}/comments` | Manager | Add a review comment on a member's goal |

---

## Goal Object

```json
{
  "id": "uuid",
  "title": "string",
  "type": "individual",
  "progress": 40,
  "dueDate": "2026-06-30",
  "status": "on-track",
  "description": "string",
  "orgGoalId": "org-level-goal-uuid-or-empty",
  "createdDate": "2026-03-01",
  "linkedTasks": [
    {
      "id": "TASK-101",
      "taskNumber": 101,
      "title": "string",
      "status": "in-progress",
      "done": false,
      "priority": "high",
      "dueDate": "2026-03-31"
    }
  ],
  "comments": [
    {
      "id": "uuid",
      "author": "Jane Doe",
      "authorUserName": "jane.doe@company.com",
      "initials": "JD",
      "role": "member",
      "text": "string",
      "date": "2026-03-01"
    }
  ]
}
```

### Goal Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | UUID of the goal |
| `title` | `string` | Goal title |
| `type` | `string` | `individual` · `growth` · `kpi` · `okr` |
| `progress` | `integer` | 0–100 |
| `dueDate` | `string` | Due date (`YYYY-MM-DD`) |
| `status` | `string` | `on-track` · `completed` · `ahead` · `at-risk` · `behind` |
| `description` | `string` | Optional description |
| `orgGoalId` | `string` | UUID of a linked org-level goal (empty if not linked) |
| `createdDate` | `string` | Date created (`YYYY-MM-DD`) |
| `linkedTasks` | `array` | Tasks linked to this goal (full task objects) |
| `comments` | `array` | All comments — both member and manager authored |

### Comment Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | UUID of the comment |
| `author` | `string` | Display name of the commenter |
| `authorUserName` | `string` | Username (email) of the commenter |
| `initials` | `string` | Two-letter initials derived from display name |
| `role` | `string` | `member` (goal owner's comment) or `manager` (reviewer's comment) |
| `text` | `string` | Comment body |
| `date` | `string` | Date posted (`YYYY-MM-DD`) |

---

## 1. List Goals

### GET `/v2/users/me/goals`

Returns all goals for the authenticated user within a team.

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |
| `type` | ❌ | Filter: `individual` · `growth` · `kpi` · `okr` |
| `status` | ❌ | Filter: `on-track` · `completed` · `ahead` · `at-risk` · `behind` |

### Response `200`

```json
{ "data": { "goals": [ <GoalObject>, ... ] } }
```

---

## 2. Create Goal

### POST `/v2/users/me/goals`

Creates a new individual goal. `title`, `type`, and `dueDate` are required.

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |
| `Content-Type` | ✅ | `application/json` |

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |

### Request Body

```json
{
  "title": "Improve customer NPS score",
  "type": "individual",
  "dueDate": "2026-06-30",
  "description": "Focus on reducing support response time to improve satisfaction.",
  "status": "on-track",
  "orgGoalId": "org-goal-uuid"
}
```

| Field | Required | Allowed Values / Notes |
|-------|----------|------------------------|
| `title` | ✅ | Any non-empty string |
| `type` | ✅ | `individual` · `growth` · `kpi` · `okr` |
| `dueDate` | ✅ | `YYYY-MM-DD` |
| `description` | ❌ | Any string |
| `status` | ❌ | `on-track` · `completed` · `ahead` · `at-risk` · `behind` (default: `on-track`) |
| `orgGoalId` | ❌ | UUID of an org-level goal to link this individual goal to |

### Response `201`

```json
{ "data": { "goal": <GoalObject> } }
```

---

## 3. Update Goal

### PATCH `/v2/users/me/goals/{goalId}`

Updates any combination of goal fields. Send only what needs to change.

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |
| `Content-Type` | ✅ | `application/json` |

### Path Parameters

| Param | Description |
|-------|-------------|
| `goalId` | UUID of the goal to update |

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |

### Request Body (all fields optional)

```json
{
  "progress": 60,
  "status": "ahead",
  "dueDate": "2026-07-31",
  "orgGoalId": "org-goal-uuid",
  "linkedTasks": [
    { "id": "existing-task-uuid", "title": "Existing task", "done": true },
    { "id": "", "title": "New inline task", "done": false }
  ]
}
```

| Field | Notes |
|-------|-------|
| `progress` | Integer 0–100 |
| `status` | `on-track` · `completed` · `ahead` · `at-risk` · `behind` |
| `dueDate` | `YYYY-MM-DD` — updates the due date |
| `orgGoalId` | `""` to unlink · `"uuid"` to link/relink · field absent = unchanged |
| `linkedTasks` | Upsert array — omit `id` (or send `""`) to create a new task; supply `id` to update |

### `orgGoalId` semantics

| Value sent | Behaviour |
|------------|-----------|
| field absent | existing org goal link unchanged |
| `"orgGoalId": ""` | **unlinks** from org goal |
| `"orgGoalId": "uuid"` | links / relinks to that org-level goal |

### Response `200`

```json
{ "data": { "goal": <GoalObject> } }
```

---

## 4. Member Comment on a Goal

### POST `/v2/users/me/goals/{goalId}/comments`

Adds a comment authored by the goal owner (`role: "member"`).

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |
| `Content-Type` | ✅ | `application/json` |

### Path Parameters

| Param | Description |
|-------|-------------|
| `goalId` | UUID of the goal |

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |

### Request Body

```json
{ "text": "Completed the Q1 customer interviews." }
```

### Response `201`

```json
{
  "data": {
    "comment": {
      "id": "uuid",
      "author": "Jane Doe",
      "authorUserName": "jane.doe@company.com",
      "initials": "JD",
      "role": "member",
      "text": "Completed the Q1 customer interviews.",
      "date": "2026-03-16"
    }
  }
}
```

---

## 5. Manager Review Comment on a Member's Goal

### POST `/v2/teams/{teamId}/members/{username}/goals/{goalId}/comments`

Allows a manager to leave a review comment on a specific member's individual goal.  
The comment is stored with `role: "manager"` on the **member's** goal partition.  
It appears inline alongside member-authored comments when the goal is fetched.

> Caller must be a member (or manager) of the given team.

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |
| `Content-Type` | ✅ | `application/json` |

### Path Parameters

| Param | Description |
|-------|-------------|
| `teamId` | UUID of the team |
| `username` | The **member's** userName (email) |
| `goalId` | UUID of the member's goal |

### Request Body

```json
{ "text": "Great progress — keep focusing on the key metrics." }
```

### Response `201`

```json
{
  "data": {
    "comment": {
      "id": "uuid",
      "author": "Alex Manager",
      "authorUserName": "alex.manager@company.com",
      "initials": "AM",
      "role": "manager",
      "text": "Great progress — keep focusing on the key metrics.",
      "date": "2026-03-16"
    }
  }
}
```

---

## Error Codes

| HTTP | Code | When |
|------|------|------|
| `400` | `VALIDATION_ERROR` | Missing required field (`title`, `type`, `dueDate`, `text`), invalid `type` or `status` value |
| `401` | `UNAUTHORIZED` | Missing or invalid Cognito JWT |
| `404` | `NOT_FOUND` | Goal not found, or member's goal not found (manager comment) |
| `405` | `METHOD_NOT_ALLOWED` | HTTP method not supported on the route |
| `500` | `INTERNAL_ERROR` | Server-side failure |
