package Companylib

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"
)

// EmailInput represents the input structure for sending emails
type EmailInput struct {
	ToEmails  []string `json:"toEmails" validate:"required,min=1"`
	CcEmails  []string `json:"ccEmails,omitempty"`
	BccEmails []string `json:"bccEmails,omitempty"`
	Subject   string   `json:"subject" validate:"required"`
	HtmlBody  string   `json:"htmlBody,omitempty"`
	TextBody  string   `json:"textBody,omitempty"`
	FromEmail string   `json:"fromEmail" validate:"required,email"`
	FromName  string   `json:"fromName,omitempty"`
}

// EmailService handles email operations using AWS SES
type EmailService struct {
	ctx       context.Context
	sesClient *ses.Client
	logger    *log.Logger

	// Configuration
	DefaultFromEmail string
	DefaultFromName  string
}

// CreateEmailService creates a new email service
func CreateEmailService(ctx context.Context, sesClient *ses.Client, logger *log.Logger) *EmailService {
	// Get default email settings from environment variables
	defaultFromEmail := os.Getenv("DEFAULT_FROM_EMAIL")
	if defaultFromEmail == "" {
		defaultFromEmail = "noreply@mvp-dev.4cl-tech.com.au" // Fallback default
	}

	defaultFromName := os.Getenv("DEFAULT_FROM_NAME")
	if defaultFromName == "" {
		defaultFromName = "4CL Tech" // Fallback default
	}

	return &EmailService{
		ctx:              ctx,
		sesClient:        sesClient,
		logger:           logger,
		DefaultFromEmail: defaultFromEmail,
		DefaultFromName:  defaultFromName,
	}
}

// SendEmail sends an email using AWS SES
func (svc *EmailService) SendEmail(input EmailInput) error {
	// Validate input
	if len(input.ToEmails) == 0 {
		return fmt.Errorf("at least one recipient email is required")
	}

	if input.Subject == "" {
		return fmt.Errorf("email subject is required")
	}

	if input.HtmlBody == "" && input.TextBody == "" {
		return fmt.Errorf("either HTML body or text body is required")
	}

	// Use defaults if not provided
	fromEmail := input.FromEmail
	if fromEmail == "" {
		fromEmail = svc.DefaultFromEmail
	}

	fromName := input.FromName
	if fromName == "" {
		fromName = svc.DefaultFromName
	}

	// Build the from address
	var fromAddress string
	if fromName != "" {
		fromAddress = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	} else {
		fromAddress = fromEmail
	}

	// Prepare destination
	destination := &types.Destination{
		ToAddresses: input.ToEmails,
	}

	if len(input.CcEmails) > 0 {
		destination.CcAddresses = input.CcEmails
	}

	if len(input.BccEmails) > 0 {
		destination.BccAddresses = input.BccEmails
	}

	// Prepare message content
	content := &types.Content{
		Data:    aws.String(input.Subject),
		Charset: aws.String("UTF-8"),
	}

	message := &types.Message{
		Subject: content,
	}

	// Build body
	body := &types.Body{}

	if input.HtmlBody != "" {
		body.Html = &types.Content{
			Data:    aws.String(input.HtmlBody),
			Charset: aws.String("UTF-8"),
		}
	}

	if input.TextBody != "" {
		body.Text = &types.Content{
			Data:    aws.String(input.TextBody),
			Charset: aws.String("UTF-8"),
		}
	}

	message.Body = body

	// Send email
	sendInput := &ses.SendEmailInput{
		Source:      aws.String(fromAddress),
		Destination: destination,
		Message:     message,
	}

	svc.logger.Printf("Sending email to %v with subject: %s", input.ToEmails, input.Subject)

	result, err := svc.sesClient.SendEmail(svc.ctx, sendInput)
	if err != nil {
		svc.logger.Printf("Failed to send email: %v", err)
		return fmt.Errorf("failed to send email: %w", err)
	}

	svc.logger.Printf("Successfully sent email. MessageId: %s", aws.ToString(result.MessageId))
	return nil
}

// SendBulkEmail sends the same email to multiple recipients individually
// This is useful for personalized emails where you want separate delivery
func (svc *EmailService) SendBulkEmail(inputs []EmailInput) []error {
	errors := make([]error, len(inputs))

	for i, input := range inputs {
		err := svc.SendEmail(input)
		if err != nil {
			svc.logger.Printf("Failed to send bulk email #%d: %v", i, err)
			errors[i] = err
		}
	}

	return errors
}

