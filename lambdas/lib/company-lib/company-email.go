package Companylib

import (
	"context"
	"fmt"
	"log"

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
	return &EmailService{
		ctx:              ctx,
		sesClient:        sesClient,
		logger:           logger,
		DefaultFromEmail: "noreply@gomovo.com", // Configure as needed
		DefaultFromName:  "Gomovo Hub",
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
