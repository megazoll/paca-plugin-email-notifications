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
		[]string{"id", "task_number", "title"},
		[][]any{
			{"task-1", 42, "Refactor UI"},
		},
	)
	tc.DB.SeedRows("project_members",
		[]string{"id", "project_id", "user_id"},
		[][]any{
			{"pm-1", testProjectID, "user-1"},
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

func TestNotificationCreated_ValidEmailUsername_SendsEmail(t *testing.T) {
	tc, p := setupTestPlugin(t)

	var sentReq fetchHostRequest
	mockHTTPClient = func(req fetchHostRequest) (*fetchHostResponse, error) {
		sentReq = req
		return &fetchHostResponse{
			Status: 200,
			Body:   `{"MessageId":"msg-12345"}`,
		}, nil
	}
	defer func() { mockHTTPClient = nil }()

	payload := NotificationEventPayload{
		ID:              "notif-1",
		RecipientUserID: "user-1",
		ProjectID:       testProjectID,
		TaskID:          "task-1",
		Type:            "assigned",
		CreatedAt:       "2026-08-14T12:00:00Z",
	}
	payloadJSON, _ := json.Marshal(payload)

	evt := &plugin.Event{
		Topic:   "notification.created",
		Payload: payloadJSON,
	}

	p.onNotificationCreated(evt)

	if sentReq.URL != "https://postbox.cloud.yandex.net/v2/email/outbound-emails" {
		t.Fatalf("expected send request to yandex postbox endpoint, got: %s", sentReq.URL)
	}

	if !strings.Contains(sentReq.Body, "alex@company.com") {
		t.Errorf("expected body to contain alex@company.com, got: %s", sentReq.Body)
	}

	// Verify log recorded
	logs, err := p.loadLogs("", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected email log to be recorded, got: %v (logs: %v)", err, logs)
	}
	if logs[0].Status != "sent" {
		t.Errorf("expected status 'sent', got %v", logs[0].Status)
	}
	_ = tc
}

func TestNotificationCreated_InvalidEmailUsername_Skipped(t *testing.T) {
	_, p := setupTestPlugin(t)

	sentCount := 0
	mockHTTPClient = func(req fetchHostRequest) (*fetchHostResponse, error) {
		sentCount++
		return &fetchHostResponse{Status: 200, Body: "{}"}, nil
	}
	defer func() { mockHTTPClient = nil }()

	payload := NotificationEventPayload{
		ID:              "notif-2",
		RecipientUserID: "user-2",
		Type:            "assigned",
	}
	payloadJSON, _ := json.Marshal(payload)

	p.onNotificationCreated(&plugin.Event{Topic: "notification.created", Payload: payloadJSON})

	if sentCount > 0 {
		t.Errorf("expected 0 emails sent for non-email username, got %d", sentCount)
	}
}

func TestNotificationCreated_WithUserEmailOverride_SendsEmail(t *testing.T) {
	tc, p := setupTestPlugin(t)

	var sentReq fetchHostRequest
	mockHTTPClient = func(req fetchHostRequest) (*fetchHostResponse, error) {
		sentReq = req
		return &fetchHostResponse{Status: 200, Body: `{"MessageId":"123"}`}, nil
	}
	defer func() { mockHTTPClient = nil }()

	// Add override for user-3
	tc.DB.SeedRows("user_email_overrides",
		[]string{"user_id", "email", "created_at", "updated_at"},
		[][]any{
			{"user-3", "maria@custom-corp.com", "2026-08-14T00:00:00Z", "2026-08-14T00:00:00Z"},
		},
	)

	payload := NotificationEventPayload{
		ID:              "notif-3",
		RecipientUserID: "user-3",
		Type:            "mentioned",
	}
	payloadJSON, _ := json.Marshal(payload)

	p.onNotificationCreated(&plugin.Event{Topic: "notification.created", Payload: payloadJSON})

	if !strings.Contains(sentReq.Body, "maria@custom-corp.com") {
		t.Fatalf("expected send request to override email maria@custom-corp.com, got body: %s", sentReq.Body)
	}
}

func TestSettingsRoutes(t *testing.T) {
	tc, _ := setupTestPlugin(t)

	// 1. Get admin settings
	req := adminReq()
	res := tc.Call("GET", "/admin/email-notifications/settings", req)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200, got %d (body: %s)", res.StatusCode, res.BodyString())
	}

	// 2. Update admin settings (switch to Resend)
	prov := ProviderResend
	updateInput := UpdateSettingsInput{
		Provider:  &prov,
		APIKey:    ptr("re_test_12345"),
		FromEmail: ptr("alerts@mydomain.com"),
	}
	body, _ := json.Marshal(updateInput)
	req.Body = body
	res = tc.Call("PATCH", "/admin/email-notifications/settings", req)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 update, got %d (body: %s)", res.StatusCode, res.BodyString())
	}

	// 3. Verify updated
	res = tc.Call("GET", "/admin/email-notifications/settings", adminReq())
	if !strings.Contains(res.BodyString(), "resend") || !strings.Contains(res.BodyString(), "alerts@mydomain.com") {
		t.Errorf("expected updated settings in response, got: %s", res.BodyString())
	}
}

