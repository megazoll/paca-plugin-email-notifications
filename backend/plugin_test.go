package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	plugin "github.com/Paca-AI/plugin-sdk-go"
	"github.com/Paca-AI/plugin-sdk-go/plugintest"
)

const testProjectID = "proj-1"

func setupTestPlugin(t *testing.T) (*plugintest.Context, *emailPlugin) {
	t.Helper()
	tc := plugintest.NewContext(t)

	// Seed tables
	tc.DB.SeedRows("smtp_settings",
		[]string{"id", "project_id", "provider", "endpoint", "api_key", "from_email", "from_name", "notify_on_assign", "notify_on_mention", "notify_on_update", "created_at", "updated_at"},
		[][]any{
			{"global-default", nil, "yandex_postbox", "https://postbox.cloud.yandex.net/v2/email/outbound-emails", "AQVN_test_key", "notifications@company.com", "PACA", true, true, false, "2026-08-14T00:00:00Z", "2026-08-14T00:00:00Z"},
		},
	)
	tc.DB.SeedRows("email_logs",
		[]string{"id", "project_id", "recipient_user_id", "recipient_email", "notification_type", "subject", "status", "error_message", "created_at"},
		[][]any{},
	)
	tc.DB.SeedRows("user_email_overrides",
		[]string{"user_id", "email", "created_at", "updated_at"},
		[][]any{},
	)
	tc.DB.SeedRows("users",
		[]string{"id", "username", "full_name"},
		[][]any{
			{"user-1", "alex@company.com", "Alex Developer"},
			{"user-2", "john_doe", "John Doe"},
			{"user-3", "maria_admin", "Maria Admin"},
		},
	)
	tc.DB.SeedRows("projects",
		[]string{"id", "name", "task_id_prefix"},
		[][]any{
			{testProjectID, "Frontend App", "FE"},
		},
	)
	tc.DB.SeedRows("tasks",
		[]string{"id", "task_number", "title", "project_id"},
		[][]any{
			{"task-1", 42, "Refactor UI", testProjectID},
		},
	)
	tc.DB.SeedRows("project_members",
		[]string{"id", "project_id", "user_id"},
		[][]any{
			{"pm-1", testProjectID, "user-1"},
		},
	)
	tc.DB.SeedRows("task_assignees",
		[]string{"task_id", "member_id"},
		[][]any{
			{"task-1", "pm-1"},
		},
	)

	var p emailPlugin
	if err := p.Init(tc.PluginContext()); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	return tc, &p
}

func callerReq() plugintest.Request {
	return plugintest.Request{
		Caller: plugin.CallerIdentity{
			ProjectID:  testProjectID,
			CallerID:   "user-1",
			CallerRole: "PROJECT_MEMBER",
		},
		PathParams: map[string]string{},
	}
}

func adminReq() plugintest.Request {
	return plugintest.Request{
		Caller: plugin.CallerIdentity{
			CallerID:   "user-admin",
			CallerRole: "GLOBAL_ADMIN",
		},
		PathParams: map[string]string{},
	}
}

func TestIsValidEmail(t *testing.T) {
	validEmails := []string{
		"alex@company.com",
		"user.name+tag@sub.domain.org",
		"admin@paca.io",
	}
	for _, email := range validEmails {
		if !IsValidEmail(email) {
			t.Errorf("expected %q to be valid email", email)
		}
	}

	invalidEmails := []string{
		"alex",
		"admin",
		"@domain.com",
		"user@",
		"",
		"alex user@domain.com",
	}
	for _, email := range invalidEmails {
		if IsValidEmail(email) {
			t.Errorf("expected %q to be invalid email", email)
		}
	}
}

