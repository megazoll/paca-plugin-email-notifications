package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	plugin "github.com/Paca-AI/plugin-sdk-go"
	"github.com/google/uuid"
)

type envelope struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type rowScanner struct {
	colMap map[string]int
	row    []any
}

func newRowScanner(columns []string, row []any) rowScanner {
	m := make(map[string]int, len(columns))
	for i, c := range columns {
		m[strings.ToLower(c)] = i
	}
	return rowScanner{colMap: m, row: row}
}

func (s rowScanner) str(col string) string {
	idx, ok := s.colMap[strings.ToLower(col)]
	if !ok || idx >= len(s.row) || s.row[idx] == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", s.row[idx]))
}

func (s rowScanner) boolVal(col string, def bool) bool {
	idx, ok := s.colMap[strings.ToLower(col)]
	if !ok || idx >= len(s.row) || s.row[idx] == nil {
		return def
	}
	if b, ok := s.row[idx].(bool); ok {
		return b
	}
	v := strings.ToLower(fmt.Sprintf("%v", s.row[idx]))
	return v == "true" || v == "1" || v == "t"
}

func (s rowScanner) intVal(col string, def int) int {
	idx, ok := s.colMap[strings.ToLower(col)]
	if !ok || idx >= len(s.row) || s.row[idx] == nil {
		return def
	}
	if f, ok := s.row[idx].(float64); ok {
		return int(f)
	}
	if i, ok := s.row[idx].(int); ok {
		return i
	}
	if i64, ok := s.row[idx].(int64); ok {
		return int(i64)
	}
	return def
}

func timeNowNano() int64 {
	return time.Now().UnixNano()
}

func ok(res *plugin.Response, data any) {
	res.JSON(200, envelope{Success: true, Data: data})
}

func badRequest(res *plugin.Response, msg string) {
	res.JSON(400, envelope{Success: false, Error: msg})
}

func serverError(res *plugin.Response, msg string) {
	res.JSON(500, envelope{Success: false, Error: msg})
}

// ── Settings Handlers ────────────────────────────────────────────────────────

func (p *emailPlugin) getProjectSettings(req *plugin.Request, res *plugin.Response) {
	projectID := req.PathParam("projectId")
	settings, err := p.loadSettings("project", projectID)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to load project settings: %v", err))
		serverError(res, "failed to load settings")
		return
	}
	ok(res, settings)
}

func (p *emailPlugin) updateProjectSettings(req *plugin.Request, res *plugin.Response) {
	projectID := req.PathParam("projectId")
	var input UpdateSettingsInput
	if err := json.Unmarshal(req.Body, &input); err != nil {
		badRequest(res, "invalid json body")
		return
	}
	settings, err := p.saveSettings("project", projectID, input)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to update project settings: %v", err))
		serverError(res, "failed to update settings: "+err.Error())
		return
	}
	ok(res, settings)
}

func (p *emailPlugin) getAdminSettings(req *plugin.Request, res *plugin.Response) {
	settings, err := p.loadSettings("global", "")
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to load global settings: %v", err))
		serverError(res, "failed to load settings: "+err.Error())
		return
	}
	ok(res, settings)
}

func (p *emailPlugin) updateAdminSettings(req *plugin.Request, res *plugin.Response) {
	var input UpdateSettingsInput
	if err := json.Unmarshal(req.Body, &input); err != nil {
		badRequest(res, "invalid json body")
		return
	}
	settings, err := p.saveSettings("global", "", input)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to update global settings: %v", err))
		serverError(res, "failed to update settings: "+err.Error())
		return
	}
	ok(res, settings)
}

// ── Logs Handlers ────────────────────────────────────────────────────────────

func (p *emailPlugin) getProjectLogs(req *plugin.Request, res *plugin.Response) {
	projectID := req.PathParam("projectId")
	logs, err := p.loadLogs(projectID, 100)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to load project logs: %v", err))
		serverError(res, "failed to load logs")
		return
	}
	ok(res, logs)
}

