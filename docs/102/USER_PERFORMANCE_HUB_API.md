# User Performance Hub — API Reference

All endpoints require a Cognito JWT in the `Authorization` header.  
Every response is wrapped in `{ "data": { ... } }` on success or `{ "error": { "code": "...", "message": "..." } }` on failure.  
Dates are `YYYY-MM-DD`, timestamps are ISO 8601 (UTC).

> **Team scoping** — all `/v2/users/me/...` endpoints require a `teamId` query parameter. Data is stored and retrieved per-user per-team, so goals, meetings, appreciations and feedback requests from one team are never visible under another team.
>
> Example: `GET /v2/users/me/goals?teamId=<teamId>`

---

## Goals

### GET `/v2/users/me/goals`
List all goals for the authenticated user.

**Query params** (all optional)
| Param | Values |
|-------|--------|
| `type` | `individual` · `growth` · `kpi` · `okr` |
| `status` | `on-track` · `completed` · `ahead` · `at-risk` · `behind` |

**Response `200`**
```json
{
  "data": {
    "goals": [
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
          { "id": "TASK-101", "title": "string", "done": false }
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
    ]
  }
}
```

---

### POST `/v2/users/me/goals`
Create a new goal.

**Request body**
```json
{
  "title": "string",          // required
  "type": "individual",       // required — individual | growth | kpi | okr
  "dueDate": "2026-06-30",    // required — YYYY-MM-DD
  "description": "string",    // optional
  "status": "on-track",       // optional — defaults to "on-track"
  "orgGoalId": "org-goal-uuid" // optional — links this goal to an org-level goal
}
```

**Response `201`** — `{ "data": { "goal": <GoalObject> } }`

---

### PATCH `/v2/users/me/goals/{goalId}`
Update progress, status, dueDate, orgGoalId, or bulk-upsert linked tasks.

**Request body** (all fields optional)
```json
{
  "progress": 60,
  "status": "ahead",
  "dueDate": "2026-07-31",
  "orgGoalId": "org-goal-uuid",
  "linkedTasks": [
    { "id": "existing-uuid-or-empty", "title": "string", "done": false }
  ]
}
```

`orgGoalId` semantics:
- **field absent**: existing org goal link unchanged
- **`"orgGoalId": ""`** (empty string): **unlinks** from the org goal
- **`"orgGoalId": "uuid"`**: links / relinks to that org goal
}
```
Omit `id` in a `linkedTask` entry to create a new task; supply it to update an existing one.

**Response `200`** — `{ "data": { "goal": <GoalObject> } }` (with updated tasks/comments)

---

### POST `/v2/users/me/goals/{goalId}/tasks`
Add a task linked to a specific goal. The task is stored as a top-level item with `goalId` as an attribute.

**Request body**
```json
{ "title": "string" }   // required
```

**Response `201`**
```json
{ "data": { "task": { "id": "uuid", "goalId": "uuid", "title": "string", "done": false } } }
```

---

### POST `/v2/users/me/goals/{goalId}/comments`
Add a comment to a goal. The comment is stored with `role: "member"` and the caller's display name and username.

**Request body**
```json
{ "text": "string" }   // required
```

**Response `201`**
```json
{
  "data": {
    "comment": {
      "id": "uuid",
      "author": "Jane Doe",
      "authorUserName": "jane.doe@company.com",
      "initials": "JD",
      "role": "member",
      "text": "string",
      "date": "2026-03-01"
    }
  }
}
```

---

### POST `/v2/teams/{teamId}/members/{username}/goals/{goalId}/comments` *(manager)*
Allows a manager to add a review comment on a specific member's goal. The comment is stored with `role: "manager"` on the **member's** goal partition — it will appear alongside the member's own comments when the goal is fetched.

**Request body**
```json
{ "text": "string" }   // required
```

**Response `201`**
```json
{
  "data": {
    "comment": {
      "id": "uuid",
      "author": "Manager Name",
      "authorUserName": "manager@company.com",
      "initials": "MN",
      "role": "manager",
      "text": "string",
      "date": "2026-03-01"
    }
  }
}
```

---

## Tasks

Tasks are **top-level items** stored as `SK = TASK#{taskId}` on the user+team partition. Each task carries a human-readable `id` in the format `TASK-101`, `TASK-102`, … — a team-scoped sequence that starts at **101** and increments atomically. A task may optionally carry a `goalId` attribute to link it to a goal.

