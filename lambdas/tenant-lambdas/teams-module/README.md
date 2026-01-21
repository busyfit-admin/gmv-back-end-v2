# Teams Management Module

This module provides comprehensive team management functionality for the tenant portal, allowing users to create teams, manage memberships, and control team access.

## Architecture

### DynamoDB Table Structure

The `TenantTeamsTable` uses a single-table design with the following access patterns:

**Primary Key:**
- `PK`: Partition key (e.g., `TEAM#uuid`)
- `SK`: Sort key (e.g., `METADATA` or `USER#username`)

**GSI1:**
- `GSI1PK`: User partition key (`USER#username`)
- `GSI1SK`: Team sort key (`TEAM#uuid`)

**Item Types:**

1. **Team Metadata** (`SK = METADATA`):
   - TeamId, TeamName, TeamDesc
   - Status (ACTIVE/INACTIVE)
   - CreatedBy, CreatedAt, UpdatedAt
   - MemberCount

2. **Team Member** (`SK = USER#username`):
   - TeamId, UserName, DisplayName
   - Role (ADMIN/MEMBER)
   - JoinedAt, IsActive

### Lambda Functions

1. **list-user-teams**: Lists all teams for a user
2. **create-team**: Creates a new team (user becomes admin)
3. **manage-team-operations**: Handles team operations (deactivate, add users, assign admins)
4. **set-current-team**: Sets the user's current active team

### Library

**company-teams-v2.go**: Core team management logic with the following services:
- CreateTeam: Create team with admin
- GetUserTeams: List user's teams
- GetTeamMetadata: Get team details
- UpdateTeamStatus: Activate/deactivate team
- AddTeamMembers: Add users to team
- UpdateMemberRole: Change user role
- IsTeamAdmin: Check admin status

## API Endpoints

### 1. List User Teams

**Endpoint:** `GET /v1/teams?currentTeam={teamId}`

**Description:** Returns all teams the authenticated user is a member of.

**Query Parameters:**
- `currentTeam` (optional): Team ID the user is currently logged into

**Response:**
```json
{
  "teams": [
    {
      "teamId": "TEAM#uuid",
      "teamName": "Engineering Team",
      "teamDesc": "Product engineering team",
      "role": "ADMIN",
      "status": "ACTIVE",
      "memberCount": 15,
      "joinedAt": "2024-01-15T10:30:00Z",
      "isLoggedIn": true
    }
  ],
  "currentTeam": "TEAM#uuid",
  "count": 3
}
```

**Example:**
```bash
curl -X GET "https://api.example.com/v1/teams?currentTeam=TEAM#123" \
  -H "Authorization: Bearer <token>"
```

### 2. Create Team

**Endpoint:** `POST /v1/teams`

**Description:** Creates a new team with the requesting user as admin.

**Request Body:**
```json
{
  "teamName": "Marketing Team",
  "teamDesc": "Digital marketing and campaigns"
}
```

**Response:**
```json
{
  "message": "Team created successfully",
  "team": {
    "teamId": "TEAM#uuid",
    "teamName": "Marketing Team",
    "teamDesc": "Digital marketing and campaigns",
    "status": "ACTIVE",
    "createdBy": "john.doe",
    "createdAt": "2024-01-20T14:30:00Z",
    "memberCount": 1
  }
}
```

**Example:**
```bash
curl -X POST "https://api.example.com/v1/teams" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"teamName":"Marketing Team","teamDesc":"Digital marketing"}'
```

### 3. Get Team Details

**Endpoint:** `GET /v1/teams/{teamId}`

**Description:** Returns detailed information about a specific team.

**Response:**
```json
{
  "teamId": "TEAM#uuid",
  "teamName": "Engineering Team",
  "teamDesc": "Product engineering",
  "status": "ACTIVE",
  "createdBy": "admin",
  "createdAt": "2024-01-10T09:00:00Z",
  "updatedAt": "2024-01-20T15:00:00Z",
  "memberCount": 15
}
```

### 4. Update Team Status (Deactivate/Activate)

**Endpoint:** `PATCH /v1/teams/{teamId}/status`

**Description:** Activate or deactivate a team. Only admins can perform this action.

**Request Body:**
```json
{
  "status": "INACTIVE"
}
```

**Valid Status Values:**
- `ACTIVE`: Team is active
- `INACTIVE`: Team is deactivated

**Response:**
```json
{
  "message": "Team status updated successfully",
  "teamId": "TEAM#uuid",
  "status": "INACTIVE"
}
```

**Example:**
```bash
curl -X PATCH "https://api.example.com/v1/teams/TEAM#123/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status":"INACTIVE"}'
```

### 5. Get Team Members

**Endpoint:** `GET /v1/teams/{teamId}/members`

**Description:** Returns all members of a team with their roles.

**Response:**
```json
{
  "members": [
    {
      "teamId": "TEAM#uuid",
      "userName": "john.doe",
      "displayName": "John Doe",
      "role": "ADMIN",
      "joinedAt": "2024-01-10T09:00:00Z",
      "isActive": true
    },
    {
      "teamId": "TEAM#uuid",
      "userName": "jane.smith",
      "displayName": "Jane Smith",
      "role": "MEMBER",
      "joinedAt": "2024-01-15T10:30:00Z",
      "isActive": true
    }
  ],
  "count": 2
}
```

### 6. Add Team Members

