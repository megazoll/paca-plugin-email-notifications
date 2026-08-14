# PACA Email Notifications Plugin

[![backend-pr-ci](https://github.com/megazoll/paca-plugin-email-notifications/actions/workflows/backend-pr-ci.yml/badge.svg)](https://github.com/megazoll/paca-plugin-email-notifications/actions/workflows/backend-pr-ci.yml)
[![frontend-pr-ci](https://github.com/megazoll/paca-plugin-email-notifications/actions/workflows/frontend-pr-ci.yml/badge.svg)](https://github.com/megazoll/paca-plugin-email-notifications/actions/workflows/frontend-pr-ci.yml)
[![mcp-pr-ci](https://github.com/megazoll/paca-plugin-email-notifications/actions/workflows/mcp-pr-ci.yml/badge.svg)](https://github.com/megazoll/paca-plugin-email-notifications/actions/workflows/mcp-pr-ci.yml)

Duplicates PACA notifications (task assignments, mentions, and updates) to user email addresses via **Yandex Cloud Postbox**, **Resend**, **SendGrid**, **Mailgun**, **Postmark**, **Brevo**, or **Webhooks**.

## Features

- **Zero Core Changes**: Uses standard PACA `paca.fetch` host function. No kernel modifications required.
- **Yandex Cloud Postbox**: Built-in support for Yandex Cloud Postbox AWS SES REST API (`https://postbox.cloud.yandex.net/v2/email/outbound-emails`) with IAM Token / API Key authentication.
- **Popular Email APIs**: Resend, SendGrid, Mailgun, Postmark, Brevo, and Generic Webhooks.
- **Username as Email**: Uses the user's `username` as destination email if it matches an email format (e.g. `user@company.com`).
- **User Email Overrides**: Allows administrators to map text handles (e.g. `admin`) to custom email addresses.
- **Granular Triggers**: Configurable triggers for task assignment, mentions in comments, and status updates.
- **Project & Global Settings**: Configure global instance defaults or override per-project.
- **Audit Logs**: Detailed delivery log tracking sent, skipped, and failed email dispatches.
- **MCP Integration**: 7 MCP tools for autonomous AI agent workflows.

## License

[Apache-2.0](LICENSE)
