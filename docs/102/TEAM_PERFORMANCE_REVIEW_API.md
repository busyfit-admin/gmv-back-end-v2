# Team Performance Review — API Reference

All endpoints require a Cognito JWT in the `Authorization` header.  
Every response is wrapped in `{ "data": { ... } }` on success or `{ "error": { "code": "...", "message": "..." } }` on failure.  
Dates are `YYYY-MM-DD`, timestamps are ISO 8601 (UTC).

> **Manager view** — all endpoints under `/v2/teams/{teamId}/...` are used by a manager reviewing their team's performance. The caller must be a member of the requested team, otherwise a `403` is returned.

> **Lambda** — routed through `ManageTeamPerformanceLambda`. Shares the same `UserPerformanceHubTable` (single-table design) and IAM role as `ManageUserPerformanceLambda`.

---

## Summary Table

| # | Method | Endpoint | Purpose |
|---|--------|----------|---------|
| 1 | GET | `/v2/teams/{teamId}/performance/members` | List all team members with review status |
| 2 | GET | `/v2/teams/{teamId}/members/{username}/goals` | Member OKRs & KPIs |
| 3 | GET | `/v2/teams/{teamId}/members/{username}/meetings` | Member 1-on-1 meeting history |
| 4 | GET | `/v2/teams/{teamId}/members/{username}/appreciations` | Appreciations received by member |
| 5 | GET | `/v2/teams/{teamId}/members/{username}/comments` | Manager comments & feedback |
| 6 | POST | `/v2/teams/{teamId}/members/{username}/comments` | Add a manager comment |
| — | GET | `/v2/teams/{teamId}/members/{username}/performance-summary` | *(optional)* All-in-one detail |

---

## 1. Team Members List

### GET `/v2/teams/{teamId}/performance/members`
Returns all members of a team enriched with their performance review lifecycle state. Use this to populate the manager's team review grid.

**Path params**
| Param | Description |
|-------|-------------|
| `teamId` | UUID of the team |

**Response `200`**
```json
{
  "data": {
    "members": [
      {
        "id": "jane.smith",
        "name": "Jane Smith",
        "initials": "JS",
        "role": "MEMBER",
        "department": "",
        "avatarColor": "",
        "overallRating": 4.5,
        "lastReviewDate": "2026-02-28",
        "isPendingReview": true,
        "hasUserUpdatedReviews": false
      }
    ]
  }
}
```

**Field notes**
| Field | Description |
|-------|-------------|
| `id` | The member's `userName` (email-based) |
| `role` | `OWNER` · `ADMIN` · `MEMBER` · `GUEST` |
| `overallRating` | `0.0` when no review record exists yet |
| `lastReviewDate` | `null` when the member has never been reviewed |
| `isPendingReview` | `true` when the manager has not yet completed the review cycle |
| `hasUserUpdatedReviews` | `false` when the member has not submitted their self-review |

**Status badge logic (frontend reference)**
| Condition | Badge |
|-----------|-------|
| `isPendingReview: true` | 🟡 Review Pending |
| `hasUserUpdatedReviews: false` | 🔴 Self-Review Not Updated |
| Header: pending count > 0 | 🟡 `N Reviews Pending` |
| Header: missing self-review count > 0 | 🔴 `N Self-Reviews Missing` |

---

## 2. Member Detail

All detail endpoints below take the same path prefix: `/v2/teams/{teamId}/members/{username}/...`

| Param | Description |
|-------|-------------|
| `teamId` | UUID of the team |
| `username` | The member's `userName` |

---

### 2.1 GET `/v2/teams/{teamId}/members/{username}/goals`
Returns all goals for a member split into OKRs and KPIs.

**Response `200`**
```json
{
  "data": {
    "okrs": [
      {
        "id": "uuid",
        "title": "Grow user retention",
        "status": "on-track",
        "progress": 75,
        "dueDate": "2026-06-30",
        "keyResults": []
      }
    ],
    "kpis": [
      {
        "id": "uuid",
        "name": "Support tickets resolved",
        "current": 14,
        "target": 100,
        "unit": "",
        "frequency": "Monthly",
        "trend": "up",
        "change": ""
      }
    ]
  }
}
```

