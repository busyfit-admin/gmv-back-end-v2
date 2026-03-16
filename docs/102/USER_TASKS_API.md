# User Tasks — API Reference

All endpoints require a Cognito JWT in the `Authorization` header.  
Pass `Organization-Id` as a header to identify the organisation.  
All responses are wrapped in `{ "data": { ... } }` on success or `{ "error": { "code": "...", "message": "..." } }` on failure.  
Dates are `YYYY-MM-DD`, timestamps are ISO 8601 (UTC).

> **Lambda** — routed through `ManageUserPerformanceLambda`.  
> **Table** — `PerfHubTable`, single-table DynamoDB design.  
> **Key pattern** — `PK = USER#{userName}#TEAM#{teamId}`, `SK = TASK#{taskId}` (e.g. `TASK#TASK-101`).  
> **Team scoping** — all endpoints require a `teamId` query parameter.

---

## Summary Table

| # | Method | Endpoint | Purpose |
|---|--------|----------|---------|
| 1 | GET | `/v2/users/me/tasks` | List all tasks for the current user in a team |
| 2 | POST | `/v2/users/me/tasks` | Create a standalone task |
| 3 | PATCH | `/v2/users/me/tasks/{taskId}` | Update any task field |
| 4 | POST | `/v2/users/me/goals/{goalId}/tasks` | Create a task already linked to a specific goal |

---

## Task Field Reference

| Field | Type | Writable | Description |
|-------|------|----------|-------------|
| `id` | `string` | read-only | Human-readable team-scoped ID: `TASK-101`, `TASK-102`, … (starts at 101, increments per team) |
| `taskNumber` | `integer` | read-only | Numeric part of the ID (101, 102, …) |
| `title` | `string` | ✅ | Short task title. **Required** on create |
| `description` | `string` | ✅ | Optional longer description |
| `status` | `string` | ✅ | `todo` · `in-progress` · `done` · `closed`. Default: `todo` |
| `done` | `boolean` | ✅ | Auto-derived from status (`true` when status is `done` or `closed`) |
| `priority` | `string` | ✅ | `low` · `medium` · `high` · `urgent` |
| `tags` | `string[]` | ✅ | Array of arbitrary label strings |
| `timeHours` | `number` | ✅ | Time logged in hours (e.g. `2.5`) |
| `timeDays` | `number` | ✅ | Time logged in days (e.g. `0.5`) |
| `dueDate` | `string` | ✅ | Due date in `YYYY-MM-DD` format |
| `goalId` | `string` | ✅ | UUID of the linked goal, or `""` / absent if not linked |
| `createdAt` | `string` | read-only | ISO 8601 UTC |
| `updatedAt` | `string` | read-only | ISO 8601 UTC — stamped on every update |

---

## 1. List Tasks

### GET `/v2/users/me/tasks`

Returns all tasks for the authenticated user within a team, sorted by `createdAt` ascending.

---

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |

---

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |
| `goalId` | ❌ | Filter to a specific goal UUID. Use `none` to return only tasks with no goal linked |
| `status` | ❌ | Filter by status: `todo` · `in-progress` · `done` · `closed` |
| `done` | ❌ | Backward-compat boolean filter: `true` returns done+closed tasks; `false` returns todo+in-progress tasks |

---

### Response `200`

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

## 2. Create Task

### POST `/v2/users/me/tasks`

Creates a new standalone task. The `id` is auto-assigned as `TASK-101`, `TASK-102`, … using a per-team atomic counter. Optionally link the task to a goal via `goalId`.

---

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |
| `Content-Type` | ✅ | `application/json` |

---

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |

---

### Request Body

```json
{
  "title": "Write unit tests",
  "description": "Cover happy path and error cases",
  "priority": "high",
  "status": "todo",
  "tags": ["backend", "q1"],
  "timeHours": 0,
  "timeDays": 0,
  "dueDate": "2026-03-31",
  "goalId": "goal-uuid"
}
```

