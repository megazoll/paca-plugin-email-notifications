import {
  PluginAPIClient,
  type PluginMCPContext,
  type PluginMCPEntry,
  type Tool,
  errorResult,
  textResult,
} from "@paca-ai/plugin-sdk-mcp";

interface SMTPSettings {
  id: string;
  project_id?: string;
  provider: string;
  endpoint: string;
  api_key?: string;
  from_email: string;
  from_name: string;
  notify_on_assigned: boolean;
  notify_on_mentioned: boolean;
  notify_on_update: boolean;
}

interface EmailLog {
  id: string;
  project_id?: string;
  recipient_email: string;
  notification_type: string;
  subject: string;
  status: string;
  error_message?: string;
  created_at: string;
}

interface UserEmailOverride {
  user_id: string;
  email: string;
  updated_at: string;
}

function formatSettings(s: SMTPSettings): string {
  return [
    `Project: ${s.project_id || "Global"}`,
    `Provider: ${s.provider}`,
    `Endpoint: ${s.endpoint}`,
    `API Key: ${s.api_key ? "configured" : "none"}`,
    `From: ${s.from_name} <${s.from_email}>`,
    `Notify on Assigned: ${s.notify_on_assigned}`,
    `Notify on Mentioned: ${s.notify_on_mentioned}`,
    `Notify on Update: ${s.notify_on_update}`,
  ].join("\n");
}

function formatLog(log: EmailLog): string {
  const statusStr =
    log.status === "sent"
      ? "SENT"
      : log.status === "skipped"
      ? "SKIPPED"
      : `FAILED (${log.error_message || "Unknown error"})`;
  return `[${log.created_at}] [${log.notification_type.toUpperCase()}] To: ${log.recipient_email} | Subject: "${log.subject}" | Status: ${statusStr}`;
}

const tools: Tool[] = [
  {
    name: "email_notifications_get_settings",
    description: "Get email notification configuration for a project or global instance.",
    inputSchema: {
      type: "object",
      properties: {
        projectId: { type: "string", description: "Optional project UUID. If omitted, returns global settings." },
      },
    },
  },
  {
    name: "email_notifications_update_settings",
    description: "Update email notification provider (Yandex Cloud Postbox, Resend, SendGrid, Mailgun, Postmark, Brevo, Webhook) and triggers.",
    inputSchema: {
      type: "object",
      properties: {
        projectId: { type: "string", description: "Optional project UUID." },
        provider: { type: "string", enum: ["yandex_postbox", "resend", "sendgrid", "mailgun", "postmark", "brevo", "webhook"] },
        endpoint: { type: "string", description: "API Endpoint URL" },
        api_key: { type: "string", description: "API Key / IAM Token" },
        from_email: { type: "string", description: "Sender email address." },
        from_name: { type: "string", description: "Sender display name." },
        notify_on_assigned: { type: "boolean", description: "Send email when assigned to a task." },
        notify_on_mentioned: { type: "boolean", description: "Send email when mentioned in comments." },
        notify_on_update: { type: "boolean", description: "Send email on task updates." },
      },
    },
  },
  {
    name: "email_notifications_get_logs",
    description: "Get recent email notification delivery logs.",
    inputSchema: {
      type: "object",
      properties: {
        projectId: { type: "string", description: "Optional project UUID." },
      },
    },
  },
  {
    name: "email_notifications_send_test",
    description: "Send a diagnostic test email to verify credentials and connectivity.",
    inputSchema: {
      type: "object",
      properties: {
        to_email: { type: "string", description: "Recipient email address." },
        projectId: { type: "string", description: "Optional project UUID." },
      },
      required: ["to_email"],
    },
  },
  {
    name: "email_notifications_list_overrides",
    description: "List all user ID to email address overrides.",
    inputSchema: {
      type: "object",
      properties: {},
    },
  },
  {
    name: "email_notifications_save_override",
    description: "Set an email override for a user ID whose login is not an email.",
    inputSchema: {
      type: "object",
      properties: {
        userId: { type: "string", description: "User UUID." },
        email: { type: "string", description: "Destination email address." },
      },
      required: ["userId", "email"],
    },
  },
  {
    name: "email_notifications_delete_override",
    description: "Delete an email override for a user ID.",
    inputSchema: {
      type: "object",
      properties: {
        userId: { type: "string", description: "User UUID." },
      },
      required: ["userId"],
    },
  },
];

const entry: PluginMCPEntry = {
  tools,

  async handleToolCall(
    name: string,
    args: Record<string, unknown>,
    context: PluginMCPContext
  ) {
    const api = new PluginAPIClient(context);
    const projectId = args.projectId as string | undefined;

    try {
      switch (name) {
        case "email_notifications_get_settings": {
          const path = projectId
            ? `projects/${projectId}/email-notifications/settings`
            : "admin/email-notifications/settings";
          const res = await api.pluginGet<SMTPSettings>(path);
          return textResult(formatSettings(res));
        }

        case "email_notifications_update_settings": {
          const path = projectId
            ? `projects/${projectId}/email-notifications/settings`
            : "admin/email-notifications/settings";
          const res = await api.pluginPatch<SMTPSettings>(path, args);
          return textResult(`Settings updated successfully:\n\n${formatSettings(res)}`);
        }

        case "email_notifications_get_logs": {
          const path = projectId
            ? `projects/${projectId}/email-notifications/logs`
            : "admin/email-notifications/logs";
          const logs = await api.pluginGet<EmailLog[]>(path);
          if (!logs || logs.length === 0) {
            return textResult("No email delivery logs recorded yet.");
          }
          return textResult(logs.map(formatLog).join("\n"));
        }

        case "email_notifications_send_test": {
          const toEmail = String(args.to_email || "");
          const path = projectId
            ? `projects/${projectId}/email-notifications/test`
            : "admin/email-notifications/test";
          const res = await api.pluginPost<{ sent: boolean; recipient: string }>(path, {
            to_email: toEmail,
          });
          return textResult(`Test email successfully sent to ${res.recipient}`);
        }

        case "email_notifications_list_overrides": {
          const overrides = await api.pluginGet<UserEmailOverride[]>("admin/email-notifications/overrides");
          if (!overrides || overrides.length === 0) {
            return textResult("No email overrides configured.");
          }
          const lines = overrides.map(
            (ov: UserEmailOverride) => `User ID: ${ov.user_id} -> Email: ${ov.email} (Updated: ${ov.updated_at})`
          );
          return textResult(lines.join("\n"));
        }

        case "email_notifications_save_override": {
          const userId = String(args.userId || "");
          const email = String(args.email || "");
          await api.pluginPost(`admin/email-notifications/overrides/${userId}`, {
            user_id: userId,
            email,
          });
          return textResult(`Email override saved for user ${userId} -> ${email}`);
        }

        case "email_notifications_delete_override": {
          const userId = String(args.userId || "");
          await api.pluginDelete(`admin/email-notifications/overrides/${userId}`);
          return textResult(`Email override deleted for user ${userId}`);
        }

        default:
          return errorResult(`Unknown tool: ${name}`);
      }
    } catch (err: any) {
      const message = err instanceof Error ? err.message : String(err);
      return errorResult(`Email Notifications Error: ${message}`);
    }
  },
};

export default entry;
