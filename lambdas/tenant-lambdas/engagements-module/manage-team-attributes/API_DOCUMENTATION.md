# Team Attributes API Documentation

## Base URL
```
https://api.{domain}/v2
```

## Authentication
All endpoints require Cognito User Pool authentication with a valid Bearer token.

```
Authorization: Bearer {cognito-token}
```

## Available Endpoints

| Method | Endpoint | Description | Access Level |
|--------|----------|-------------|--------------|
| GET | `/v2/teams/{teamId}/attributes` | List team attributes | Team Members |
| POST | `/v2/teams/{teamId}/attributes` | Create custom attribute | Team Admins |
| PATCH | `/v2/teams/{teamId}/attributes/{attributeId}` | Update custom attribute | Team Admins |
| DELETE | `/v2/teams/{teamId}/attributes/{attributeId}` | Delete custom attribute | Team Admins |

**Note**: Default attributes (provided automatically when teams are created) cannot be updated or deleted.

---

## Endpoints

### 1. Create Custom Team Attribute

Creates a new custom attribute for a team. Only team admins can create custom attributes.

**Endpoint**: `POST /v2/teams/{teamId}/attributes`

**Path Parameters**:
- `teamId` (required): The unique identifier of the team (format: `TEAM-{id}` or `TEAM#{uuid}`)
  - **Important**: If the team ID contains special characters like `#`, it must be URL-encoded (e.g., `TEAM%236f5f4dd7-7c78-47f2-aa6c-12ff66e9aa58`)

**Request Headers**:
```
Authorization: Bearer {token}
Content-Type: application/json
```

**Request Body**:
```json
{
  "attributeType": "SKILL",
  "name": "Data Analysis",
  "description": "Ability to analyze and interpret complex data sets"
}
```

**Request Body Parameters**:
- `attributeType` (required): Type of attribute. Valid values: `SKILL`, `VALUE`, `MILESTONE`, `METRIC`
- `name` (required): Display name for the attribute (max 100 characters)
- `description` (optional): Detailed description of the attribute (max 500 characters)

**Success Response** (201 Created):
```json
{
  "message": "Attribute created successfully",
  "attribute": {
    "attributeId": "ATTR-123e4567-e89b-12d3-a456-426614174000",
    "teamId": "TEAM-abc123",
    "attributeType": "SKILL",
    "name": "Data Analysis",
    "description": "Ability to analyze and interpret complex data sets",
    "createdBy": "john.doe"
  }
}
```

**Error Responses**:

400 Bad Request:
```json
{
  "error": "Invalid attribute type. Must be SKILL, VALUE, MILESTONE, or METRIC"
}
```

403 Forbidden:
```json
{
  "error": "Only team admins can create custom attributes"
}
```

404 Not Found:
```json
{
  "error": "Team not found"
}
```

**cURL Example**:
```bash
curl -X POST https://api.example.com/v2/teams/TEAM-abc123/attributes \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1..." \
  -H "Content-Type: application/json" \
  -d '{
    "attributeType": "SKILL",
    "name": "Data Analysis",
    "description": "Ability to analyze and interpret complex data sets"
  }'
```

---

### 2. List Team Attributes

Retrieves all attributes for a team, grouped by type. All team members can access this endpoint.

**Endpoint**: `GET /v2/teams/{teamId}/attributes`

**Path Parameters**:
- `teamId` (required): The unique identifier of the team (format: `TEAM-{id}` or `TEAM#{uuid}`)
  - **Important**: If the team ID contains special characters like `#`, it must be URL-encoded (e.g., `TEAM%236f5f4dd7-7c78-47f2-aa6c-12ff66e9aa58`)

**Query Parameters**:
- `type` (optional): Filter results by attribute type. Valid values: `SKILL`, `VALUE`, `MILESTONE`, `METRIC`

**Request Headers**:
```
Authorization: Bearer {token}
```

**Success Response** (200 OK):

