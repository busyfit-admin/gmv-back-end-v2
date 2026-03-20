# Org Performance API Documentation

## Overview
Org Performance APIs are exposed under `/v2` and implemented via split lambdas in `lambdas/tenant-lambdas/org-performance/`.

- `manage-performance-cycles`: cycles, quarters, meeting notes, analytics
- `manage-performance-kpis`: KPI CRUD, sub-KPIs, KPI values
- `manage-performance-okrs`: OKR CRUD, key-result updates
- `manage-performance-goals`: goals, value history, teams, sub-items, ladder-up, tasks

## Base URL
`https://{api-id}.execute-api.{region}.amazonaws.com/{stage}`

> All endpoints below are prefixed with `/v2`.

## Authentication & Authorization
- Requires Cognito JWT in `Authorization: Bearer <token>`.
- User identity source:
  1. `requestContext.authorizer.claims.sub`
  2. fallback header: `X-Cognito-Id`
- Most endpoints require org-admin access (`IsOrgAdmin`).

## Headers
- `Content-Type: application/json` (for body endpoints)
- `Authorization: Bearer <token>`
- `organization-id` or `Organization-Id` (required for `GET/POST /v2/kpis` and `GET/POST /v2/okrs`)

## Common Response Shapes

### Success
- `200/201`: JSON object payload from service
- `204`: empty body

### Error
```json
{
  "error": "short message",
  "message": "short message: detailed error"
}
```

## Common Query Params
- `page` (int)
- `pageSize` (int)
- `sortBy` (string)
- `order` (`asc|desc`)

---

## 1) Performance Cycles

### `GET /v2/organizations/{orgId}/performance-cycles`
- **Purpose:** list cycles for org
- **Input:** query: `status`, `fiscalYear`, `includeQuarters`, `includeKPIs`, `includeOKRs`, pagination params
- **Output (200):**
```json
{
  "performanceCycles": [
    {
      "id": "cycle-...",
      "organizationId": "ORG#...",
      "name": "FY 2027",
      "fiscalYear": "2027",
      "status": "PLANNING",
      "createdAt": "...",
      "updatedAt": "..."
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20,
  "totalPages": 1
}
```
- **Errors:** `401`, `403`, `500`

### `POST /v2/organizations/{orgId}/performance-cycles`
- **Purpose:** create cycle
- **Input body (typical):**
```json
{
  "name": "FY 2027",
  "fiscalYear": "2027",
  "startDate": "2027-01-01",
  "endDate": "2027-12-31",
  "description": "",
  "status": "PLANNING"
}
```
- **Output (201):** created cycle object (includes `id`, `organizationId`, timestamps)
- **Errors:** `400`, `401`, `403`, `422`

### `GET /v2/performance-cycles/{cycleId}`
- **Purpose:** get cycle details
- **Input query:** `includeQuarters` (default `true`), `includeKPIs` (default `true`), `includeOKRs` (default `true`), `includeAnalytics` (default `false`)
- **Output (200):** cycle object with optional `quarters`, `kpis`, `okrs`, `analytics`
- **Errors:** `401`, `403`, `404`

### `PATCH /v2/performance-cycles/{cycleId}`
- **Purpose:** update cycle fields
- **Input body:** partial JSON patch
```json
{
  "status": "STARTED",
  "description": "Updated description"
}
```
- **Output (200):** updated cycle object
- **Errors:** `400`, `401`, `403`, `404`, `500`

### `DELETE /v2/performance-cycles/{cycleId}`
- **Purpose:** delete cycle (with related data cleanup)
- **Output:** `204`
- **Errors:** `401`, `403`, `404`, `500`

### `GET /v2/performance-cycles/{cycleId}/analytics`
- **Purpose:** cycle analytics summary
- **Output (200):**
```json
{
  "cycleId": "cycle-...",
  "summary": {
    "totalKPIs": 10,
    "kpisOnTrack": 6,
    "kpisAtRisk": 3,
    "kpisBehind": 1,
    "totalOKRs": 4,
    "okrsCompleted": 1,
    "averageKPIProgress": 72.5,
    "averageOKRProgress": 61.0
  },
  "kpiTrends": [],
  "departmentPerformance": []
}
```
- **Errors:** `401`, `403`, `500`

