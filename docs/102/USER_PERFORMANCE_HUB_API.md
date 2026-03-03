# User Performance Hub — API Reference

All endpoints require a Cognito JWT in the `Authorization` header.  
Every response is wrapped in `{ "data": { ... } }` on success or `{ "error": { "code": "...", "message": "..." } }` on failure.  
Dates are `YYYY-MM-DD`, timestamps are ISO 8601 (UTC).

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
        "createdDate": "2026-03-01",
        "linkedTasks": [
          { "id": "uuid", "title": "string", "done": false }
        ],
        "comments": [
          { "id": "uuid", "author": "Jane Doe", "initials": "JD", "text": "string", "date": "2026-03-01" }
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
  "status": "on-track"        // optional — defaults to "on-track"
}
```

**Response `201`** — `{ "data": { "goal": <GoalObject> } }`

---

### PATCH `/v2/users/me/goals/{goalId}`
Update progress, status, or bulk-upsert linked tasks.

**Request body** (all fields optional)
```json
{
  "progress": 60,
  "status": "ahead",
  "linkedTasks": [
    { "id": "existing-uuid-or-empty", "title": "string", "done": false }
  ]
}
```
Omit `id` in a `linkedTask` entry to create a new task; supply it to update an existing one.

**Response `200`** — `{ "data": { "goal": <GoalObject> } }` (with updated tasks/comments)

---

### POST `/v2/users/me/goals/{goalId}/tasks`
Add a single linked task to a goal.

**Request body**
```json
{ "title": "string" }   // required
```

**Response `201`**
```json
{ "data": { "task": { "id": "uuid", "title": "string", "done": false } } }
```

---

### PATCH `/v2/users/me/goals/{goalId}/tasks/{taskId}`
Toggle a linked task's completion state.

**Request body**
```json
{ "done": true }
```

**Response `200`**
```json
{ "data": { "task": { "id": "uuid", "goalId": "uuid", "done": true } } }
```

---

### POST `/v2/users/me/goals/{goalId}/comments`
Add a comment to a goal. Author name and initials are resolved from the authenticated user.

**Request body**
```json
{ "text": "string" }   // required
```

**Response `201`**
```json
{
  "data": {
    "comment": { "id": "uuid", "author": "Jane Doe", "initials": "JD", "text": "string", "date": "2026-03-01" }
  }
}
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
