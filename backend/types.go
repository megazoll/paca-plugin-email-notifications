package main

import (
	"time"
)

// SMTPSettings holds configuration for email delivery.
type SMTPSettings struct {
	ID                string `json:"id"`
	Scope             string `json:"scope"` // "global" or "project"
	ProjectID         string `json:"project_id,omitempty"`
	Enabled           bool   `json:"enabled"`
	Host              string `json:"host"`
	Port              int    `json:"port"`
	Username          string `json:"username"`
	Password          string `json:"password,omitempty"` // Omitted in some responses or masked
	FromEmail         string `json:"from_email"`
	FromName          string `json:"from_name"`
	Security          string `json:"security"` // "tls", "starttls", "none"
	WebhookURL        string `json:"webhook_url"`
	WebhookAPIKey     string `json:"webhook_api_key,omitempty"`
	NotifyOnAssigned  bool   `json:"notify_on_assigned"`
	NotifyOnMentioned bool   `json:"notify_on_mentioned"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// UpdateSettingsInput carries update fields for SMTP settings.
type UpdateSettingsInput struct {
	Enabled           *bool   `json:"enabled"`
	Host              *string `json:"host"`
	Port              *int    `json:"port"`
	Username          *string `json:"username"`
	Password          *string `json:"password"`
	FromEmail         *string `json:"from_email"`
	FromName          *string `json:"from_name"`
	Security          *string `json:"security"`
	WebhookURL        *string `json:"webhook_url"`
	WebhookAPIKey     *string `json:"webhook_api_key"`
	NotifyOnAssigned  *bool   `json:"notify_on_assigned"`
	NotifyOnMentioned *bool   `json:"notify_on_mentioned"`
}

// EmailLog records every processed notification email attempt.
type EmailLog struct {
	ID               string `json:"id"`
	ProjectID        string `json:"project_id,omitempty"`
	NotificationID   string `json:"notification_id,omitempty"`
	RecipientUserID  string `json:"recipient_user_id"`
	RecipientEmail   string `json:"recipient_email"`
	NotificationType string `json:"notification_type"`
	Subject          string `json:"subject"`
	BodyText         string `json:"body_text"`
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
	ToEmail string `json:"to_email"`
	Subject string `json:"subject,omitempty"`
	Message string `json:"message,omitempty"`
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

// SMTPHostPayload is the payload passed to the host's paca.smtp_send bridge.
type SMTPHostPayload struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	FromName string   `json:"from_name"`
	To       []string `json:"to"`
	Subject  string   `json:"subject"`
	BodyText string   `json:"body_text"`
	BodyHTML string   `json:"body_html"`
	Security string   `json:"security"`
}

// SMTPHostResult is the result returned by paca.smtp_send bridge.
type SMTPHostResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func nowStr() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