---

## 2) Quarters & Meeting Notes

### `GET /v2/performance-cycles/{cycleId}/quarters`
- **Purpose:** list quarters for cycle
- **Output (200):** `{ "quarters": [ ... ] }`
- **Errors:** `401`, `403`, `404`, `500`

### `POST /v2/performance-cycles/{cycleId}/quarters`
- **Purpose:** create quarter
- **Input body (typical):**
```json
{
  "name": "Q1",
  "startDate": "2027-01-01",
  "endDate": "2027-03-31",
  "status": "PLANNING"
}
```
- **Output (201):** created quarter object
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `GET /v2/quarters/{quarterId}`
- **Input query:** `includeKPIs`, `includeOKRs`, `includeMeetingNotes`, `includePendingReviews`
- **Output (200):** quarter object + optional nested fields
- **Errors:** `401`, `403`, `404`

### `PATCH /v2/quarters/{quarterId}`
- **Input:** partial JSON patch
- **Output (200):** updated quarter object
- **Errors:** `400`, `401`, `403`, `404`, `500`

### `DELETE /v2/quarters/{quarterId}`
- **Output:** `204`
- **Errors:** `401`, `403`, `404`, `500`

### `GET /v2/quarters/{quarterId}/meeting-notes`
- **Input query:** `sortBy`, `order`
- **Output (200):** `{ "meetingNotes": [ ... ] }`
- **Errors:** `401`, `403`, `404`, `500`

### `POST /v2/quarters/{quarterId}/meeting-notes`
- **Input body (typical):**
```json
{
  "title": "Weekly review",
  "date": "2027-01-10",
  "notes": "Discussion notes"
}
```
- **Output (201):** meeting note object
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `PATCH /v2/meeting-notes/{noteId}`
- **Input:** partial patch
- **Output (200):** updated note object
- **Errors:** `400`, `401`, `403`, `500`

### `DELETE /v2/meeting-notes/{noteId}`
- **Output:** `204`
- **Errors:** `401`, `500`

### `GET /v2/quarters/{quarterId}/analytics`
- **Output (200):** quarter-scoped analytics object
- **Errors:** `401`, `403`, `404`, `500`

---

## 3) KPI APIs

### `GET /v2/kpis`
- **Required header:** `organization-id` / `Organization-Id`
- **Input query:** `cycleId`, `quarterId`, `department`, `owner`, `status`, `parentKpiId`, `includeSubKPIs`, pagination params
- **Output (200):**
```json
{
  "kpis": [ { "id": "kpi-...", "name": "Revenue", "status": "PLANNING" } ],
  "total": 1,
  "page": 1,
  "pageSize": 20,
  "totalPages": 1
}
```
- **Errors:** `400`, `401`, `403`, `500`

### `POST /v2/kpis`
- **Required header:** `organization-id` / `Organization-Id`
- **Input body (required fields):**
```json
{
  "cycleId": "cycle-...",
  "name": "Revenue Growth",
  "owner": "user@company.com",
  "targetValue": 100
}
```
- **Validation rules (service):**
  - `name` required, <= 200 chars
  - `owner` required
  - `targetValue` required
  - `status` in `PLANNING|STARTED|FINALIZED|CLOSED` (if provided)
  - `reportingFrequency` in `daily|weekly|monthly|quarterly|annually` (if provided)
  - threshold rule: `green >= amber >= red` (if thresholds provided)
  - `trend` in `up|down|stable` (if provided)
  - `incentiveImpact` in `yes|no` (if provided)
- **Output (201):** created KPI object
- **Errors:** `400`, `401`, `403`, `422`

### `GET /v2/kpis/{kpiId}`
- **Input query:** `includeSubKPIs`, `includeValueHistory`
- **Output (200):** KPI object + optional `subKPIs`, `valueHistory`
- **Errors:** `401`, `403`, `404`

### `PATCH /v2/kpis/{kpiId}`
- **Input:** partial patch
- **Output (200):** updated KPI object
- **Errors:** `400`, `401`, `403`, `404`, `500`