func (p *emailPlugin) getAdminLogs(req *plugin.Request, res *plugin.Response) {
	logs, err := p.loadLogs("", 100)
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to load admin logs: %v", err))
		serverError(res, "failed to load logs")
		return
	}
	ok(res, logs)
}

// ── Test Email Handlers ──────────────────────────────────────────────────────

func (p *emailPlugin) testProjectEmail(req *plugin.Request, res *plugin.Response) {
	projectID := req.PathParam("projectId")
	var input TestEmailInput
	if err := json.Unmarshal(req.Body, &input); err != nil {
		badRequest(res, "invalid json body")
		return
	}
	if !IsValidEmail(input.ToEmail) {
		badRequest(res, "recipient email address is invalid")
		return
	}

	settings, err := p.loadSettings("project", projectID)
	if err != nil || settings.APIKey == "" {
		// Fallback to global settings
		settings, err = p.loadSettings("global", "")
	}
	if err != nil {
		serverError(res, "failed to load email settings")
		return
	}

	if input.Provider != nil {
		settings.Provider = *input.Provider
	}
	if input.Endpoint != nil {
		settings.Endpoint = *input.Endpoint
	}
	if input.APIKey != nil {
		settings.APIKey = *input.APIKey
	}
	if input.FromEmail != nil {
		settings.FromEmail = *input.FromEmail
	}
	if input.FromName != nil {
		settings.FromName = *input.FromName
	}

	testMail := FormatTestEmail(input.ToEmail)
	err = SendEmail(settings, OutboundEmail{
		FromEmail: settings.FromEmail,
		FromName:  settings.FromName,
		ToEmail:   input.ToEmail,
		Subject:   testMail.Subject,
		BodyHTML:  testMail.BodyHTML,
		BodyText:  testMail.BodyText,
	})

	status := "sent"
	errMsg := ""
	if err != nil {
		status = "failed"
		errMsg = err.Error()
	}

	_ = p.recordLog(EmailLog{
		ID:               uuid.NewString(),
		ProjectID:        projectID,
		RecipientEmail:   input.ToEmail,
		NotificationType: "test",
		Subject:          testMail.Subject,
		Status:           status,
		ErrorMessage:     errMsg,
		CreatedAt:        nowStr(),
	})

	if err != nil {
		serverError(res, "failed to send test email: "+err.Error())
		return
	}

	ok(res, map[string]any{"sent": true, "recipient": input.ToEmail})
}

func (p *emailPlugin) testAdminEmail(req *plugin.Request, res *plugin.Response) {
	var input TestEmailInput
	if err := json.Unmarshal(req.Body, &input); err != nil {
		badRequest(res, "invalid json body")
		return
	}
	if !IsValidEmail(input.ToEmail) {
		badRequest(res, "recipient email address is invalid")
		return
	}

	settings, err := p.loadSettings("global", "")
	if err != nil {
		serverError(res, "failed to load email settings")
		return
	}

	if input.Provider != nil {
		settings.Provider = *input.Provider
	}
	if input.Endpoint != nil {
		settings.Endpoint = *input.Endpoint
	}
	if input.APIKey != nil {
		settings.APIKey = *input.APIKey
	}
	if input.FromEmail != nil {
		settings.FromEmail = *input.FromEmail
	}
	if input.FromName != nil {
		settings.FromName = *input.FromName
	}

	testMail := FormatTestEmail(input.ToEmail)
	err = SendEmail(settings, OutboundEmail{
		FromEmail: settings.FromEmail,
		FromName:  settings.FromName,
		ToEmail:   input.ToEmail,
		Subject:   testMail.Subject,
		BodyHTML:  testMail.BodyHTML,
		BodyText:  testMail.BodyText,
	})

	status := "sent"
	errMsg := ""
	if err != nil {
		status = "failed"
		errMsg = err.Error()
	}

	_ = p.recordLog(EmailLog{
		ID:               uuid.NewString(),
		RecipientEmail:   input.ToEmail,
		NotificationType: "test",
		Subject:          testMail.Subject,
		Status:           status,
		ErrorMessage:     errMsg,
		CreatedAt:        nowStr(),
	})

	if err != nil {
		serverError(res, "failed to send test email: "+err.Error())
		return
	}

	ok(res, map[string]any{"sent": true, "recipient": input.ToEmail})
}

