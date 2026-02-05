# Send Invitations API - Frontend Integration Guide

## API Endpoint

```
POST /v2/organization/send-invitations
```

**Base URL Examples:**
- Dev: `https://mvp-dev.4cl-tech.com.au/v2/organization/send-invitations`
- Production: `https://your-domain.com/v2/organization/send-invitations`

## Authentication

Required: Cognito JWT token in Authorization header

```javascript
headers: {
  'Authorization': `Bearer ${cognitoJwtToken}`,
  'Content-Type': 'application/json'
}
```

---

## Request Structure

### TypeScript Interface

```typescript
interface SendInvitationsRequest {
  emailAddresses: string[];           // Required: List of email addresses
  organizationName?: string;          // Optional: Organization name (auto-detected if omitted)
  invitationLink?: string;            // Optional: Custom invitation link
  customMessage?: string;             // Optional: Personal message (max 500 chars)
}
```

### Sample Request - Minimal

```json
{
  "emailAddresses": [
    "john.doe@example.com",
    "jane.smith@example.com"
  ]
}
```

### Sample Request - Full

```json
{
  "emailAddresses": [
    "john.doe@example.com",
    "jane.smith@example.com",
    "alex.wilson@company.com"
  ],
  "organizationName": "4CL Tech",
  "invitationLink": "https://mvp-dev.4cl-tech.com.au/accept-invitation?token=abc123xyz",
  "customMessage": "We're excited to have you join our team! Looking forward to working together."
}
```

---

## Response Structures

### TypeScript Interfaces

```typescript
interface InvitationResult {
  email: string;
  success: boolean;
  error?: string;  // Only present if success is false
}

interface SendInvitationsResponse {
  message: string;
  totalSent: number;
  successCount: number;
  failedCount: number;
  results: InvitationResult[];
}
```

### Success Response (200 OK)

All invitations sent successfully.

```json
{
  "message": "Sent 3 invitations successfully, 0 failed",
  "totalSent": 3,
  "successCount": 3,
  "failedCount": 0,
  "results": [
    {
      "email": "john.doe@example.com",
      "success": true
    },
    {
      "email": "jane.smith@example.com",
      "success": true
    },
    {
      "email": "alex.wilson@company.com",
      "success": true
    }
  ]
}
```

### Partial Success Response (207 Multi-Status)

Some invitations sent, others failed.

```json
{
  "message": "Sent 2 invitations successfully, 1 failed",
  "totalSent": 3,
  "successCount": 2,
  "failedCount": 1,
  "results": [
    {
      "email": "john.doe@example.com",
      "success": true
    },
    {
      "email": "invalid-email-format",
      "success": false,
      "error": "invalid email address"
    },
    {
      "email": "alex.wilson@company.com",
      "success": true
    }
  ]
}
```

### Error Response (400 Bad Request)

Invalid request body or missing required fields.

```json
{
  "error": "At least one email address is required",
  "details": "At least one email address is required: validation failed"
}
```

### Error Response (401 Unauthorized)

Missing or invalid authentication token.

```json
{
  "error": "Unauthorized",
  "details": "Unauthorized: cognito ID not found in request"
}
```

### Error Response (500 Internal Server Error)

All invitations failed to send.

```json
{
  "error": "Failed to send invitations",
  "details": "Failed to send invitations: SES service unavailable"
}
```

---

## Frontend Implementation Examples

### React with Axios

```typescript
import axios from 'axios';

interface SendInvitationsRequest {
  emailAddresses: string[];
  organizationName?: string;
  invitationLink?: string;
  customMessage?: string;
}

interface InvitationResult {
  email: string;
  success: boolean;
  error?: string;
}

interface SendInvitationsResponse {
  message: string;
  totalSent: number;
  successCount: number;
  failedCount: number;
  results: InvitationResult[];
}

const sendInvitations = async (
  request: SendInvitationsRequest,
  token: string
): Promise<SendInvitationsResponse> => {
  try {
    const response = await axios.post<SendInvitationsResponse>(
      'https://mvp-dev.4cl-tech.com.au/v2/organization/send-invitations',
      request,
      {
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
      }
    );

    return response.data;
  } catch (error) {
    if (axios.isAxiosError(error) && error.response) {
      throw new Error(error.response.data.error || 'Failed to send invitations');
    }
    throw error;
  }
};

// Usage in a React component
const InvitationForm: React.FC = () => {
  const [emails, setEmails] = useState<string[]>([]);
  const [customMessage, setCustomMessage] = useState('');
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<InvitationResult[]>([]);

  const handleSendInvitations = async () => {
    setLoading(true);
    try {
      const token = await getCurrentUserToken(); // Your auth method
      
      const response = await sendInvitations(
        {
          emailAddresses: emails,
          customMessage: customMessage || undefined,
        },
        token
      );

      setResults(response.results);
      
      // Show success message
      alert(`${response.successCount} invitations sent successfully!`);
      
      if (response.failedCount > 0) {
        console.warn('Failed invitations:', 
          response.results.filter(r => !r.success)
        );
      }
    } catch (error) {
      console.error('Error sending invitations:', error);
      alert('Failed to send invitations. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      {/* Your form UI here */}
      <button onClick={handleSendInvitations} disabled={loading}>
        {loading ? 'Sending...' : 'Send Invitations'}
      </button>
      
      {/* Display results */}
      {results.length > 0 && (
        <div>
          <h3>Results</h3>
          {results.map((result, index) => (
            <div key={index}>
              {result.email}: {result.success ? '✓ Sent' : `✗ Failed - ${result.error}`}
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
```

