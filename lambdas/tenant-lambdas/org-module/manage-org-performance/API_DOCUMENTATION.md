# Manage Org Performance API Documentation

## Overview
This Lambda handles organization performance management APIs under the `/v2` prefix.

- Lambda: `lambdas/tenant-lambdas/org-module/manage-org-performance/manage-org-performance.go`
- Core logic: `lambdas/lib/company-lib/company-org-performance.go`
- Swagger source: `swagger-docs/tenant/tenant-apis.yaml`

## Authentication & Authorization
- All endpoints require Cognito-authenticated requests.
- User identity is read from `requestContext.authorizer.claims.sub` (fallback header: `X-Cognito-Id`).
- Most endpoints enforce organization admin access via `orgSVC.IsOrgAdmin(orgID, userName)`.

## Common Conventions
- Base path: `/v2`
- Content type: `application/json`
- CORS preflight: `OPTIONS` returns `200`.
- Error body shape:

```json
{
  "error": "short message",
  "message": "detailed message"
}
```

## Common Query Options
List endpoints generally support these query params:
- `page` (int)
- `pageSize` (int)
- `sortBy` (string)
- `order` (`asc` | `desc`)

---

## API Endpoints

### Performance Cycles

#### List Performance Cycles
- **Method/Path:** `GET /v2/organizations/{organizationId}/performance-cycles`
- **Auth:** Org Admin for `{organizationId}`
- **Query Params:**
  - `includeQuarters` (bool, default `false`)
  - `includeKPIs` (bool, default `false`)
  - `includeOKRs` (bool, default `false`)
  - `status`, `fiscalYear`
  - common pagination/sort options
- **Responses:** `200`, `403`, `500`

#### Create Performance Cycle
- **Method/Path:** `POST /v2/organizations/{organizationId}/performance-cycles`
- **Auth:** Org Admin for `{organizationId}`
- **Body:** JSON payload for cycle fields (validated by service layer)
- **Responses:** `201`, `400`, `403`, `422`

#### Get Performance Cycle
- **Method/Path:** `GET /v2/performance-cycles/{cycleId}`
- **Auth:** Org Admin of owning org
- **Query Params:**
  - `includeQuarters` (default `true`)
  - `includeKPIs` (default `true`)
  - `includeOKRs` (default `true`)
  - `includeAnalytics` (default `false`)
- **Responses:** `200`, `403`, `404`

#### Update Performance Cycle
- **Method/Path:** `PATCH /v2/performance-cycles/{cycleId}`
- **Auth:** Org Admin of owning org
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `404`, `500`

#### Delete Performance Cycle
- **Method/Path:** `DELETE /v2/performance-cycles/{cycleId}`
- **Auth:** Org Admin of owning org
- **Responses:** `204`, `403`, `404`, `500`

#### Performance Cycle Analytics
- **Method/Path:** `GET /v2/performance-cycles/{cycleId}/analytics`
- **Auth:** Org Admin of owning org (with header-based fallback org check)
- **Responses:** `200`, `403`, `500`

---

### Quarters

#### List Quarters in Cycle
- **Method/Path:** `GET /v2/performance-cycles/{cycleId}/quarters`
- **Auth:** Org Admin of cycle org
- **Responses:** `200`, `403`, `404`, `500`

#### Create Quarter
- **Method/Path:** `POST /v2/performance-cycles/{cycleId}/quarters`
- **Auth:** Org Admin of cycle org
- **Body:** JSON payload for quarter fields
- **Responses:** `201`, `400`, `403`, `404`, `422`

#### Get Quarter
- **Method/Path:** `GET /v2/quarters/{quarterId}`
- **Auth:** Org Admin of owning org
- **Query Params:**
  - `includeKPIs`
  - `includeOKRs`
  - `includeMeetingNotes`
  - `includePendingReviews`
- **Responses:** `200`, `403`, `404`

#### Update Quarter
- **Method/Path:** `PATCH /v2/quarters/{quarterId}`
- **Auth:** Org Admin of owning org
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `404`, `500`

#### Delete Quarter
- **Method/Path:** `DELETE /v2/quarters/{quarterId}`
- **Auth:** Org Admin of owning org
- **Responses:** `204`, `403`, `404`, `500`

#### Quarter Analytics
- **Method/Path:** `GET /v2/quarters/{quarterId}/analytics`
- **Auth:** Org Admin of owning org
- **Responses:** `200`, `403`, `404`, `500`

---

### KPIs

