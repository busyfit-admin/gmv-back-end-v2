# AI Chat API

## Overview

| Property | Value |
|---|---|
| Lambda | `AIChatHandlerLambda` |
| Bedrock model | `amazon.nova-pro-v1:0` (Amazon Nova Pro) |
| DynamoDB table | `AIChatHistoryTable-{Environment}` |
| Chat history TTL | 6 months (auto-deleted via DynamoDB TTL on `expiresAt` attribute) |

### Required request headers

| Header | Required | Description |
|---|---|---|
| `Authorization` | ✅ | Cognito JWT issued by the tenant User Pool |
| `Organization-Id` | ✅ | Caller's organisation ID |

### Caller permissions

This endpoint is accessible to **all authenticated users**: admins, performance-admins, and regular members. The AI assistant automatically scopes its answers to the caller's identity (username, team, org) resolved from the Cognito token.

### Date / timestamp format

All dates returned by tool results use **ISO 8601 UTC** strings (e.g. `2024-07-01T00:00:00Z`).

---

## Summary

| # | Method | Path | Purpose |
|---|---|---|---|
| 1 | `POST` | `/v2/ai/chat` | Send a message; receive a grounded AI response |

---

## 1. POST /v2/ai/chat

Send a natural-language message to the AI performance management assistant. The assistant can autonomously call internal data-retrieval tools to answer questions about goals, tasks, meetings, team performance, KPIs, OKRs, and more.

Multi-turn conversations are supported via the `chatId` field. When a `chatId` is provided, the Lambda loads the most recent 20 messages from chat history and feeds them to the model as conversation context. Conversation history persists for **6 months** after the last message timestamp.

### Headers

| Header | Description |
|---|---|
| `Authorization` | `Bearer <cognito-jwt>` |
| `Organization-Id` | Caller's organisation ID |
| `Content-Type` | `application/json` |

### Request body

```json
{
  "chatId": "3f7a1c2d-8e5b-4f0a-9012-abc123def456",
  "message": "What is the progress on my team's Q3 goals?",
  "context": {
    "teamId": "team-uuid",
    "orgId": "org-uuid",
    "targetUserId": "jane.smith"
  }
}
```

| Field | Type | Required | Description |
|---|---|---|---|
| `chatId` | `string` (UUID) | ❌ | Existing session ID. Omit to start a new conversation; a new UUID will be generated and returned. |
| `message` | `string` | ✅ | The user's natural-language input. |
| `context.teamId` | `string` | ❌ | Scopes tools to a specific team. Defaults to the caller's team derived from Cognito claims. |
| `context.orgId` | `string` | ❌ | Scopes tools to a specific organisation. Defaults to the caller's org. |
| `context.targetUserId` | `string` | ❌ | For manager/admin use: the username of the member the request concerns. |

### Response 200

```json
{
  "chatId": "3f7a1c2d-8e5b-4f0a-9012-abc123def456",
  "response": "Your team currently has 12 active Q3 goals. 5 are on-track, 3 are ahead of schedule, 2 are at-risk, and 2 are behind. The highest-priority at-risk goal is \"Increase Trial Conversion\" owned by john.doe with 34% progress against a 60% target.",
  "toolsUsed": ["get_team_org_goals", "get_user_goals_for_org_goal"]
}
```

| Field | Type | Description |
|---|---|---|
| `chatId` | `string` | The session UUID (echoed back or newly generated). Save this for subsequent requests in the same conversation. |
| `response` | `string` | The AI assistant's natural-language answer. |
| `toolsUsed` | `string[]` | Internal data-retrieval tool names invoked by the model during this turn (for transparency / debugging). |

### Error responses

| Status | Cause |
|---|---|
| `400` | Request body is missing or `message` is empty. |
| `401` | Cognito token is absent or invalid. |
| `500` | Bedrock service error, downstream DynamoDB failure, or unhandled exception. |

---

## AI Tool Capabilities

The assistant has access to **50+ read-only tools** covering the following data domains. It selects tools automatically based on the user's question.

### Employee & User

| Tool | Description |
|---|---|
| `get_employee_information` | Full employee profile by username |
| `get_all_employees` | All employees in the organisation |
| `find_employee_by_email` | Look up employee by email |
| `get_all_employee_groups` | Cognito group memberships |

### Team

| Tool | Description |
|---|---|
| `get_team_information` | Team metadata (name, description, status) |
| `get_all_org_teams` | All teams in an organisation |
| `get_team_members` | Full member list for a team |
| `get_team_member_directory` | Simplified directory (userName, displayName, initials, role) |
| `get_user_teams` | Teams a user belongs to |
| `is_team_admin` | Whether a user is a team admin |

### Organisation

| Tool | Description |
|---|---|
| `get_org_info` | Organisation details and settings |
| `get_org_admins` | Admin user list |
| `get_org_users` | All org members |
| `is_org_admin` | Whether a user is an org admin |

