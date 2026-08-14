# Paca Email Notifications Plugin

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Version](https://img.shields.io/badge/version-0.1.0-green.svg)](plugin.json)

An official extension for **Paca AI** that automatically duplicates task assignments and user @mentions to email via native SMTP delivery.

---

## Features

- 📧 **Automatic Notification Duplication**: Automatically delivers emails when tasks are assigned or members are mentioned.
- 👤 **Username as Email Address**: Zero configuration required for users whose usernames are already valid email addresses (e.g. `alice@example.com`).
- 🛡️ **Safe Fallback**: If a username is a plain handle (e.g. `admin`, `alex123`), notifications are safely skipped by default to prevent delivery errors.
- 🔀 **User Email Overrides**: Administrators can explicitly map usernames/user IDs to custom email addresses.
- ⚙️ **Custom SMTP Gateways**: Supports STARTTLS (port 587), SSL/TLS (port 465), and Plain SMTP (port 25) with custom authentication and sender headers.
- 🏢 **Multi-level Configuration**: Global system-wide SMTP gateway with optional per-project overrides.
- 📋 **Delivery Audit Logs**: Comprehensive logging of all outbound emails, timestamps, subjects, and delivery status.
- 🧪 **Live Connection Testing**: Instantly test SMTP configurations with a single click or MCP tool call.
- 🤖 **MCP & AI Tooling**: 7 dedicated MCP tools allowing AI agents to manage email settings, send tests, and inspect delivery status.

---

## Architecture & How It Works

1. **Event Interception**: When tasks are assigned or users are mentioned, Paca core appends a `notification.created` event to `events.StreamPluginEvents`.
2. **Recipient Email Resolution**:
   - The plugin inspects the `user_email_overrides` table for an explicit mapping.
   - If no override exists, it verifies whether the recipient's `username` is a valid RFC 5322 email address.
   - If neither condition is met, delivery is skipped cleanly without error.
3. **Host SMTP Bridge**: Delivery is executed via the `paca.smtp_send` WebAssembly host function registered in the Paca core runtime.
4. **Audit Logging**: Every dispatch result (success or failure) is stored in the `email_logs` database table.

---

## MCP Tools

| Tool Name | Description |
|---|---|
| `email_notifications_get_settings` | Get email notification and SMTP settings (global or project). |
| `email_notifications_update_settings` | Update email notification triggers and SMTP server configuration. |
| `email_notifications_get_logs` | Get recent outbound email delivery logs. |
| `email_notifications_send_test` | Send a test email to verify SMTP configuration and TLS connectivity. |
| `email_notifications_list_overrides` | List user email override mappings. |
| `email_notifications_save_override` | Save an email address override for a user ID. |
| `email_notifications_delete_override` | Remove an email address override for a user ID. |

---

## Development

```bash
# Backend (Go / WebAssembly)
cd backend
go test -v ./...
GOOS=wasip1 GOARCH=wasm go build -o plugin.wasm .

# Frontend (React 19 + Module Federation)
cd frontend
bun install
bun run build

# MCP (TypeScript)
cd mcp
bun install
bun run build
```

---

## License

Apache-2.0. See [LICENSE](LICENSE) for details.
