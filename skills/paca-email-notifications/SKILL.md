---
name: paca-email-notifications
description: Email notification manager for PACA AI (Email Notifications plugin). Allows configuring SMTP gateways, managing notification triggers (assignment, mentions), routing via username email or overrides, and inspecting delivery logs.
---

# PACA Email Notifications Plugin

Duplicates PACA notifications (task assignments, mentions) to user email addresses via native SMTP delivery.

## Key Concepts

- **Username as Email**: Uses the recipient's `username` as their email address if it is formatted as a valid email address (e.g. `user@company.com`). If it is not (e.g. `admin`), email sending is skipped unless an explicit override is configured.
- **User Email Overrides**: Maps user IDs to custom email addresses when usernames are handles.
- **SMTP Gateway**: Supports STARTTLS (587), TLS/SSL (465), and Plain SMTP (25) with custom authentication and sender headers.
- **Project Overrides**: Individual projects can inherit global SMTP settings or configure project-specific mail servers.

## Bun Development Workflow

```bash
# Frontend
cd frontend
bun install
bun run dev
bun run build

# MCP Server
cd mcp
bun install
bun run build

# Backend WASM Plugin
cd backend
GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm .
```

## Available MCP Tools

- `email_notifications_get_settings` — Get email notification and SMTP settings for a project or global instance.
- `email_notifications_update_settings` — Update email notification triggers and SMTP server configuration.
- `email_notifications_get_logs` — Inspect recent outbound email notification delivery logs.
- `email_notifications_send_test` — Send a test email to verify SMTP configuration and TLS connectivity.
- `email_notifications_list_overrides` — List user email override mappings.
- `email_notifications_save_override` — Set an email override for a user ID.
- `email_notifications_delete_override` — Delete an email override for a user ID.