// ── Overrides Handlers ───────────────────────────────────────────────────────

func (p *emailPlugin) listAdminOverrides(req *plugin.Request, res *plugin.Response) {
	overrides, err := p.loadOverrides()
	if err != nil {
		p.log.Error(fmt.Sprintf("failed to load overrides: %v", err))
		serverError(res, "failed to load overrides")
		return
	}
	ok(res, overrides)
}

func (p *emailPlugin) saveAdminOverride(req *plugin.Request, res *plugin.Response) {
	var input SaveOverrideInput
	if err := json.Unmarshal(req.Body, &input); err != nil {
		badRequest(res, "invalid json body")
		return
	}
	if input.UserID == "" {
		badRequest(res, "user_id is required")
		return
	}
	if !IsValidEmail(input.Email) {
		badRequest(res, "invalid email address")
		return
	}

	if err := p.storeOverride(input.UserID, input.Email); err != nil {
		p.log.Error(fmt.Sprintf("failed to save override: %v", err))
		serverError(res, "failed to save override")
		return
	}
	ok(res, map[string]any{"saved": true, "user_id": input.UserID, "email": input.Email})
}

func (p *emailPlugin) deleteAdminOverride(req *plugin.Request, res *plugin.Response) {
	userID := req.PathParam("userId")
	if userID == "" {
		badRequest(res, "userId parameter is required")
		return
	}
	if err := p.removeOverride(userID); err != nil {
		p.log.Error(fmt.Sprintf("failed to delete override: %v", err))
		serverError(res, "failed to delete override")
		return
	}
	ok(res, map[string]any{"deleted": true, "user_id": userID})
}

// ── Database Helper Methods ──────────────────────────────────────────────────

func (p *emailPlugin) loadSettings(scope, projectID string) (*SMTPSettings, error) {
	var query string
	var args []any
	if scope == "project" && projectID != "" {
		query = `SELECT id, project_id, provider, endpoint, api_key, from_email, from_name, notify_on_assign, notify_on_mention, notify_on_update, created_at, updated_at FROM smtp_settings WHERE project_id = $1`
		args = []any{projectID}
	} else {
		query = `SELECT id, project_id, provider, endpoint, api_key, from_email, from_name, notify_on_assign, notify_on_mention, notify_on_update, created_at, updated_at FROM smtp_settings WHERE project_id IS NULL`
	}

	res, err := p.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	if len(res.Rows) == 0 {
		return &SMTPSettings{
			ID:                "default",
			ProjectID:         projectID,
			Provider:          ProviderYandexPostbox,
			Endpoint:          "https://postbox.cloud.yandex.net/v2/email/outbound-emails",
			APIKey:            "",
			FromEmail:         "no-reply@paca.local",
			FromName:          "Paca",
			NotifyOnAssigned:  true,
			NotifyOnMentioned: true,
			NotifyOnUpdate:    false,
			CreatedAt:         nowStr(),
			UpdatedAt:         nowStr(),
		}, nil
	}

	return scanSMTPSettings(res.Columns, res.Rows[0]), nil
}

func scanSMTPSettings(columns []string, row []any) *SMTPSettings {
	sc := newRowScanner(columns, row)
	prov := EmailProviderType(sc.str("provider"))
	if prov == "" {
		prov = ProviderYandexPostbox
	}
	endpoint := sc.str("endpoint")
	if endpoint == "" && prov == ProviderYandexPostbox {
		endpoint = "https://postbox.cloud.yandex.net/v2/email/outbound-emails"
	}

	return &SMTPSettings{
		ID:                sc.str("id"),
		ProjectID:         sc.str("project_id"),
		Provider:          prov,
		Endpoint:          endpoint,
		APIKey:            sc.str("api_key"),
		FromEmail:         sc.str("from_email"),
		FromName:          sc.str("from_name"),
		NotifyOnAssigned:  sc.boolVal("notify_on_assign", true),
		NotifyOnMentioned: sc.boolVal("notify_on_mention", true),
		NotifyOnUpdate:    sc.boolVal("notify_on_update", false),
		CreatedAt:         sc.str("created_at"),
		UpdatedAt:         sc.str("updated_at"),
	}
}

