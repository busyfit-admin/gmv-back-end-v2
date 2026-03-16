# Org Goal — Linked User Goals API

Allows org admins to see, for any OKR or KPI at the org/quarter level, which **individual user goals** have been linked to it — along with a rolled-up status summary (how many are on-track, in progress, completed, etc.).

All endpoints require a Cognito JWT in the `Authorization` header and the `Organization-Id` header.

> **Lambda** — `ManagePerformanceGoalsLambda` (org-performance module, `RouteGroupGoals`).  
> **Data flow** — org-level goal exists in `OrgPerformanceTable`; user goals are in `UserPerformanceHubTable` and reference the org goal via the `orgGoalId` field.  
> **DynamoDB access pattern** — GSI `OrgGoalIdIndex` on `UserPerformanceHubTable` (hash key: `orgGoalId`).

---

## Summary Table

| # | Method | Endpoint | Caller | Purpose |
|---|--------|----------|--------|---------|
| 1 | GET | `/v2/goals/{goalId}/user-goals` | Org Admin | List all user goals linked to an org OKR/KPI with status summary |

---

## How Linking Works

When a user creates or updates an **individual goal** in the performance hub, they can supply `orgGoalId` to link it to an org-level OKR or KPI.

```
User creates        →  POST /v2/users/me/goals
                       body: { "orgGoalId": "<org-okr-or-kpi-uuid>", ... }

Org admin views     →  GET /v2/goals/{goalId}/user-goals
                       path param = the same org OKR/KPI UUID
```

---

## 1. List User Goals for an Org Goal

### GET `/v2/goals/{goalId}/user-goals`

Returns all user-level individual goals that have been linked to this org-level goal (OKR or KPI), plus a summary of how many goals fall into each status bucket.

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |

### Path Parameters

| Param | Description |
|-------|-------------|
| `goalId` | UUID of the org-level OKR or KPI |

### Query Parameters

| Param | Required | Allowed Values | Description |
|-------|----------|----------------|-------------|
| `status` | ❌ | `on-track` · `ahead` · `at-risk` · `behind` · `completed` | Filter to only return user goals with this status |

### Response `200`

```json
{
  "data": {
    "orgGoalId": "org-goal-uuid",
    "userGoals": [
      {
        "goalId": "uuid",
        "userName": "alice@company.com",
        "teamId": "team-uuid",
        "title": "Improve customer NPS by 10 points",
        "type": "individual",
        "progress": 65,
        "status": "on-track",
        "dueDate": "2026-06-30",
        "updatedAt": "2026-03-15T10:00:00Z",
        "linkedTasks": [
          {
            "id": "TASK-101",
            "taskNumber": 101,
            "title": "Run NPS survey for Q1 cohort",
            "status": "done",
            "done": true,
            "priority": "high",
            "dueDate": "2026-03-31",
            "updatedAt": "2026-03-14T09:00:00Z"
          },
          {
            "id": "TASK-104",
            "taskNumber": 104,
            "title": "Analyse detractor feedback",
            "status": "in-progress",
            "done": false,
            "priority": "medium",
            "dueDate": "2026-04-15",
            "updatedAt": "2026-03-15T10:00:00Z"
          }
        ]
      },
      {
        "goalId": "uuid",
        "userName": "bob@company.com",
        "teamId": "team-uuid",
        "title": "Reduce support ticket backlog",
        "type": "individual",
        "progress": 20,
        "status": "at-risk",
        "dueDate": "2026-06-30",
        "updatedAt": "2026-03-10T08:30:00Z",
        "linkedTasks": []
      }
    ],
    "summary": {
      "total": 7,
      "onTrack": 3,
      "ahead": 1,
      "atRisk": 2,
      "behind": 0,
      "completed": 1
    }
  }
}
```

### Response Fields

#### `userGoals[]` item

| Field | Type | Description |
|-------|------|-------------|
| `goalId` | `string` | UUID of the user's goal |
| `userName` | `string` | Username (email) of the goal owner |
| `teamId` | `string` | Team the goal belongs to (parsed from DDB partition key) |
| `title` | `string` | Goal title |
| `type` | `string` | `individual` · `growth` · `kpi` · `okr` |
| `progress` | `integer` | 0–100 |
| `status` | `string` | Current status (see values below) |
| `dueDate` | `string` | Due date `YYYY-MM-DD` |
| `updatedAt` | `string` | ISO 8601 timestamp of last update |
| `linkedTasks` | `array` | Tasks linked to this goal (always present; empty array if none) |

#### `linkedTasks[]` item

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Human-readable task ID (e.g. `TASK-101`) |
| `taskNumber` | `integer` | Numeric part of the task ID |
| `title` | `string` | Task title |
| `status` | `string` | `todo` · `in-progress` · `done` · `closed` |
| `done` | `boolean` | Shorthand for whether the task is complete |
| `priority` | `string` | `low` · `medium` · `high` · `urgent` |
| `dueDate` | `string` | Task due date `YYYY-MM-DD` |
| `description` | `string` | Optional task description (omitted if empty) |
| `timeHours` | `number` | Estimated effort in hours (omitted if 0) |
| `timeDays` | `number` | Estimated effort in days (omitted if 0) |
| `updatedAt` | `string` | ISO 8601 timestamp of last update |

#### `summary` object

| Field | Description |
|-------|-------------|
| `total` | Total number of linked user goals (after optional status filter) |
| `onTrack` | Goals with status `on-track` |
| `ahead` | Goals with status `ahead` |
| `atRisk` | Goals with status `at-risk` |
| `behind` | Goals with status `behind` |
| `completed` | Goals with status `completed` |

> **Note:** If a `status` query param is supplied, `summary.total` reflects the filtered count only.

---

## Error Codes

| HTTP | When |
|------|------|
| `401` | Missing or invalid Cognito JWT |
| `403` | Caller is not an org admin for the org that owns this goal |
| `404` | Org-level goal not found (`goalId` does not exist in `OrgPerformanceTable`) |
| `500` | DynamoDB query failure or `PERF_HUB_TABLE` not configured |

---

## Infrastructure Notes

### DynamoDB GSI

A **sparse GSI** named `OrgGoalIdIndex` was added to `UserPerformanceHubTable`:

| Attribute | Role |
|-----------|------|
| `orgGoalId` | GSI hash key (only present on GoalRecord items that have been linked) |

Because `orgGoalId` is `omitempty` on user goal items and absent on all other record types (tasks, comments, meetings, etc.), only linked goal records appear in this index — no additional filtering is needed.

### IAM

`OrganizationLambdaRole` has been granted `dynamodb:Query` on `${UserPerformanceHubTable.Arn}/index/OrgGoalIdIndex`.

### Environment Variable

`ManagePerformanceGoalsLambda` receives `PERF_HUB_TABLE: !Ref UserPerformanceHubTable`.
