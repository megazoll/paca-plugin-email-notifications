package main

import (
	"fmt"
)

// MailSender defines the interface for delivering email messages.
type MailSender interface {
	SendEmail(settings *SMTPSettings, to string, subject, bodyText, bodyHTML string) error
}

// DefaultMailSender implements MailSender using host SMTP and fallback mechanisms.
type DefaultMailSender struct{}

func (s *DefaultMailSender) SendEmail(settings *SMTPSettings, to string, subject, bodyText, bodyHTML string) error {
	if settings != nil && !settings.Enabled {
		return fmt.Errorf("email notifications are disabled in settings")
	}

	if settings != nil && settings.Host != "" {
		fromEmail := settings.FromEmail
		if fromEmail == "" {
			fromEmail = "notifications@paca.local"
		}
		fromName := settings.FromName
		if fromName == "" {
			fromName = "Paca"
		}

		payload := SMTPHostPayload{
			Host:     settings.Host,
			Port:     settings.Port,
			Username: settings.Username,
			Password: settings.Password,
			From:     fromEmail,
			FromName: fromName,
			To:       []string{to},
			Subject:  subject,
			BodyText: bodyText,
			BodyHTML: bodyHTML,
			Security: settings.Security,
		}
		return sendViaHostSMTP(payload)
	}

	// When no SMTP server is configured, log-only mode succeeds without error
	return nil
}
