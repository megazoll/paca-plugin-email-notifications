---
name: paca-email-notifications
description: Email notification manager for PACA AI (Email Notifications plugin). Allows configuring email providers (Yandex Cloud Postbox, Resend, SendGrid, Mailgun, Postmark, Brevo, Webhooks), managing notification triggers (assignment, mentions), routing via username email or overrides, and inspecting delivery logs.
---

# PACA Email Notifications Plugin

Duplicates PACA notifications (task assignments, mentions, updates) to user email addresses via Yandex Cloud Postbox (AWS SES REST API), Resend, SendGrid, Mailgun, Postmark, Brevo, or Webhooks.

## Key Concepts

- **Username as Email**: Uses the recipient's `username` as their email address if it is formatted as a valid email address (e.g. `user@company.com`). If it is not (e.g. `admin`), email sending is safely skipped unless an explicit override is configured.
- **User Email Overrides**: Maps user IDs to custom email addresses when usernames are handles.
- **Email Providers**:
  - ⭐️ **Yandex Cloud Postbox**: AWS SES v2 REST API compatible (`https://postbox.cloud.yandex.net/v2/email/outbound-emails`) via `paca.fetch`.
  - **Resend**: `https://api.resend.com/emails`
  - **SendGrid**: `https://api.sendgrid.com/v3/mail/send`
  - **Mailgun**: `https://api.mailgun.net/v3/.../messages`
  - **Postmark**: `https://api.postmarkapp.com/email`
  - **Brevo**: `https://api.brevo.com/v3/smtp/email`
  - **Custom Webhook**: Any HTTP POST endpoint or SMTP Relay.
- **Project Overrides**: Individual projects can inherit global settings or configure project-specific credentials.

## Development Workflow

```bash
# Frontend
cd frontend
bun install
bun run build

# MCP Server
cd mcp
bun install
bun run build

# Backend WASM Plugin
cd backend
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o backend.wasm .
```

## Available MCP Tools

- `email_notifications_get_settings` — Get email notification settings for a project or global instance.
- `email_notifications_update_settings` — Update email notification provider, credentials, and triggers.
- `email_notifications_get_logs` — Inspect recent outbound email notification delivery logs.
- `email_notifications_send_test` — Send a test email to verify credentials and connectivity.
- `email_notifications_list_overrides` — List user email override mappings.
- `email_notifications_save_override` — Set an email override for a user ID.
- `email_notifications_delete_override` — Delete an email override for a user ID.
