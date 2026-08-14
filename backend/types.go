package main

import (
	"time"
)

type EmailProviderType string

const (
	ProviderYandexPostbox EmailProviderType = "yandex_postbox"
	ProviderResend        EmailProviderType = "resend"
	ProviderSendGrid      EmailProviderType = "sendgrid"
	ProviderMailgun       EmailProviderType = "mailgun"
	ProviderPostmark      EmailProviderType = "postmark"
	ProviderBrevo         EmailProviderType = "brevo"
	ProviderWebhook       EmailProviderType = "webhook"
)

// SMTPSettings holds configuration for email delivery.
type SMTPSettings struct {
	ID                string            `json:"id"`
	ProjectID         string            `json:"project_id,omitempty"`
	Provider          EmailProviderType `json:"provider"`
	Endpoint          string            `json:"endpoint"`
	APIKey            string            `json:"api_key,omitempty"`
	FromEmail         string            `json:"from_email"`
	FromName          string            `json:"from_name"`
	NotifyOnAssigned  bool              `json:"notify_on_assigned"`
	NotifyOnMentioned bool              `json:"notify_on_mentioned"`
	NotifyOnUpdate    bool              `json:"notify_on_update"`
	CreatedAt         string            `json:"created_at"`
	UpdatedAt         string            `json:"updated_at"`
}

// UpdateSettingsInput carries update fields for settings.
type UpdateSettingsInput struct {
	Provider          *EmailProviderType `json:"provider"`
	Endpoint          *string            `json:"endpoint"`
	APIKey            *string            `json:"api_key"`
	FromEmail         *string            `json:"from_email"`
	FromName          *string            `json:"from_name"`
	NotifyOnAssigned  *bool              `json:"notify_on_assigned"`
	NotifyOnMentioned *bool              `json:"notify_on_mentioned"`
	NotifyOnUpdate    *bool              `json:"notify_on_update"`
}

// EmailLog records every processed notification email attempt.
type EmailLog struct {
	ID               string `json:"id"`
	ProjectID        string `json:"project_id,omitempty"`
	RecipientUserID  string `json:"recipient_user_id"`
	RecipientEmail   string `json:"recipient_email"`
	NotificationType string `json:"notification_type"`
	Subject          string `json:"subject"`
	Status           string `json:"status"` // "sent", "failed", "skipped"
	ErrorMessage     string `json:"error_message,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// UserEmailOverride maps a user ID to an explicit email.
type UserEmailOverride struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// SaveOverrideInput carries data to create/update an override.
type SaveOverrideInput struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

// TestEmailInput carries test email dispatch parameters.
type TestEmailInput struct {
	ToEmail   string             `json:"to_email"`
	Provider  *EmailProviderType `json:"provider,omitempty"`
	Endpoint  *string            `json:"endpoint,omitempty"`
	APIKey    *string            `json:"api_key,omitempty"`
	FromEmail *string            `json:"from_email,omitempty"`
	FromName  *string            `json:"from_name,omitempty"`
	Subject   string             `json:"subject,omitempty"`
	Message   string             `json:"message,omitempty"`
}

// NotificationEventPayload is the data shape of the notification.created event.
type NotificationEventPayload struct {
	ID              string `json:"id"`
	RecipientUserID string `json:"recipient_user_id"`
	ActorMemberID   string `json:"actor_member_id,omitempty"`
	Type            string `json:"type"` // "assigned", "mentioned", etc.
	TaskID          string `json:"task_id,omitempty"`
	ProjectID       string `json:"project_id,omitempty"`
	CreatedAt       string `json:"created_at"`
}

func nowStr() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
