# Team Feeds API Documentation

All endpoints require a valid Cognito JWT token in the `Authorization` header.  
All responses follow the standard envelope:

```json
{
  "data":  { ... } | null,
  "meta":  { "total": 0, "page": 1, "limit": 20 } | null,
  "error": { "code": "ERROR_CODE", "message": "Human-readable message" } | null
}
```

---

## Table of Contents

1. [Common Headers](#common-headers)
2. [Post Types Reference](#post-types-reference)
3. [Feed & Posts](#1-feed--posts)
   - [GET /v2/teams/{teamId}/feed](#11-list-team-feed)
   - [POST /v2/teams/{teamId}/posts](#12-create-post)
   - [GET /v2/teams/{teamId}/posts/{postId}](#13-get-post)
   - [PUT /v2/teams/{teamId}/posts/{postId}](#14-update-post)
   - [DELETE /v2/teams/{teamId}/posts/{postId}](#15-delete-post)
4. [Likes](#2-likes)
   - [GET /v2/posts/{postId}/likes](#21-get-post-likes)
   - [POST /v2/posts/{postId}/likes](#22-like-a-post)
   - [DELETE /v2/posts/{postId}/likes](#23-unlike-a-post)
   - [POST /v2/posts/{postId}/comments/{commentId}/likes](#24-like-a-comment)
   - [DELETE /v2/posts/{postId}/comments/{commentId}/likes](#25-unlike-a-comment)
5. [Comments](#3-comments)
   - [GET /v2/posts/{postId}/comments](#31-list-comments)
   - [POST /v2/posts/{postId}/comments](#32-add-comment)
   - [PUT /v2/posts/{postId}/comments/{commentId}](#33-edit-comment)
   - [DELETE /v2/posts/{postId}/comments/{commentId}](#34-delete-comment)
6. [Poll Votes](#4-poll-votes)
   - [POST /v2/posts/{postId}/poll/vote](#41-cast-vote)
   - [DELETE /v2/posts/{postId}/poll/vote](#42-retract-vote)
   - [GET /v2/posts/{postId}/poll/results](#43-get-poll-results)
7. [Checklist Items](#5-checklist-items)
   - [POST /v2/posts/{postId}/checklist/items](#51-add-checklist-item)
   - [PATCH /v2/posts/{postId}/checklist/items/{itemId}](#52-toggle-checklist-item)
   - [DELETE /v2/posts/{postId}/checklist/items/{itemId}](#53-delete-checklist-item)
8. [Task Updates](#6-task-updates)
   - [PATCH /v2/posts/{postId}/task/status](#61-update-task-status)
   - [PATCH /v2/posts/{postId}/task/time](#62-log-task-time)
9. [Error Codes Reference](#error-codes-reference)

---

## Common Headers

| Header          | Required | Description                                |
|-----------------|----------|--------------------------------------------|
| `Authorization` | Yes      | `Bearer <cognito_jwt_token>`               |
| `Content-Type`  | Yes*     | `application/json` (*for POST/PUT/PATCH)   |

---

## Post Types Reference

| Type        | Description                        | Extra Fields Required                              |
|-------------|------------------------------------|----------------------------------------------------|
| `update`    | General team update / announcement | `content`                                          |
| `kudos`     | Recognize a team member            | `content`, `recipientUserId`                       |
| `task`      | Create a trackable task            | `taskSummary`, `assigneeUserId`, `dueDate`, `urgency` |
| `poll`      | Create a vote / survey             | `question`, `options[]`                            |
| `checklist` | Create a to-do checklist           | `title`, `items[]`                                 |
| `event`     | Announce a team event              | `content`, `eventDate`, `eventTime`                |

---

## 1. Feed & Posts

### 1.1 List Team Feed

Retrieve a paginated, reverse-chronological list of posts for a team.  
Caller must be a member of the team.

```
GET /v2/teams/{teamId}/feed
```

**Lambda:** `ManageFeedPostsLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `teamId` | string | Yes      | Team ID     |

**Query Parameters**

| Param  | Type    | Default | Description                                     |
|--------|---------|---------|-------------------------------------------------|
| `page` | integer | `1`     | Page number (1-based)                           |
| `limit`| integer | `20`    | Items per page (max recommended: 50)            |
| `type` | string  | —       | Filter by post type: `update \| kudos \| task \| poll \| checklist \| event` |

**Success Response — 200**

```json
{
  "data": [
    {
      "postId": "abc123",
      "teamId": "team-001",
      "type": "update",
      "content": "Great sprint everyone!",
      "tags": [
        { "type": "goal", "refId": "goal-99", "name": "Q1 Revenue" }
      ],
      "author": {
        "userId": "user@example.com",
        "name": "Jane Smith",
        "role": "ADMIN",
        "profilePic": null
      },
      "likeCount": 4,
      "commentCount": 2,
      "createdAt": "2026-02-26T08:00:00Z",
      "updatedAt": "2026-02-26T08:00:00Z"
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20 },
  "error": null
}
```

**Error Responses**

| Status | Code        | When                        |
|--------|-------------|-----------------------------|
| 403    | `FORBIDDEN` | Caller not a team member    |
| 500    | `INTERNAL_ERROR` | DynamoDB failure       |

---

### 1.2 Create Post

Create a new post in the team feed. The caller becomes the post author.

```
POST /v2/teams/{teamId}/posts
```

**Lambda:** `ManageFeedPostsLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `teamId` | string | Yes      | Team ID     |

**Request Body**

```json
{
  "type": "update",
  "content": "Text content of the post",
  "tags": [
    { "type": "goal", "refId": "goal-99", "name": "Q1 Revenue" }
  ]
}
```

**Fields by Post Type**

<details>
<summary><strong>update</strong></summary>

```json
{
  "type": "update",
  "content": "Sprint retrospective summary...",
  "tags": []
}
```

| Field     | Type     | Required | Description              |
|-----------|----------|----------|--------------------------|
| `type`    | string   | Yes      | `"update"`               |
| `content` | string   | Yes      | Post body text           |
| `tags`    | array    | No       | Array of tag objects     |

</details>

<details>
<summary><strong>kudos</strong></summary>

```json
{
  "type": "kudos",
  "content": "Amazing work this week!",
  "recipientUserId": "jane@company.com"
}
```

| Field             | Type   | Required | Description                      |
|-------------------|--------|----------|----------------------------------|
| `type`            | string | Yes      | `"kudos"`                        |
| `content`         | string | Yes      | Recognition message              |
| `recipientUserId` | string | Yes      | Username of the person being recognised |

</details>

<details>
<summary><strong>task</strong></summary>

```json
{
  "type": "task",
  "taskSummary": "Update API docs",
  "taskDescription": "Full description of the task...",
  "assigneeUserId": "dev@company.com",
  "dueDate": "2026-03-15",
  "urgency": "High"
}
```

| Field             | Type   | Required | Description                           |
|-------------------|--------|----------|---------------------------------------|
| `type`            | string | Yes      | `"task"`                              |
| `taskSummary`     | string | Yes      | Short task title                      |
| `taskDescription` | string | No       | Detailed description                  |
| `assigneeUserId`  | string | Yes      | Username of the assignee              |
| `dueDate`         | string | Yes      | Due date (`YYYY-MM-DD`)               |
| `urgency`         | string | Yes      | `Low \| Medium \| High`               |

</details>

<details>
<summary><strong>poll</strong></summary>

```json
{
  "type": "poll",
  "question": "Which framework should we adopt?",
  "options": [
    { "optionId": "opt-1", "text": "React" },
    { "optionId": "opt-2", "text": "Vue" },
    { "optionId": "opt-3", "text": "Angular" }
  ]
}
```

| Field      | Type   | Required | Description                      |
|------------|--------|----------|----------------------------------|
| `type`     | string | Yes      | `"poll"`                         |
| `question` | string | Yes      | The poll question                |
| `options`  | array  | Yes      | Min 2 options, each with `optionId` and `text` |

</details>

<details>
<summary><strong>checklist</strong></summary>

```json
{
  "type": "checklist",
  "title": "Daily Stand-up Checklist",
  "items": [
    { "itemId": "item-1", "text": "Review open PRs", "completed": false },
    { "itemId": "item-2", "text": "Update Jira tickets", "completed": false }
  ],
  "isRecurring": true,
  "recurringFrequency": "Daily"
}
```

| Field                | Type    | Required | Description                                        |
|----------------------|---------|----------|----------------------------------------------------|
| `type`               | string  | Yes      | `"checklist"`                                      |
| `title`              | string  | Yes      | Checklist title                                    |
| `items`              | array   | No       | Initial items (each with `itemId`, `text`, `completed`) |
| `isRecurring`        | boolean | No       | Whether checklist repeats                          |
| `recurringFrequency` | string  | No       | `Daily \| Weekly \| Bi-Weekly \| Monthly`          |

</details>

<details>
<summary><strong>event</strong></summary>

```json
{
  "type": "event",
  "content": "Team lunch — everyone is welcome!",
  "eventDate": "2026-03-01",
  "eventTime": "12:30",
  "location": "Level 3 Boardroom"
}
```

| Field       | Type   | Required | Description                    |
|-------------|--------|----------|--------------------------------|
| `type`      | string | Yes      | `"event"`                      |
| `content`   | string | Yes      | Event description              |
| `eventDate` | string | Yes      | Date (`YYYY-MM-DD`)            |
| `eventTime` | string | Yes      | Time (`HH:MM` 24-hr)           |
| `location`  | string | No       | Location or meeting link       |

</details>

**Success Response — 201**

```json
{
  "data": {
    "postId": "abc123",
    "teamId": "team-001",
    "type": "update",
    "content": "Text content",
    "author": { "userId": "user@example.com", "name": "Jane Smith", "profilePic": null },
    "likeCount": 0,
    "commentCount": 0,
    "createdAt": "2026-02-26T08:00:00Z",
    "updatedAt": "2026-02-26T08:00:00Z"
  },
  "meta": null,
  "error": null
}
```

**Error Responses**

| Status | Code              | When                              |
|--------|-------------------|-----------------------------------|
| 400    | `BAD_REQUEST`     | Invalid/missing required fields   |
| 403    | `FORBIDDEN`       | Caller not a team member          |
| 500    | `INTERNAL_ERROR`  | DynamoDB or dependency failure    |

---

### 1.3 Get Post

Retrieve a single post by ID.

```
GET /v2/teams/{teamId}/posts/{postId}
```

**Lambda:** `ManageFeedPostsLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `teamId` | string | Yes      | Team ID     |
| `postId` | string | Yes      | Post ID     |

**Success Response — 200**

Same shape as a single item in the [List Team Feed](#11-list-team-feed) response.  
Task posts include: `taskSummary`, `taskDescription`, `assigneeUserId`, `assigneeName`, `dueDate`, `urgency`, `taskStatus`, `timeSpentHours`.  
Poll posts include: `pollQuestion`, `pollOptions[]` (with per-option vote counts omitted at fetch; use [Get Poll Results](#43-get-poll-results) for live counts).  
Checklist posts include: `checklistTitle`, `isRecurring`, `recurringFrequency`, `checklistItems[]`.  
Event posts include: `eventTitle`, `eventDate`, `eventTime`, `location`.

**Error Responses**

| Status | Code        | When                     |
|--------|-------------|--------------------------|
| 403    | `FORBIDDEN` | Caller not a team member |
| 404    | `NOT_FOUND` | Post does not exist      |

---

### 1.4 Update Post

Edit a post's content or type-specific fields. Only the post author or a team admin can update.

```
PUT /v2/teams/{teamId}/posts/{postId}
```

**Lambda:** `ManageFeedPostsLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `teamId` | string | Yes      | Team ID     |
| `postId` | string | Yes      | Post ID     |

**Request Body**  
Same fields as [Create Post](#12-create-post) — only provide fields you want to update. The `type` field cannot be changed after creation.

```json
{
  "content": "Updated post content"
}
```

**Success Response — 200**

Updated post object (same shape as Get Post).

**Error Responses**

| Status | Code              | When                                      |
|--------|-------------------|-------------------------------------------|
| 403    | `FORBIDDEN`       | Caller is neither author nor team admin   |
| 404    | `NOT_FOUND`       | Post does not exist                       |
| 500    | `INTERNAL_ERROR`  | DynamoDB failure                          |

---

### 1.5 Delete Post

Delete a post and all its associated records (likes, comments, checklist items, poll votes). Only the post author or a team admin can delete.

```
DELETE /v2/teams/{teamId}/posts/{postId}
```

**Lambda:** `ManageFeedPostsLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `teamId` | string | Yes      | Team ID     |
| `postId` | string | Yes      | Post ID     |

**Request Body:** None

**Success Response — 204**  
Empty body.

**Error Responses**

| Status | Code        | When                                    |
|--------|-------------|-----------------------------------------|
| 403    | `FORBIDDEN` | Caller is neither author nor team admin |
| 404    | `NOT_FOUND` | Post does not exist                     |

---

## 2. Likes

### 2.1 Get Post Likes

Returns a count and the list of users who liked a post.

```
GET /v2/posts/{postId}/likes
```

**Lambda:** `ManagePostLikesLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID     |

**Success Response — 200**

```json
{
  "data": {
    "postId": "abc123",
    "likeCount": 3,
    "likedByCurrentUser": true,
    "likedBy": ["user1@co.com", "user2@co.com", "user3@co.com"]
  },
  "meta": null,
  "error": null
}
```

---

### 2.2 Like a Post

Toggle a like on a post (idempotent — liking an already-liked post is a no-op).

```
POST /v2/posts/{postId}/likes
```

**Lambda:** `ManagePostLikesLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID     |

**Request Body:** None

**Success Response — 200**

```json
{
  "data": { "postId": "abc123", "liked": true, "likeCount": 5 },
  "meta": null,
  "error": null
}
```

---

### 2.3 Unlike a Post

Remove the caller's like from a post.

```
DELETE /v2/posts/{postId}/likes
```

**Lambda:** `ManagePostLikesLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID     |

**Request Body:** None

**Success Response — 200**

```json
{
  "data": { "postId": "abc123", "liked": false, "likeCount": 4 },
  "meta": null,
  "error": null
}
```

---

### 2.4 Like a Comment

```
POST /v2/posts/{postId}/comments/{commentId}/likes
```

**Lambda:** `ManagePostLikesLambda`

**Path Parameters**

| Param       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| `postId`    | string | Yes      | Post ID     |
| `commentId` | string | Yes      | Comment ID  |

**Request Body:** None

**Success Response — 200**

```json
{
  "data": { "commentId": "cmt-001", "liked": true, "likeCount": 2 },
  "meta": null,
  "error": null
}
```

---

### 2.5 Unlike a Comment

```
DELETE /v2/posts/{postId}/comments/{commentId}/likes
```

**Lambda:** `ManagePostLikesLambda`

**Path Parameters**

| Param       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| `postId`    | string | Yes      | Post ID     |
| `commentId` | string | Yes      | Comment ID  |

**Request Body:** None

**Success Response — 200**

```json
{
  "data": { "commentId": "cmt-001", "liked": false, "likeCount": 1 },
  "meta": null,
  "error": null
}
```

---

## 3. Comments

### 3.1 List Comments

Retrieve all comments on a post, ordered by creation time (oldest first).

```
GET /v2/posts/{postId}/comments
```

**Lambda:** `ManagePostCommentsLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID     |

**Query Parameters**

| Param  | Type    | Default | Description       |
|--------|---------|---------|-------------------|
| `page` | integer | `1`     | Page number       |
| `limit`| integer | `20`    | Items per page    |

**Success Response — 200**

```json
{
  "data": [
    {
      "commentId": "cmt-001",
      "postId": "abc123",
      "content": "Great update!",
      "author": {
        "userId": "user@example.com",
        "name": "Jane Smith",
        "profilePic": null
      },
      "parentCommentId": null,
      "likeCount": 1,
      "createdAt": "2026-02-26T09:00:00Z",
      "updatedAt": "2026-02-26T09:00:00Z"
    }
  ],
  "meta": { "total": 1, "page": 1, "limit": 20 },
  "error": null
}
```

---

### 3.2 Add Comment

Add a comment (or reply) to a post.

```
POST /v2/posts/{postId}/comments
```

**Lambda:** `ManagePostCommentsLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID     |

**Request Body**

```json
{
  "content": "Your comment text here",
  "parentCommentId": "cmt-001"
}
```

| Field             | Type   | Required | Description                                    |
|-------------------|--------|----------|------------------------------------------------|
| `content`         | string | Yes      | Comment text                                   |
| `parentCommentId` | string | No       | ID of parent comment (for threaded replies)    |

**Success Response — 201**

```json
{
  "data": {
    "commentId": "cmt-002",
    "postId": "abc123",
    "content": "Your comment text here",
    "author": { "userId": "user@example.com", "name": "Jane Smith", "profilePic": null },
    "parentCommentId": "cmt-001",
    "likeCount": 0,
    "createdAt": "2026-02-26T10:00:00Z",
    "updatedAt": "2026-02-26T10:00:00Z"
  },
  "meta": null,
  "error": null
}
```

**Error Responses**

| Status | Code          | When                   |
|--------|---------------|------------------------|
| 400    | `BAD_REQUEST` | `content` is empty     |
| 404    | `NOT_FOUND`   | Post does not exist    |

---

### 3.3 Edit Comment

Update a comment's text. Only the comment author can edit.

```
PUT /v2/posts/{postId}/comments/{commentId}
```

**Lambda:** `ManagePostCommentsLambda`

**Path Parameters**

| Param       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| `postId`    | string | Yes      | Post ID     |
| `commentId` | string | Yes      | Comment ID  |

**Request Body**

```json
{
  "content": "Updated comment text"
}
```

| Field     | Type   | Required | Description          |
|-----------|--------|----------|----------------------|
| `content` | string | Yes      | New comment content  |

**Success Response — 200**

Updated comment object (same shape as Add Comment response data).

**Error Responses**

| Status | Code        | When                                  |
|--------|-------------|---------------------------------------|
| 403    | `FORBIDDEN` | Caller is not the comment author      |
| 404    | `NOT_FOUND` | Comment or post does not exist        |

---

### 3.4 Delete Comment

Delete a comment. Only the comment author or a team admin can delete.

```
DELETE /v2/posts/{postId}/comments/{commentId}
```

**Lambda:** `ManagePostCommentsLambda`

**Path Parameters**

| Param       | Type   | Required | Description |
|-------------|--------|----------|-------------|
| `postId`    | string | Yes      | Post ID     |
| `commentId` | string | Yes      | Comment ID  |

**Request Body:** None

**Success Response — 204**  
Empty body.

**Error Responses**

| Status | Code        | When                                      |
|--------|-------------|-------------------------------------------|
| 403    | `FORBIDDEN` | Caller is neither author nor team admin   |
| 404    | `NOT_FOUND` | Comment or post does not exist            |

---

## 4. Poll Votes

Applies only to posts of `type: "poll"`.

### 4.1 Cast Vote

Vote for an option in a poll. A user can only vote for one option at a time — casting a new vote replaces the previous one.

```
POST /v2/posts/{postId}/poll/vote
```

**Lambda:** `ManagePollVotesLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID (must be a poll-type post) |

**Request Body**

```json
{
  "optionId": "opt-2"
}
```

| Field      | Type   | Required | Description                                         |
|------------|--------|----------|-----------------------------------------------------|
| `optionId` | string | Yes      | Must match one of the poll's configured option IDs  |

**Success Response — 200**

```json
{
  "data": {
    "postId": "abc123",
    "votedOptionId": "opt-2",
    "votedAt": "2026-02-26T11:00:00Z"
  },
  "meta": null,
  "error": null
}
```

**Error Responses**

| Status | Code          | When                          |
|--------|---------------|-------------------------------|
| 400    | `BAD_REQUEST` | `optionId` missing or invalid |
| 404    | `NOT_FOUND`   | Post does not exist           |

---

### 4.2 Retract Vote

Remove the caller's vote from a poll.

```
DELETE /v2/posts/{postId}/poll/vote
```

**Lambda:** `ManagePollVotesLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID     |

**Request Body:** None

**Success Response — 200**

```json
{
  "data": { "postId": "abc123", "retracted": true },
  "meta": null,
  "error": null
}
```

---

### 4.3 Get Poll Results

Retrieve live vote counts per option and the caller's current vote.

```
GET /v2/posts/{postId}/poll/results
```

**Lambda:** `ManagePollVotesLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID     |

**Success Response — 200**

```json
{
  "data": {
    "postId": "abc123",
    "question": "Which framework should we adopt?",
    "totalVotes": 8,
    "userVotedOptionId": "opt-1",
    "options": [
      { "optionId": "opt-1", "text": "React",   "votes": 5 },
      { "optionId": "opt-2", "text": "Vue",     "votes": 2 },
      { "optionId": "opt-3", "text": "Angular", "votes": 1 }
    ]
  },
  "meta": null,
  "error": null
}
```

| Field               | Type    | Description                                               |
|---------------------|---------|-----------------------------------------------------------|
| `totalVotes`        | integer | Sum of votes across all options                           |
| `userVotedOptionId` | string  | Which option the caller voted for (`null` if not voted)   |
| `options[].votes`   | integer | Vote count for that option                                |

---

## 5. Checklist Items

Applies only to posts of `type: "checklist"`.

### 5.1 Add Checklist Item

Append a new item to an existing checklist post.

```
POST /v2/posts/{postId}/checklist/items
```

**Lambda:** `ManageChecklistItemsLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID (must be a checklist-type post) |

**Request Body**

```json
{
  "text": "Review pull requests"
}
```

| Field  | Type   | Required | Description        |
|--------|--------|----------|--------------------|
| `text` | string | Yes      | Item text content  |

**Success Response — 201**

```json
{
  "data": {
    "itemId": "item-uuid",
    "postId": "abc123",
    "text": "Review pull requests",
    "completed": false,
    "createdAt": "2026-02-26T12:00:00Z"
  },
  "meta": null,
  "error": null
}
```

**Error Responses**

| Status | Code          | When                        |
|--------|---------------|-----------------------------|
| 400    | `BAD_REQUEST` | `text` is empty             |
| 403    | `FORBIDDEN`   | Caller not a team member    |
| 404    | `NOT_FOUND`   | Post does not exist         |

---

### 5.2 Toggle Checklist Item

Mark an item as complete or incomplete.

```
PATCH /v2/posts/{postId}/checklist/items/{itemId}
```

**Lambda:** `ManageChecklistItemsLambda`

**Path Parameters**

| Param    | Type   | Required | Description     |
|----------|--------|----------|-----------------|
| `postId` | string | Yes      | Post ID         |
| `itemId` | string | Yes      | Checklist item ID |

**Request Body**

```json
{
  "completed": true
}
```

| Field       | Type    | Required | Description                            |
|-------------|---------|----------|----------------------------------------|
| `completed` | boolean | Yes      | `true` to mark done, `false` to unmark |

**Success Response — 200**

```json
{
  "data": {
    "itemId": "item-uuid",
    "completed": true,
    "updatedAt": "2026-02-26T12:30:00Z"
  },
  "meta": null,
  "error": null
}
```

**Error Responses**

| Status | Code        | When                     |
|--------|-------------|--------------------------|
| 403    | `FORBIDDEN` | Caller not a team member |
| 404    | `NOT_FOUND` | Item does not exist      |

---

### 5.3 Delete Checklist Item

Remove an item from a checklist post. Only the post author or a team admin can delete.

```
DELETE /v2/posts/{postId}/checklist/items/{itemId}
```

**Lambda:** `ManageChecklistItemsLambda`

**Path Parameters**

| Param    | Type   | Required | Description      |
|----------|--------|----------|------------------|
| `postId` | string | Yes      | Post ID          |
| `itemId` | string | Yes      | Checklist item ID |

**Request Body:** None

**Success Response — 204**  
Empty body.

**Error Responses**

| Status | Code        | When                                      |
|--------|-------------|-------------------------------------------|
| 403    | `FORBIDDEN` | Caller is neither author nor team admin   |
| 404    | `NOT_FOUND` | Item does not exist                       |

---

## 6. Task Updates

Applies only to posts of `type: "task"`.

### 6.1 Update Task Status

Change the status of a task. Only the assignee, post author, or a team admin can update.

```
PATCH /v2/posts/{postId}/task/status
```

**Lambda:** `ManageTaskUpdatesLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID (must be a task-type post) |

**Request Body**

```json
{
  "status": "in-progress"
}
```

| Field    | Type   | Required | Description                           |
|----------|--------|----------|---------------------------------------|
| `status` | string | Yes      | `todo \| in-progress \| done`         |

**Success Response — 200**

```json
{
  "data": {
    "postId": "abc123",
    "status": "in-progress",
    "updatedAt": "2026-02-26T14:00:00Z"
  },
  "meta": null,
  "error": null
}
```

**Error Responses**

| Status | Code          | When                                        |
|--------|---------------|---------------------------------------------|
| 400    | `BAD_REQUEST` | Invalid status value                        |
| 403    | `FORBIDDEN`   | Caller is not assignee, author, or admin    |
| 404    | `NOT_FOUND`   | Post does not exist                         |

---

### 6.2 Log Task Time

Add hours worked to a task's time-spent counter (cumulative).

```
PATCH /v2/posts/{postId}/task/time
```

**Lambda:** `ManageTaskUpdatesLambda`

**Path Parameters**

| Param    | Type   | Required | Description |
|----------|--------|----------|-------------|
| `postId` | string | Yes      | Post ID (must be a task-type post) |

**Request Body**

```json
{
  "hours": 1.5
}
```

| Field   | Type    | Required | Description                                |
|---------|---------|----------|--------------------------------------------|
| `hours` | number  | Yes      | Hours to add (positive decimal, e.g. `1.5`) |

**Success Response — 200**

```json
{
  "data": {
    "postId": "abc123",
    "timeSpentHours": 3.5,
    "updatedAt": "2026-02-26T15:00:00Z"
  },
  "meta": null,
  "error": null
}
```

**Error Responses**

| Status | Code          | When                              |
|--------|---------------|-----------------------------------|
| 400    | `BAD_REQUEST` | `hours` is missing or <= 0        |
| 403    | `FORBIDDEN`   | Caller not a team member          |
| 404    | `NOT_FOUND`   | Post does not exist               |

---

## Error Codes Reference

| HTTP Status | Code                | Description                                              |
|-------------|---------------------|----------------------------------------------------------|
| 400         | `BAD_REQUEST`       | Missing or invalid request body field                    |
| 401         | `UNAUTHORIZED`      | Missing, expired, or invalid Cognito JWT                 |
| 403         | `FORBIDDEN`         | Caller lacks permission (not a member / not author/admin)|
| 404         | `NOT_FOUND`         | Requested resource does not exist                        |
| 405         | `METHOD_NOT_ALLOWED`| HTTP method is not supported for this route              |
| 500         | `INTERNAL_ERROR`    | Unexpected server or DynamoDB failure                    |

---

## DynamoDB Table — TeamFeedTable

Single-table design with one Global Secondary Index.

| Record Type     | PK                          | SK                              | GSI1PK              | GSI1SK                   |
|-----------------|-----------------------------|---------------------------------|---------------------|--------------------------|
| Post metadata   | `POST#{postId}`             | `#METADATA`                     | `TEAM#{teamId}`     | `{createdAt}#{postId}`   |
| Post like       | `POST#{postId}`             | `LIKE#{userId}`                 | —                   | —                        |
| Comment         | `POST#{postId}`             | `CMMNT#{createdAt}#{commentId}` | `COMMENT#{commentId}` | `META`                 |
| Comment like    | `COMMENT#{commentId}`       | `LIKE#{userId}`                 | —                   | —                        |
| Poll vote       | `POST#{postId}`             | `VOTE#{userId}`                 | —                   | —                        |
| Checklist item  | `POST#{postId}`             | `ITEM#{itemId}`                 | —                   | —                        |

**GSI1** (GSI1PK + GSI1SK) is used exclusively to list team feed posts in reverse-chronological order.