#### List KPIs
- **Method/Path:** `GET /v2/kpis`
- **Auth:** Org Admin from header org ID
- **Required Header:** `organization-id` or `Organization-Id`
- **Query Params:**
  - `cycleId`, `quarterId`, `department`, `owner`, `status`, `parentKpiId`
  - `includeSubKPIs`
  - common pagination/sort options
- **Responses:** `200`, `400`, `403`, `500`

#### Create KPI
- **Method/Path:** `POST /v2/kpis`
- **Auth:** Org Admin from header org ID
- **Required Header:** `organization-id` or `Organization-Id`
- **Body:** KPI payload
- **Responses:** `201`, `400`, `403`, `422`

#### Get KPI
- **Method/Path:** `GET /v2/kpis/{kpiId}`
- **Auth:** Org Admin of owning org
- **Query Params:** `includeSubKPIs`, `includeValueHistory`
- **Responses:** `200`, `403`, `404`

#### Update KPI
- **Method/Path:** `PATCH /v2/kpis/{kpiId}`
- **Auth:** Org Admin of owning org
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `404`, `500`

#### Delete KPI
- **Method/Path:** `DELETE /v2/kpis/{kpiId}`
- **Auth:** Org Admin of owning org
- **Query Params:** `deleteSubKPIs` (bool)
- **Responses:** `204`, `403`, `404`, `500`

#### Create Sub-KPI
- **Method/Path:** `POST /v2/kpis/{kpiId}/sub-kpis`
- **Auth:** Org Admin of parent KPI org
- **Body:** KPI payload for child
- **Responses:** `201`, `400`, `403`, `404`, `422`

#### Add KPI Value Entry
- **Method/Path:** `POST /v2/kpis/{kpiId}/values`
- **Auth:** Org Admin of KPI org
- **Body:** Value entry payload
- **Responses:** `201`, `400`, `403`, `404`, `422`

---

### OKRs & Key Results

#### List OKRs
- **Method/Path:** `GET /v2/okrs`
- **Auth:** Org Admin from header org ID
- **Required Header:** `organization-id` or `Organization-Id`
- **Query Params:**
  - `cycleId`, `quarterId`, `owner`, `status`
  - `includeKeyResults`
  - common pagination/sort options
- **Responses:** `200`, `400`, `403`, `500`

#### Create OKR
- **Method/Path:** `POST /v2/okrs`
- **Auth:** Org Admin from header org ID
- **Required Header:** `organization-id` or `Organization-Id`
- **Body:** OKR payload
- **Responses:** `201`, `400`, `403`, `422`

#### Get OKR
- **Method/Path:** `GET /v2/okrs/{okrId}`
- **Auth:** Org Admin of owning org
- **Query Params:** `includeKeyResults` (default `true`), `includeProgressHistory`
- **Responses:** `200`, `403`, `404`

#### Update OKR
- **Method/Path:** `PATCH /v2/okrs/{okrId}`
- **Auth:** Org Admin of owning org
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `404`, `500`

#### Delete OKR
- **Method/Path:** `DELETE /v2/okrs/{okrId}`
- **Auth:** Org Admin of owning org
- **Responses:** `204`, `403`, `404`, `500`

#### Update Key Result
- **Method/Path:** `PATCH /v2/key-results/{keyResultId}`
- **Auth:** Org Admin of owning org (validated after update response)
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `500`

---

### Meeting Notes

#### List Quarter Meeting Notes
- **Method/Path:** `GET /v2/quarters/{quarterId}/meeting-notes`
- **Auth:** Org Admin of quarter org
- **Query Params:** `sortBy`, `order`
- **Responses:** `200`, `403`, `404`, `500`

#### Create Meeting Note
- **Method/Path:** `POST /v2/quarters/{quarterId}/meeting-notes`
- **Auth:** Org Admin of quarter org
- **Body:** Meeting note payload
- **Responses:** `201`, `400`, `403`, `404`, `422`

#### Update Meeting Note
- **Method/Path:** `PATCH /v2/meeting-notes/{noteId}`
- **Auth:** Org Admin of owning org (validated after update response)
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `500`

#### Delete Meeting Note
- **Method/Path:** `DELETE /v2/meeting-notes/{noteId}`
- **Auth:** Service operation, no explicit pre-check in handler
- **Responses:** `204`, `500`

---

### Goals

#### Get Goal Details
- **Method/Path:** `GET /v2/goals/{goalId}`
- **Auth:** Org Admin of owning org
- **Query Params:**
  - `includeValueHistory`
  - `includeTaggedTeams`
  - `includeSubItems`
  - `includeLadderUp`
  - `includePrivateTasks`
