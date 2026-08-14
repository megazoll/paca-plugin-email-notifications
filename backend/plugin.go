package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	plugin "github.com/Paca-AI/plugin-sdk-go"
	"github.com/google/uuid"
)

var plainMentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_.+-]+(?:@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)?|[a-zA-Z0-9_.-]+)`)

// emailPlugin implements plugin.Plugin.
type emailPlugin struct {
	db    *plugin.DB
	cache *plugin.Cache
	log   *plugin.Logger
}

// Init registers routes and event subscribers.
func (p *emailPlugin) Init(ctx *plugin.Context) error {
	p.db = ctx.DB()
	p.cache = ctx.Cache()
	p.log = ctx.Log()

	// ── Project-scope routes ───────────────────────────────────────────────
	ctx.Route("GET", "/projects/:projectId/email-notifications/settings", p.getProjectSettings)
	ctx.Route("PATCH", "/projects/:projectId/email-notifications/settings", p.updateProjectSettings)
	ctx.Route("GET", "/projects/:projectId/email-notifications/logs", p.getProjectLogs)
	ctx.Route("POST", "/projects/:projectId/email-notifications/test", p.testProjectEmail)

	// ── Admin-scope routes ──────────────────────────────────────────────────
	ctx.Route("GET", "/admin/email-notifications/settings", p.getAdminSettings)
	ctx.Route("PATCH", "/admin/email-notifications/settings", p.updateAdminSettings)
	ctx.Route("GET", "/admin/email-notifications/logs", p.getAdminLogs)
	ctx.Route("POST", "/admin/email-notifications/test", p.testAdminEmail)
	ctx.Route("GET", "/admin/email-notifications/overrides", p.listAdminOverrides)
	ctx.Route("PUT", "/admin/email-notifications/overrides/:userId", p.saveAdminOverride)
	ctx.Route("POST", "/admin/email-notifications/overrides/:userId", p.saveAdminOverride)
	ctx.Route("DELETE", "/admin/email-notifications/overrides/:userId", p.deleteAdminOverride)

	// ── Event subscriptions ────────────────────────────────────────────────
	ctx.On("task.created", p.onTaskActivityEvent)
	ctx.On("task.updated", p.onTaskActivityEvent)
	ctx.On("task.comment.added", p.onCommentActivityEvent)
	ctx.On("comment", p.onCommentActivityEvent)
	ctx.On("notification.created", p.onNotificationCreated)

	return nil
}

func (p *emailPlugin) Shutdown() {}

func (p *emailPlugin) getEffectiveSettings(projectID string) (*SMTPSettings, error) {
	if projectID != "" {
		settings, err := p.loadSettings("project", projectID)
		if err == nil && settings.APIKey != "" {
			return settings, nil
		}
	}
	return p.loadSettings("global", "")
}

// onTaskActivityEvent handles task.created and task.updated stream events.
func (p *emailPlugin) onTaskActivityEvent(evt *plugin.Event) {
	var payload struct {
		ID           string          `json:"id"`
		TaskID       string          `json:"task_id"`
		ProjectID    string          `json:"project_id"`
		ActivityType string          `json:"activity_type"`
		ActorID      string          `json:"actor_id,omitempty"`
		Content      json.RawMessage `json:"content,omitempty"`
		CreatedAt    string          `json:"created_at"`
	}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		p.log.Error(fmt.Sprintf("failed to decode task event payload: %v", err))
		return
	}

	taskID := strings.TrimSpace(payload.TaskID)
	projectID := strings.TrimSpace(payload.ProjectID)
	if taskID == "" {
		return
	}

	settings, err := p.getEffectiveSettings(projectID)
	if err != nil || settings.APIKey == "" {
		p.log.Debug("no email provider configured, skipping task notification")
		return
	}

	// Load task info
	taskNumber := 0
	taskTitle := ""
	taskRes, _ := p.db.Query(`SELECT task_number, title FROM tasks WHERE id = $1`, taskID)
	if len(taskRes.Rows) > 0 {
		sc := newRowScanner(taskRes.Columns, taskRes.Rows[0])
		taskNumber = sc.intVal("task_number", 0)
		taskTitle = sc.str("title")
	}

	// Load project info
	projectName := "Project"
	taskPrefix := ""
	if projectID != "" {
		projRes, _ := p.db.Query(`SELECT name, task_id_prefix FROM projects WHERE id = $1`, projectID)
		if len(projRes.Rows) > 0 {
			sc := newRowScanner(projRes.Columns, projRes.Rows[0])
			if name := sc.str("name"); name != "" {
				projectName = name
			}
			taskPrefix = sc.str("task_id_prefix")
		}
	}

	// Load actor info
	actorUserID := ""
	actorName := ""
	if payload.ActorID != "" {
		pmRes, _ := p.db.Query(`SELECT user_id FROM project_members WHERE id = $1`, payload.ActorID)
		if len(pmRes.Rows) > 0 {
			sc := newRowScanner(pmRes.Columns, pmRes.Rows[0])
			actorUserID = sc.str("user_id")
		} else {
			actorUserID = payload.ActorID
		}
		if actorUserID != "" {
			uRes, _ := p.db.Query(`SELECT full_name, username FROM users WHERE id = $1`, actorUserID)
			if len(uRes.Rows) > 0 {
				sc := newRowScanner(uRes.Columns, uRes.Rows[0])
				actorName = sc.str("full_name")
				if actorName == "" {
					actorName = sc.str("username")
				}
			}
		}
	}

	// Find assignees of the task (two-step lookup for maximum compatibility)
	assigneeUserIDs := make([]string, 0)
	assigneesRes, err := p.db.Query(`SELECT member_id FROM task_assignees WHERE task_id = $1`, taskID)
	if err == nil {
		for _, row := range assigneesRes.Rows {
			sc := newRowScanner(assigneesRes.Columns, row)
			memberID := sc.str("member_id")
			if memberID != "" {
				pmRes, _ := p.db.Query(`SELECT user_id FROM project_members WHERE id = $1`, memberID)
				if len(pmRes.Rows) > 0 {
					pmSc := newRowScanner(pmRes.Columns, pmRes.Rows[0])
					uid := pmSc.str("user_id")
					if uid != "" {
						assigneeUserIDs = append(assigneeUserIDs, uid)
					}
				}
			}
		}
	}

	activityType := payload.ActivityType
	if activityType == "" {
		activityType = evt.Topic
	}

	if activityType == "task.created" || activityType == "task.assigned" {
		if !settings.NotifyOnAssigned {
			return
		}
		for _, uid := range assigneeUserIDs {
			if uid == actorUserID {
				continue
			}
			p.dispatchEmailNotification(settings, projectID, taskID, uid, actorName, projectName, taskPrefix, taskNumber, taskTitle, "assigned", payload.CreatedAt)
		}
	} else if activityType == "task.updated" {
		contentStr := string(payload.Content)
		hasAssigneeChange := strings.Contains(contentStr, `"field":"assignee"`) || strings.Contains(contentStr, `"assignee"`)

		if hasAssigneeChange && settings.NotifyOnAssigned {
			for _, uid := range assigneeUserIDs {
				if uid == actorUserID {
					continue
				}
				p.dispatchEmailNotification(settings, projectID, taskID, uid, actorName, projectName, taskPrefix, taskNumber, taskTitle, "assigned", payload.CreatedAt)
			}
		} else if settings.NotifyOnUpdate {
			for _, uid := range assigneeUserIDs {
				if uid == actorUserID {
					continue
				}
				p.dispatchEmailNotification(settings, projectID, taskID, uid, actorName, projectName, taskPrefix, taskNumber, taskTitle, "updated", payload.CreatedAt)
			}
		}
	}
}

// onCommentActivityEvent handles task.comment.added and comment stream events.
func (p *emailPlugin) onCommentActivityEvent(evt *plugin.Event) {
	var payload struct {
		ID           string          `json:"id"`
		TaskID       string          `json:"task_id"`
		ProjectID    string          `json:"project_id"`
		ActorID      string          `json:"actor_id,omitempty"`
		Content      json.RawMessage `json:"content,omitempty"`
		CreatedAt    string          `json:"created_at"`
	}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		p.log.Error(fmt.Sprintf("failed to decode comment event payload: %v", err))
		return
	}

	taskID := strings.TrimSpace(payload.TaskID)
	projectID := strings.TrimSpace(payload.ProjectID)
	if taskID == "" {
		return
	}

	settings, err := p.getEffectiveSettings(projectID)
	if err != nil || settings.APIKey == "" || !settings.NotifyOnMentioned {
		return
	}

	taskNumber := 0
	taskTitle := ""
	taskRes, _ := p.db.Query(`SELECT task_number, title FROM tasks WHERE id = $1`, taskID)
	if len(taskRes.Rows) > 0 {
		sc := newRowScanner(taskRes.Columns, taskRes.Rows[0])
		taskNumber = sc.intVal("task_number", 0)
		taskTitle = sc.str("title")
	}

	projectName := "Project"
	taskPrefix := ""
	if projectID != "" {
		projRes, _ := p.db.Query(`SELECT name, task_id_prefix FROM projects WHERE id = $1`, projectID)
		if len(projRes.Rows) > 0 {
			sc := newRowScanner(projRes.Columns, projRes.Rows[0])
			if name := sc.str("name"); name != "" {
				projectName = name
			}
			taskPrefix = sc.str("task_id_prefix")
		}
	}

	actorUserID := ""
	actorName := ""
	if payload.ActorID != "" {
		pmRes, _ := p.db.Query(`SELECT user_id FROM project_members WHERE id = $1`, payload.ActorID)
		if len(pmRes.Rows) > 0 {
			sc := newRowScanner(pmRes.Columns, pmRes.Rows[0])
			actorUserID = sc.str("user_id")
		} else {
			actorUserID = payload.ActorID
		}
		if actorUserID != "" {
			uRes, _ := p.db.Query(`SELECT full_name, username FROM users WHERE id = $1`, actorUserID)
			if len(uRes.Rows) > 0 {
				sc := newRowScanner(uRes.Columns, uRes.Rows[0])
				actorName = sc.str("full_name")
				if actorName == "" {
					actorName = sc.str("username")
				}
			}
		}
	}

	notifiedUsers := make(map[string]bool)

	// 1. Structured BlockNote team mentions
	var blocks []struct {
		Content []struct {
			Type  string         `json:"type"`
			Props map[string]any `json:"props"`
		} `json:"content"`
	}
	if err := json.Unmarshal(payload.Content, &blocks); err == nil {
		for _, block := range blocks {
			for _, item := range block.Content {
				if item.Type == "teamMention" && item.Props != nil {
					if idStr, ok := item.Props["id"].(string); ok && idStr != "" {
						if idStr != actorUserID && !notifiedUsers[idStr] {
							notifiedUsers[idStr] = true
							p.dispatchEmailNotification(settings, projectID, taskID, idStr, actorName, projectName, taskPrefix, taskNumber, taskTitle, "mentioned", payload.CreatedAt)
						}
					}
				}
			}
		}
	}

	// 2. Plain text mentions (@username or @email)
	contentStr := string(payload.Content)
	matches := plainMentionRegex.FindAllStringSubmatch(contentStr, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		rawMention := strings.TrimSpace(match[1])
		if rawMention == "" {
			continue
		}
		uRes, err := p.db.Query(`SELECT id FROM users WHERE username = $1`, rawMention)
		if (err != nil || len(uRes.Rows) == 0) && strings.Contains(rawMention, "@") {
			// Try case-insensitive fallback
			uRes, _ = p.db.Query(`SELECT id FROM users WHERE LOWER(username) = LOWER($1)`, rawMention)
		}
		if err == nil && len(uRes.Rows) > 0 {
			uid := newRowScanner(uRes.Columns, uRes.Rows[0]).str("id")
			if uid != "" && uid != actorUserID && !notifiedUsers[uid] {
				notifiedUsers[uid] = true
				p.dispatchEmailNotification(settings, projectID, taskID, uid, actorName, projectName, taskPrefix, taskNumber, taskTitle, "mentioned", payload.CreatedAt)
			}
		}
	}
}

// onNotificationCreated handles direct notification.created events.
func (p *emailPlugin) onNotificationCreated(evt *plugin.Event) {
	payload, err := plugin.JSONPayload[NotificationEventPayload](evt)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to decode notification.created payload: %v", err))
		return
	}

	if payload.RecipientUserID == "" {
		return
	}

	settings, err := p.getEffectiveSettings(payload.ProjectID)
	if err != nil || settings.APIKey == "" {
		return
	}

	if payload.Type == "assigned" && !settings.NotifyOnAssigned {
		return
	}
	if payload.Type == "mentioned" && !settings.NotifyOnMentioned {
		return
	}

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

	p.dispatchEmailNotification(settings, payload.ProjectID, payload.TaskID, payload.RecipientUserID, actorName, projectName, taskPrefix, taskNumber, taskTitle, payload.Type, payload.CreatedAt)
}

func (p *emailPlugin) dispatchEmailNotification(settings *SMTPSettings, projectID, taskID, recipientUserID, actorName, projectName, taskPrefix string, taskNumber int, taskTitle, notifType, createdAt string) {
	if recipientUserID == "" {
		return
	}

	recipientEmail, recipientName, err := p.resolveRecipientEmail(recipientUserID)
	if err != nil {
		p.log.Warn(fmt.Sprintf("could not resolve recipient for user %s: %v", recipientUserID, err))
		return
	}

	formatted := FormatNotificationEmail(
		notifType,
		recipientName,
		actorName,
		projectName,
		taskPrefix,
		taskNumber,
		taskTitle,
		createdAt,
	)

	if recipientEmail == "" || !IsValidEmail(recipientEmail) {
		p.log.Info(fmt.Sprintf("user %s username is not an email and has no override, skipped", recipientUserID))
		_ = p.recordLog(EmailLog{
			ID:               uuid.NewString(),
			ProjectID:        projectID,
			RecipientUserID:  recipientUserID,
			RecipientEmail:   recipientEmail,
			NotificationType: notifType,
			Subject:          formatted.Subject,
			Status:           "skipped",
			ErrorMessage:     "Recipient username is not an email address and no user email override is configured",
			CreatedAt:        nowStr(),
		})
		return
	}

	sendErr := SendEmail(settings, OutboundEmail{
		FromEmail: settings.FromEmail,
		FromName:  settings.FromName,
		ToEmail:   recipientEmail,
		Subject:   formatted.Subject,
		BodyHTML:  formatted.BodyHTML,
		BodyText:  formatted.BodyText,
	})

	status := "sent"
	errMsg := ""
	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
		p.log.Warn(fmt.Sprintf("failed to send notification email to %s: %v", recipientEmail, sendErr))
	} else {
		p.log.Info(fmt.Sprintf("notification email sent successfully to %s for %s", recipientEmail, notifType))
	}

	_ = p.recordLog(EmailLog{
		ID:               uuid.NewString(),
		ProjectID:        projectID,
		RecipientUserID:  recipientUserID,
		RecipientEmail:   recipientEmail,
		NotificationType: notifType,
		Subject:          formatted.Subject,
		Status:           status,
		ErrorMessage:     errMsg,
		CreatedAt:        nowStr(),
	})
}
