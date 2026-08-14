package main

import (
	"fmt"
	"time"

	plugin "github.com/Paca-AI/plugin-sdk-go"
)

func timeNowNano() int64 {
	return time.Now().UnixNano()
}

// emailPlugin implements plugin.Plugin.
type emailPlugin struct {
	db     *plugin.DB
	cache  *plugin.Cache
	log    *plugin.Logger
	sender MailSender
}

// Init registers routes and event subscribers.
func (p *emailPlugin) Init(ctx *plugin.Context) error {
	p.db = ctx.DB()
	p.cache = ctx.Cache()
	p.log = ctx.Log()
	if p.sender == nil {
		p.sender = &DefaultMailSender{}
	}

	// ── Project-scope routes ───────────────────────────────────────────────
	ctx.Route("GET", "/projects/:projectId/email-notifications/settings", p.getProjectSettings)
	ctx.Route("PATCH", "/projects/:projectId/email-notifications/settings", p.updateProjectSettings)
	ctx.Route("GET", "/projects/:projectId/email-notifications/logs", p.getProjectLogs)
	ctx.Route("POST", "/projects/:projectId/email-notifications/test", p.testProjectEmail)

	// ── Admin-scope routes ──────────────────────────────────────────────────
	ctx.Route("GET", "/email-notifications/admin-settings", p.getAdminSettings)
	ctx.Route("PATCH", "/email-notifications/admin-settings", p.updateAdminSettings)
	ctx.Route("GET", "/email-notifications/admin-logs", p.getAdminLogs)
	ctx.Route("POST", "/email-notifications/admin-test", p.testAdminEmail)
	ctx.Route("GET", "/email-notifications/admin-overrides", p.listAdminOverrides)
	ctx.Route("POST", "/email-notifications/admin-overrides", p.saveAdminOverride)
	ctx.Route("DELETE", "/email-notifications/admin-overrides/:userId", p.deleteAdminOverride)

	// ── Event subscriptions ────────────────────────────────────────────────
	ctx.On("notification.created", p.onNotificationCreated)

	return nil
}

func (p *emailPlugin) Shutdown() {}

// onNotificationCreated handles incoming in-app notification events.
func (p *emailPlugin) onNotificationCreated(evt *plugin.Event) {
	payload, err := plugin.JSONPayload[NotificationEventPayload](evt)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to decode notification.created payload: %v", err))
		return
	}

	if payload.RecipientUserID == "" {
		p.log.Debug("notification has no recipient_user_id, ignoring")
		return
	}

	// Resolve email: username (if valid email) or explicit override
	recipientEmail, recipientName, err := p.resolveRecipientEmail(payload.RecipientUserID)
	if err != nil {
		p.log.Warn(fmt.Sprintf("could not resolve recipient for notification user %s: %v", payload.RecipientUserID, err))
		return
	}

	// If username does not look like a valid email (and no override), skip silently
	if recipientEmail == "" || !IsValidEmail(recipientEmail) {
		p.log.Debug(fmt.Sprintf("recipient username is not a valid email, skipping email duplication for user %s", payload.RecipientUserID))
		return
	}

	// Load settings (project-specific or global)
	settings, err := p.loadSettings("project", payload.ProjectID)
	if err != nil || !settings.Enabled || settings.Host == "" {
		settings, err = p.loadSettings("global", "")
	}
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to load settings for notification email: %v", err))
		return
	}

	if !settings.Enabled {
		return
	}

	// Check type filters
	if payload.Type == "assigned" && !settings.NotifyOnAssigned {
		return
	}
	if payload.Type == "mentioned" && !settings.NotifyOnMentioned {
		return
	}

	// Fetch Project Info
	projectName := "Project"
	taskPrefix := ""
	if payload.ProjectID != "" {
		projRes, _ := p.db.Query(`SELECT name, task_id_prefix FROM projects WHERE id = $1`, payload.ProjectID)
		if len(projRes.Rows) > 0 {
			sc := newRowScanner(projRes.Columns, projRes.Rows[0])
			if name := sc.str("name"); name != "" {
				projectName = name
			}
			taskPrefix = sc.str("task_id_prefix")
		}
	}

	// Fetch Task Info if task_id is present
	taskNumber := 0
	taskTitle := ""
	if payload.TaskID != "" {
		taskRes, _ := p.db.Query(`SELECT task_number, title FROM tasks WHERE id = $1`, payload.TaskID)
		if len(taskRes.Rows) > 0 {
			sc := newRowScanner(taskRes.Columns, taskRes.Rows[0])
			taskNumber = sc.intVal("task_number", 0)
			taskTitle = sc.str("title")
		}
	}

	// Fetch Actor Info if actor_member_id is present
	actorName := ""
	if payload.ActorMemberID != "" {
		pmRes, _ := p.db.Query(`SELECT user_id FROM project_members WHERE id = $1`, payload.ActorMemberID)
		if len(pmRes.Rows) > 0 {
			pmSc := newRowScanner(pmRes.Columns, pmRes.Rows[0])
			actorUserID := pmSc.str("user_id")
			if actorUserID != "" {
				uRes, _ := p.db.Query(`SELECT full_name, username FROM users WHERE id = $1`, actorUserID)
				if len(uRes.Rows) > 0 {
					uSc := newRowScanner(uRes.Columns, uRes.Rows[0])
					actorName = uSc.str("full_name")
					if actorName == "" {
						actorName = uSc.str("username")
					}
				}
			}
		}
	}

	// Format Email
	formatted := FormatNotificationEmail(
		payload.Type,
		recipientName,
		actorName,
		projectName,
		taskPrefix,
		taskNumber,
		taskTitle,
		payload.CreatedAt,
	)

	// Send Email
	sendErr := p.sender.SendEmail(settings, recipientEmail, formatted.Subject, formatted.BodyText, formatted.BodyHTML)

	status := "sent"
	errMsg := ""
	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
		p.log.Warn(fmt.Sprintf("failed to send notification email to %s: %v", recipientEmail, sendErr))
	} else {
		p.log.Info(fmt.Sprintf("notification email sent successfully to %s for %s", recipientEmail, payload.Type))
	}

	// Record in audit log
	_ = p.recordLog(EmailLog{
		ID:               fmt.Sprintf("log-%d", timeNowNano()),
		ProjectID:        payload.ProjectID,
		NotificationID:   payload.ID,
		RecipientUserID:  payload.RecipientUserID,
		RecipientEmail:   recipientEmail,
		NotificationType: payload.Type,
		Subject:          formatted.Subject,
		BodyText:         formatted.BodyText,
		Status:           status,
		ErrorMessage:     errMsg,
		CreatedAt:        nowStr(),
	})
}