Without filter:
```json
{
  "teamId": "TEAM-abc123",
  "attributes": {
    "skills": [
      {
        "attributeId": "ATTR-skill-001",
        "teamId": "TEAM-abc123",
        "attributeType": "SKILL",
        "name": "Leadership",
        "description": "Demonstrated ability to lead and inspire team members",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      },
      {
        "attributeId": "ATTR-skill-002",
        "teamId": "TEAM-abc123",
        "attributeType": "SKILL",
        "name": "Communication",
        "description": "Effective verbal and written communication skills",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      },
      {
        "attributeId": "ATTR-skill-003",
        "teamId": "TEAM-abc123",
        "attributeType": "SKILL",
        "name": "Problem Solving",
        "description": "Ability to analyze and resolve complex challenges",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      }
    ],
    "values": [
      {
        "attributeId": "ATTR-value-001",
        "teamId": "TEAM-abc123",
        "attributeType": "VALUE",
        "name": "Integrity",
        "description": "Consistently demonstrates honesty and strong moral principles",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      },
      {
        "attributeId": "ATTR-value-002",
        "teamId": "TEAM-abc123",
        "attributeType": "VALUE",
        "name": "Teamwork",
        "description": "Works collaboratively and supports team success",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      },
      {
        "attributeId": "ATTR-value-003",
        "teamId": "TEAM-abc123",
        "attributeType": "VALUE",
        "name": "Innovation",
        "description": "Brings creative ideas and embraces new approaches",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      }
    ],
    "milestones": [
      {
        "attributeId": "ATTR-milestone-001",
        "teamId": "TEAM-abc123",
        "attributeType": "MILESTONE",
        "name": "First Quarter Achievement",
        "description": "Successfully completed first quarter objectives",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      },
      {
        "attributeId": "ATTR-milestone-002",
        "teamId": "TEAM-abc123",
        "attributeType": "MILESTONE",
        "name": "Project Completion",
        "description": "Delivered project on time and within scope",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      },
      {
        "attributeId": "ATTR-milestone-003",
        "teamId": "TEAM-abc123",
        "attributeType": "MILESTONE",
        "name": "Team Goal Achievement",
        "description": "Contributed significantly to achieving team goals",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      }
    ],
    "metrics": [
      {
        "attributeId": "ATTR-metric-001",
        "teamId": "TEAM-abc123",
        "attributeType": "METRIC",
        "name": "Productivity",
        "description": "Measures output and efficiency in task completion",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      },
      {
        "attributeId": "ATTR-metric-002",
        "teamId": "TEAM-abc123",
        "attributeType": "METRIC",
        "name": "Quality",
        "description": "Measures the standard of work delivered",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      },
      {
        "attributeId": "ATTR-metric-003",
        "teamId": "TEAM-abc123",
        "attributeType": "METRIC",
        "name": "Engagement",
        "description": "Measures active participation and involvement",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      }
    ]
  },
  "total": 12
}
```

**Error Responses**:

400 Bad Request (invalid type filter):
```json
{
  "error": "Invalid type filter. Must be SKILL, VALUE, MILESTONE, or METRIC"
}
```

403 Forbidden:
```json
{
  "error": "User is not a member of this team"
}
```

404 Not Found:
```json
{
  "error": "Team not found"
}
```

**cURL Examples**:

Get all attributes:
```bash
curl -X GET https://api.example.com/v2/teams/TEAM-abc123/attributes \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1..."
```

Get only skills:
```bash
curl -X GET "https://api.example.com/v2/teams/TEAM-abc123/attributes?type=SKILL" \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1..."
```

Get only values:
```bash
curl -X GET "https://api.example.com/v2/teams/TEAM-abc123/attributes?type=VALUE" \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1..."
```

---

### 3. Update Custom Team Attribute

Updates an existing custom attribute for a team. Only team admins can update custom attributes. Default attributes cannot be updated.

