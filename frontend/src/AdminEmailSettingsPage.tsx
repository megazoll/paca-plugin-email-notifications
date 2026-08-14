import { useState, useEffect } from "react";
import { PluginQueryClientProvider } from "@paca-ai/plugin-sdk-react";
import type { AdminPageProps } from "@paca-ai/plugin-sdk-react";
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
  Users,
  Plus,
  Trash2,
} from "lucide-react";
import {
  useAdminEmailSettings,
  useUpdateAdminEmailSettings,
  useAdminEmailLogs,
  useSendAdminTestEmail,
  useAdminEmailOverrides,
  useSaveAdminEmailOverride,
  useDeleteAdminEmailOverride,
} from "./api";
import type { UpdateSettingsInput } from "./types";

export default function AdminEmailSettingsPage(props: AdminPageProps) {
  return (
    <PluginQueryClientProvider>
      <Content {...props} />
    </PluginQueryClientProvider>
  );
}

function Content(props: AdminPageProps) {
  const { api } = props;

  const { data: settings, isLoading: loadingSettings, refetch: refetchSettings } =
    useAdminEmailSettings(api);
  const updateSettingsMutation = useUpdateAdminEmailSettings(api);
  const { data: logs = [], isLoading: loadingLogs, refetch: refetchLogs } =
    useAdminEmailLogs(api);
  const sendTestMutation = useSendAdminTestEmail(api);
  const { data: overrides = [], isLoading: loadingOverrides, refetch: refetchOverrides } =
    useAdminEmailOverrides(api);
  const saveOverrideMutation = useSaveAdminEmailOverride(api);
  const deleteOverrideMutation = useDeleteAdminEmailOverride(api);

  const [formData, setFormData] = useState<UpdateSettingsInput>({
    enabled: true,
    host: "",
    port: 587,
    username: "",
    password: "",
    from_email: "notifications@paca.local",
    from_name: "Paca",
    security: "starttls",
    notify_on_assigned: true,
    notify_on_mentioned: true,
  });

  const [testRecipient, setTestRecipient] = useState("");
  const [saveSuccessMessage, setSaveSuccessMessage] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  // New Override State
  const [newUserId, setNewUserId] = useState("");
  const [newUserEmail, setNewUserEmail] = useState("");
  const [overrideError, setOverrideError] = useState<string | null>(null);

  useEffect(() => {
    if (settings) {
      setFormData({
        enabled: settings.enabled,
        host: settings.host || "",
        port: settings.port || 587,
        username: settings.username || "",
        password: settings.password || "",
        from_email: settings.from_email || "notifications@paca.local",
        from_name: settings.from_name || "Paca",
        security: settings.security || "starttls",
        notify_on_assigned: settings.notify_on_assigned,
        notify_on_mentioned: settings.notify_on_mentioned,
      });
    }
  }, [settings]);

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaveSuccessMessage(null);
    try {
      await updateSettingsMutation.mutateAsync(formData);
      setSaveSuccessMessage("Global SMTP configuration saved successfully.");
      setTimeout(() => setSaveSuccessMessage(null), 4000);
    } catch (err: any) {
      console.error("Failed to save admin settings:", err);
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
        message: `Global test email successfully dispatched to ${testRecipient.trim()}.`,
      });
    } catch (err: any) {
      setTestResult({
        success: false,
        message: err?.message || "Failed to send test email. Verify SMTP host, port, credentials, and security.",
      });
    }
  };

  const handleAddOverride = async (e: React.FormEvent) => {
    e.preventDefault();
    setOverrideError(null);
    if (!newUserId.trim() || !newUserEmail.trim()) return;
    try {
      await saveOverrideMutation.mutateAsync({
        userId: newUserId.trim(),
        email: newUserEmail.trim(),
      });
      setNewUserId("");
      setNewUserEmail("");
    } catch (err: any) {
      setOverrideError(err?.message || "Failed to add email override.");
    }
  };

  const handleDeleteOverride = async (userId: string) => {
    try {
      await deleteOverrideMutation.mutateAsync(userId);
    } catch (err: any) {
      console.error("Failed to delete override:", err);
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
    <div className="max-w-5xl space-y-8 p-8">
      {/* Page Title */}
      <div className="flex items-center gap-3 border-b border-border pb-6">
        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
          <Mail className="h-6 w-6" />
        </div>
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-foreground">
            Email Notifications & SMTP Gateway
          </h1>
          <p className="text-sm text-muted-foreground">
            Configure system-wide SMTP mail routing, user email overrides, and audit notification delivery logs.
          </p>
        </div>
      </div>

      {/* Info Card */}
      <div className="flex items-start gap-3.5 rounded-xl border border-border bg-card/70 p-5 text-sm text-foreground shadow-sm">
        <Info className="mt-0.5 h-5 w-5 shrink-0 text-blue-500" />
        <div>
          <div className="font-semibold text-foreground mb-1">Username as Email Policy</div>
          <p className="text-muted-foreground leading-relaxed">
            Paca automatically checks if a user's <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">username</code> is a valid email address (e.g., <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">user@organization.com</code>). If it is, notifications are sent directly to that address. If a user has a plain handle (e.g., <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-foreground">admin</code>), you can define a mapping in the <strong>User Email Overrides</strong> section below.
          </p>
        </div>
      </div>

      {/* Section 1: Global SMTP Server Configuration */}
      <form onSubmit={handleSave} className="space-y-6">
        <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Server className="h-5 w-5 text-primary" />
              <h2 className="text-base font-semibold text-foreground">Global SMTP Server</h2>
            </div>
            <label className="flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={formData.enabled}
                onChange={(e) => setFormData({ ...formData, enabled: e.target.checked })}
                className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
              />
              <span className="text-sm font-medium text-foreground">Enable Email Delivery</span>
            </label>
          </div>

          <div className="grid grid-cols-1 gap-4 pt-2 sm:grid-cols-2">
            <div>
              <label className="block text-xs font-medium text-foreground">SMTP Host / Server</label>
              <input
                type="text"
                required
                placeholder="smtp.mailgun.org, smtp.sendgrid.net, etc."
                value={formData.host || ""}
                onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-foreground">Port</label>
              <input
                type="number"
                required
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
                <option value="none">Plain SMTP (Port 25)</option>
              </select>
            </div>

            <div>
              <label className="block text-xs font-medium text-foreground">SMTP Username / API User</label>
              <input
                type="text"
                placeholder="postmaster@yourdomain.com"
                value={formData.username || ""}
                onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-foreground">SMTP Password / API Key</label>
              <input
                type="password"
                placeholder="••••••••••••••••"
                value={formData.password || ""}
                onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-foreground">Default Sender Email (From)</label>
              <input
                type="email"
                required
                placeholder="notifications@paca.local"
                value={formData.from_email || ""}
                onChange={(e) => setFormData({ ...formData, from_email: e.target.value })}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            <div className="sm:col-span-2">
              <label className="block text-xs font-medium text-foreground">Default Sender Name</label>
              <input
                type="text"
                placeholder="Paca Notifications"
                value={formData.from_name || ""}
                onChange={(e) => setFormData({ ...formData, from_name: e.target.value })}
                className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <div className="mt-6 border-t border-border pt-4">
            <div className="mb-3 flex items-center gap-2">
              <Bell className="h-4 w-4 text-muted-foreground" />
              <h3 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Default Notification Triggers
              </h3>
            </div>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <label className="flex items-center gap-2.5 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.notify_on_assigned}
                  onChange={(e) => setFormData({ ...formData, notify_on_assigned: e.target.checked })}
                  className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                />
                <span className="text-sm text-foreground">Notify assignee when assigned to a task</span>
              </label>

              <label className="flex items-center gap-2.5 cursor-pointer">
                <input
                  type="checkbox"
                  checked={formData.notify_on_mentioned}
                  onChange={(e) => setFormData({ ...formData, notify_on_mentioned: e.target.checked })}
                  className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                />
                <span className="text-sm text-foreground">Notify member when mentioned in comment / task</span>
              </label>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-4">
          <button
            type="submit"
            disabled={updateSettingsMutation.isPending}
            className="flex items-center gap-2 rounded-md bg-primary px-5 py-2.5 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            {updateSettingsMutation.isPending ? (
              <RefreshCw className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            Save Global Settings
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

      {/* Section 2: Global Test Email */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
        <div className="mb-2 flex items-center gap-2">
          <Send className="h-5 w-5 text-primary" />
          <h2 className="text-base font-semibold text-foreground">Test Mail Delivery</h2>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          Verify outbound SMTP server connectivity and TLS handshake by sending a test message.
        </p>

        <form onSubmit={handleSendTest} className="flex flex-col gap-3 sm:flex-row">
          <input
            type="email"
            required
            placeholder="admin@yourcompany.com"
            value={testRecipient}
            onChange={(e) => setTestRecipient(e.target.value)}
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
          />
          <button
            type="submit"
            disabled={sendTestMutation.isPending || !testRecipient.trim()}
            className="flex items-center justify-center gap-2 rounded-md bg-secondary px-5 py-2 text-sm font-medium text-secondary-foreground shadow-sm hover:bg-secondary/80 disabled:opacity-50"
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

      {/* Section 3: User Email Overrides */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
        <div className="mb-2 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Users className="h-5 w-5 text-primary" />
            <h2 className="text-base font-semibold text-foreground">User Email Overrides</h2>
          </div>
          <button
            onClick={() => refetchOverrides()}
            className="flex items-center gap-1.5 rounded-md border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loadingOverrides ? "animate-spin" : ""}`} />
            Refresh
          </button>
        </div>
        <p className="text-xs text-muted-foreground mb-4">
          Override or assign an explicit email address for users whose system username is not an email.
        </p>

        {/* Add Override Form */}
        <form onSubmit={handleAddOverride} className="mb-6 flex flex-col gap-3 rounded-lg border border-border bg-muted/20 p-4 sm:flex-row sm:items-end">
          <div className="flex-1">
            <label className="block text-xs font-medium text-foreground">User ID</label>
            <input
              type="text"
              required
              placeholder="e.g. 0192e2fb-..."
              value={newUserId}
              onChange={(e) => setNewUserId(e.target.value)}
              className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-1.5 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <div className="flex-1">
            <label className="block text-xs font-medium text-foreground">Email Address</label>
            <input
              type="email"
              required
              placeholder="user@example.com"
              value={newUserEmail}
              onChange={(e) => setNewUserEmail(e.target.value)}
              className="mt-1 block w-full rounded-md border border-input bg-background px-3 py-1.5 text-sm text-foreground shadow-sm focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary"
            />
          </div>
          <button
            type="submit"
            disabled={saveOverrideMutation.isPending || !newUserId.trim() || !newUserEmail.trim()}
            className="flex items-center justify-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow transition-colors hover:bg-primary/90 disabled:opacity-50"
          >
            <Plus className="h-4 w-4" /> Add Mapping
          </button>
        </form>

        {overrideError && (
          <div className="mb-4 text-xs font-medium text-destructive">
            {overrideError}
          </div>
        )}

        {overrides.length === 0 ? (
          <div className="py-6 text-center text-xs text-muted-foreground">
            No email overrides configured. All users with valid email usernames will receive notifications directly.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-border bg-muted/50 text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 font-medium">User ID</th>
                  <th className="px-3 py-2 font-medium">Mapped Email</th>
                  <th className="px-3 py-2 font-medium">Updated At</th>
                  <th className="px-3 py-2 font-medium text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {overrides.map((ov) => (
                  <tr key={ov.user_id} className="hover:bg-muted/30">
                    <td className="px-3 py-2.5 font-mono text-foreground">{ov.user_id}</td>
                    <td className="px-3 py-2.5 font-medium text-primary">{ov.email}</td>
                    <td className="px-3 py-2.5 text-muted-foreground">
                      {new Date(ov.updated_at).toLocaleString()}
                    </td>
                    <td className="px-3 py-2.5 text-right">
                      <button
                        onClick={() => handleDeleteOverride(ov.user_id)}
                        disabled={deleteOverrideMutation.isPending}
                        className="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                        title="Delete override"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Section 4: System Audit Logs */}
      <div className="rounded-xl border border-border bg-card p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-base font-semibold text-foreground">Global Email Delivery Logs</h2>
            <p className="text-xs text-muted-foreground">Recent 100 outbound notification emails across all projects.</p>
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
            No outbound emails recorded yet.
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-border bg-muted/50 text-muted-foreground">
                <tr>
                  <th className="px-3 py-2 font-medium">Time</th>
                  <th className="px-3 py-2 font-medium">Project ID</th>
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
                    <td className="px-3 py-2.5 font-mono text-muted-foreground">
                      {log.project_id || "Global"}
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
