# Send Invitations API - Quick Reference

## Endpoint
```
POST /v2/organization/send-invitations
```

## Headers
```json
{
  "Authorization": "Bearer <JWT_TOKEN>",
  "Content-Type": "application/json"
}
```

## Request Body
```typescript
{
  emailAddresses: string[];      // Required
  organizationName?: string;     // Optional
  invitationLink?: string;       // Optional
  customMessage?: string;        // Optional (max 500 chars)
}
```

## Response
```typescript
{
  message: string;
  totalSent: number;
  successCount: number;
  failedCount: number;
  results: Array<{
    email: string;
    success: boolean;
    error?: string;
  }>;
}
```

## Status Codes
- **200**: All sent successfully
- **207**: Partial success
- **400**: Bad request
- **401**: Unauthorized
- **500**: All failed

## Quick Example
```javascript
const response = await fetch('https://mvp-dev.4cl-tech.com.au/v2/organization/send-invitations', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    emailAddresses: ['user1@example.com', 'user2@example.com'],
    customMessage: 'Welcome to the team!'
  })
});

const data = await response.json();
console.log(`Success: ${data.successCount}, Failed: ${data.failedCount}`);
```