### `DELETE /v2/kpis/{kpiId}`
- **Input query:** `deleteSubKPIs` (bool)
- **Output:** `204`
- **Errors:** `401`, `403`, `404`, `500`

### `POST /v2/kpis/{kpiId}/sub-kpis`
- **Purpose:** create child KPI under parent KPI
- **Input:** same shape/validation as KPI create
- **Output (201):** created KPI object with `parentKpiId`
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `POST /v2/kpis/{kpiId}/values`
- **Input body (typical):**
```json
{
  "value": 75,
  "date": "2027-02-10",
  "comment": "Mid-cycle update"
}
```
- **Output (201):** created KPI value entry
- **Side effect:** updates KPI `currentValue`
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `GET /v2/kpis/{kpiId}/targets`
- **Purpose:** list all targets (milestone sub-goals) defined for the KPI
- **Output (200):**
```json
{
  "targets": [
    {
      "id": "kpi-target-...",
      "kpiId": "kpi-...",
      "name": "Q1 milestone",
      "targetValue": 50,
      "targetValueType": "%",
      "currentValue": 20,
      "status": "ON_TRACK",
      "createdAt": "...",
      "updatedAt": "..."
    }
  ]
}
```
- **Errors:** `401`, `403`, `404`

### `POST /v2/kpis/{kpiId}/targets`
- **Purpose:** add a new target (milestone sub-goal) to the KPI
- **Input body (typical):**
```json
{
  "name": "Q1 milestone",
  "targetValue": 50,
  "targetValueType": "%",
  "status": "ON_TRACK",
  "description": "Reach 50% by end of Q1"
}
```
- **`targetValueType`** (optional): unit/type of the target value — e.g. `"%"`, `"count"`, `"users"`, `"revenue"`, or any custom string
- **Output (201):** created KPI target object
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `PATCH /v2/kpi-targets/{kpiTargetId}`
- **Purpose:** update progress or status on a KPI target
- **Input body (typical):**
```json
{
  "status": "AT_RISK",
  "currentValue": 18,
  "targetValueType": "%",
  "comment": "Behind schedule"
}
```
- **Output (200):** updated KPI target object
- **Errors:** `400`, `401`, `403`, `500`

### `DELETE /v2/kpi-targets/{kpiTargetId}`
- **Purpose:** permanently delete a KPI target
- **Output:** `204` (no content)
- **Errors:** `401`, `403`, `404`, `500`

### `GET /v2/kpi-targets/{kpiTargetId}/comments`
- **Purpose:** list all comments for the specified KPI target, ordered by creation time
- **Output (200):**
```json
{
  "comments": [
    {
      "id": "comment-...",
      "targetId": "kpi-target-...",
      "text": "Looking good this quarter",
      "addedBy": "user@company.com",
      "createdAt": "2024-06-01T10:00:00Z"
    }
  ]
}
```
- **Errors:** `401`, `403`, `404`, `500`

### `POST /v2/kpi-targets/{kpiTargetId}/comments`
- **Purpose:** add a comment to a KPI target; `addedBy` is set automatically from the authenticated user
- **Required body field:** `text`
- **Input body:**
```json
{
  "text": "Adjusted forecast based on Q2 results"
}
```
- **Output (201):** created comment object
- **Errors:** `400`, `401`, `403`, `404`, `422`

---

## 4) OKR APIs

### `GET /v2/okrs`
- **Required header:** `organization-id` / `Organization-Id`
- **Input query:** `cycleId`, `quarterId`, `owner`, `status`, `includeKeyResults`, pagination params
- **Output (200):** paginated OKRs
```json
{
  "okrs": [ { "id": "okr-...", "name": "Expand market" } ],
  "total": 1,
  "page": 1,
  "pageSize": 20,
  "totalPages": 1
}
```
- **Errors:** `400`, `401`, `403`, `500`