func TestTestEmailRoutes(t *testing.T) {
	tc, _ := setupTestPlugin(t)

	sentCount := 0
	mockHTTPClient = func(req fetchHostRequest) (*fetchHostResponse, error) {
		sentCount++
		return &fetchHostResponse{Status: 200, Body: `{"status":"ok"}`}, nil
	}
	defer func() { mockHTTPClient = nil }()

	req := adminReq()
	testInput := TestEmailInput{
		ToEmail: "test.receiver@company.com",
	}
	body, _ := json.Marshal(testInput)
	req.Body = body

	res := tc.Call("POST", "/admin/email-notifications/test", req)
	if res.StatusCode != 200 {
		t.Fatalf("expected 200 test email dispatch, got %d (body: %s)", res.StatusCode, res.BodyString())
	}
	if sentCount != 1 {
		t.Errorf("expected 1 test email sent, got %d", sentCount)
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

func TestTaskCreated_ActivityEvent_SendsEmail(t *testing.T) {
	tc, p := setupTestPlugin(t)

	// Seed task_assignees
	tc.DB.SeedRows("task_assignees",
		[]string{"task_id", "member_id"},
		[][]any{
			{"task-1", "pm-1"},
		},
	)

	var sentReq fetchHostRequest
	mockHTTPClient = func(req fetchHostRequest) (*fetchHostResponse, error) {
		sentReq = req
		return &fetchHostResponse{
			Status: 200,
			Body:   `{"MessageId":"msg-task-created"}`,
		}, nil
	}
	defer func() { mockHTTPClient = nil }()

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

	p.onTaskActivityEvent(evt)

	if sentReq.URL != "https://postbox.cloud.yandex.net/v2/email/outbound-emails" {
		t.Fatalf("expected send request to yandex postbox endpoint, got: %s", sentReq.URL)
	}

	if !strings.Contains(sentReq.Body, "alex@company.com") {
		t.Errorf("expected body to contain alex@company.com, got: %s", sentReq.Body)
	}

	logs, err := p.loadLogs("", 10)
	if err != nil || len(logs) == 0 {
		t.Fatalf("expected email log, got %v", err)
	}
	if logs[0].Status != "sent" {
		t.Errorf("expected status 'sent', got %v", logs[0].Status)
	}
}

func TestComment_Mention_SendsEmail(t *testing.T) {
	_, p := setupTestPlugin(t)

	var sentReq fetchHostRequest
	mockHTTPClient = func(req fetchHostRequest) (*fetchHostResponse, error) {
		sentReq = req
		return &fetchHostResponse{
			Status: 200,
			Body:   `{"MessageId":"msg-comment-mention"}`,
		}, nil
	}
	defer func() { mockHTTPClient = nil }()

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

	p.onCommentActivityEvent(evt)

	if sentReq.URL != "https://postbox.cloud.yandex.net/v2/email/outbound-emails" {
		t.Fatalf("expected send request to yandex postbox, got: %s", sentReq.URL)
	}

	if !strings.Contains(sentReq.Body, "alex@company.com") {
		t.Errorf("expected body to contain alex@company.com, got: %s", sentReq.Body)
	}
}
