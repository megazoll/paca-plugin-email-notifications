package main

import (
	"encoding/json"
	"sync"
	"testing"

	plugin "github.com/Paca-AI/plugin-sdk-go"
	"github.com/Paca-AI/plugin-sdk-go/plugintest"
)

const (
	testProjectID = "proj-1"
	testUser1ID   = "user-1"
	testUser2ID   = "user-2"
	testUser3ID   = "user-3"
)

type mockSender struct {
	mu    sync.Mutex
	sends []struct {
		Settings *SMTPSettings
		To       string
		Subject  string
		BodyText string
		BodyHTML string
	}
}

func (m *mockSender) SendEmail(settings *SMTPSettings, to string, subject, bodyText, bodyHTML string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sends = append(m.sends, struct {
		Settings *SMTPSettings
		To       string
		Subject  string
		BodyText string
		BodyHTML string
	}{
		Settings: settings,
		To:       to,
		Subject:  subject,
		BodyText: bodyText,
		BodyHTML: bodyHTML,
	})
	return nil
}

func setupTestContext(t *testing.T) (*plugintest.Context, *emailPlugin, *mockSender) {
	t.Helper()
	tc := plugintest.NewContext(t)

	// Seed users table
	tc.DB.SeedRows("users",
		[]string{"id", "username", "full_name"},
		[][]any{
			{testUser1ID, "alice@example.com", "Alice Smith"},
			{testUser2ID, "bob_plain_username", "Bob Jones"},
			{testUser3ID, "charlie_dev", "Charlie Brown"},
		},
	)

	// Seed projects table
	tc.DB.SeedRows("projects",
		[]string{"id", "name", "task_id_prefix"},
		[][]any{
			{testProjectID, "Alpha Project", "ALP"},
		},
	)

	// Seed tasks table
	tc.DB.SeedRows("tasks",
		[]string{"id", "project_id", "task_number", "title"},
		[][]any{
			{"task-1", testProjectID, 42, "Fix authentication bug"},
		},
	)

	// Seed project_members table
	tc.DB.SeedRows("project_members",
		[]string{"id", "project_id", "user_id"},
		[][]any{
			{"member-actor", testProjectID, testUser3ID},
		},
	)

	// Seed default global SMTP settings
	tc.DB.SeedRows("smtp_settings",
		[]string{"id", "scope", "project_id", "enabled", "host", "port", "username", "password", "from_email", "from_name", "security", "webhook_url", "webhook_api_key", "notify_on_assigned", "notify_on_mentioned", "created_at", "updated_at"},
		[][]any{
			{"smtp-global", "global", nil, true, "smtp.example.com", 587, "user", "pass", "notifications@paca.local", "Paca", "starttls", "", "", true, true, "2026-08-01T00:00:00Z", "2026-08-01T00:00:00Z"},
		},
	)

	tc.DB.SeedRows("email_logs",
		[]string{"id", "project_id", "notification_id", "recipient_user_id", "recipient_email", "notification_type", "subject", "body_text", "status", "error_message", "created_at"},
		[][]any{},
	)

	tc.DB.SeedRows("user_email_overrides",
		[]string{"user_id", "email", "created_at", "updated_at"},
		[][]any{},
	)

	sender := &mockSender{}
	p := &emailPlugin{
		sender: sender,
	}

	if err := p.Init(tc.PluginContext()); err != nil {
		t.Fatal("Init failed:", err)
	}

	return tc, p, sender
}

func callerReq() plugintest.Request {
	return plugintest.Request{
		Caller: plugin.CallerIdentity{
			ProjectID:  testProjectID,
			CallerID:   testUser1ID,
			CallerRole: "PROJECT_MEMBER",
		},
		PathParams: map[string]string{},
	}
}