### Performance Cycles, Quarters, KPIs, OKRs

| Tool | Description |
|---|---|
| `get_performance_cycles` | List cycles with optional filters |
| `get_performance_cycle_details` | Single cycle detail |
| `get_cycle_analytics` | Aggregated cycle analytics |
| `get_all_quarters` | Quarters within a cycle |
| `get_quarter_details` | Single quarter detail |
| `get_quarter_analytics` | Quarter analytics |
| `get_quarter_meeting_notes` | Meeting notes for a quarter |
| `get_all_kpis` | Org-level KPIs |
| `get_kpi_detail` | Single KPI detail |
| `get_all_okrs` | Org-level OKRs |
| `get_okr_detail` | Single OKR detail |

### Org Goals

| Tool | Description |
|---|---|
| `get_org_goal_detail` | Full org goal, optionally with history/teams/sub-items |
| `get_team_org_goals` | Goals assigned to a team |
| `get_org_goal_sub_items` | Sub-milestones / key results for a goal |
| `get_user_goals_for_org_goal` | User-level goals linked to an org goal + status summary |
| `get_goal_ladder_up` | Ladder-up requests for a goal |
| `get_goal_value_history` | Value-update history with optional date range |
| `get_goal_tasks` | Tasks linked to an org goal |
| `get_goal_tagged_teams` | Teams tagged to an org goal |

### User Goals, Tasks, Meetings & Appreciations

| Tool | Description |
|---|---|
| `get_my_goals` | Goals for a user/team combination |
| `get_my_goal` | Single goal detail |
| `get_goal_linked_tasks` | Tasks linked to a user goal |
| `get_goal_comments` | Comments on a user goal |
| `get_all_tasks` | Task list with filters |
| `get_task` | Single task detail |
| `get_my_meetings` | 1-on-1 meeting records |
| `get_meeting` | Single meeting detail |
| `get_my_appreciations` | Appreciation records |
| `get_my_feedback_requests` | Feedback requests |

### Manager / Admin — Team Member Performance

| Tool | Description |
|---|---|
| `get_team_performance_members` | Full member list with performance summaries |
| `get_member_goals` | Goals for a specific member |
| `get_member_tasks` | Tasks for a specific member |
| `get_member_meetings` | Meetings for a specific member |
| `get_member_appreciations` | Appreciations for a specific member |
| `get_member_manager_comments` | Manager review comments for a member |
| `get_member_performance_summary` | Comprehensive performance summary |

---

## Chat History Schema

| Attribute | Key | Type | Description |
|---|---|---|---|
| `chatId` | Partition key | `String` | UUID identifying the conversation session |
| `msgKey` | Sort key | `String` | `{epoch_ms_padded}#{uuid}` — enables chronological ordering |
| `role` | — | `String` | `"user"` or `"assistant"` |
| `messageText` | — | `String` | The message content |
| `userId` | — | `String` | Cognito sub of the user who initiated the turn |
| `createdAt` | — | `String` | ISO 8601 UTC timestamp |
| `expiresAt` | TTL | `Number` | Unix epoch seconds; record auto-deleted after 6 months |

---

## Frontend Integration Notes

1. **First message**: Do not include `chatId`. The response will return a newly generated `chatId`.
2. **Subsequent messages**: Always include the `chatId` from the previous response to maintain context.
3. **New conversation**: Generate a new UUID client-side (or drop `chatId`) to start a fresh session at any time.
4. **`toolsUsed`**: This field is informational. You may choose to display it, log it, or ignore it.
5. **Timeout**: The Lambda timeout is 300 seconds. The AI model response plus up to 5 tool-use iterations should typically complete in 5–30 seconds depending on data volume.

---

## Example conversations

### Example 1 — Goal progress enquiry

**Request**
```json
{
  "message": "How are my Q3 goals going?",
  "context": { "teamId": "team-abc123" }
}
```

**Response**
```json
{
  "chatId": "new-uuid-here",
  "response": "You have 4 active Q3 goals. 2 are on-track (Progress Update System at 72%, Code Review Process at 55%), 1 is ahead (Documentation Initiative at 91%), and 1 is at-risk (API Performance at 28% — target is 50% by end of month).",
  "toolsUsed": ["get_my_goals"]
}
```

### Example 2 — Manager team overview

**Request**
```json
{
  "chatId": "existing-session-uuid",
  "message": "Who on my team is behind on their goals?",
  "context": { "teamId": "team-abc123" }
}
```

**Response**
```json
{
  "chatId": "existing-session-uuid",
  "response": "Two members are behind on their goals this quarter: jane.smith (2 goals behind — sales pipeline at 15%, customer success at 22%) and mark.jones (1 goal behind — product adoption at 18%, target 40%).",
  "toolsUsed": ["get_team_performance_members", "get_member_goals"]
}
```