- **Responses:** `200`, `403`, `404`

#### Update Goal
- **Method/Path:** `PATCH /v2/goals/{goalId}`
- **Auth:** Org Admin of owning org
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `404`, `500`

#### List Goal Value History
- **Method/Path:** `GET /v2/goals/{goalId}/value-history`
- **Auth:** Org Admin of owning org
- **Query Params:** `startDate`, `endDate` + common pagination/sort options
- **Responses:** `200`, `403`, `404`, `500`

#### Add Goal Value Entry
- **Method/Path:** `POST /v2/goals/{goalId}/value-history`
- **Auth:** Org Admin of owning org
- **Body:** Value history payload
- **Responses:** `201`, `400`, `403`, `404`, `422`

#### List Tagged Teams
- **Method/Path:** `GET /v2/goals/{goalId}/teams`
- **Auth:** Org Admin of owning org
- **Responses:** `200`, `403`, `404`, `500`

#### Tag Team to Goal
- **Method/Path:** `POST /v2/goals/{goalId}/teams`
- **Auth:** Org Admin of owning org
- **Body:** Team tag payload
- **Responses:** `201`, `400`, `403`, `404`, `422`

#### Remove Team Tag
- **Method/Path:** `DELETE /v2/goals/{goalId}/teams/{teamId}`
- **Auth:** Org Admin of owning org
- **Responses:** `204`, `403`, `404`, `500`

#### List Goal Sub-items
- **Method/Path:** `GET /v2/goals/{goalId}/sub-items`
- **Auth:** Org Admin of owning org
- **Responses:** `200`, `403`, `404`, `500`

#### Add Goal Sub-item
- **Method/Path:** `POST /v2/goals/{goalId}/sub-items`
- **Auth:** Org Admin of owning org
- **Body:** Sub-item payload
- **Responses:** `201`, `400`, `403`, `404`, `422`

#### Update Sub-item
- **Method/Path:** `PATCH /v2/sub-items/{subItemId}`
- **Auth:** Org Admin of owning org (validated after update response)
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `500`

#### Delete Sub-item
- **Method/Path:** `DELETE /v2/sub-items/{subItemId}`
- **Auth:** Service operation, no explicit pre-check in handler
- **Responses:** `204`, `500`

#### List Ladder-Up Items
- **Method/Path:** `GET /v2/goals/{goalId}/ladder-up`
- **Auth:** Org Admin of owning org
- **Query Params:** `status`
- **Responses:** `200`, `403`, `404`, `500`

#### Approve Ladder-Up Item
- **Method/Path:** `PATCH /v2/ladder-up/{ladderId}/approve`
- **Auth:** Org Admin of owning org (validated after update response)
- **Body:** Action payload
- **Responses:** `200`, `400`, `403`, `500`

#### Reject Ladder-Up Item
- **Method/Path:** `PATCH /v2/ladder-up/{ladderId}/reject`
- **Auth:** Org Admin of owning org (validated after update response)
- **Body:** Action payload
- **Responses:** `200`, `400`, `403`, `500`

---

### Goal Tasks (Private Tasks)

#### List Goal Tasks
- **Method/Path:** `GET /v2/goals/{goalId}/tasks`
- **Auth:** User-context access (`userName`) enforced in service layer
- **Query Params:** `status` + common pagination/sort options
- **Responses:** `200`, `500`

#### Create Goal Task
- **Method/Path:** `POST /v2/goals/{goalId}/tasks`
- **Auth:** User-context access (`userName`) enforced in service layer
- **Body:** Task payload
- **Responses:** `201`, `400`, `422`

#### Update Goal Task
- **Method/Path:** `PATCH /v2/goals/{goalId}/tasks/{taskId}`
- **Auth:** User-context access (`userName`) enforced in service layer
- **Body:** Partial JSON patch
- **Responses:** `200`, `400`, `403`, `500`

#### Delete Goal Task
- **Method/Path:** `DELETE /v2/goals/{goalId}/tasks/{taskId}`
- **Auth:** User-context access (`userName`) enforced in service layer
- **Responses:** `204`, `403`, `500`

---

## Notes
- Unsupported route/method combinations return `405 Method not allowed`.
- Unknown route fragments inside routed prefixes can return `404 Route not found` for specific branches.
- Detailed field-level validation rules are implemented in `PerformanceService` and may evolve independent of this document.