### Task fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Human-readable ID: `TASK-101`, `TASK-102`, … (team-scoped) |
| `taskNumber` | `integer` | Numeric part of the ID (101, 102, …) |
| `title` | `string` | Short task title (**required** on create) |
| `description` | `string` | Optional longer description |
| `status` | `string` | `todo` · `in-progress` · `done` · `closed` (default: `todo`) |
| `done` | `boolean` | Derived from status — `true` when status is `done` or `closed` |
| `priority` | `string` | `low` · `medium` · `high` · `urgent` |
| `tags` | `string[]` | Arbitrary label array |
| `timeHours` | `number` | Time logged in hours |
| `timeDays` | `number` | Time logged in days |
| `dueDate` | `string` | Due date (`YYYY-MM-DD`) |
| `goalId` | `string` | UUID of linked goal, or empty if unlinked |
| `createdAt` | `string` | ISO 8601 UTC |
| `updatedAt` | `string` | ISO 8601 UTC |

---

### GET `/v2/users/me/tasks`
List all tasks for the authenticated user within a team.

**Query params**
| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |
| `goalId` | optional | Filter by goal UUID; use `none` for tasks with no goal linked |
| `status` | optional | Filter by status: `todo`, `in-progress`, `done`, `closed` |
| `done` | optional | Backward-compat boolean filter (`true` = done/closed, `false` = todo/in-progress) |

**Response `200`**
```json
{
  "data": {
    "tasks": [
      {
        "id": "TASK-101",
        "taskNumber": 101,
        "title": "Write unit tests",
        "description": "Cover happy path and error cases",
        "status": "in-progress",
        "done": false,
        "priority": "high",
        "tags": ["backend", "q1"],
        "timeHours": 2.5,
        "timeDays": 0,
        "dueDate": "2026-03-31",
        "goalId": "goal-uuid-or-empty",
        "createdAt": "2026-03-16T10:00:00Z",
        "updatedAt": "2026-03-16T10:00:00Z"
      }
    ]
  }
}
```

---

### POST `/v2/users/me/tasks`
Create a standalone task. Optionally link it to a goal by including `goalId` in the body.

**Query params**: `teamId` (required)

**Request body**
```json
{
  "title": "string",           // required
  "description": "string",     // optional
  "priority": "high",          // optional: low | medium | high | urgent
  "status": "todo",            // optional: todo | in-progress | done | closed (default: todo)
  "tags": ["backend", "q1"],   // optional
  "timeHours": 0,              // optional
  "timeDays": 0,               // optional
  "dueDate": "2026-03-31",     // optional (YYYY-MM-DD)
  "goalId": "uuid"             // optional
}
```

**Response `201`**
```json
{
  "data": {
    "task": {
      "id": "TASK-101",
      "taskNumber": 101,
      "title": "string",
      "description": "string",
      "status": "todo",
      "done": false,
      "priority": "high",
      "tags": ["backend"],
      "timeHours": 0,
      "timeDays": 0,
      "dueDate": "2026-03-31",
      "goalId": "uuid-or-empty",
      "createdAt": "RFC3339",
      "updatedAt": "RFC3339"
    }
  }
}
```

---

### PATCH `/v2/users/me/tasks/{taskId}`
Update any combination of task fields. Only include the fields you want to change.

> `taskId` is the `TASK-101` style identifier returned on create.

**Query params**: `teamId` (required)

**Request body** (all fields optional — include only what to change)
```json
{
  "title": "Updated title",
  "description": "Updated description",
  "status": "in-progress",
  "priority": "urgent",
  "tags": ["backend", "hotfix"],
  "timeHours": 4.5,
  "timeDays": 0.5,
  "dueDate": "2026-04-01",
  "goalId": "new-goal-uuid"
}
```

**`status` / `done` semantics:**
- Send `status` (preferred) — `done` is automatically derived (`true` when status is `done` or `closed`)
- Send `done: true` (backward-compat) — status is set to `done`; `done: false` sets status to `todo`

**`goalId` semantics:**
- **field absent**: existing goal linkage unchanged
- **`"goalId": ""`** (empty string): **unlinks** the task from any goal
- **`"goalId": "uuid"`**: relinks the task to the specified goal (goal must exist)

