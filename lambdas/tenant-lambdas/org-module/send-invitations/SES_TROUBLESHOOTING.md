# AWS SES Troubleshooting Guide

## Common Error: "Email address is not verified"

### Error Message
```
MessageRejected: Email address is not verified. The following identities failed 
the check in region AP-SOUTHEAST-2: recipient@example.com
```

### Cause
Your AWS SES account is in **sandbox mode**, which restricts sending emails to only verified addresses.

---

## Solutions

### 🔧 Option 1: Verify Individual Email Addresses (Testing Only)

Use this for testing with specific email addresses.

#### Steps:

1. **Verify the recipient email:**
```bash
aws ses verify-email-identity \
  --email-address sivala84@gmail.com \
  --region ap-southeast-2
```

2. **Check your inbox:**
   - AWS will send a verification email to `sivala84@gmail.com`
   - Click the verification link in the email

3. **Confirm verification status:**
```bash
aws ses get-identity-verification-attributes \
  --identities sivala84@gmail.com \
  --region ap-southeast-2
```

Expected output:
```json
{
  "VerificationAttributes": {
    "sivala84@gmail.com": {
      "VerificationStatus": "Success"
    }
  }
}
```

4. **Retry sending the invitation**

#### Limitations:
- ❌ Must verify each recipient email individually
- ❌ Not suitable for production
- ❌ Cannot send to arbitrary email addresses
- ✅ Good for testing/development

---

### 🚀 Option 2: Request Production Access (Recommended)

Move your SES account out of sandbox mode to send to any email address.

#### Method A: AWS CLI

```bash
aws sesv2 put-account-details \
  --production-access-enabled \
  --mail-type TRANSACTIONAL \
  --website-url https://mvp-dev.4cl-tech.com.au \
  --use-case-description "Sending team invitation emails to organization members. Users will receive HTML-formatted invitations to join teams and collaborate." \
  --additional-contact-email-addresses vishal@4cl-tech.com.au \
  --region ap-southeast-2
```

#### Method B: AWS Console

1. Open [SES Console](https://console.aws.amazon.com/ses/)
2. Navigate to **Account dashboard**
3. Click **Request production access**
4. Fill out the form:
   - **Mail type**: Transactional
   - **Website URL**: https://mvp-dev.4cl-tech.com.au
   - **Use case**: 
     ```
     Sending team invitation emails to organization members.
     Our platform allows organizations to invite team members via email.
     Users receive HTML-formatted invitations with details about joining
     their organization's workspace. Typical volume: 100-500 invitations/day.
     ```
   - **Additional contact email**: your-email@4cl-tech.com.au

5. Click **Submit request**

#### Approval Timeline:
- Usually within **24 hours**
- May require additional information
- Check email for AWS support updates

#### After Approval:
✅ Send to any email address  
✅ Higher sending limits  
✅ Better deliverability metrics  

---

## Verify Current SES Status

### Check if you're in sandbox mode:

```bash
# Check account status
aws sesv2 get-account --region ap-southeast-2
```

Look for:
```json
{
  "ProductionAccessEnabled": false  // ← You're in sandbox mode
}
```

### Check sending quota:

```bash
aws ses get-send-quota --region ap-southeast-2
```

Output:
```json
{
  "Max24HourSend": 200.0,        // Max emails per 24 hours
  "MaxSendRate": 1.0,            // Max emails per second
  "SentLast24Hours": 5.0         // Sent in last 24 hours
}
```

### List verified email identities:

```bash
aws ses list-identities --region ap-southeast-2
```

---

## Testing After Fixes

### Test 1: Verify Domain is Working

```bash
aws ses get-identity-verification-attributes \
  --identities mvp-dev.4cl-tech.com.au \
  --region ap-southeast-2
```

Should show:
```json
{
  "VerificationAttributes": {
    "mvp-dev.4cl-tech.com.au": {
      "VerificationStatus": "Success"
    }
  }
}
```

### Test 2: Send a Test Email

```bash
aws ses send-email \
  --from noreply@mvp-dev.4cl-tech.com.au \
  --destination "ToAddresses=sivala84@gmail.com" \
  --message "Subject={Data=Test Email},Body={Text={Data=This is a test}}" \
  --region ap-southeast-2
```

### Test 3: Use the API

```bash
curl -X POST https://mvp-dev.4cl-tech.com.au/v2/organization/send-invitations \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "emailAddresses": ["sivala84@gmail.com"],
    "customMessage": "Test invitation"
  }'
```

---

## Best Practices

### For Development:
1. ✅ Verify your own email addresses for testing
2. ✅ Use sandbox mode to avoid accidental email sends
3. ✅ Test with a small list of verified emails

### For Production:
1. ✅ Request production access before launch
2. ✅ Implement proper error handling for bounces
3. ✅ Monitor SES metrics in CloudWatch
4. ✅ Set up SNS notifications for bounces/complaints
5. ✅ Maintain good sender reputation

---

## Additional DNS Configuration

Even with production access, ensure proper DNS records:

### 1. DKIM Records (for deliverability)

After verifying your domain, AWS provides DKIM tokens. Add them to Route 53:

```bash
# Get DKIM tokens
aws ses verify-domain-dkim \
  --domain mvp-dev.4cl-tech.com.au \
  --region ap-southeast-2
```

Add the returned CNAME records to Route 53.

### 2. SPF Record

Add this TXT record to your Route 53 hosted zone:

```
Name: mvp-dev.4cl-tech.com.au
Type: TXT
Value: v=spf1 include:amazonses.com ~all
TTL: 300
```

### 3. DMARC Record (Optional but recommended)

```
Name: _dmarc.mvp-dev.4cl-tech.com.au
Type: TXT
Value: v=DMARC1; p=quarantine; rua=mailto:dmarc@4cl-tech.com.au
TTL: 300
```

---

## Monitoring and Debugging

### CloudWatch Logs

Check Lambda logs for detailed errors:

```bash
aws logs tail /aws/lambda/SendInvitationsLambda-dev --follow
```

### SES Sending Statistics

```bash
# Get sending statistics
aws ses get-send-statistics --region ap-southeast-2

# Check bounce notifications
aws sns list-subscriptions --region ap-southeast-2
```

### Common Issues:

| Error | Cause | Solution |
|-------|-------|----------|
| "Email address is not verified" | Sandbox mode | Verify email or request production access |
| "Daily sending quota exceeded" | Hit sending limit | Request quota increase or wait 24 hours |
| "Maximum sending rate exceeded" | Sending too fast | Implement rate limiting/backoff |
| "Domain not verified" | DNS records not set | Add verification TXT record to Route 53 |

---

## Quick Reference Commands

```bash
# Verify email
aws ses verify-email-identity --email-address EMAIL --region ap-southeast-2

# Check verification
aws ses get-identity-verification-attributes --identities EMAIL --region ap-southeast-2

# Request production access
aws sesv2 put-account-details --production-access-enabled --region ap-southeast-2

# Check if in sandbox
aws sesv2 get-account --region ap-southeast-2

# View sending quota
aws ses get-send-quota --region ap-southeast-2

# List verified identities
aws ses list-identities --region ap-southeast-2
```

---

## Support

For issues:
- AWS SES Documentation: https://docs.aws.amazon.com/ses/
- AWS Support: Open a ticket in AWS Console
- Check CloudWatch logs for detailed error messages
