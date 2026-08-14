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
  scope: string;
  project_id?: string;
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  from_email: string;
  from_name: string;
  security: string;
  notify_on_assigned: boolean;
  notify_on_mentioned: boolean;
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
    `Scope: ${s.scope} ${s.project_id ? `(Project: ${s.project_id})` : ""}`,
    `Enabled: ${s.enabled}`,
    `Host: ${s.host || "N/A"}`,
    `Port: ${s.port}`,
    `Security: ${s.security}`,
    `From: ${s.from_name} <${s.from_email}>`,
    `Notify on Assigned: ${s.notify_on_assigned}`,
    `Notify on Mentioned: ${s.notify_on_mentioned}`,
  ].join("\n");
}

function formatLog(log: EmailLog): string {
  const statusStr = log.status === "sent" ? "SENT" : `FAILED (${log.error_message || "Unknown error"})`;
  return `[${log.created_at}] [${log.notification_type.toUpperCase()}] To: ${log.recipient_email} | Subject: "${log.subject}" | Status: ${statusStr}`;
}

const tools: Tool[] = [
  {
    name: "email_notifications_get_settings",
    description: "Get email notification and SMTP settings for a project or global instance.",
    inputSchema: {
      type: "object",
      properties: {
        projectId: { type: "string", description: "Optional project UUID context. If omitted, returns global settings." },
      },
    },
  },
  {
    name: "email_notifications_update_settings",
    description: "Update email notification triggers and SMTP server configuration.",
    inputSchema: {
      type: "object",
      properties: {
        projectId: { type: "string", description: "Optional project UUID context." },
        enabled: { type: "boolean", description: "Enable or disable email delivery." },
        host: { type: "string", description: "SMTP server host." },
        port: { type: "number", description: "SMTP server port (e.g. 587 or 465)." },
        username: { type: "string", description: "SMTP authentication username." },
        password: { type: "string", description: "SMTP authentication password." },
        from_email: { type: "string", description: "Sender email address." },
        from_name: { type: "string", description: "Sender display name." },
        security: { type: "string", enum: ["starttls", "tls", "none"], description: "Security mode." },
        notify_on_assigned: { type: "boolean", description: "Send email when assigned to a task." },
        notify_on_mentioned: { type: "boolean", description: "Send email when mentioned in comments." },
      },
    },
  },
  {
    name: "email_notifications_get_logs",
    description: "Get recent email notification delivery logs.",
    inputSchema: {
      type: "object",
      properties: {
        projectId: { type: "string", description: "Optional project UUID context." },
      },
    },
  },
  {
    name: "email_notifications_send_test",
    description: "Send a test email to verify SMTP configuration and connectivity.",
    inputSchema: {
      type: "object",
      properties: {
        projectId: { type: "string", description: "Optional project UUID context." },
        toEmail: { type: "string", description: "Recipient email address for test message." },
      },
      required: ["toEmail"],
    },
  },
  {
    name: "email_notifications_list_overrides",
    description: "List user email override mappings.",
    inputSchema: {
      type: "object",
      properties: {},
    },
  },
  {
    name: "email_notifications_save_override",
    description: "Save an email override mapping for a user whose username is not a valid email.",
    inputSchema: {
      type: "object",
      properties: {
        userId: { type: "string", description: "User UUID." },
        email: { type: "string", description: "Valid email address." },
      },
      required: ["userId", "email"],
    },
  },
  {
    name: "email_notifications_delete_override",
    description: "Delete an email override mapping for a user.",
    inputSchema: {
      type: "object",
      properties: {
        userId: { type: "string", description: "User UUID to remove." },
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
    context: PluginMCPContext,
  ) {
    const api = new PluginAPIClient(context);

    try {
      switch (name) {
        case "email_notifications_get_settings": {
          const { projectId } = args as { projectId?: string };
          const path = projectId
            ? `projects/${projectId}/email-notifications/settings`
            : "email-notifications/admin-settings";
          const settings = await api.pluginGet<SMTPSettings>(path);
          return textResult(formatSettings(settings));
        }

        case "email_notifications_update_settings": {
          const { projectId, ...input } = args as { projectId?: string } & Record<string, unknown>;
          const path = projectId
            ? `projects/${projectId}/email-notifications/settings`
            : "email-notifications/admin-settings";
          const updated = await api.pluginPatch<SMTPSettings>(path, input);
          return textResult(`Settings updated successfully:\n\n${formatSettings(updated)}`);
        }

        case "email_notifications_get_logs": {
          const { projectId } = args as { projectId?: string };
          const path = projectId
            ? `projects/${projectId}/email-notifications/logs`
            : "email-notifications/admin-logs";
          const logs = await api.pluginGet<EmailLog[]>(path);
          if (logs.length === 0) return textResult("No email logs recorded.");
          return textResult(`Email Logs (${logs.length}):\n\n` + logs.map(formatLog).join("\n"));
        }

        case "email_notifications_send_test": {
          const { projectId, toEmail } = args as { projectId?: string; toEmail: string };
          const path = projectId
            ? `projects/${projectId}/email-notifications/test`
            : "email-notifications/admin-test";
          const res = await api.pluginPost<{ sent: boolean; recipient: string }>(path, { to_email: toEmail });
          return textResult(`Test email dispatched successfully to ${toEmail}.`);
        }

        case "email_notifications_list_overrides": {
          const overrides = await api.pluginGet<UserEmailOverride[]>("email-notifications/admin-overrides");
          if (overrides.length === 0) return textResult("No user email overrides found.");
          return textResult(
            `User Email Overrides (${overrides.length}):\n\n` +
              overrides.map((o) => `User ID: ${o.user_id} -> Email: ${o.email} (Updated: ${o.updated_at})`).join("\n"),
          );
        }

        case "email_notifications_save_override": {
          const { userId, email } = args as { userId: string; email: string };
          await api.pluginPost("email-notifications/admin-overrides", { user_id: userId, email });
          return textResult(`Email override saved: ${userId} -> ${email}`);
        }

        case "email_notifications_delete_override": {
          const { userId } = args as { userId: string };
          await api.pluginDelete(`email-notifications/admin-overrides/${userId}`);
          return textResult(`Email override deleted for user ${userId}`);
        }

        default:
          return errorResult(`Unknown tool: ${name}`);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      return errorResult(`Tool ${name} failed: ${message}`);
    }
  },
};

export default entry;
