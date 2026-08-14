package main

import (
	"fmt"
	"html"
	"net/mail"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail checks if a string looks like a valid email address.
func IsValidEmail(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 254 {
		return false
	}
	if !emailRegex.MatchString(s) {
		return false
	}
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}

// FormattedEmail holds the subject and bodies for an outgoing email.
type FormattedEmail struct {
	Subject  string
	BodyText string
	BodyHTML string
}

// FormatNotificationEmail creates a rich HTML and text email based on notification context.
func FormatNotificationEmail(
	notifType string,
	recipientName string,
	actorName string,
	projectName string,
	taskPrefix string,
	taskNumber int,
	taskTitle string,
	createdAt string,
) FormattedEmail {
	taskLabel := ""
	if taskNumber > 0 {
		if taskPrefix != "" {
			taskLabel = fmt.Sprintf("%s-%d", taskPrefix, taskNumber)
		} else {
			taskLabel = fmt.Sprintf("#%d", taskNumber)
		}
	}

	var subject, actionDescription string
	switch notifType {
	case "assigned":
		if taskLabel != "" {
			subject = fmt.Sprintf("[Paca] [%s] Assigned to %s: %s", projectName, taskLabel, taskTitle)
			actionDescription = fmt.Sprintf("You have been assigned to task <strong>%s: %s</strong>", html.EscapeString(taskLabel), html.EscapeString(taskTitle))
		} else {
			subject = fmt.Sprintf("[Paca] [%s] New assignment in %s", projectName, projectName)
			actionDescription = fmt.Sprintf("You have been assigned to a task in <strong>%s</strong>", html.EscapeString(projectName))
		}
	case "mentioned":
		if taskLabel != "" {
			subject = fmt.Sprintf("[Paca] [%s] Mentioned in %s: %s", projectName, taskLabel, taskTitle)
			actionDescription = fmt.Sprintf("You were mentioned by <strong>%s</strong> in task <strong>%s: %s</strong>", html.EscapeString(actorName), html.EscapeString(taskLabel), html.EscapeString(taskTitle))
		} else {
			subject = fmt.Sprintf("[Paca] [%s] You were mentioned in %s", projectName, projectName)
			actionDescription = fmt.Sprintf("You were mentioned by <strong>%s</strong> in <strong>%s</strong>", html.EscapeString(actorName), html.EscapeString(projectName))
		}
	default:
		subject = fmt.Sprintf("[Paca] [%s] Notification: %s", projectName, notifType)
		actionDescription = fmt.Sprintf("You received a new notification (<strong>%s</strong>) in <strong>%s</strong>", html.EscapeString(notifType), html.EscapeString(projectName))
	}

	actorDisplay := "System"
	if actorName != "" {
		actorDisplay = actorName
	}

	// Plain text version
	var textBuilder strings.Builder
	textBuilder.WriteString(fmt.Sprintf("Hello %s,\n\n", recipientName))
	switch notifType {
	case "assigned":
		textBuilder.WriteString(fmt.Sprintf("You have been assigned to task %s: %s in project \"%s\" by %s.\n\n", taskLabel, taskTitle, projectName, actorDisplay))
	case "mentioned":
		textBuilder.WriteString(fmt.Sprintf("You were mentioned by %s in task %s: %s (project: \"%s\").\n\n", actorDisplay, taskLabel, taskTitle, projectName))
	default:
		textBuilder.WriteString(fmt.Sprintf("You received a new %s notification in project \"%s\".\n\n", notifType, projectName))
	}
	if createdAt != "" {
		textBuilder.WriteString(fmt.Sprintf("Time: %s\n\n", createdAt))
	}
	textBuilder.WriteString("--\nPaca AI Notifications")

	// HTML version
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif; background-color: #0f172a; color: #f8fafc; margin: 0; padding: 24px; }
.card { max-width: 560px; margin: 0 auto; background: #1e293b; border: 1px solid #334155; border-radius: 12px; overflow: hidden; box-shadow: 0 10px 25px -5px rgba(0,0,0,0.3); }
.header { background: linear-gradient(135deg, #6366f1 0%%, #8b5cf6 100%%); padding: 20px 24px; color: #ffffff; font-weight: 700; font-size: 18px; }
.content { padding: 24px; color: #cbd5e1; font-size: 15px; line-height: 1.6; }
.greeting { font-size: 16px; font-weight: 600; color: #f8fafc; margin-bottom: 12px; }
.desc { margin-bottom: 20px; }
.details { background: #0f172a; border-radius: 8px; padding: 16px; margin-bottom: 20px; border: 1px solid #334155; }
.detail-row { display: flex; justify-content: space-between; margin-bottom: 8px; font-size: 13px; }
.detail-row:last-child { margin-bottom: 0; }
.detail-label { color: #94a3b8; }
.detail-value { font-weight: 500; color: #f1f5f9; }
.footer { padding: 16px 24px; background: #0f172a; border-top: 1px solid #334155; font-size: 12px; color: #64748b; text-align: center; }
</style>
</head>
<body>
<div class="card">
  <div class="header">Paca Notification</div>
  <div class="content">
    <div class="greeting">Hello %s,</div>
    <div class="desc">%s</div>
    <div class="details">
      <table style="width: 100%%; font-size: 14px; color: #cbd5e1; border-collapse: collapse;">
        <tr><td style="padding: 4px 0; color: #94a3b8; width: 30%%;">Project:</td><td style="padding: 4px 0; font-weight: 600; color: #f8fafc;">%s</td></tr>
        %s
        <tr><td style="padding: 4px 0; color: #94a3b8;">Triggered by:</td><td style="padding: 4px 0; color: #f8fafc;">%s</td></tr>
        %s
      </table>
    </div>
  </div>
  <div class="footer">
    Sent automatically by Paca Email Notifications plugin.
  </div>
</div>
</body>
</html>`,
		html.EscapeString(recipientName),
		actionDescription,
		html.EscapeString(projectName),
		func() string {
			if taskLabel != "" {
				return fmt.Sprintf(`<tr><td style="padding: 4px 0; color: #94a3b8;">Task:</td><td style="padding: 4px 0; font-weight: 600; color: #60a5fa;">%s (%s)</td></tr>`, html.EscapeString(taskLabel), html.EscapeString(taskTitle))
			}
			return ""
		}(),
		html.EscapeString(actorDisplay),
		func() string {
			if createdAt != "" {
				return fmt.Sprintf(`<tr><td style="padding: 4px 0; color: #94a3b8;">Time:</td><td style="padding: 4px 0; color: #94a3b8;">%s</td></tr>`, html.EscapeString(createdAt))
			}
			return ""
		}(),
	)

	return FormattedEmail{
		Subject:  subject,
		BodyText: textBuilder.String(),
		BodyHTML: htmlBody,
	}
}

// FormatTestEmail creates subject and body for test email verification.
func FormatTestEmail(toEmail string) FormattedEmail {
	subject := "[Paca] Test Email Notification"
	bodyText := fmt.Sprintf("Hello,\n\nThis is a test email sent from the Paca Email Notifications plugin.\nIf you received this message, your SMTP configuration is working correctly!\n\nRecipient: %s\n\n--\nPaca AI Notifications", toEmail)
	bodyHTML := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="utf-8">
<style>
body { font-family: sans-serif; background: #0f172a; color: #f8fafc; padding: 20px; }
.card { max-width: 500px; margin: 0 auto; background: #1e293b; border-radius: 8px; padding: 24px; border: 1px solid #334155; }
h2 { color: #22c55e; margin-top: 0; }
</style></head>
<body>
<div class="card">
  <h2>✓ Test Email Successful</h2>
  <p>This is a test email sent from your Paca Email Notifications plugin.</p>
  <p>Your email settings and SMTP delivery are working properly!</p>
  <p style="color: #94a3b8; font-size: 13px;">Recipient: %s</p>
</div>
</body></html>`, html.EscapeString(toEmail))

	return FormattedEmail{
		Subject:  subject,
		BodyText: bodyText,
		BodyHTML: bodyHTML,
	}
}