### `POST /v2/okrs`
- **Required body:** `cycleId`
- **Input body (typical):**
```json
{
  "cycleId": "cycle-...",
  "quarterId": "quarter-...",
  "name": "Improve retention",
  "owner": "user@company.com",
  "status": "DRAFT",
  "keyResults": [
    { "name": "KR1", "targetValue": 50, "targetValueType": "%" }
  ]
}
```
- **Output (201):** created OKR object (with created key results when provided)
- **Errors:** `400`, `401`, `403`, `422`

### `GET /v2/okrs/{okrId}`
- **Input query:** `includeKeyResults` (default `true`), `includeProgressHistory`
- **Output (200):** OKR object + optional `keyResults`, `progressHistory`
- **Errors:** `401`, `403`, `404`

### `PATCH /v2/okrs/{okrId}`
- **Input:** partial patch
- **Output (200):** updated OKR object
- **Errors:** `400`, `401`, `403`, `404`, `500`

### `DELETE /v2/okrs/{okrId}`
- **Output:** `204`
- **Errors:** `401`, `403`, `404`, `500`

### `GET /v2/okrs/{okrId}/key-results`
- **Purpose:** list all key results for the specified OKR
- **Output (200):**
```json
{
  "keyResults": [
    {
      "id": "kr-...",
      "name": "KR1",
      "targetValue": 50,
      "targetValueType": "%",
      "currentValue": 20,
      "status": "ON_TRACK",
      "okrId": "okr-..."
    }
  ]
}
```
- **Errors:** `401`, `403`, `404`

### `POST /v2/okrs/{okrId}/key-results`
- **Purpose:** add a new key result to an existing OKR
- **Input body (typical):**
```json
{
  "name": "KR2",
  "targetValue": 100,
  "targetValueType": "users",
  "status": "ON_TRACK"
}
```
- **`targetValueType`** (optional): unit/type of the target value — e.g. `"%"`, `"count"`, `"users"`, `"revenue"`, or any custom string
- **Output (201):** created key result object
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `PATCH /v2/key-results/{keyResultId}`
- **Input body (typical):**
```json
{
  "status": "ON_TRACK",
  "currentValue": 20,
  "targetValueType": "users",
  "comment": "Updated"
}
```
- **Output (200):** updated key result object
- **Errors:** `400`, `401`, `403`, `500`

### `DELETE /v2/key-results/{keyResultId}`
- **Purpose:** permanently delete a key result
- **Output:** `204` (no content)
- **Errors:** `401`, `403`, `404`, `500`

### `GET /v2/key-results/{keyResultId}/comments`
- **Purpose:** list all comments for the specified key result, ordered by creation time
- **Output (200):**
```json
{
  "comments": [
    {
      "id": "comment-...",
      "keyResultId": "kr-...",
      "text": "Great progress this week",
      "addedBy": "user@company.com",
      "createdAt": "2024-06-01T10:00:00Z"
    }
  ]
}
```
- **Errors:** `401`, `403`, `404`, `500`

### `POST /v2/key-results/{keyResultId}/comments`
- **Purpose:** add a comment to a key result; `addedBy` is set automatically from the authenticated user
- **Required body field:** `text`
- **Input body:**
```json
{
  "text": "Completed the user interviews milestone"
}
```
- **Output (201):** created comment object
- **Errors:** `400`, `401`, `403`, `404`, `422`

---

## 5) Goal APIs

### `GET /v2/goals/{goalId}`
- **Input query:**
  - `includeValueHistory`
  - `includeTaggedTeams`
  - `includeSubItems`
  - `includeLadderUp`
  - `includePrivateTasks`
- **Output (200):** normalized goal view
```json
{
  "id": "kpi-or-okr-id",
  "name": "Goal Name",
  "type": "kpi|okr",
  "owner": "user@company.com",
  "currentValue": 40,
  "targetValue": 100,
  "progress": 40,
  "organizationId": "ORG#..."
}
```
- **Errors:** `401`, `403`, `404`

### `PATCH /v2/goals/{goalId}`
- **Input:** partial patch
- **Output (200):** updated underlying KPI/OKR payload
- **Errors:** `400`, `401`, `403`, `404`, `500`