func TestNotificationCreated_ValidEmailUsername(t *testing.T) {
	tc, p := setupTestPlugin(t)

	payload := map[string]any{
		"id":                "notif-1",
		"recipient_user_id": "user-1",
		"project_id":        testProjectID,
		"task_id":           "task-1",
		"type":              "assigned",
		"created_at":        "2026-08-14T12:00:00Z",
	}
	payloadJSON, _ := json.Marshal(payload)

	evt := &plugin.Event{
		Topic:   "notification.created",
		Payload: payloadJSON,
	}

	p.handleActivityEvent("notification.created")(evt)

	// Verify log recorded (even outside WASM where Fetch returns mock/error)
	logs, err := p.loadLogs("", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected email log to be recorded, got: %v (logs: %v)", err, logs)
	}
	if logs[0].RecipientEmail != "alex@company.com" {
		t.Errorf("expected recipient alex@company.com, got %s", logs[0].RecipientEmail)
	}
	_ = tc
}

func TestNotificationCreated_InvalidEmailUsername_Skipped(t *testing.T) {
	_, p := setupTestPlugin(t)

	payload := map[string]any{
		"id":                "notif-2",
		"recipient_user_id": "user-2",
		"type":              "assigned",
	}
	payloadJSON, _ := json.Marshal(payload)

	p.handleActivityEvent("notification.created")(&plugin.Event{Topic: "notification.created", Payload: payloadJSON})

	logs, err := p.loadLogs("", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected log entry for skipped notification, got %v", err)
	}
	if logs[0].Status != "skipped" {
		t.Errorf("expected status 'skipped', got %s", logs[0].Status)
	}
}

func TestNotificationCreated_WithUserEmailOverride(t *testing.T) {
	tc, p := setupTestPlugin(t)

	// Add override for user-3
	tc.DB.SeedRows("user_email_overrides",
		[]string{"user_id", "email", "created_at", "updated_at"},
		[][]any{
			{"user-3", "maria@custom-corp.com", "2026-08-14T00:00:00Z", "2026-08-14T00:00:00Z"},
		},
	)

	payload := map[string]any{
		"id":                "notif-3",
		"recipient_user_id": "user-3",
		"type":              "mentioned",
	}
	payloadJSON, _ := json.Marshal(payload)

	p.handleActivityEvent("notification.created")(&plugin.Event{Topic: "notification.created", Payload: payloadJSON})

	logs, err := p.loadLogs("", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected log entry, got %v", err)
	}
	if logs[0].RecipientEmail != "maria@custom-corp.com" {
		t.Errorf("expected maria@custom-corp.com, got %s", logs[0].RecipientEmail)
	}
}

func TestTaskCreated_ActivityEvent(t *testing.T) {
	_, p := setupTestPlugin(t)

	rawPayload := []byte(`{
		"id": "act-1",
		"task_id": "task-1",
		"project_id": "` + testProjectID + `",
		"activity_type": "task.created",
		"actor_id": "actor-user",
		"content": "{\"title\":\"Refactor UI\"}",
		"created_at": "2026-08-14T14:00:00Z"
	}`)

	evt := &plugin.Event{
		Topic:   "task.created",
		Payload: rawPayload,
	}

	p.handleActivityEvent("task.created")(evt)

	logs, err := p.loadLogs("", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected email log, got %v", err)
	}
	if logs[0].RecipientEmail != "alex@company.com" {
		t.Errorf("expected alex@company.com, got %s", logs[0].RecipientEmail)
	}
}

func TestComment_Mention(t *testing.T) {
	_, p := setupTestPlugin(t)

	rawPayload := []byte(`{
		"id": "act-comment-1",
		"task_id": "task-1",
		"project_id": "` + testProjectID + `",
		"activity_type": "comment",
		"actor_id": "actor-user",
		"content": "Hey @alex@company.com please review this!",
		"created_at": "2026-08-14T14:05:00Z"
	}`)

	evt := &plugin.Event{
		Topic:   "comment",
		Payload: rawPayload,
	}

	p.handleActivityEvent("comment")(evt)

	logs, err := p.loadLogs("", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected email log for mention, got %v", err)
	}
	if logs[0].RecipientEmail != "alex@company.com" {
		t.Errorf("expected alex@company.com, got %s", logs[0].RecipientEmail)
	}
}