// SendTemplateEmail sends an email using AWS SES templates
func (svc *EmailService) SendTemplateEmail(templateName string, templateData interface{}, input EmailInput) error {
	// This would require SES template functionality
	// For now, we'll just use the regular SendEmail method
	return svc.SendEmail(input)
}

// ValidateEmailAddress validates if an email address is deliverable using SES
func (svc *EmailService) ValidateEmailAddress(email string) error {
	input := &ses.GetIdentityVerificationAttributesInput{
		Identities: []string{email},
	}

	result, err := svc.sesClient.GetIdentityVerificationAttributes(svc.ctx, input)
	if err != nil {
		svc.logger.Printf("Failed to validate email %s: %v", email, err)
		return fmt.Errorf("failed to validate email: %w", err)
	}

	if attrs, exists := result.VerificationAttributes[email]; exists {
		if attrs.VerificationStatus != types.VerificationStatusSuccess {
			return fmt.Errorf("email %s is not verified for sending", email)
		}
	}

	return nil
}

// GetSendingQuota returns the current sending quota for the account
func (svc *EmailService) GetSendingQuota() (*ses.GetSendQuotaOutput, error) {
	return svc.sesClient.GetSendQuota(svc.ctx, &ses.GetSendQuotaInput{})
}

// GetSendingStatistics returns sending statistics
func (svc *EmailService) GetSendingStatistics() (*ses.GetSendStatisticsOutput, error) {
	return svc.sesClient.GetSendStatistics(svc.ctx, &ses.GetSendStatisticsInput{})
}

// InvitationEmailInput represents the input for sending invitation emails
type InvitationEmailInput struct {
	EmailAddresses   []string `json:"emailAddresses" validate:"required,min=1"`
	OrganizationName string   `json:"organizationName"`
	TeamName         string   `json:"teamName,omitempty"`
	InviterName      string   `json:"inviterName"`
	InvitationLink   string   `json:"invitationLink"`
	CustomMessage    string   `json:"customMessage,omitempty"`
}