### `GET /v2/goals/{goalId}/value-history`
- **Input query:** `startDate`, `endDate`, pagination params
- **Output (200):**
```json
{
  "valueHistory": [ { "id": "goal-value-...", "value": 65, "date": "2027-01-15" } ],
  "total": 1,
  "page": 1,
  "pageSize": 20,
  "totalPages": 1
}
```
- **Errors:** `401`, `403`, `404`, `500`

### `POST /v2/goals/{goalId}/value-history`
- **Input body (typical):** `{ "value": 68, "date": "2027-01-20", "comment": "Progress" }`
- **Output (201):** created history entry
- **Side effect:** updates `currentValue` on the underlying KPI or OKR to the submitted `value`
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `GET /v2/goals/{goalId}/teams`
- **Output (200):** `{ "teams": [ ... ] }`
- **Errors:** `401`, `403`, `404`, `500`

### `POST /v2/goals/{goalId}/teams`
- **Required body field:** `teamId`
- **Input body (typical):** `{ "teamId": "TEAM#...", "alignmentReason": "Supports KPI" }`
- **Output (201):** tagged team record
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `DELETE /v2/goals/{goalId}/teams/{teamId}`
- **Output:** `204`
- **Errors:** `401`, `403`, `404`, `500`

### `GET /v2/goals/{goalId}/sub-items`
- **Output (200):** `{ "subItems": [ ... ] }`
- **Errors:** `401`, `403`, `404`, `500`

### `POST /v2/goals/{goalId}/sub-items`
- **Input body (typical):** `{ "title": "Sub work", "status": "PLANNING" }`
- **Output (201):** created sub-item
- **Errors:** `400`, `401`, `403`, `404`, `422`

### `PATCH /v2/sub-items/{subItemId}`
- **Input:** partial patch
- **Output (200):** updated sub-item
- **Errors:** `400`, `401`, `403`, `500`

### `DELETE /v2/sub-items/{subItemId}`
- **Output:** `204`
- **Errors:** `401`, `500`

### `GET /v2/goals/{goalId}/ladder-up`
- **Input query:** `status`
- **Output (200):** `{ "ladderUpItems": [ ... ] }`
- **Errors:** `401`, `403`, `404`, `500`

### `PATCH /v2/ladder-up/{ladderUpId}/approve`
- **Input body:** optional action metadata
- **Output (200):** updated ladder-up item (`status: APPROVED`)
- **Errors:** `400`, `401`, `403`, `404`, `500`

### `PATCH /v2/ladder-up/{ladderUpId}/reject`
- **Input body:** optional action metadata
- **Output (200):** updated ladder-up item (`status: REJECTED`)
- **Errors:** `400`, `401`, `403`, `404`, `500`

### `GET /v2/goals/{goalId}/tasks`
- **Input query:** `status`, pagination params
- **Output (200):**
```json
{
  "tasks": [
    { "id": "task-...", "title": "Follow up", "status": "todo", "userId": "user@company.com" }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20,
  "totalPages": 1
}
```
- **Errors:** `401`, `500`

### `POST /v2/goals/{goalId}/tasks`
- **Required body field:** `title`
- **Input body (typical):**
```json
{
  "title": "Prepare QBR notes",
  "description": "Draft by Friday",
  "status": "todo"
}
```
- **Validation rules:**
  - `title` required, <= 200 chars
  - `status` in `todo|in-progress|completed`
- **Output (201):** created task
- **Errors:** `400`, `401`, `422`

### `PATCH /v2/goals/{goalId}/tasks/{taskId}`
- **Input:** partial patch
- **Rules:** only task owner can update
- **Output (200):** updated task
- **Errors:** `400`, `401`, `403`, `500`

### `DELETE /v2/goals/{goalId}/tasks/{taskId}`
- **Rules:** only task owner can delete
- **Output:** `204`
- **Errors:** `401`, `403`, `500`

---

## Non-Functional Behavior
- `OPTIONS` preflight supported for all org-performance routes.
- Unknown route group path in split lambdas returns `404 Route not found`.
- Unsupported method for matched path returns `405 Method not allowed`.
- Path segments are URL-decoded in handler (supports encoded IDs like `%23`).