func decodeEnvelopeData[T any](t *testing.T, res *plugin.Response) T {
	t.Helper()
	var env struct {
		Success bool   `json:"success"`
		Data    T      `json:"data"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("failed to decode response %s: %v", res.BodyString(), err)
	}
	if !env.Success {
		t.Fatalf("expected success true, got false with error: %s", env.Error)
	}
	return env.Data
}

func TestIsValidEmail(t *testing.T) {
	valid := []string{
		"test@example.com",
		"user.name+tag@sub.domain.org",
		"first.last@company.co.uk",
		"a@b.cd",
	}
	for _, e := range valid {
		if !IsValidEmail(e) {
			t.Errorf("expected %q to be valid email", e)
		}
	}

	invalid := []string{
		"plain_username",
		"admin",
		"user@",
		"@domain.com",
		"user@domain",
		"user @domain.com",
		"",
	}
	for _, e := range invalid {
		if IsValidEmail(e) {
			t.Errorf("expected %q to be invalid email", e)
		}
	}
}

func TestNotificationCreated_ValidEmailUsername_SendsEmail(t *testing.T) {
	tc, _, sender := setupTestContext(t)

	// Dispatch notification.created event for Alice (username is alice@example.com)
	evtPayload, _ := json.Marshal(NotificationEventPayload{
		ID:              "notif-1",
		RecipientUserID: testUser1ID,
		ActorMemberID:   "member-actor",
		Type:            "assigned",
		TaskID:          "task-1",
		ProjectID:       testProjectID,
		CreatedAt:       "2026-08-14T10:00:00Z",
	})

	plugin.DispatchEvent(tc.PluginContext(), "notification.created", evtPayload)

	if len(sender.sends) != 1 {
		t.Fatalf("expected 1 email send, got %d", len(sender.sends))
	}

	send := sender.sends[0]
	if send.To != "alice@example.com" {
		t.Errorf("expected recipient 'alice@example.com', got %q", send.To)
	}
	if send.Subject != "[Paca] [Alpha Project] Assigned to ALP-42: Fix authentication bug" {
		t.Errorf("unexpected subject: %q", send.Subject)
	}

	// Verify log recorded in DB
	rows := tc.DB.AllRows("email_logs")
	if len(rows) != 1 {
		t.Fatalf("expected 1 log row in email_logs, got %d", len(rows))
	}
}

func TestNotificationCreated_InvalidEmailUsername_Skipped(t *testing.T) {
	tc, _, sender := setupTestContext(t)

	// Dispatch event for Bob (username is bob_plain_username - not an email)
	evtPayload, _ := json.Marshal(NotificationEventPayload{
		ID:              "notif-2",
		RecipientUserID: testUser2ID,
		ActorMemberID:   "member-actor",
		Type:            "assigned",
		TaskID:          "task-1",
		ProjectID:       testProjectID,
		CreatedAt:       "2026-08-14T10:00:00Z",
	})

	plugin.DispatchEvent(tc.PluginContext(), "notification.created", evtPayload)

	if len(sender.sends) != 0 {
		t.Fatalf("expected 0 email sends for non-email username, got %d", len(sender.sends))
	}

	rows := tc.DB.AllRows("email_logs")
	if len(rows) != 0 {
		t.Fatalf("expected 0 log rows in email_logs, got %d", len(rows))
	}
}

func TestNotificationCreated_WithUserEmailOverride_SendsEmail(t *testing.T) {
	tc, _, sender := setupTestContext(t)

	// Add an override for Bob
	tc.DB.SeedRows("user_email_overrides",
		[]string{"user_id", "email", "created_at", "updated_at"},
		[][]any{
			{testUser2ID, "bob.override@company.com", "2026-08-01T00:00:00Z", "2026-08-01T00:00:00Z"},
		},
	)

	evtPayload, _ := json.Marshal(NotificationEventPayload{
		ID:              "notif-3",
		RecipientUserID: testUser2ID,
		ActorMemberID:   "member-actor",
		Type:            "mentioned",
		TaskID:          "task-1",
		ProjectID:       testProjectID,
		CreatedAt:       "2026-08-14T10:00:00Z",
	})

	plugin.DispatchEvent(tc.PluginContext(), "notification.created", evtPayload)

	if len(sender.sends) != 1 {
		t.Fatalf("expected 1 email send via override, got %d", len(sender.sends))
	}

	send := sender.sends[0]
	if send.To != "bob.override@company.com" {
		t.Errorf("expected recipient 'bob.override@company.com', got %q", send.To)
	}
	if send.Subject != "[Paca] [Alpha Project] Mentioned in ALP-42: Fix authentication bug" {
		t.Errorf("unexpected subject: %q", send.Subject)
	}
}

func TestSettingsRoutes(t *testing.T) {
	tc, _, _ := setupTestContext(t)

	// 1. GET admin settings
	res := tc.Call("GET", "/email-notifications/admin-settings", callerReq())
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 for GET admin settings, got %d: %s", res.StatusCode, res.BodyString())
	}
	settings := decodeEnvelopeData[SMTPSettings](t, res)
	if settings.Host != "smtp.example.com" {
		t.Errorf("expected host 'smtp.example.com', got %q", settings.Host)
	}

	// 2. PATCH admin settings
	patchReq := callerReq()
	patchReq.Body = []byte(`{"host":"smtp.newhost.com","port":465,"security":"tls"}`)
	res = tc.Call("PATCH", "/email-notifications/admin-settings", patchReq)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 for PATCH admin settings, got %d: %s", res.StatusCode, res.BodyString())
	}
	updated := decodeEnvelopeData[SMTPSettings](t, res)
	if updated.Host != "smtp.newhost.com" || updated.Port != 465 || updated.Security != "tls" {
		t.Errorf("settings not updated properly: %+v", updated)
	}
}

func TestTestEmailRoutes(t *testing.T) {
	tc, _, sender := setupTestContext(t)

	testReq := callerReq()
	testReq.Body = []byte(`{"to_email":"developer@paca.ai"}`)

	res := tc.Call("POST", "/email-notifications/admin-test", testReq)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 for admin test email, got %d: %s", res.StatusCode, res.BodyString())
	}

	if len(sender.sends) != 1 {
		t.Fatalf("expected 1 test send, got %d", len(sender.sends))
	}
	if sender.sends[0].To != "developer@paca.ai" {
		t.Errorf("expected test recipient 'developer@paca.ai', got %q", sender.sends[0].To)
	}
}

func TestOverridesRoutes(t *testing.T) {
	tc, _, _ := setupTestContext(t)

	// 1. Save override
	saveReq := callerReq()
	saveReq.Body = []byte(`{"user_id":"user-2","email":"bob.custom@example.com"}`)
	res := tc.Call("POST", "/email-notifications/admin-overrides", saveReq)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 for saving override, got %d: %s", res.StatusCode, res.BodyString())
	}

	// 2. List overrides
	res = tc.Call("GET", "/email-notifications/admin-overrides", callerReq())
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 for listing overrides, got %d: %s", res.StatusCode, res.BodyString())
	}
	overrides := decodeEnvelopeData[[]UserEmailOverride](t, res)
	if len(overrides) != 1 {
		t.Fatalf("expected 1 override, got %d", len(overrides))
	}
	if overrides[0].Email != "bob.custom@example.com" {
		t.Errorf("expected email 'bob.custom@example.com', got %q", overrides[0].Email)
	}

	// 3. Delete override
	delReq := callerReq()
	delReq.PathParams = map[string]string{"userId": "user-2"}
	res = tc.Call("DELETE", "/email-notifications/admin-overrides/:userId", delReq)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 for deleting override, got %d: %s", res.StatusCode, res.BodyString())
	}
}