// InvitationEmailResult represents the result of sending an invitation email
type InvitationEmailResult struct {
	Email   string `json:"email"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// SendInvitationEmails sends invitation emails to multiple recipients
// Returns a slice of results indicating success/failure for each email
func (svc *EmailService) SendInvitationEmails(input InvitationEmailInput) ([]InvitationEmailResult, error) {
	svc.logger.Printf("Sending invitation emails to %d recipients", len(input.EmailAddresses))

	// Validate input
	if len(input.EmailAddresses) == 0 {
		return nil, fmt.Errorf("at least one email address is required")
	}

	// Prepare results slice
	results := make([]InvitationEmailResult, len(input.EmailAddresses))

	// Build HTML email template
	htmlBody := svc.buildInvitationEmailHTML(input)
	textBody := svc.buildInvitationEmailText(input)

	// Create subject
	subject := "You're invited to join our organization"
	if input.OrganizationName != "" {
		subject = fmt.Sprintf("EGS Hub: You're invited to join %s", input.OrganizationName)
	}

	// Send email to each recipient
	for i, email := range input.EmailAddresses {
		results[i].Email = email

		emailInput := EmailInput{
			ToEmails: []string{email},
			Subject:  subject,
			HtmlBody: htmlBody,
			TextBody: textBody,
		}

		err := svc.SendEmail(emailInput)
		if err != nil {
			svc.logger.Printf("Failed to send invitation to %s: %v", email, err)
			results[i].Success = false
			results[i].Error = err.Error()
		} else {
			svc.logger.Printf("Successfully sent invitation to %s", email)
			results[i].Success = true
		}
	}

	return results, nil
}

// buildInvitationEmailHTML builds the HTML body for the invitation email
func (svc *EmailService) buildInvitationEmailHTML(input InvitationEmailInput) string {
	organizationName := input.OrganizationName
	if organizationName == "" {
		organizationName = "our organization"
	}

	invitationLink := input.InvitationLink
	if invitationLink == "" {
		invitationLink = "#"
	}

	// Build invitation message with team name if provided
	invitationMessage := ""
	if input.TeamName != "" {
		invitationMessage = fmt.Sprintf("You are invited to join <strong>%s</strong> team in <strong>%s</strong>.", input.TeamName, organizationName)
	} else {
		invitationMessage = fmt.Sprintf("You are invited to join <strong>%s</strong>.", organizationName)
	}

	customMessage := ""
	if input.CustomMessage != "" {
		customMessage = fmt.Sprintf(`
			<div style="background-color: #f8f9fa; padding: 15px; border-left: 4px solid #007bff; margin: 20px 0;">
				<p style="margin: 0; color: #495057; font-style: italic;">"%s"</p>
			</div>
		`, input.CustomMessage)
	}

	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Invitation to Join</title>
</head>
<body style="margin: 0; padding: 0; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #f4f4f4;">
	<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
		<tr>
			<td align="center" style="padding: 40px 20px;">
				<table role="presentation" width="600" cellspacing="0" cellpadding="0" border="0" style="background-color: #ffffff; border-radius: 8px; box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);">
					<!-- Header -->
					<tr>
						<td style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 40px; text-align: center; border-radius: 8px 8px 0 0;">
							<h1 style="margin: 0; color: #ffffff; font-size: 28px; font-weight: 600;">You're Invited!</h1>
						</td>
					</tr>
					
					<!-- Content -->
					<tr>
						<td style="padding: 40px;">
							<p style="margin: 0 0 20px; color: #333333; font-size: 16px; line-height: 1.6;">
								Hello,
							</p>
							
							<p style="margin: 0 0 20px; color: #333333; font-size: 16px; line-height: 1.6;">
								%s
							</p>
							
							%s
							
							<p style="margin: 0 0 30px; color: #333333; font-size: 16px; line-height: 1.6;">
								Click the button below to accept the invitation and get started:
							</p>
							
							<!-- CTA Button -->
							<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" border="0">
								<tr>
									<td align="center">
										<a href="%s" style="display: inline-block; padding: 16px 40px; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: #ffffff; text-decoration: none; border-radius: 6px; font-size: 16px; font-weight: 600; box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);">
											Accept Invitation
										</a>
									</td>
								</tr>
							</table>
							
							<p style="margin: 30px 0 0; color: #666666; font-size: 14px; line-height: 1.6;">
								If the button doesn't work, copy and paste this link into your browser:<br>
								<a href="%s" style="color: #667eea; word-break: break-all;">%s</a>
							</p>
						</td>
					</tr>
					
					<!-- Footer -->
					<tr>
						<td style="background-color: #f8f9fa; padding: 30px; text-align: center; border-radius: 0 0 8px 8px;">
							<p style="margin: 0; color: #999999; font-size: 12px;">
								If you didn't expect this invitation, you can safely ignore this email.
							</p>
						</td>
					</tr>
				</table>
				
				<!-- Footer Text -->
				<p style="margin: 20px 0 0; color: #999999; font-size: 12px; text-align: center;">
					© 2026 Gomovo Hub. All rights reserved.
				</p>
			</td>
		</tr>
	</table>
</body>
</html>
	`, invitationMessage, customMessage, invitationLink, invitationLink, invitationLink)
}

// buildInvitationEmailText builds the plain text body for the invitation email
func (svc *EmailService) buildInvitationEmailText(input InvitationEmailInput) string {
	organizationName := input.OrganizationName
	if organizationName == "" {
		organizationName = "our organization"
	}

	invitationLink := input.InvitationLink
	if invitationLink == "" {
		invitationLink = "#"
	}

	// Build invitation message with team name if provided
	invitationMessage := ""
	if input.TeamName != "" {
		invitationMessage = fmt.Sprintf("You are invited to join %s team in %s.", input.TeamName, organizationName)
	} else {
		invitationMessage = fmt.Sprintf("You are invited to join %s.", organizationName)
	}

	customMessage := ""
	if input.CustomMessage != "" {
		customMessage = fmt.Sprintf("\n\nPersonal message:\n\"%s\"\n", input.CustomMessage)
	}

	return fmt.Sprintf(`You're Invited!

Hello,

%s
%s
To accept the invitation, visit:
%s

If you didn't expect this invitation, you can safely ignore this email.

© 2026 Gomovo Hub. All rights reserved.
`, invitationMessage, customMessage, invitationLink)
}
