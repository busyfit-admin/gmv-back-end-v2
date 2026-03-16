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

KPIs and OKRs are returned in the same `goals` array but use **different field sets** because they are stored differently in DynamoDB.

#### KPI item

```json
{
  "id": "kpi-uuid",
  "type": "kpi",
  "name": "Support Tickets Resolved",
  "description": "Number of support tickets resolved per month",
  "owner": "jane.smith@company.com",
  "status": "active",
  "cycleId": "cycle-uuid",
  "quarterId": "quarter-uuid",
  "currentValue": 42,
  "targetValue": 100,
  "unit": "tickets",
  "deadline": "2026-03-31",
  "progress": 42.0,
  "createdAt": "2026-01-10T08:00:00Z",
  "updatedAt": "2026-03-01T12:00:00Z"
}
```

#### OKR item

```json
{
  "id": "okr-uuid",
  "type": "okr",
  "name": "Strengthen product foundation and improve user onboarding.",
  "objective": "Strengthen product foundation and improve user onboarding.",
  "owner": "Vishal",
  "status": "FINALIZED",
  "cycleId": "cycle-uuid",
  "quarterId": "quarter-uuid",
  "timeBound": "Half-Yearly",
  "deadline": "Half-Yearly",
  "confidenceScore": 70,
  "keyResults": [
    {
      "description": "Reduce customer onboarding time from 7 days → 3 days",
      "target": "7 days",
      "currentValue": 0,
      "unit": "Count",
      "weight": 20,
      "owner": ""
    }
  ],
  "currentValue": null,
  "targetValue": null,
  "unit": "",
  "progress": 0,
  "createdAt": "2026-03-11T02:21:59Z",
  "updatedAt": "2026-03-11T02:21:59Z"
}
```

> **Note:** For OKRs, `progress` is always `0` at the OKR level — progress lives on individual key results. `currentValue`/`targetValue`/`unit` are also `null`/empty at the OKR level for the same reason.

```json
{
  "goals": [ ...see above... ],
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

| Field | Type | Present on | Description |
|-------|------|------------|-------------|
| `id` | `string` | both | Unique goal ID |
| `type` | `string` | both | `"kpi"` or `"okr"` |
| `name` | `string` | both | Display name. For OKRs this mirrors `objective`. |
| `objective` | `string` | OKR only | The raw objective text stored in DynamoDB |
| `description` | `string` | KPI only | KPI description |
| `owner` | `string` | both | For KPIs: `owner` field. For OKRs: `objectiveOwner` field. |
| `status` | `string` | both | e.g. `active`, `completed`, `FINALIZED`, `on-track` |
| `cycleId` | `string` | both | Performance cycle this goal belongs to |
| `quarterId` | `string\|null` | both | Quarter identifier, if set |
| `deadline` | `string` | both | KPI: `endDate` (`YYYY-MM-DD`). OKR: `timeBound` (e.g. `"Half-Yearly"`). |
| `timeBound` | `string` | OKR only | Raw time-bound descriptor (e.g. `"Half-Yearly"`, `"Monthly"`) |
| `currentValue` | `number\|null` | KPI only | Current measured value. `null` for OKRs. |
| `targetValue` | `number\|null` | KPI only | Target value. `null` for OKRs. |
| `unit` | `string` | KPI only | Unit of measurement. Empty for OKRs. |
| `progress` | `number` | KPI only | `(currentValue / targetValue) × 100`. Always `0` for OKRs. |
| `confidenceScore` | `number` | OKR only | Confidence score (0–100) |
| `keyResults` | `array` | OKR only | Inline key result objects (each has `description`, `target`, `currentValue`, `unit`, `weight`, `owner`) |
| `createdAt` | `string` | both | ISO 8601 creation timestamp |
| `updatedAt` | `string` | both | ISO 8601 last update timestamp |

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