### Vanilla JavaScript with Fetch

```javascript
async function sendInvitations(emailAddresses, customMessage, token) {
  const requestBody = {
    emailAddresses: emailAddresses,
    customMessage: customMessage || undefined
  };

  try {
    const response = await fetch(
      'https://mvp-dev.4cl-tech.com.au/v2/organization/send-invitations',
      {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody)
      }
    );

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to send invitations');
    }

    const data = await response.json();
    return data;
  } catch (error) {
    console.error('Error sending invitations:', error);
    throw error;
  }
}

// Usage
async function handleSendInvitations() {
  const emails = ['user1@example.com', 'user2@example.com'];
  const message = 'Welcome to the team!';
  const token = getUserToken(); // Your auth method

  try {
    const result = await sendInvitations(emails, message, token);
    
    console.log('Success:', result.successCount);
    console.log('Failed:', result.failedCount);
    console.log('Results:', result.results);

    // Display results to user
    result.results.forEach(item => {
      if (item.success) {
        console.log(`✓ ${item.email} - Invitation sent`);
      } else {
        console.error(`✗ ${item.email} - ${item.error}`);
      }
    });
  } catch (error) {
    alert('Failed to send invitations: ' + error.message);
  }
}
```

### Angular Service

```typescript
import { Injectable } from '@angular/core';
import { HttpClient, HttpHeaders } from '@angular/common/http';
import { Observable } from 'rxjs';

interface SendInvitationsRequest {
  emailAddresses: string[];
  organizationName?: string;
  invitationLink?: string;
  customMessage?: string;
}

interface InvitationResult {
  email: string;
  success: boolean;
  error?: string;
}

interface SendInvitationsResponse {
  message: string;
  totalSent: number;
  successCount: number;
  failedCount: number;
  results: InvitationResult[];
}

@Injectable({
  providedIn: 'root'
})
export class InvitationService {
  private apiUrl = 'https://mvp-dev.4cl-tech.com.au/v2/organization';

  constructor(private http: HttpClient) {}

  sendInvitations(
    request: SendInvitationsRequest,
    token: string
  ): Observable<SendInvitationsResponse> {
    const headers = new HttpHeaders({
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json'
    });

    return this.http.post<SendInvitationsResponse>(
      `${this.apiUrl}/send-invitations`,
      request,
      { headers }
    );
  }
}

// Usage in component
export class InvitationComponent {
  constructor(private invitationService: InvitationService) {}

  sendInvites(emails: string[], message: string) {
    const token = this.authService.getToken(); // Your auth service

    this.invitationService.sendInvitations(
      {
        emailAddresses: emails,
        customMessage: message
      },
      token
    ).subscribe({
      next: (response) => {
        console.log('Invitations sent:', response);
        this.showSuccessMessage(response);
      },
      error: (error) => {
        console.error('Error:', error);
        this.showErrorMessage(error);
      }
    });
  }
}
```

---

## Error Handling Best Practices