func (p *emailPlugin) saveSettings(scope, projectID string, input UpdateSettingsInput) (*SMTPSettings, error) {
	current, err := p.loadSettings(scope, projectID)
	if err != nil {
		return nil, err
	}

	if input.Provider != nil {
		current.Provider = *input.Provider
	}
	if input.Endpoint != nil {
		current.Endpoint = *input.Endpoint
	}
	if input.APIKey != nil {
		current.APIKey = *input.APIKey
	}
	if input.FromEmail != nil {
		current.FromEmail = *input.FromEmail
	}
	if input.FromName != nil {
		current.FromName = *input.FromName
	}
	if input.NotifyOnAssigned != nil {
		current.NotifyOnAssigned = *input.NotifyOnAssigned
	}
	if input.NotifyOnMentioned != nil {
		current.NotifyOnMentioned = *input.NotifyOnMentioned
	}
	if input.NotifyOnUpdate != nil {
		current.NotifyOnUpdate = *input.NotifyOnUpdate
	}

	now := nowStr()
	current.UpdatedAt = now

	// Check if already exists in table
	var existingQuery string
	var existingArgs []any
	if scope == "project" && projectID != "" {
		existingQuery = `SELECT id FROM smtp_settings WHERE project_id = $1`
		existingArgs = []any{projectID}
	} else {
		existingQuery = `SELECT id FROM smtp_settings WHERE project_id IS NULL`
	}

	res, _ := p.db.Query(existingQuery, existingArgs...)
	if len(res.Rows) > 0 {
		id := fmt.Sprintf("%v", res.Rows[0][0])
		current.ID = id
		_, err = p.db.Exec(
			`UPDATE smtp_settings SET provider = $1, endpoint = $2, api_key = $3, from_email = $4, from_name = $5, notify_on_assign = $6, notify_on_mention = $7, notify_on_update = $8, updated_at = $9 WHERE id = $10`,
			string(current.Provider), current.Endpoint, current.APIKey, current.FromEmail, current.FromName, current.NotifyOnAssigned, current.NotifyOnMentioned, current.NotifyOnUpdate, now, id,
		)
	} else {
		id := uuid.NewString()
		current.ID = id
		current.CreatedAt = now
		var pid any
		if projectID != "" {
			pid = projectID
		}
		_, err = p.db.Exec(
			`INSERT INTO smtp_settings (id, project_id, provider, endpoint, api_key, from_email, from_name, notify_on_assign, notify_on_mention, notify_on_update, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			id, pid, string(current.Provider), current.Endpoint, current.APIKey, current.FromEmail, current.FromName, current.NotifyOnAssigned, current.NotifyOnMentioned, current.NotifyOnUpdate, now, now,
		)
	}

	return current, err
}

func (p *emailPlugin) loadLogs(projectID string, limit int) ([]EmailLog, error) {
	if limit <= 0 {
		limit = 100
	}
	var query string
	var args []any
	if projectID != "" {
		query = `SELECT id, project_id, recipient_user_id, recipient_email, notification_type, subject, status, error_message, created_at FROM email_logs WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2`
		args = []any{projectID, limit}
	} else {
		query = `SELECT id, project_id, recipient_user_id, recipient_email, notification_type, subject, status, error_message, created_at FROM email_logs ORDER BY created_at DESC LIMIT $1`
		args = []any{limit}
	}

	res, err := p.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	logs := make([]EmailLog, 0, len(res.Rows))
	for _, row := range res.Rows {
		sc := newRowScanner(res.Columns, row)
		logs = append(logs, EmailLog{
			ID:               sc.str("id"),
			ProjectID:        sc.str("project_id"),
			RecipientUserID:  sc.str("recipient_user_id"),
			RecipientEmail:   sc.str("recipient_email"),
			NotificationType: sc.str("notification_type"),
			Subject:          sc.str("subject"),
			Status:           sc.str("status"),
			ErrorMessage:     sc.str("error_message"),
			CreatedAt:        sc.str("created_at"),
		})
	}
	return logs, nil
}

func (p *emailPlugin) recordLog(l EmailLog) error {
	id := l.ID
	if id == "" {
		id = uuid.NewString()
	}
	ruid := l.RecipientUserID
	if ruid == "" {
		ruid = ""
	}
	pid := l.ProjectID
	errMsg := l.ErrorMessage
	now := l.CreatedAt
	if now == "" {
		now = nowStr()
	}

	_, err := p.db.Query(
		`INSERT INTO email_logs (id, project_id, recipient_user_id, recipient_email, notification_type, subject, status, error_message, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id`,
		id, pid, ruid, l.RecipientEmail, l.NotificationType, l.Subject, l.Status, errMsg, now,
	)
	if err != nil {
		p.log.Error("email: recordLog failed: " + err.Error())
		return err
	}
	p.log.Info(fmt.Sprintf("email: recorded log: id=%s recipient=%s status=%s", id, l.RecipientEmail, l.Status))
	return nil
}

func (p *emailPlugin) loadOverrides() ([]UserEmailOverride, error) {
	res, err := p.db.Query(`SELECT user_id, email, created_at, updated_at FROM user_email_overrides ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	overrides := make([]UserEmailOverride, 0, len(res.Rows))
	for _, row := range res.Rows {
		sc := newRowScanner(res.Columns, row)
		overrides = append(overrides, UserEmailOverride{
			UserID:    sc.str("user_id"),
			Email:     sc.str("email"),
			CreatedAt: sc.str("created_at"),
			UpdatedAt: sc.str("updated_at"),
		})
	}
	return overrides, nil
}

func (p *emailPlugin) storeOverride(userID, email string) error {
	now := nowStr()
	res, _ := p.db.Query(`SELECT user_id FROM user_email_overrides WHERE user_id = $1`, userID)
	if len(res.Rows) > 0 {
		_, err := p.db.Exec(`UPDATE user_email_overrides SET email = $1, updated_at = $2 WHERE user_id = $3`, email, now, userID)
		return err
	}
	_, err := p.db.Exec(`INSERT INTO user_email_overrides (user_id, email, created_at, updated_at) VALUES ($1, $2, $3, $4)`, userID, email, now, now)
	return err
}

func (p *emailPlugin) removeOverride(userID string) error {
	_, err := p.db.Exec(`DELETE FROM user_email_overrides WHERE user_id = $1`, userID)
	return err
}

// resolveRecipientEmail gets the email address for a user ID.
// 1. Checks user_email_overrides table.
// 2. Queries users table for username: if username is valid email, returns it.
// 3. Otherwise returns empty string (meaning: do not send email).
func (p *emailPlugin) resolveRecipientEmail(userID string) (email string, fullName string, err error) {
	// 1. Check override
	res, _ := p.db.Query(`SELECT email FROM user_email_overrides WHERE user_id = $1`, userID)
	if len(res.Rows) > 0 {
		sc := newRowScanner(res.Columns, res.Rows[0])
		ovEmail := sc.str("email")
		if IsValidEmail(ovEmail) {
			email = ovEmail
		}
	}

	// 2. Query user info
	uRes, err := p.db.Query(`SELECT username, full_name FROM users WHERE id = $1`, userID)
	if err != nil {
		return "", "", err
	}
	if len(uRes.Rows) == 0 {
		return "", "", sql.ErrNoRows
	}

	uSc := newRowScanner(uRes.Columns, uRes.Rows[0])
	username := uSc.str("username")
	fullName = uSc.str("full_name")
	if fullName == "" {
		fullName = username
	}

	// If no override email was found, use username if it's a valid email
	if email == "" && IsValidEmail(username) {
		email = username
	}

	return email, fullName, nil
}