| Field | Required | Allowed Values |
|-------|----------|----------------|
| `title` | ✅ | Any non-empty string |
| `description` | ❌ | Any string |
| `priority` | ❌ | `low` · `medium` · `high` · `urgent` |
| `status` | ❌ | `todo` · `in-progress` · `done` · `closed` (default: `todo`) |
| `tags` | ❌ | Array of strings |
| `timeHours` | ❌ | Number ≥ 0 |
| `timeDays` | ❌ | Number ≥ 0 |
| `dueDate` | ❌ | `YYYY-MM-DD` |
| `goalId` | ❌ | UUID of an existing goal |

---

### Response `201`

```json
{
  "data": {
    "task": {
      "id": "TASK-101",
      "taskNumber": 101,
      "title": "Write unit tests",
      "description": "Cover happy path and error cases",
      "status": "todo",
      "done": false,
      "priority": "high",
      "tags": ["backend", "q1"],
      "timeHours": 0,
      "timeDays": 0,
      "dueDate": "2026-03-31",
      "goalId": "goal-uuid-or-empty",
      "createdAt": "2026-03-16T10:00:00Z",
      "updatedAt": "2026-03-16T10:00:00Z"
    }
  }
}
```

---

## 3. Update Task

### PATCH `/v2/users/me/tasks/{taskId}`

Updates one or more fields on an existing task. Send **only the fields you want to change** — absent fields are left untouched.

> `taskId` is the **`TASK-101` string** (NOT a UUID).

---

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |
| `Content-Type` | ✅ | `application/json` |

---

### Path Parameters

| Param | Description |
|-------|-------------|
| `taskId` | Task ID in `TASK-101` format |

---

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |

---

### Request Body (all fields optional)

```json
{
  "title": "Updated title",
  "description": "Updated description",
  "status": "done",
  "priority": "urgent",
  "tags": ["backend", "hotfix"],
  "timeHours": 4.5,
  "timeDays": 0.5,
  "dueDate": "2026-04-01",
  "goalId": "new-goal-uuid"
}
```

---

### Field Behaviour Notes

**`status` and `done`:**

| You send | Result |
|----------|--------|
| `"status": "done"` | `status = done`, `done = true` |
| `"status": "closed"` | `status = closed`, `done = true` |
| `"status": "in-progress"` | `status = in-progress`, `done = false` |
| `"done": true` *(legacy)* | `status = done`, `done = true` |
| `"done": false` *(legacy)* | `status = todo`, `done = false` |

Prefer sending `status` — `done` is kept in sync automatically.

**`goalId`:**

| Value | Behaviour |
|-------|-----------|
| field absent | existing goal linkage unchanged |
| `"goalId": ""` | **unlinks** the task from any goal |
| `"goalId": "uuid"` | links / relinks to that goal (goal must exist) |

**`tags`:**

| Value | Behaviour |
|-------|-----------|
| field absent | existing tags unchanged |
| `"tags": []` | **clears** all tags |
| `"tags": ["a","b"]` | **replaces** tags entirely |

---

### Response `200`

```json
{ "data": { "task": { "id": "TASK-101", "teamId": "uuid", "updated": true } } }
```

---

## 4. Create Task Linked to a Goal (Shorthand)

### POST `/v2/users/me/goals/{goalId}/tasks`

Shorthand to create a task and immediately link it to a specific goal. The goal must exist for the current user in the given team.

---

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |
| `Content-Type` | ✅ | `application/json` |

---

### Path Parameters

| Param | Description |
|-------|-------------|
| `goalId` | UUID of the goal to link the task to |

---

### Query Parameters

| Param | Required | Description |
|-------|----------|-------------|
| `teamId` | ✅ | Team scope |

---

### Request Body

```json
{ "title": "string" }
```

---

### Response `201`

```json
{
  "data": {
    "task": {
      "id": "uuid",
      "goalId": "goal-uuid",
      "title": "string",
      "done": false
    }
  }
}
```

---

## Error Codes

| HTTP | Code | When |
|------|------|------|
| `400` | `VALIDATION_ERROR` | Missing `title`, invalid `priority` or `status` value, malformed request body |
| `401` | `UNAUTHORIZED` | Missing or invalid Cognito JWT |
| `404` | `NOT_FOUND` | Task not found, or linked `goalId` does not exist |
| `405` | `METHOD_NOT_ALLOWED` | HTTP method not supported on the route |
| `500` | `INTERNAL_ERROR` | Server-side failure (e.g. DynamoDB error, counter allocation failure) |