**Endpoint**: `PATCH /v2/teams/{teamId}/attributes/{attributeId}`

**Path Parameters**:
- `teamId` (required): The unique identifier of the team (format: `TEAM-{id}` or `TEAM#{uuid}`)
  - **Important**: If the team ID contains special characters like `#`, it must be URL-encoded (e.g., `TEAM%236f5f4dd7-7c78-47f2-aa6c-12ff66e9aa58`)
- `attributeId` (required): The unique identifier of the attribute to update (format: `ATTR-{uuid}`)

**Request Headers**:
```
Authorization: Bearer {token}
Content-Type: application/json
```

**Request Body**:
```json
{
  "name": "Advanced Data Analysis",
  "description": "Ability to perform advanced statistical analysis and machine learning"
}
```

**Request Body Parameters**:
- `name` (optional): New display name for the attribute (max 100 characters)
- `description` (optional): New description for the attribute (max 500 characters)
- **Note**: At least one field must be provided

**Success Response** (200 OK):
```json
{
  "message": "Attribute updated successfully",
  "attribute": {
    "attributeId": "ATTR-123e4567-e89b-12d3-a456-426614174000",
    "teamId": "TEAM-abc123",
    "attributeType": "SKILL",
    "name": "Advanced Data Analysis",
    "description": "Ability to perform advanced statistical analysis and machine learning"
  }
}
```

**Error Responses**:

400 Bad Request (no fields provided):
```json
{
  "error": "At least one field (name or description) must be provided for update"
}
```

403 Forbidden:
```json
{
  "error": "Only team admins can update custom attributes"
}
```

404 Not Found:
```json
{
  "error": "Attribute not found"
}
```

**cURL Example**:
```bash
curl -X PATCH https://api.example.com/v2/teams/TEAM-abc123/attributes/ATTR-123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Advanced Data Analysis",
    "description": "Ability to perform advanced statistical analysis and machine learning"
  }'
```

---

### 4. Delete Custom Team Attribute

Deletes a custom attribute from a team. Only team admins can delete custom attributes. Default attributes cannot be deleted.

**Endpoint**: `DELETE /v2/teams/{teamId}/attributes/{attributeId}`

**Path Parameters**:
- `teamId` (required): The unique identifier of the team (format: `TEAM-{id}` or `TEAM#{uuid}`)
  - **Important**: If the team ID contains special characters like `#`, it must be URL-encoded (e.g., `TEAM%236f5f4dd7-7c78-47f2-aa6c-12ff66e9aa58`)
- `attributeId` (required): The unique identifier of the attribute to delete (format: `ATTR-{uuid}`)

**Request Headers**:
```
Authorization: Bearer {token}
```

**Success Response** (200 OK):
```json
{
  "message": "Attribute deleted successfully",
  "attributeId": "ATTR-123e4567-e89b-12d3-a456-426614174000",
  "deletedBy": "john.doe"
}
```

**Error Responses**:

403 Forbidden:
```json
{
  "error": "Only team admins can delete custom attributes"
}
```

404 Not Found:
```json
{
  "error": "Attribute not found"
}
```

**cURL Example**:
```bash
curl -X DELETE https://api.example.com/v2/teams/TEAM-abc123/attributes/ATTR-123e4567-e89b-12d3-a456-426614174000 \
  -H "Authorization: Bearer eyJhbGciOiJSUzI1..."
```

---

## Team Initialization

### Automatic Default Attributes

When a new team is created, the system should automatically initialize 12 default attributes (3 per type):
- **3 Skills**: Leadership, Communication, Problem Solving
- **3 Values**: Integrity, Teamwork, Innovation
- **3 Milestones**: First Quarter Achievement, Project Completion, Team Goal Achievement
- **3 Metrics**: Productivity, Quality, Engagement

### Integration with Team Creation

When creating a new team via the teams API, ensure you call the initialization:

