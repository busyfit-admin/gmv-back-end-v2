# Team Goals — API Reference

All endpoints require a Cognito JWT in the `Authorization` header.  
Pass `Organization-Id` as a header to identify the organisation.  
Callers must be an **org admin** — a `403` is returned otherwise.  
Dates are `YYYY-MM-DD`, timestamps are ISO 8601 (UTC).

> **Lambda** — routed through `ManagePerformanceGoalsLambda` (same lambda used for `/v2/goals/...`).  
> Uses the same `OrgPerformanceTable` (single-table DynamoDB design).

---

## Summary Table

| # | Method | Endpoint | Purpose |
|---|--------|----------|---------|
| 1 | GET | `/v2/teams/{teamId}/goals` | List all OKRs & KPIs tagged to a team |

---

## 1. List Team Goals

### GET `/v2/teams/{teamId}/goals`

Returns a paginated list of all OKRs and KPIs that have been tagged to a specific team.  
This is the reverse of the goal→team tagging flow (`POST /v2/goals/{goalId}/teams`).

---

### Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | ✅ | `Bearer <cognito-jwt>` |
| `Organization-Id` | ✅ | UUID of the organisation |

---

### Path Parameters

| Param | Description |
|-------|-------------|
| `teamId` | UUID of the team whose goals you want to list |

---

### Query Parameters

| Param | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | `string` | ❌ | Filter by goal type. Accepted values: `kpi`, `okr`. Omit to return both. |
| `cycleId` | `string` | ❌ | Filter by performance cycle ID |
| `status` | `string` | ❌ | Filter by goal status (e.g. `active`, `completed`, `on-track`) |
| `page` | `integer` | ❌ | Page number (default: `1`) |
| `pageSize` | `integer` | ❌ | Items per page (default: `20`) |
| `sortBy` | `string` | ❌ | Field to sort by (e.g. `name`, `status`, `createdAt`) |
| `order` | `string` | ❌ | Sort direction: `asc` or `desc` |

---

### Response `200`

```json
{
  "goals": [
    {
      "id": "kpi-uuid-here",
      "type": "kpi",
      "name": "Support Tickets Resolved",
      "description": "Number of support tickets resolved per month",
      "owner": "jane.smith@company.com",
      "status": "active",
      "cycleId": "cycle-uuid-here",
      "quarterId": "q1-2026",
      "currentValue": 42,
      "targetValue": 100,
      "unit": "tickets",
      "deadline": "2026-03-31",
      "progress": 42.0,
      "createdAt": "2026-01-10T08:00:00Z",
      "updatedAt": "2026-03-01T12:00:00Z"
    },
    {
      "id": "okr-uuid-here",
      "type": "okr",
      "name": "Grow User Retention",
      "description": "Increase 30-day retention to 80%",
      "owner": "team.lead@company.com",
      "status": "on-track",
      "cycleId": "cycle-uuid-here",
      "quarterId": "q1-2026",
      "currentValue": 72,
      "targetValue": 80,
      "unit": "%",
      "deadline": "2026-03-31",
      "progress": 90.0,
      "createdAt": "2026-01-10T08:00:00Z",
      "updatedAt": "2026-03-10T09:00:00Z"
    }
  ],
  "total": 12,
  "page": 1,
  "pageSize": 20,
  "totalPages": 1
}
```

---

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `goals` | `array` | Array of goal objects (mixed OKRs and KPIs unless filtered) |
| `total` | `integer` | Total matched goals before pagination |
| `page` | `integer` | Current page number |
| `pageSize` | `integer` | Number of items per page |
| `totalPages` | `integer` | Total number of pages |

#### Goal Object Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Unique goal ID |
| `type` | `string` | `"kpi"` or `"okr"` |
| `name` | `string` | Goal title |
| `description` | `string` | Goal description |
| `owner` | `string` | `userName` of the goal owner |
| `status` | `string` | Current status (e.g. `active`, `completed`, `on-track`, `at-risk`) |
| `cycleId` | `string` | Performance cycle this goal belongs to |
| `quarterId` | `string\|null` | Quarter identifier, if set |
| `currentValue` | `number\|null` | Current measured value |
| `targetValue` | `number\|null` | Target value to reach |
| `unit` | `string` | Unit of measurement (e.g. `%`, `tickets`, `$`) |
| `deadline` | `string` | Target end date (`YYYY-MM-DD`) |
| `progress` | `number` | Percentage towards target (`0–100`). `0` when `targetValue` is 0. |
| `createdAt` | `string` | ISO 8601 creation timestamp |
| `updatedAt` | `string` | ISO 8601 last update timestamp |

---

### Error Responses

| Status | Meaning |
|--------|---------|
| `400` | Missing or invalid `Organization-Id` header |
| `403` | Caller is not an org admin |
| `500` | Internal server error |

---

### Example Requests

#### Fetch all goals tagged to a team
```http
GET /v2/teams/team-abc-123/goals
Authorization: Bearer <token>
Organization-Id: org-xyz-456
```

#### Fetch only KPIs for a specific cycle
```http
GET /v2/teams/team-abc-123/goals?type=kpi&cycleId=cycle-q1-2026
Authorization: Bearer <token>
Organization-Id: org-xyz-456
```

#### Fetch active OKRs, page 2
```http
GET /v2/teams/team-abc-123/goals?type=okr&status=active&page=2&pageSize=10
Authorization: Bearer <token>
Organization-Id: org-xyz-456
```

---

## Related Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v2/goals/{goalId}/teams` | List teams tagged to a specific goal |
| POST | `/v2/goals/{goalId}/teams` | Tag a team to a goal |
| DELETE | `/v2/goals/{goalId}/teams/{teamId}` | Remove a team tag from a goal |
| GET | `/v2/goals` | List all goals for the organisation |
| GET | `/v2/goals/{goalId}` | Get a single goal's details |