**Endpoint:** `POST /v1/teams/{teamId}/members`

**Description:** Add new members to a team. Only admins can add members.

**Request Body:**
```json
{
  "userNames": ["jane.smith", "bob.jones"]
}
```

**Response:**
```json
{
  "message": "Successfully added 2 members to team",
  "teamId": "TEAM#uuid",
  "count": 2
}
```

**Example:**
```bash
curl -X POST "https://api.example.com/v1/teams/TEAM#123/members" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"userNames":["jane.smith","bob.jones"]}'
```

### 7. Update Member Role (Assign Admin)

**Endpoint:** `POST /v1/teams/{teamId}/members/{username}/role`

**Description:** Update a team member's role to ADMIN or MEMBER. Only admins can perform this action.

**Request Body:**
```json
{
  "userName": "jane.smith",
  "role": "ADMIN"
}
```

**Valid Roles:**
- `ADMIN`: Can manage team (add/remove members, change roles, deactivate team)
- `MEMBER`: Regular team member

**Response:**
```json
{
  "message": "Member role updated successfully",
  "teamId": "TEAM#uuid",
  "userName": "jane.smith",
  "role": "ADMIN"
}
```

**Example:**
```bash
curl -X POST "https://api.example.com/v1/teams/TEAM#123/members/jane.smith/role" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"userName":"jane.smith","role":"ADMIN"}'
```

### 8. Set Current Team

**Endpoint:** `PATCH /v1/current-team` or `PUT /v1/current-team`

**Description:** Sets the user's current active team. The user must be a member of the specified team.

**Request Body:**
```json
{
  "teamId": "TEAM#uuid"
}
```

**Response:**
```json
{
  "message": "Current team updated successfully",
  "currentTeamId": "TEAM#uuid"
}
```

**Example:**
```bash
curl -X PATCH "https://api.example.com/v1/current-team" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"teamId":"TEAM#123"}'
```

## Authorization

All endpoints require Cognito authentication. The username is extracted from:
1. Cognito authorizer claims (`cognito:username` or `custom:userName`)
2. `X-User-Name` header (for testing)

## Admin Permissions

The following operations require the requesting user to be a team admin:
- Deactivate/activate team
- Add members to team
- Update member roles

**Protection:** Cannot demote the last admin of a team.

## Error Responses

### 401 Unauthorized
```json
{
  "error": "Unauthorized",
  "message": "username not found in request"
}
```

### 403 Forbidden
```json
{
  "error": "Only admins can add members",
  "message": "user john.doe is not an admin of team TEAM#123"
}
```

```json
{
  "error": "User is not a member of this team",
  "message": "User is not a member of this team"
}
```

### 400 Bad Request
```json
{
  "error": "Cannot demote the last admin of the team",
  "message": "Cannot demote the last admin of the team"
}
```

### 404 Not Found
```json
{
  "error": "Team not found",
  "message": "team not found: TEAM#invalid"
}
```

## Deployment

### Build Lambdas

```bash
cd lambdas/tenant-lambdas/teams-module

# Build list-user-teams
cd list-user-teams && make build && cd ..

# Build create-team
cd create-team && make build && cd ..

# Build manage-team-operations
cd manage-team-operations && make build && cd ..

# Build set-current-team
cd set-current-team && make build && cd ..
```

### Deploy CloudFormation Stack

```bash
cd cfn/tenant-cfn
make deploy ENVIRONMENT=dev
```

## Testing

### Create a Team
```bash
curl -X POST "https://api.example.com/v1/teams" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "teamName": "Test Team",
    "teamDesc": "Team for testing"
  }'
```

### List Your Teams
```bash
curl -X GET "https://api.example.com/v1/teams" \
  -H "Authorization: Bearer <token>"
```

### Add Members
```bash
curl -X POST "https://api.example.com/v1/teams/TEAM#123/members" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "userNames": ["user1", "user2"]
  }'
```

### Promote to Admin
```bash
curl -X POST "https://api.example.com/v1/teams/TEAM#123/members/user1/role" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "userName": "user1",
    "role": "ADMIN"
  }'
```

### Deactivate Team
```bash
curl -X PATCH "https://api.example.com/v1/teams/TEAM#123/status" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "INACTIVE"
  }'
```

### Set Current Team
```bash
curl -X PATCH "https://api.example.com/v1/current-team" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "teamId": "TEAM#123"
  }'
```

## Database Queries

### Get all teams for a user
```
Query GSI1:
GSI1PK = "USER#john.doe"
```

### Get all members of a team
```
Query Primary:
PK = "TEAM#uuid"
SK begins_with "USER#"
```

### Get team metadata
```
GetItem:
PK = "TEAM#uuid"
SK = "METADATA"
```

## Best Practices

1. **Team Creation**: Every user who creates a team automatically becomes an admin
2. **Admin Protection**: System prevents demotion of the last admin
3. **Atomic Operations**: Uses DynamoDB transactions for consistency
4. **Soft Deletes**: Members marked as inactive rather than deleted
5. **GSI for User Queries**: Efficient lookup of all teams for a user
6. **Member Count**: Maintained automatically for quick retrieval

## Future Enhancements

- Team invitations with pending status
- Team deletion (requires special permissions)
- Team transfer (change ownership)
- Team activity logs
- Member removal functionality
- Team categories/tags
- Team avatars
- Member activity tracking