```go
// After creating the team
attributeSvc := companylib.CreateTeamAttributeServiceV2(ctx, ddbClient, logger)
attributeSvc.TeamAttributesTable = "TeamAttributesTable"
attributeSvc.TeamAttributesTeamIdIndex = "TeamId-AttributeType-index"

err := attributeSvc.InitializeDefaultAttributes(teamId, createdByUsername)
if err != nil {
    log.Printf("Warning: Failed to initialize default attributes: %v", err)
    // Team creation succeeded, but default attributes failed
    // This is non-critical and can be retried
}
```

**Important Notes**:
- Default attributes are created with `isDefault: true` and cannot be deleted
- If initialization fails, the team is still valid, but won't have default attributes
- Team admins can still create custom attributes even if defaults are missing
- The GET endpoint will return an empty list if no attributes exist for a team

---

## Data Models

### TeamAttribute

```typescript
interface TeamAttribute {
  attributeId: string;      // Format: ATTR-{uuid}
  teamId: string;           // Format: TEAM-{id}
  attributeType: AttributeType;
  name: string;
  description: string;
  isDefault: boolean;       // true for system defaults, false for custom
  createdAt: string;        // ISO 8601 timestamp
  createdBy: string;        // Username
  updatedAt: string;        // ISO 8601 timestamp
}
```

### AttributeType

```typescript
enum AttributeType {
  SKILL = "SKILL",
  VALUE = "VALUE",
  MILESTONE = "MILESTONE",
  METRIC = "METRIC"
}
```

---

## Access Control

### Team Member (Any Role)
- ✅ GET /v2/teams/{teamId}/attributes
- ❌ POST /v2/teams/{teamId}/attributes
- ❌ PATCH /v2/teams/{teamId}/attributes/{attributeId}
- ❌ DELETE /v2/teams/{teamId}/attributes/{attributeId}

### Team Admin
- ✅ GET /v2/teams/{teamId}/attributes
- ✅ POST /v2/teams/{teamId}/attributes
- ✅ PATCH /v2/teams/{teamId}/attributes/{attributeId}
- ✅ DELETE /v2/teams/{teamId}/attributes/{attributeId}

### Non-Team Member
- ❌ All endpoints (403 Forbidden)

---

## Rate Limiting

API Gateway throttles requests based on account limits:
- Burst: 5,000 requests
- Steady state: 10,000 requests per second

Per-user limits (recommended):
- 100 requests per minute per user

---

## Error Codes

| Status Code | Description |
|------------|-------------|
| 200 | Success - Request completed successfully |
| 201 | Created - Resource created successfully |
| 400 | Bad Request - Invalid request parameters |
| 401 | Unauthorized - Missing or invalid authentication |
| 403 | Forbidden - User lacks required permissions |
| 404 | Not Found - Resource does not exist |
| 429 | Too Many Requests - Rate limit exceeded |
| 500 | Internal Server Error - Server encountered an error |
| 503 | Service Unavailable - Service temporarily unavailable |

---

## Best Practices

### For Frontend Developers

1. **Cache attribute lists** - Attributes don't change frequently, cache for 5-10 minutes
2. **Show loading states** - API calls may take 200-500ms
3. **Handle empty states** - New teams start with only 12 default attributes
4. **Validate input** - Check attribute types before sending requests
5. **Show defaults separately** - UI can distinguish default vs custom attributes using `isDefault` flag

### For Backend Integration

1. **Initialize on team creation** - Call `InitializeDefaultAttributes()` when creating new teams
2. **Batch operations** - Use DynamoDB batch operations when possible
3. **Monitor GSI usage** - Most queries use the GSI, ensure it's properly provisioned
4. **Implement caching** - Use ElastiCache or similar for frequently accessed data

### Example Frontend Usage (React)

