# Send Invitations Lambda

A Lambda function that sends HTML-formatted invitation emails to multiple recipients using AWS SES.

## Overview

This Lambda function enables sending professional, branded invitation emails to team members or organization participants. It supports bulk email sending with individual tracking of success/failure for each recipient.

## Features

- 📧 **HTML Email Templates**: Beautiful, responsive HTML emails with gradient styling
- 📊 **Bulk Sending**: Send to multiple recipients in a single API call
- ✅ **Individual Tracking**: Get success/failure status for each email
- 🎨 **Customization**: Support for custom messages and organization branding
- 🔗 **Dynamic Links**: Configurable invitation acceptance links
- 🌍 **Environment-aware**: Email settings configured per environment via CloudFormation

## Prerequisites

### AWS SES Setup

1. **Verify your domain** in AWS SES:
   ```bash
   aws ses verify-domain-identity \
     --domain mvp-dev.4cl-tech.com.au \
     --region ap-southeast-2
   ```

2. **Add DNS records** to Route 53:
   - Add the TXT verification record provided by SES
   - Configure DKIM records for better deliverability
   - Add SPF record: `v=spf1 include:amazonses.com ~all`

3. **Request production access** (if needed):
   - By default, SES is in sandbox mode (can only send to verified emails)
   - Request production access in AWS Console or via CLI to send to any email

## API Endpoint

**POST** `/org/send-invitations`

### Request Headers
- `Authorization`: Cognito JWT token (required)
- `Content-Type`: application/json

### Request Body

```json
{
  "emailAddresses": [
    "user1@example.com",
    "user2@example.com",
    "user3@example.com"
  ],
  "organizationName": "4CL Tech",
  "invitationLink": "https://mvp-dev.4cl-tech.com.au/accept-invitation?token=abc123",
  "customMessage": "We're excited to have you join our team!"
}
```

#### Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `emailAddresses` | string[] | Yes | List of email addresses to send invitations to |
| `organizationName` | string | No | Name of the organization (auto-detected from user's org if not provided) |
| `invitationLink` | string | No | Custom invitation acceptance link (defaults to APP_BASE_URL/accept-invitation) |
| `customMessage` | string | No | Personal message to include in the invitation |

### Response

**Success (200 OK)**
```json
{
  "message": "Sent 3 invitations successfully, 0 failed",
  "totalSent": 3,
  "successCount": 3,
  "failedCount": 0,
  "results": [
    {
      "email": "user1@example.com",
      "success": true
    },
    {
      "email": "user2@example.com",
      "success": true
    },
    {
      "email": "user3@example.com",
      "success": true
    }
  ]
}
```

**Partial Success (207 Multi-Status)**
```json
{
  "message": "Sent 2 invitations successfully, 1 failed",
  "totalSent": 3,
  "successCount": 2,
  "failedCount": 1,
  "results": [
    {
      "email": "user1@example.com",
      "success": true
    },
    {
      "email": "invalid-email",
      "success": false,
      "error": "invalid email address"
    },
    {
      "email": "user3@example.com",
      "success": true
    }
  ]
}
```

**Error Responses**
- `400 Bad Request`: Invalid request body or missing required fields
- `401 Unauthorized`: Missing or invalid authentication token
- `500 Internal Server Error`: All emails failed to send

## Environment Variables

Configured via CloudFormation AccountMappings:

| Variable | Description | Example |
|----------|-------------|---------|
| `DEFAULT_FROM_EMAIL` | Sender email address (must be SES-verified) | noreply@mvp-dev.4cl-tech.com.au |
| `DEFAULT_FROM_NAME` | Display name for sender | 4CL Tech |
| `APP_BASE_URL` | Base URL for invitation links | https://mvp-dev.4cl-tech.com.au |
| `ORGANIZATION_TABLE` | DynamoDB table for organizations | Organizations-Table-dev |
| `EMPLOYEE_TABLE` | DynamoDB table for employees | Employee-Table-dev |

## Email Template

The invitation email includes:
- Gradient header with personalized greeting
- Inviter's name and organization name
- Optional custom message in a styled callout box
- Prominent call-to-action button
- Plain text fallback link
- Responsive design for mobile and desktop

### Customization

To customize the email template, modify the `buildInvitationEmailHTML` function in:
```
lambdas/lib/company-lib/company-email.go
```

## Deployment

### Build
```bash
cd lambdas/tenant-lambdas/org-module/send-invitations
make build
```

### Deploy via CloudFormation
```bash
cd cfn/tenant-cfn
sam build
sam deploy --guided
```

The Lambda is automatically configured in the CloudFormation template with all necessary permissions and environment variables.

## IAM Permissions

The Lambda requires the following permissions (configured via OrganizationLambdaRole):
- `ses:SendEmail` - Send emails via SES
- `ses:SendRawEmail` - Send raw email messages
- `dynamodb:Query` - Query organization and employee tables
- `dynamodb:GetItem` - Get specific items from DynamoDB

## Testing

### Local Testing
```bash
# Test with sample event
sam local invoke SendInvitationsLambda -e test-event.json
```

### Via API Gateway
```bash
curl -X POST https://your-api.execute-api.ap-southeast-2.amazonaws.com/org/send-invitations \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "emailAddresses": ["test@example.com"],
    "customMessage": "Welcome to the team!"
  }'
```

## Monitoring

### CloudWatch Metrics
- Lambda invocations
- Error rate
- Duration

### CloudWatch Logs
Check logs for detailed execution information:
```bash
aws logs tail /aws/lambda/SendInvitationsLambda-dev --follow
```

### SES Metrics
Monitor email sending in SES dashboard:
- Delivery rate
- Bounce rate
- Complaint rate

## Troubleshooting

### Email not sending
1. Verify domain in SES: `aws ses get-identity-verification-attributes --identities mvp-dev.4cl-tech.com.au`
2. Check SES sandbox status: Production access may be required
3. Review CloudWatch logs for detailed error messages

### Invalid email addresses
- Ensure email format is valid
- Check if recipient's domain accepts emails
- Verify SPF/DKIM records are configured

### Rate limiting
- SES has sending quotas (adjust in AWS Console)
- Implement exponential backoff for retries
- Consider using SES sending pools for high volume

## Related Documentation

- [AWS SES Developer Guide](https://docs.aws.amazon.com/ses/)
- [Email Service Library](../../../lib/company-lib/company-email.go)
- [Organization Module](../)

## Support

For issues or questions, contact the development team or refer to the main project documentation.
