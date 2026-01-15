# Team Attributes API (V2)

This module provides APIs for managing team-specific attributes (skills, values, milestones, and metrics) in the engagements system.

## Overview

The V2 implementation consolidates skills, values, milestones, and metrics into a single unified table structure. Each team gets 3 default options for each attribute type upon initialization, and team admins can create custom attributes as needed.

## Architecture

### Library File
- **File**: `lambdas/lib/company-lib/company-skills-values-engagements-v2.go`
- **Service**: `TeamAttributeServiceV2`
- **Table Structure**: Single DynamoDB table with TeamId-AttributeType GSI

### API Endpoints
- **File**: `lambdas/tenant-lambdas/engagements-module/manage-team-attributes/manage-team-attributes.go`
- **Base Path**: `/teams/{teamId}/attributes`

## Database Schema

### Team Attributes Table

```
Primary Key: AttributeId (String)
Sort Key: TeamId (String)
GSI: TeamId-AttributeType-index
  - PK: TeamId
  - SK: AttributeType
```

### Attribute Structure

```json
{
  "attributeId": "ATTR-{uuid}",
  "teamId": "TEAM-{id}",
  "attributeType": "SKILL|VALUE|MILESTONE|METRIC",
  "name": "Attribute Name",
  "description": "Attribute Description",
  "isDefault": true/false,
  "createdAt": "2026-01-15T10:30:00Z",
  "createdBy": "username",
  "updatedAt": "2026-01-15T10:30:00Z"
}
```

## Default Attributes

Each team automatically receives 3 default attributes per type:

### Skills (3 defaults)
1. **Leadership** - Demonstrated ability to lead and inspire team members
2. **Communication** - Effective verbal and written communication skills
3. **Problem Solving** - Ability to analyze and resolve complex challenges

### Values (3 defaults)
1. **Integrity** - Consistently demonstrates honesty and strong moral principles
2. **Teamwork** - Works collaboratively and supports team success
3. **Innovation** - Brings creative ideas and embraces new approaches

### Milestones (3 defaults)
1. **First Quarter Achievement** - Successfully completed first quarter objectives
2. **Project Completion** - Delivered project on time and within scope
3. **Team Goal Achievement** - Contributed significantly to achieving team goals

### Metrics (3 defaults)
1. **Productivity** - Measures output and efficiency in task completion
2. **Quality** - Measures the standard of work delivered
3. **Engagement** - Measures active participation and involvement

## API Endpoints

### 1. Create Custom Attribute

**Endpoint**: `POST /teams/{teamId}/attributes`

**Authorization**: Team Admin only

**Request Body**:
```json
{
  "attributeType": "SKILL|VALUE|MILESTONE|METRIC",
  "name": "Custom Attribute Name",
  "description": "Optional description"
}
```

**Response** (201 Created):
```json
{
  "message": "Attribute created successfully",
  "attribute": {
    "attributeId": "ATTR-123e4567-e89b-12d3-a456-426614174000",
    "teamId": "TEAM-abc123",
    "attributeType": "SKILL",
    "name": "Custom Attribute Name",
    "description": "Optional description",
    "createdBy": "john.doe"
  }
}
```

**Error Responses**:
- `400 Bad Request` - Invalid request body or attribute type
- `403 Forbidden` - User is not a team admin
- `500 Internal Server Error` - Server error

### 2. List Team Attributes

**Endpoint**: `GET /teams/{teamId}/attributes`

**Authorization**: Any team member

**Query Parameters**:
- `type` (optional): Filter by attribute type (`SKILL`, `VALUE`, `MILESTONE`, or `METRIC`)

**Response** (200 OK):
```json
{
  "teamId": "TEAM-abc123",
  "attributes": {
    "skills": [
      {
        "attributeId": "ATTR-123",
        "teamId": "TEAM-abc123",
        "attributeType": "SKILL",
        "name": "Leadership",
        "description": "Demonstrated ability to lead...",
        "isDefault": true,
        "createdAt": "2026-01-15T10:00:00Z",
        "createdBy": "system",
        "updatedAt": "2026-01-15T10:00:00Z"
      }
    ],
    "values": [...],
    "milestones": [...],
    "metrics": [...]
  },
  "total": 12
}
```

**Error Responses**:
- `403 Forbidden` - User is not a team member
- `500 Internal Server Error` - Server error

## Environment Variables

The Lambda function requires the following environment variables:

```bash
EMPLOYEE_TABLE=TenantEmployeeTable
EMPLOYEE_TABLE_COGNITO_ID_INDEX=CognitoId-index
TEAMS_TABLE=TenantTeamsTable
TEAM_ATTRIBUTES_TABLE=TenantTeamAttributesTable
TEAM_ATTRIBUTES_TEAMID_INDEX=TeamId-AttributeType-index
```

## Service Methods

### TeamAttributeServiceV2

```go
// Initialize default attributes for a team
InitializeDefaultAttributes(teamId string, createdBy string) error

// Create a custom attribute (admin only)
CreateCustomAttribute(attr TeamAttribute) error

// List attributes for a team
ListTeamAttributes(teamId string, attributeType *TeamAttributeType) ([]TeamAttribute, error)

// Get attributes grouped by type
GetAttributesByType(teamId string) (GroupedAttributes, error)

// Update an attribute
UpdateAttribute(attributeId, teamId, name, description string) error

// Delete a custom attribute (cannot delete defaults)
DeleteAttribute(attributeId, teamId string) error
```

## Access Control

### Team Admin
- Can create custom attributes
- Can view all attributes
- Can update custom attributes
- Can delete custom attributes (not defaults)

### Team Member
- Can view all attributes
- Cannot create, update, or delete attributes

## Usage Example

### Initialize Team Attributes
```go
svc := companylib.CreateTeamAttributeServiceV2(ctx, ddbClient, logger)
svc.TeamAttributesTable = "TenantTeamAttributesTable"
svc.TeamAttributesTeamIdIndex = "TeamId-AttributeType-index"

// When a new team is created
err := svc.InitializeDefaultAttributes("TEAM-abc123", "admin-user")
```

### Create Custom Attribute (via API)
```bash
curl -X POST https://api.example.com/teams/TEAM-abc123/attributes \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{
    "attributeType": "SKILL",
    "name": "Data Analysis",
    "description": "Ability to analyze and interpret data"
  }'
```

### List Attributes (via API)
```bash
# Get all attributes
curl -X GET https://api.example.com/teams/TEAM-abc123/attributes \
  -H "Authorization: Bearer {token}"

# Get only skills
curl -X GET https://api.example.com/teams/TEAM-abc123/attributes?type=SKILL \
  -H "Authorization: Bearer {token}"
```

## Building and Deploying

### Build Lambda
```bash
cd lambdas/tenant-lambdas/engagements-module/manage-team-attributes
make build
```

### Deploy
The Lambda should be configured with:
- Runtime: Custom (Go 1.x)
- Handler: bootstrap
- Timeout: 30 seconds
- Memory: 256 MB

### API Gateway Integration
- Path: `/teams/{teamId}/attributes`
- Methods: GET, POST
- Authorization: Cognito User Pool

## Migration from V1

If migrating from the old separate tables structure:

1. Deploy V2 library and Lambda
2. Run migration script to:
   - Copy data from old tables to new unified table
   - Set `isDefault=true` for existing attributes
   - Add TeamId to all records
3. Update API Gateway routes
4. Test thoroughly before deprecating V1

## Future Enhancements

- Bulk attribute creation
- Attribute templates
- Attribute usage analytics
- Soft delete with archival
- Attribute versioning
- Custom attribute validation rules
