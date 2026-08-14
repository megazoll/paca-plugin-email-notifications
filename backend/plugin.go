package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	plugin "github.com/Paca-AI/plugin-sdk-go"
	"github.com/google/uuid"
)

var (
	plainMentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_.+-]+(?:@[a-zA-Z0-9-]+\.[a-zA-Z0-9-.]+)?|[a-zA-Z0-9_.-]+)`)
	supportedTopics   = []string{
		"task.created",
		"task.updated",
		"task.deleted",
		"comment",
		"task.comment.added",
		"task.comment.updated",
		"notification.created",
	}
)

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

	// ── Register event handlers ─────────────────────────────────────────────
	for _, topic := range supportedTopics {
		ctx.On(topic, p.handleActivityEvent(topic))
	}

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

// handleActivityEvent returns an EventHandler closure that processes activity and notification events.
func (p *emailPlugin) handleActivityEvent(topic string) plugin.EventHandler {
	return func(evt *plugin.Event) {
		var payload map[string]any
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			p.log.Warn(fmt.Sprintf("email: failed to decode event payload for topic %s: %v", topic, err))
			return
		}

		projectID, _ := payload["project_id"].(string)
		taskID, _ := payload["task_id"].(string)
		createdAt, _ := payload["created_at"].(string)
		if createdAt == "" {
			createdAt = nowStr()
		}

		// Load settings
		settings, err := p.getEffectiveSettings(projectID)
		if err != nil || settings.APIKey == "" {
			p.log.Debug("email: no email provider configured or API key empty, skipping")
			return
		}

		// Resolve actor info
		actorName, actorUserID := p.resolveActor(topic, payload)

		// Fetch Task details
		taskNumber := 0
		taskTitle := ""
		if taskID != "" {
			taskRes, _ := p.db.Query(`SELECT task_number, title, project_id FROM tasks WHERE id = $1`, taskID)
			if len(taskRes.Rows) > 0 {
				sc := newRowScanner(taskRes.Columns, taskRes.Rows[0])
				taskNumber = sc.intVal("task_number", 0)
				taskTitle = sc.str("title")
				if projectID == "" {
					projectID = sc.str("project_id")
				}
			}
		}

		// Fetch Project details
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

		switch topic {
		case "notification.created":
			recipientUserID, _ := payload["recipient_user_id"].(string)
			notifType, _ := payload["type"].(string)
			if notifType == "" {
				notifType = "assigned"
			}
			if recipientUserID != "" {
				if (notifType == "assigned" && settings.NotifyOnAssigned) || (notifType == "mentioned" && settings.NotifyOnMentioned) {
					p.dispatchEmailNotification(settings, projectID, taskID, recipientUserID, actorName, projectName, taskPrefix, taskNumber, taskTitle, notifType, createdAt)
				}
			}

		case "task.created":
			if !settings.NotifyOnAssigned || taskID == "" {
				return
			}
			assigneeUserIDs := p.getTaskAssigneeUserIDs(taskID)
			for _, uid := range assigneeUserIDs {
				if uid == actorUserID {
					continue
				}
				p.dispatchEmailNotification(settings, projectID, taskID, uid, actorName, projectName, taskPrefix, taskNumber, taskTitle, "assigned", createdAt)
			}

		case "task.updated":
			if taskID == "" {
				return
			}
			rawContent, _ := payload["content"].(string)
			hasAssigneeChange := strings.Contains(rawContent, `"field":"assignee"`) || strings.Contains(rawContent, `"assignee"`)
			assigneeUserIDs := p.getTaskAssigneeUserIDs(taskID)

			if hasAssigneeChange && settings.NotifyOnAssigned {
				for _, uid := range assigneeUserIDs {
					if uid == actorUserID {
						continue
					}
					p.dispatchEmailNotification(settings, projectID, taskID, uid, actorName, projectName, taskPrefix, taskNumber, taskTitle, "assigned", createdAt)
				}
			} else if settings.NotifyOnUpdate {
				for _, uid := range assigneeUserIDs {
					if uid == actorUserID {
						continue
					}
					p.dispatchEmailNotification(settings, projectID, taskID, uid, actorName, projectName, taskPrefix, taskNumber, taskTitle, "updated", createdAt)
				}
			}

		case "comment", "task.comment.added", "task.comment.updated":
			if !settings.NotifyOnMentioned {
				return
			}
			rawContent, _ := payload["content"].(string)
			mentionedUserIDs := p.extractMentionedUserIDs(rawContent)
			for _, uid := range mentionedUserIDs {
				if uid == actorUserID {
					continue
				}
				p.dispatchEmailNotification(settings, projectID, taskID, uid, actorName, projectName, taskPrefix, taskNumber, taskTitle, "mentioned", createdAt)
			}
		}
	}
}

func (p *emailPlugin) getTaskAssigneeUserIDs(taskID string) []string {
	res, err := p.db.Query(`SELECT member_id FROM task_assignees WHERE task_id = $1`, taskID)
	if err != nil || len(res.Rows) == 0 {
		return nil
	}
	uids := make([]string, 0, len(res.Rows))
	for _, row := range res.Rows {
		sc := newRowScanner(res.Columns, row)
		memberID := sc.str("member_id")
		if memberID != "" {
			pmRes, _ := p.db.Query(`SELECT user_id FROM project_members WHERE id = $1`, memberID)
			if len(pmRes.Rows) > 0 {
				uid := newRowScanner(pmRes.Columns, pmRes.Rows[0]).str("user_id")
				if uid != "" {
					uids = append(uids, uid)
				}
			}
		}
	}
	return uids
}

func (p *emailPlugin) extractMentionedUserIDs(rawContent string) []string {
	if rawContent == "" {
		return nil
	}
	found := make(map[string]bool)
	var uids []string

	// 1. BlockNote structured mentions
	var blocks []struct {
		Content []struct {
			Type  string         `json:"type"`
			Props map[string]any `json:"props"`
		} `json:"content"`
	}
	if err := json.Unmarshal([]byte(rawContent), &blocks); err == nil {
		for _, block := range blocks {
			for _, item := range block.Content {
				if item.Type == "teamMention" && item.Props != nil {
					if idStr, ok := item.Props["id"].(string); ok && idStr != "" {
						if !found[idStr] {
							found[idStr] = true
							uids = append(uids, idStr)
						}
					}
				}
			}
		}
	}

	// 2. Plain-text @mentions
	matches := plainMentionRegex.FindAllStringSubmatch(rawContent, -1)
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
			uRes, _ = p.db.Query(`SELECT id FROM users WHERE LOWER(username) = LOWER($1)`, rawMention)
		}
		if err == nil && len(uRes.Rows) > 0 {
			uid := newRowScanner(uRes.Columns, uRes.Rows[0]).str("id")
			if uid != "" && !found[uid] {
				found[uid] = true
				uids = append(uids, uid)
			}
		}
	}

	return uids
}

func (p *emailPlugin) resolveActor(topic string, payload map[string]any) (name, id string) {
	actorID, _ := payload["actor_id"].(string)
	if actorID == "" {
		return "Someone", ""
	}

	switch topic {
	case "task.comment.added", "task.comment.updated", "task.comment.deleted", "comment":
		name = p.lookupName(
			`SELECT u.full_name AS name FROM project_members pm JOIN users u ON u.id = pm.user_id WHERE pm.id = $1`,
			actorID,
		)
		if name == "" {
			name = p.lookupName(
				`SELECT u.username AS name FROM project_members pm JOIN users u ON u.id = pm.user_id WHERE pm.id = $1`,
				actorID,
			)
		}
		pmRes, _ := p.db.Query(`SELECT user_id FROM project_members WHERE id = $1`, actorID)
		if len(pmRes.Rows) > 0 {
			actorID = newRowScanner(pmRes.Columns, pmRes.Rows[0]).str("user_id")
		}
	default:
		name = p.lookupName(`SELECT full_name AS name FROM users WHERE id = $1`, actorID)
		if name == "" {
			name = p.lookupName(`SELECT username AS name FROM users WHERE id = $1`, actorID)
		}
	}
	if name == "" {
		name = "Someone"
	}
	return name, actorID
}

func (p *emailPlugin) lookupName(sqlStr, param string) string {
	result, err := p.db.Query(sqlStr, param)
	if err != nil || len(result.Rows) == 0 {
		return ""
	}
	return newRowScanner(result.Columns, result.Rows[0]).str("name")
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