**`tags` semantics:**
- **field absent**: existing tags unchanged
- **`"tags": []`**: clears all tags
- **`"tags": ["a","b"]`**: replaces tags entirely

**Response `200`**
```json
{ "data": { "task": { "id": "TASK-101", "teamId": "uuid", "updated": true } } }
```

---

## 1-on-1 Meetings

### GET `/v2/users/me/meetings`
List all meeting records for the authenticated user.

**Query params** (optional)
| Param | Values |
|-------|--------|
| `status` | `scheduled` · `completed` |

**Response `200`**
```json
{
  "data": {
    "meetings": [
      {
        "id": "uuid",
        "date": "2026-03-10",
        "summary": "string",
        "managerName": "string",
        "managerRole": "string",
        "tags": ["performance", "growth"],
        "actionItems": ["Follow up on Q1 goals"],
        "status": "scheduled",
        "createdAt": "2026-03-01T10:00:00Z"
      }
    ]
  }
}
```

---

### POST `/v2/users/me/meetings`
Create a meeting record.

**Request body**
```json
{
  "date": "2026-03-10",            // required — YYYY-MM-DD
  "summary": "string",             // optional
  "managerName": "string",         // optional
  "managerRole": "string",         // optional
  "tags": ["string"],              // optional
  "actionItems": ["string"]        // optional
}
```

**Response `201`** — `{ "data": { "meeting": <MeetingObject> } }`

---

## Appreciations

### GET `/v2/users/me/appreciations`
List all appreciations received by the authenticated user (newest first).

**Response `200`**
```json
{
  "data": {
    "appreciations": [
      {
        "id": "uuid",
        "fromUser": "john.doe",
        "fromName": "John Doe",
        "initials": "JD",
        "message": "Great job on the sprint!",
        "badgeType": "string",
        "date": "2026-03-01",
        "createdAt": "2026-03-01T09:00:00Z"
      }
    ]
  }
}
```
> Appreciations are written to a user's record externally (e.g. from a kudos/team feed flow). This endpoint is read-only.

---

## Feedback Requests

### POST `/v2/users/me/feedback-requests`
Send a feedback request to another user.

**Request body**
```json
{
  "toUsername": "jane.smith",   // required — target user's username (email)
  "message": "string"           // required
}
```

**Response `201`**
```json
{
  "data": {
    "feedbackRequest": {
      "id": "uuid",
      "to": "jane.smith",
      "from": "john.doe",
      "message": "string",
      "status": "pending",
      "date": "2026-03-01",
      "createdAt": "2026-03-01T10:00:00Z"
    }
  }
}
```

---

### GET `/v2/users/me/feedback-requests`
List feedback requests sent by the authenticated user (newest first).

**Query params** (optional)
| Param | Values |
|-------|--------|
| `status` | `pending` · `completed` |

**Response `200`**
```json
{
  "data": {
    "feedbackRequests": [
      {
        "id": "uuid",
        "to": "jane.smith",
        "from": "john.doe",
        "message": "string",
        "status": "pending",
        "date": "2026-03-01",
        "createdAt": "2026-03-01T10:00:00Z"
      }
    ]
  }
}
```

---

## Team Member Directory

### GET `/v2/teams/{teamId}/members/directory`
Returns basic profile cards for every member of a team.  
Caller must be a member of the team, otherwise returns `403`.

**Response `200`**
```json
{
  "data": {
    "teamId": "uuid",
    "members": [
      {
        "userName": "jane.smith",
        "displayName": "Jane Smith",
        "initials": "JS",
        "role": "MEMBER"
      }
    ]
  }
}
```
`role` values: `OWNER` · `ADMIN` · `MEMBER` · `GUEST`

---

## Error Responses

All errors follow the same envelope:
```json
{ "error": { "code": "ERROR_CODE", "message": "Human readable message" } }
```

| HTTP | Code | When |
|------|------|------|
| `400` | `VALIDATION_ERROR` | Missing required field or bad value |
| `401` | `UNAUTHORIZED` | Missing / invalid Cognito token |
| `403` | `FORBIDDEN` | Not a member of the requested team |
| `404` | `NOT_FOUND` | Goal / task / meeting not found |
| `405` | `METHOD_NOT_ALLOWED` | Wrong HTTP verb for the route |
| `500` | `INTERNAL_ERROR` | DynamoDB or unexpected server error |