**Field notes**
- `okrs` — goals stored with `type = "okr"`. `keyResults` is an empty array until the schema is extended with sub-key-result records.
- `kpis` — goals stored with `type = "kpi"`. `current` maps to the goal's `progress` value (0–100). `target` defaults to `100`.
- `status` values: `on-track` · `at-risk` · `behind` · `completed` · `ahead`

---

### 2.2 GET `/v2/teams/{teamId}/members/{username}/meetings`
Returns all 1-on-1 meeting records for a member (newest first).

**Response `200`**
```json
{
  "data": {
    "meetings": [
      {
        "id": "uuid",
        "date": "2026-03-01",
        "title": "Q1 check-in",
        "notes": "Q1 check-in",
        "actionItems": ["Follow up on goal progress", "Schedule next 1-on-1"]
      }
    ]
  }
}
```

> `title` and `notes` are both sourced from the meeting's `summary` field. A separate `title` attribute may be added in a future schema revision.

---

### 2.3 GET `/v2/teams/{teamId}/members/{username}/appreciations`
Returns all appreciations received by a member (newest first).

**Response `200`**
```json
{
  "data": {
    "appreciations": [
      {
        "id": "uuid",
        "from": "john.doe",
        "fromInitials": "JD",
        "date": "2026-02-20",
        "message": "Outstanding work on the sprint demo!",
        "category": "teamwork"
      }
    ]
  }
}
```

---

### 2.4 GET `/v2/teams/{teamId}/members/{username}/comments`
Returns all manager comments written for a member (newest first).

**Response `200`**
```json
{
  "data": {
    "comments": [
      {
        "id": "uuid",
        "author": "Alice Manager",
        "authorInitials": "AM",
        "date": "2026-03-05",
        "text": "Strong delivery this quarter, keep it up.",
        "type": "feedback"
      }
    ]
  }
}
```

`type` values: `feedback` · `coaching` · `general`

---

### 2.5 POST `/v2/teams/{teamId}/members/{username}/comments`
Adds a manager comment for a team member. Author name and initials are resolved from the authenticated manager's employee record.

**Request body**
```json
{
  "text": "string",          // required
  "type": "feedback"         // optional — feedback | coaching | general (default: "general")
}
```

**Response `201`**
```json
{
  "data": {
    "comment": {
      "id": "uuid",
      "author": "Alice Manager",
      "authorInitials": "AM",
      "date": "2026-03-05",
      "text": "Strong delivery this quarter, keep it up.",
      "type": "feedback"
    }
  }
}
```

---

## 3. Performance Summary (All-in-One)

### GET `/v2/teams/{teamId}/members/{username}/performance-summary`
Returns everything about a member in a single call — use this when latency matters more than granular caching.

**Response `200`**
```json
{
  "data": {
    "profile": {
      "id": "jane.smith",
      "name": "Jane Smith",
      "initials": "JS",
      "role": "MEMBER",
      "department": "",
      "avatarColor": "",
      "overallRating": 4.5,
      "lastReviewDate": "2026-02-28",
      "isPendingReview": true,
      "hasUserUpdatedReviews": false
    },
    "okrs": [ /* same shape as §2.1 */ ],
    "kpis": [ /* same shape as §2.1 */ ],
    "meetings": [ /* same shape as §2.2 */ ],
    "appreciations": [ /* same shape as §2.3 */ ],
    "comments": [ /* same shape as §2.4 */ ]
  }
}
```

---

## DynamoDB Storage Notes

`UserPerformanceHubTable` single-table design — new SK patterns added for team performance:

| PK | SK | Record type |
|----|-----|-------------|
| `USER#{memberUserName}#TEAM#{teamId}` | `MGRCMT#{commentId}` | `ManagerCommentRecord` — manager-written comment |
| `TEAM#{teamId}` | `REVIEW#MEMBER#{memberUserName}` | `TeamMemberReviewRecord` — review lifecycle state |

All other SK patterns (`GOAL#`, `MEETING#`, `APPR#`, `FBREQ#`, `TASK#`) are read directly from the member's own partition — managers have read-only access to these records.

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
| `403` | `FORBIDDEN` | Caller is not a member of the requested team |
| `404` | `NOT_FOUND` | Member or resource not found |
| `405` | `METHOD_NOT_ALLOWED` | Wrong HTTP verb for the route |
| `500` | `INTERNAL_ERROR` | DynamoDB or unexpected server error |