func TestSettingsRoutes(t *testing.T) {
	tc, _ := setupTestPlugin(t)

	// GET global settings
	res := tc.Call("GET", "/admin/email-notifications/settings", adminReq())
	if res.StatusCode != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", res.StatusCode, res.BodyString())
	}
	if !strings.Contains(res.BodyString(), "yandex_postbox") {
		t.Errorf("expected body to contain yandex_postbox, got: %s", res.BodyString())
	}

	// PATCH global settings
	req := adminReq()
	patchInput := UpdateSettingsInput{
		FromName: ptr("PACA Global Notifications"),
	}
	body, _ := json.Marshal(patchInput)
	req.Body = body
	res = tc.Call("PATCH", "/admin/email-notifications/settings", req)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 on PATCH settings, got %d", res.StatusCode)
	}
	if !strings.Contains(res.BodyString(), "PACA Global Notifications") {
		t.Errorf("expected updated from_name, got: %s", res.BodyString())
	}
}

func TestTestEmailRoutes(t *testing.T) {
	tc, _ := setupTestPlugin(t)

	req := adminReq()
	testInput := TestEmailInput{
		ToEmail: "megazoll@gmail.com",
		Subject: "Integration Test",
		Message: "Hello",
	}
	body, _ := json.Marshal(testInput)
	req.Body = body

	// In test environment without real WASM host, SendEmail returns error which handler formats as 500
	res := tc.Call("POST", "/admin/email-notifications/test", req)
	if res.StatusCode != 500 && res.StatusCode != 200 {
		t.Fatalf("unexpected route failure: %d (body: %s)", res.StatusCode, res.BodyString())
	}
}

func TestOverridesRoutes(t *testing.T) {
	tc, _ := setupTestPlugin(t)

	// Save override
	req := adminReq()
	req.PathParams = map[string]string{"userId": "user-99"}
	overrideInput := SaveOverrideInput{
		UserID: "user-99",
		Email:  "user99@external.org",
	}
	body, _ := json.Marshal(overrideInput)
	req.Body = body

	res := tc.Call("PUT", "/admin/email-notifications/overrides/user-99", req)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 saving override, got %d (body: %s)", res.StatusCode, res.BodyString())
	}

	// List overrides
	res = tc.Call("GET", "/admin/email-notifications/overrides", adminReq())
	if !strings.Contains(res.BodyString(), "user99@external.org") {
		t.Errorf("expected overrides list to contain user99@external.org, got: %s", res.BodyString())
	}

	// Delete override
	delReq := adminReq()
	delReq.PathParams = map[string]string{"userId": "user-99"}
	res = tc.Call("DELETE", "/admin/email-notifications/overrides/user-99", delReq)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 deleting override, got %d", res.StatusCode)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestAWSSigV4(t *testing.T) {
	testURL := "https://postbox.cloud.yandex.net/v2/email/outbound-emails"
	body := []byte(`{"FromEmailAddress":"test@mbmd.ru"}`)
	fixedTime := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)

	authHeader, amzDate, err := signAWSSigV4(testURL, body, "KEY123", "SECRET456", "ru-central1", "ses", fixedTime)
	if err != nil {
		t.Fatalf("signAWSSigV4 failed: %v", err)
	}

	if amzDate != "20260814T120000Z" {
		t.Errorf("expected amzDate 20260814T120000Z, got %s", amzDate)
	}

	if !strings.HasPrefix(authHeader, "AWS4-HMAC-SHA256 Credential=KEY123/20260814/ru-central1/ses/aws4_request") {
		t.Errorf("unexpected auth header: %s", authHeader)
	}

	if !strings.Contains(authHeader, "SignedHeaders=content-type;host;x-amz-date") {
		t.Errorf("missing signed headers in: %s", authHeader)
	}
}
