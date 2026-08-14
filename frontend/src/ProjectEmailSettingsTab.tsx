import { useState, useEffect } from "react";
import { PluginQueryClientProvider } from "@paca-ai/plugin-sdk-react";
import type { ProjectPageProps } from "@paca-ai/plugin-sdk-react";
import {
  Mail,
  CheckCircle2,
  XCircle,
  RefreshCw,
  Send,
  Save,
  Server,
  Bell,
  AlertTriangle,
  Info,
} from "lucide-react";
import {
  useProjectEmailSettings,
  useUpdateProjectEmailSettings,
  useProjectEmailLogs,
  useSendProjectTestEmail,
} from "./api";
import type { UpdateSettingsInput } from "./types";

export default function ProjectEmailSettingsTab(props: ProjectPageProps) {
  return (
    <PluginQueryClientProvider>
      <Content {...props} />
    </PluginQueryClientProvider>
  );
}

function Content(props: ProjectPageProps) {
  const { api } = props;
  const projectId = api.projectId;

  const { data: settings, isLoading: loadingSettings, refetch: refetchSettings } =
    useProjectEmailSettings(api, projectId);
  const updateSettingsMutation = useUpdateProjectEmailSettings(api, projectId);
  const { data: logs = [], isLoading: loadingLogs, refetch: refetchLogs } =
    useProjectEmailLogs(api, projectId);
  const sendTestMutation = useSendProjectTestEmail(api, projectId);

  const [formData, setFormData] = useState<UpdateSettingsInput>({
    enabled: true,
    host: "",
    port: 587,
    username: "",
    password: "",
    from_email: "",
    from_name: "",
    security: "starttls",
    notify_on_assigned: true,
    notify_on_mentioned: true,
  });

  const [useCustomSMTP, setUseCustomSMTP] = useState(false);
  const [testRecipient, setTestRecipient] = useState("");
  const [saveSuccessMessage, setSaveSuccessMessage] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  useEffect(() => {
    if (settings) {
      setFormData({
        enabled: settings.enabled,
        host: settings.host || "",
        port: settings.port || 587,
        username: settings.username || "",
        password: settings.password || "",
        from_email: settings.from_email || "",
        from_name: settings.from_name || "",
        security: settings.security || "starttls",
        notify_on_assigned: settings.notify_on_assigned,
        notify_on_mentioned: settings.notify_on_mentioned,
      });
      setUseCustomSMTP(Boolean(settings.host && settings.host.trim().length > 0));
    }
  }, [settings]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaveSuccessMessage(null);
    try {
      const payload: UpdateSettingsInput = {
        enabled: formData.enabled,
        notify_on_assigned: formData.notify_on_assigned,
        notify_on_mentioned: formData.notify_on_mentioned,
      };

      if (useCustomSMTP) {
        payload.host = formData.host;
        payload.port = Number(formData.port) || 587;
        payload.username = formData.username;
        payload.password = formData.password;
        payload.from_email = formData.from_email;
        payload.from_name = formData.from_name;
        payload.security = formData.security;
      } else {
        payload.host = "";
        payload.username = "";
        payload.password = "";
      }

      await updateSettingsMutation.mutateAsync(payload);
      setSaveSuccessMessage("Settings saved successfully.");
      setTimeout(() => setSaveSuccessMessage(null), 4000);
    } catch (err: any) {
      console.error("Failed to save project email settings:", err);
    }
  };

  const handleSendTest = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!testRecipient.trim()) return;
    setTestResult(null);
    try {
      await sendTestMutation.mutateAsync(testRecipient.trim());
      setTestResult({
        success: true,
        message: `Test email sent successfully to ${testRecipient.trim()}.`,
      });
    } catch (err: any) {
      setTestResult({
        success: false,
        message: err?.message || "Failed to send test email. Check SMTP settings.",
      });
    }
  };

  if (loadingSettings) {
    return (
      <div className="flex h-64 items-center justify-center">
        <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl space-y-8 p-6">
      {/* Header */}
      <div>
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Mail className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-xl font-semibold tracking-tight text-foreground">
              Email Notifications
            </h2>
            <p className="text-sm text-muted-foreground">
              Automatically duplicate task and mention notifications to project members via email.
            </p>
          </div>
        </div>
      </div>

      {/* Info notice about username as email */}
      <div className="flex items-start gap-3 rounded-lg border border-border bg-card/60 p-4 text-sm text-foreground">
        <Info className="mt-0.5 h-5 w-5 shrink-0 text-blue-500" />
        <div>
          <span className="font-medium">How email routing works:</span> Paca sends notifications to each user's <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">username</code> if it is a valid email address (e.g. <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">alice@company.com</code>). If a user's username is not an email, notifications will be skipped unless an admin configures an override for that user.
        </div>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Section 1: General Notification Triggers */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="mb-4 flex items-center gap-2">
            <Bell className="h-5 w-5 text-primary" />
            <h3 className="text-base font-semibold text-foreground">Notification Preferences</h3>
          </div>

          <div className="space-y-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <div>
                <span className="font-medium text-foreground">Enable email notifications for this project</span>
                <p className="text-xs text-muted-foreground">
                  Master switch for sending email duplicates in this project.
                </p>
              </div>
            </label>

            <div className="ml-7 space-y-3 pt-2">
              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  disabled={!formData.enabled}
                  checked={formData.notify_on_assigned}
                  onChange={(e) => setFormData({ ...formData, notify_on_assigned: e.target.checked })}
                  className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary disabled:opacity-50"
                />
                <span className="text-sm text-foreground">
                  Notify assignee when a task is assigned
                </span>
              </label>

              <label className="flex items-center gap-3 cursor-pointer">
                <input
                  type="checkbox"
                  disabled={!formData.enabled}
                  checked={formData.notify_on_mentioned}
                  onChange={(e) => setFormData({ ...formData, notify_on_mentioned: e.target.checked })}
                  className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary disabled:opacity-50"
                />
                <span className="text-sm text-foreground">
                  Notify member when @mentioned in a task comment or description
                </span>
              </label>
            </div>
          </div>
        </div>

        {/* Section 2: SMTP Delivery */}
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="mb-4 flex items-center gap-2">
            <Server className="h-5 w-5 text-primary" />
            <h3 className="text-base font-semibold text-foreground">SMTP Server Configuration</h3>
          </div>

          <div className="space-y-4">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={useCustomSMTP}
                onChange={(e) => setUseCustomSMTP(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <div>
                <span className="font-medium text-foreground">Use custom SMTP server for this project</span>
                <p className="text-xs text-muted-foreground">
                  If disabled, emails will be routed through the global SMTP server configured by administrators.
                </p>
              </div>
            </label>

            {useCustomSMTP ? (
              <div className="grid grid-cols-1 gap-4 pt-4 sm:grid-cols-2">
                <div>
                  <label className="block text-xs font-medium text-foreground">SMTP Host</label>
                  <input
                    type="text"
                    required={useCustomSMTP}
                    placeholder="smtp.example.com"
                    value={formData.host || ""}
                    onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                    className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-foreground">Port</label>
                  <input
                    type="number"
                    placeholder="587"
                    value={formData.port || 587}
                    onChange={(e) => setFormData({ ...formData, port: Number(e.target.value) })}
                    className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-foreground">Security Mode</label>
                  <select
                    value={formData.security || "starttls"}
                    onChange={(e) => setFormData({ ...formData, security: e.target.value as any })}
                    className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  >
                    <option value="starttls">STARTTLS (Port 587 recommended)</option>
                    <option value="tls">TLS / SSL (Port 465 recommended)</option>
                    <option value="none">None (Plain SMTP, Port 25)</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-medium text-foreground">SMTP Username</label>
                  <input
                    type="text"
                    placeholder="smtp-user@example.com"
                    value={formData.username || ""}
                    onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                    className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-foreground">SMTP Password</label>
                  <input
                    type="password"
                    placeholder="••••••••••••"
                    value={formData.password || ""}
                    onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                    className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>

                <div>
                  <label className="block text-xs font-medium text-foreground">From Email</label>
                  <input
                    type="email"
                    placeholder="notifications@myproject.org"
                    value={formData.from_email || ""}
                    onChange={(e) => setFormData({ ...formData, from_email: e.target.value })}
                    className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>

                <div className="sm:col-span-2">
                  <label className="block text-xs font-medium text-foreground">From Name</label>
                  <input
                    type="text"
                    placeholder="Project Notifications"
                    value={formData.from_name || ""}
                    onChange={(e) => setFormData({ ...formData, from_name: e.target.value })}
                    className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </div>
            ) : (
              <div className="rounded-lg bg-muted/40 p-4 text-xs text-muted-foreground">
                Using global server settings. Emails will be sent from the administrator-configured mail gateway.
              </div>
            )}
          </div>
        </div>

        {/* Save Button & Feedback */}
        <div className="flex items-center gap-4">
          <button
            type="submit"
            disabled={updateSettingsMutation.isPending}
            className="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            {updateSettingsMutation.isPending ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Save Settings
          </button>

          {saveSuccessMessage && (
            <span className="flex items-center gap-1.5 text-sm font-medium text-green-600">
              <CheckCircle2 className="h-4 w-4" /> {saveSuccessMessage}
            </span>
          )}

          {updateSettingsMutation.isError && (
            <span className="flex items-center gap-1.5 text-sm font-medium text-destructive">
              <XCircle className="h-4 w-4" /> Failed to save settings.
            </span>
          )}
        </div>
      </form>

      {/* Section 3: Test Email */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
        <div className="mb-3 flex items-center gap-2">
          <Send className="h-5 w-5 text-primary" />
          <h3 className="text-base font-semibold text-foreground">Send Test Email</h3>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          Send a sample test email to verify your project SMTP connectivity and delivery.
        </p>

        <form onSubmit={handleSendTest} className="flex flex-col gap-3 sm:flex-row">
          <input
            type="email"
            required
            placeholder="recipient@example.com"
            value={testRecipient}
            onChange={(e) => setTestRecipient(e.target.value)}
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
          <button
            type="submit"
            disabled={sendTestMutation.isPending || !testRecipient.trim()}
            className="flex items-center justify-center gap-2 rounded-md bg-secondary px-4 py-2 text-sm font-medium text-secondary-foreground shadow-sm hover:bg-secondary/80 disabled:opacity-50"
          >
            {sendTestMutation.isPending ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
            Send Test
          </button>
        </form>

        {testResult && (
          <div
            className={`mt-4 flex items-start gap-2 rounded-lg p-3 text-sm ${
              testResult.success
                ? "bg-green-50 text-green-800 dark:bg-green-950/40 dark:text-green-300"
                : "bg-red-50 text-red-800 dark:bg-red-950/40 dark:text-red-300"
            }`}
          >
            {testResult.success ? (
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-green-600 dark:text-green-400" />
            ) : (
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-red-600 dark:text-red-400" />
            )}
            <span>{testResult.message}</span>
          </div>
        )}
      </div>

      {/* Section 4: Email Delivery Logs */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h3 className="text-base font-semibold text-foreground">Recent Email Logs</h3>
          </div>
          <button
            onClick={() => refetchLogs()}
            className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loadingLogs ? "animate-spin" : ""}`} />
            Refresh
          </button>
        </div>

        {logs.length === 0 ? (
          <div className="py-8 text-center text-sm text-muted-foreground">
            No email notifications recorded for this project yet.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-border bg-muted/50 text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 font-medium">Time</th>
                  <th className="px-3 py-2 font-medium">Type</th>
                  <th className="px-3 py-2 font-medium">Recipient</th>
                  <th className="px-3 py-2 font-medium">Subject</th>
                  <th className="px-3 py-2 font-medium">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {logs.map((log) => (
                  <tr key={log.id} className="hover:bg-muted/30">
                    <td className="whitespace-nowrap px-3 py-2.5 text-muted-foreground">
                      {new Date(log.created_at).toLocaleString()}
                    </td>
                    <td className="px-3 py-2.5">
                      <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-[10px] font-medium text-blue-800 dark:bg-blue-900/40 dark:text-blue-300">
                        {log.notification_type}
                      </span>
                    </td>
                    <td className="px-3 py-2.5 font-medium text-foreground">{log.recipient_email}</td>
                    <td className="max-w-xs truncate px-3 py-2.5 text-foreground" title={log.subject}>
                      {log.subject}
                    </td>
                    <td className="px-3 py-2.5">
                      {log.status === "sent" ? (
                        <span className="inline-flex items-center gap-1 text-green-600 dark:text-green-400">
                          <CheckCircle2 className="h-3.5 w-3.5" /> Sent
                        </span>
                      ) : (
                        <span
                          className="inline-flex items-center gap-1 text-destructive"
                          title={log.error_message || "Failed"}
                        >
                          <XCircle className="h-3.5 w-3.5" /> Failed
                        </span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