```typescript
// Fetch team attributes
const fetchTeamAttributes = async (teamId: string) => {
  // URL-encode the team ID to handle special characters like #
  const encodedTeamId = encodeURIComponent(teamId);
  const response = await fetch(
    `https://api.example.com/v2/teams/${encodedTeamId}/attributes`,
    {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    }
  );
  
  if (!response.ok) {
    throw new Error('Failed to fetch attributes');
  }
  
  return await response.json();
};

// Create custom attribute
const createAttribute = async (
  teamId: string,
  attributeType: string,
  name: string,
  description: string
) => {
  // URL-encode the team ID to handle special characters like #
  const encodedTeamId = encodeURIComponent(teamId);
  const response = await fetch(
    `https://api.example.com/v2/teams/${encodedTeamId}/attributes`,
    {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        attributeType,
        name,
        description,
      }),
    }
  );
  
  if (!response.ok) {
    throw new Error('Failed to create attribute');
  }
  
  return await response.json();
};

// Update custom attribute
const updateAttribute = async (
  teamId: string,
  attributeId: string,
  name?: string,
  description?: string
) => {
  const encodedTeamId = encodeURIComponent(teamId);
  const response = await fetch(
    `https://api.example.com/v2/teams/${encodedTeamId}/attributes/${attributeId}`,
    {
      method: 'PATCH',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        ...(name && { name }),
        ...(description && { description }),
      }),
    }
  );
  
  if (!response.ok) {
    throw new Error('Failed to update attribute');
  }
  
  return await response.json();
};

// Delete custom attribute
const deleteAttribute = async (
  teamId: string,
  attributeId: string
) => {
  const encodedTeamId = encodeURIComponent(teamId);
  const response = await fetch(
    `https://api.example.com/v2/teams/${encodedTeamId}/attributes/${attributeId}`,
    {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    }
  );
  
  if (!response.ok) {
    throw new Error('Failed to delete attribute');
  }
  
  return await response.json();
};
```

---

## Common Issues & Troubleshooting

### 404 Not Found Error

**Issue**: Getting 404 when calling `/v2/teams/TEAM#xxx/attributes`

**Cause**: The `#` character in the team ID is not URL-encoded, causing the URL to be truncated

**Solution**: URL-encode the team ID before making the request:
```typescript
const encodedTeamId = encodeURIComponent(teamId);
fetch(`/v2/teams/${encodedTeamId}/attributes`);
```

### Empty Attributes List

**Issue**: GET request returns empty list or `total: 0`

**Possible Causes**:
1. Default attributes were not initialized when the team was created
2. Team does not exist
3. User is not a member of the team (would return 403, not empty list)

**Solution**: 
- For new teams: Ensure `InitializeDefaultAttributes()` is called during team creation
- For existing teams: Team admins can manually create attributes via POST endpoint
- Check CloudWatch logs for any errors during team creation

### 403 Forbidden When Creating Attributes

**Issue**: Team admins getting 403 when trying to POST new attributes

**Possible Causes**:
1. User's role in the team is not set to "Admin"
2. Team membership data is stale

**Solution**:
- Verify user's role via GET `/v2/teams/{teamId}/members`
- Check that the user's role is exactly "Admin" (case-sensitive)
- Refresh team membership cache

---

## Changelog

### Version 2.1.0 (2026-01-23)
- **Added**: PATCH `/v2/teams/{teamId}/attributes/{attributeId}` endpoint documentation
- **Added**: DELETE `/v2/teams/{teamId}/attributes/{attributeId}` endpoint documentation  
- **Enhanced**: Complete access control documentation for all endpoints
- **Enhanced**: Frontend integration examples with TypeScript for all CRUD operations
- **Enhanced**: Comprehensive error handling examples
- **Updated**: Swagger/OpenAPI specification with new endpoints

### Version 2.0.0 (2026-01-15)
- Initial release of unified Team Attributes API
- Consolidated skills, values, milestones, and metrics into single table
- Added team-based access control
- Implemented default attributes (3 per type)
- Support for custom attribute creation by team admins
