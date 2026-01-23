# Manage Team Operations API

This Lambda function handles team operations including retrieving team details, managing team members, and updating team status.

## Authentication
All endpoints require authentication via AWS Cognito. The Cognito ID is extracted from the authorizer context.

## Available APIs

### 1. Get Team Details
Retrieves metadata and details for a specific team.

- **Method:** `GET`
- **Path:** `/teams/{teamId}`
- **Path Parameters:**
  - `teamId` (string, required): The unique identifier of the team
- **Response:**
  - `200 OK`: Returns team metadata
  - `404 Not Found`: Team not found
  - `401 Unauthorized`: User not authenticated

**Example Response:**
```json
{
  "teamId": "team-123",
  "teamName": "Engineering",
  "status": "ACTIVE",
  "createdAt": "2026-01-01T00:00:00Z",
  ...
}
```

---

### 2. Get Team Members
Retrieves all members of a specific team.

- **Method:** `GET`
- **Path:** `/teams/{teamId}/members`
- **Path Parameters:**
  - `teamId` (string, required): The unique identifier of the team
- **Response:**
  - `200 OK`: Returns list of team members with count
  - `500 Internal Server Error`: Failed to retrieve team members
  - `401 Unauthorized`: User not authenticated

**Example Response:**
```json
{
  "members": [
    {
      "userName": "john.doe",
      "role": "ADMIN",
      "joinedAt": "2026-01-01T00:00:00Z"
    },
    {
      "userName": "jane.smith",
      "role": "MEMBER",
      "joinedAt": "2026-01-05T00:00:00Z"
    }
  ],
  "count": 2
}
```

---

### 3. Update Team Status
Updates the status of a team (activate or deactivate).

- **Method:** `PATCH`
- **Path:** `/teams/{teamId}/status`
- **Path Parameters:**
  - `teamId` (string, required): The unique identifier of the team
- **Request Body:**
```json
{
  "status": "ACTIVE" // or "INACTIVE"
}
```
- **Authorization:** Only team admins can update team status
- **Response:**
  - `200 OK`: Team status updated successfully
  - `400 Bad Request`: Invalid status value
  - `403 Forbidden`: User is not an admin
  - `500 Internal Server Error`: Failed to update team status

**Example Response:**
```json
{
  "message": "Team status updated successfully",
  "teamId": "team-123",
  "status": "INACTIVE"
}
```

---

### 4. Add Team Members
Adds one or more members to a team.

- **Method:** `POST`
- **Path:** `/teams/{teamId}/members`
- **Path Parameters:**
  - `teamId` (string, required): The unique identifier of the team
- **Request Body:**
```json
{
  "userNames": ["john.doe", "jane.smith"]
}
```
- **Authorization:** Only team admins can add members
- **Constraints:** Cannot add members to an inactive team
- **Response:**
  - `200 OK`: Members added successfully
  - `400 Bad Request`: Invalid input or inactive team
  - `403 Forbidden`: User is not an admin
  - `500 Internal Server Error`: Failed to add team members

**Example Response:**
```json
{
  "message": "Successfully added 2 members to team",
  "teamId": "team-123",
  "count": 2
}
```

---

### 5. Update Member Role
Updates the role of a team member (promote to admin or demote to member).

- **Method:** `POST`
- **Path:** `/teams/{teamId}/members/{username}/role`
- **Path Parameters:**
  - `teamId` (string, required): The unique identifier of the team
- **Request Body:**
```json
{
  "userName": "john.doe",
  "role": "ADMIN" // or "MEMBER"
}
```
- **Authorization:** Only team admins can update member roles
- **Constraints:** Cannot demote the last admin of a team
- **Response:**
  - `200 OK`: Member role updated successfully
  - `400 Bad Request`: Invalid role or attempting to demote last admin
  - `403 Forbidden`: User is not an admin
  - `500 Internal Server Error`: Failed to update member role

**Example Response:**
```json
{
  "message": "Member role updated successfully",
  "teamId": "team-123",
  "userName": "john.doe",
  "role": "ADMIN"
}
```

---

## Error Responses

All error responses follow this format:
```json
{
  "error": "Brief error message",
  "message": "Detailed error message with context"
}
```

Common HTTP status codes:
- `400 Bad Request`: Invalid input or request
- `401 Unauthorized`: Authentication failed
- `403 Forbidden`: User lacks required permissions
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server-side error

---

## Environment Variables

The Lambda function requires the following environment variables:
- `EMPLOYEE_TABLE`: DynamoDB table name for employee data
- `EMPLOYEE_TABLE_COGNITO_ID_INDEX`: GSI name for Cognito ID lookups
- `TEAMS_TABLE`: DynamoDB table name for teams data

---

## Dependencies

- AWS Lambda Go SDK
- AWS SDK for Go v2 (DynamoDB, SES)
- AWS X-Ray SDK for Go
- Company Library (`company-lib`) for teams and employee services