```typescript
async function sendInvitationsWithErrorHandling(
  emails: string[],
  token: string
) {
  try {
    const response = await sendInvitations({ emailAddresses: emails }, token);
    
    // Handle different response codes
    if (response.failedCount === 0) {
      // All successful (200)
      showSuccessToast(`All ${response.successCount} invitations sent!`);
    } else if (response.successCount > 0) {
      // Partial success (207)
      showWarningToast(
        `${response.successCount} sent, ${response.failedCount} failed`
      );
      
      // Display failed emails to user
      const failedEmails = response.results
        .filter(r => !r.success)
        .map(r => `${r.email}: ${r.error}`);
      
      showFailedEmailsList(failedEmails);
    } else {
      // All failed (500)
      showErrorToast('All invitations failed to send');
    }
    
    return response;
  } catch (error: any) {
    // Handle specific error codes
    if (error.response?.status === 400) {
      showErrorToast('Invalid request. Please check the email addresses.');
    } else if (error.response?.status === 401) {
      showErrorToast('Session expired. Please login again.');
      redirectToLogin();
    } else if (error.response?.status === 500) {
      showErrorToast('Server error. Please try again later.');
    } else {
      showErrorToast('Network error. Please check your connection.');
    }
    
    throw error;
  }
}
```

---

## Validation Examples

### Email Validation (Frontend)

```typescript
function validateEmails(emails: string[]): { valid: string[], invalid: string[] } {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  
  const valid: string[] = [];
  const invalid: string[] = [];
  
  emails.forEach(email => {
    if (emailRegex.test(email.trim())) {
      valid.push(email.trim());
    } else {
      invalid.push(email);
    }
  });
  
  return { valid, invalid };
}

// Usage before sending
function handleSubmit(emails: string[]) {
  const { valid, invalid } = validateEmails(emails);
  
  if (invalid.length > 0) {
    alert(`Invalid emails: ${invalid.join(', ')}`);
    return;
  }
  
  sendInvitations({ emailAddresses: valid }, token);
}
```

### Custom Message Validation

```typescript
function validateCustomMessage(message: string): boolean {
  const MAX_LENGTH = 500;
  
  if (message.length > MAX_LENGTH) {
    alert(`Message too long. Maximum ${MAX_LENGTH} characters.`);
    return false;
  }
  
  return true;
}
```

---

## Testing

### Mock Response for Testing

```typescript
// Mock successful response
const mockSuccessResponse: SendInvitationsResponse = {
  message: "Sent 3 invitations successfully, 0 failed",
  totalSent: 3,
  successCount: 3,
  failedCount: 0,
  results: [
    { email: "test1@example.com", success: true },
    { email: "test2@example.com", success: true },
    { email: "test3@example.com", success: true }
  ]
};

// Mock partial success response
const mockPartialResponse: SendInvitationsResponse = {
  message: "Sent 2 invitations successfully, 1 failed",
  totalSent: 3,
  successCount: 2,
  failedCount: 1,
  results: [
    { email: "test1@example.com", success: true },
    { email: "invalid", success: false, error: "invalid email address" },
    { email: "test3@example.com", success: true }
  ]
};
```

### cURL Example for API Testing

```bash
# Basic request
curl -X POST https://mvp-dev.4cl-tech.com.au/v2/organization/send-invitations \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "emailAddresses": ["test@example.com"]
  }'

# Full request with all options
curl -X POST https://mvp-dev.4cl-tech.com.au/v2/organization/send-invitations \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "emailAddresses": ["user1@example.com", "user2@example.com"],
    "organizationName": "4CL Tech",
    "invitationLink": "https://mvp-dev.4cl-tech.com.au/accept?token=abc",
    "customMessage": "Welcome to the team!"
  }'
```

---

## Rate Limiting Considerations

AWS SES has sending limits:
- **Sandbox mode**: Can only send to verified email addresses
- **Production mode**: Default quota is ~50,000 emails/day
- Consider implementing client-side throttling for bulk invitations
- Add retry logic with exponential backoff for failed requests

```typescript
async function sendInvitationsInBatches(
  emails: string[],
  batchSize: number = 10,
  delayMs: number = 1000
) {
  const results = [];
  
  for (let i = 0; i < emails.length; i += batchSize) {
    const batch = emails.slice(i, i + batchSize);
    
    try {
      const response = await sendInvitations({ emailAddresses: batch }, token);
      results.push(...response.results);
      
      // Delay between batches to avoid rate limiting
      if (i + batchSize < emails.length) {
        await delay(delayMs);
      }
    } catch (error) {
      console.error(`Batch ${i / batchSize} failed:`, error);
      // Add failed batch to results
      batch.forEach(email => {
        results.push({ email, success: false, error: 'Batch failed' });
      });
    }
  }
  
  return results;
}

function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}
```

---

## Support

For issues or questions:
- Check CloudWatch logs for detailed error messages
- Verify AWS SES domain verification status
- Ensure Cognito token is valid and not expired
- Contact the backend team for API-related issues
