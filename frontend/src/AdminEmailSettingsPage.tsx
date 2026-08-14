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
  Bell,
  Info,
  Users,
  Plus,
  Trash2,
  ShieldCheck,
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
import type { EmailProviderType, UpdateSettingsInput } from "./types";

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
    provider: "yandex_postbox",
    endpoint: "https://postbox.cloud.yandex.net/v2/email/outbound-emails",
    api_key: "",
    from_email: "notifications@yourdomain.com",
    from_name: "PACA Notifications",
    notify_on_assigned: true,
    notify_on_mentioned: true,
    notify_on_update: false,
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
        provider: settings.provider || "yandex_postbox",
        endpoint: settings.endpoint || "https://postbox.cloud.yandex.net/v2/email/outbound-emails",
        api_key: settings.api_key || "",
        from_email: settings.from_email || "notifications@yourdomain.com",
        from_name: settings.from_name || "PACA Notifications",
        notify_on_assigned: settings.notify_on_assigned,
        notify_on_mentioned: settings.notify_on_mentioned,
        notify_on_update: settings.notify_on_update,
      });
    }
  }, [settings]);

  const handleProviderChange = (newProvider: EmailProviderType) => {
    let defaultEndpoint = formData.endpoint;
    if (newProvider === "yandex_postbox") {
      defaultEndpoint = "https://postbox.cloud.yandex.net/v2/email/outbound-emails";
    } else if (newProvider === "resend") {
      defaultEndpoint = "https://api.resend.com/emails";
    } else if (newProvider === "sendgrid") {
      defaultEndpoint = "https://api.sendgrid.com/v3/mail/send";
    } else if (newProvider === "postmark") {
      defaultEndpoint = "https://api.postmarkapp.com/email";
    } else if (newProvider === "brevo") {
      defaultEndpoint = "https://api.brevo.com/v3/smtp/email";
    }
    setFormData({
      ...formData,
      provider: newProvider,
      endpoint: defaultEndpoint,
    });
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaveSuccessMessage(null);
    try {
      await updateSettingsMutation.mutateAsync(formData);
      setSaveSuccessMessage("Global email configuration saved successfully.");
      refetchSettings();
      setTimeout(() => setSaveSuccessMessage(null), 4000);
    } catch (err: any) {
      alert("Failed to save settings: " + (err.message || String(err)));
    }
  };

  const handleSendTest = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!testRecipient || !testRecipient.includes("@")) {
      setTestResult({ success: false, message: "Please enter a valid email address." });
      return;
    }

    setTestResult(null);
    try {
      await sendTestMutation.mutateAsync({
        to_email: testRecipient,
        provider: formData.provider,
        endpoint: formData.endpoint,
        api_key: formData.api_key,
        from_email: formData.from_email,
        from_name: formData.from_name,
      });
      setTestResult({
        success: true,
        message: `Test email successfully dispatched to ${testRecipient}!`,
      });
      refetchLogs();
    } catch (err: any) {
      setTestResult({
        success: false,
        message: `Delivery failed: ${err.message || String(err)}`,
      });
    }
  };

  const handleAddOverride = async (e: React.FormEvent) => {
    e.preventDefault();
    setOverrideError(null);
    if (!newUserId.trim() || !newUserEmail.trim()) {
      setOverrideError("User ID and email address are required.");
      return;
    }
    if (!newUserEmail.includes("@")) {
      setOverrideError("Please enter a valid email address.");
      return;
    }

    try {
      await saveOverrideMutation.mutateAsync({
        userId: newUserId.trim(),
        email: newUserEmail.trim(),
      });
      setNewUserId("");
      setNewUserEmail("");
      refetchOverrides();
    } catch (err: any) {
      setOverrideError(err.message || "Failed to save override");
    }
  };

  const handleDeleteOverride = async (userId: string) => {
    if (!confirm(`Delete email override for user ${userId}?`)) return;
    try {
      await deleteOverrideMutation.mutateAsync(userId);
      refetchOverrides();
    } catch (err: any) {
      alert("Failed to delete override: " + err.message);
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
    <div className="mx-auto max-w-5xl space-y-8 p-8">
      {/* Header */}
      <div className="border-b border-border/40 pb-5">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Mail className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Email Notifications Settings
            </h1>
            <p className="text-sm text-muted-foreground">
              Configure global outbound email delivery for Paca notifications (task assignments, mentions, updates).
            </p>
          </div>
        </div>
      </div>

      {saveSuccessMessage && (
        <div className="flex items-center gap-2 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-3 text-sm text-emerald-600 dark:text-emerald-400">
          <CheckCircle2 className="h-4 w-4 shrink-0" />
          <span>{saveSuccessMessage}</span>
        </div>
      )}

      {/* Info Banner */}
      <div className="flex gap-3 rounded-xl border border-primary/20 bg-primary/5 p-4 text-sm text-foreground">
        <Info className="h-5 w-5 shrink-0 text-primary" />
        <div className="space-y-1">
          <div className="font-semibold">How recipient email routing works:</div>
          <div className="text-xs text-muted-foreground leading-relaxed">
            1. <strong>Direct Email Usernames:</strong> If a user's login username is an email address (e.g. <code>alice@company.com</code>), notifications are sent directly to that address.<br />
            2. <strong>User Email Overrides:</strong> If a user has a text login (e.g. <code>admin</code>, <code>john</code>), you can map their User ID to an email in the table below.<br />
            3. <strong>Safe Skipping:</strong> Users without a valid email username or override will not receive email notifications.
          </div>
        </div>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Provider Configuration */}
        <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
          <div className="mb-4 flex items-center gap-2">
            <ShieldCheck className="h-5 w-5 text-primary" />
            <h2 className="text-lg font-semibold text-foreground">Global Email Provider</h2>
          </div>

          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-xs font-medium text-foreground">
                Provider Type
              </label>
              <select
                value={formData.provider}
                onChange={(e) => handleProviderChange(e.target.value as EmailProviderType)}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
              >
                <option value="yandex_postbox">⭐️ Yandex Cloud Postbox (AWS SES REST API)</option>
                <option value="resend">Resend</option>
                <option value="sendgrid">SendGrid</option>
                <option value="mailgun">Mailgun</option>
                <option value="postmark">Postmark</option>
                <option value="brevo">Brevo (Sendinblue)</option>
                <option value="webhook">Custom Webhook / HTTP-to-SMTP Relay</option>
              </select>
            </div>

            <div>
              <label className="mb-1.5 block text-xs font-medium text-foreground">
                API Endpoint URL
              </label>
              <input
                type="text"
                value={formData.endpoint || ""}
                onChange={(e) => setFormData({ ...formData, endpoint: e.target.value })}
                placeholder="https://postbox.cloud.yandex.net/v2/email/outbound-emails"
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none font-mono text-xs"
              />
            </div>

            <div>
              <label className="mb-1.5 block text-xs font-medium text-foreground">
                API Key / IAM Token
              </label>
              <input
                type="password"
                value={formData.api_key || ""}
                onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
                placeholder={
                  formData.provider === "yandex_postbox"
                    ? "IAM Token (X-YaCloud-SubjectToken)"
                    : "API Key (Bearer token)"
                }
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
              />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">
                  Default Sender Email (From)
                </label>
                <input
                  type="email"
                  value={formData.from_email || ""}
                  onChange={(e) => setFormData({ ...formData, from_email: e.target.value })}
                  placeholder="no-reply@yourdomain.com"
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">
                  Sender Display Name
                </label>
                <input
                  type="text"
                  value={formData.from_name || ""}
                  onChange={(e) => setFormData({ ...formData, from_name: e.target.value })}
                  placeholder="PACA Notifications"
                  className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
                />
              </div>
            </div>
          </div>
        </div>

        {/* Global Notification Triggers */}
        <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
          <div className="mb-4 flex items-center gap-2">
            <Bell className="h-5 w-5 text-primary" />
            <h2 className="text-lg font-semibold text-foreground">Default Notification Triggers</h2>
          </div>
          <div className="space-y-3">
            <label className="flex items-center gap-3 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={formData.notify_on_assigned}
                onChange={(e) => setFormData({ ...formData, notify_on_assigned: e.target.checked })}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/20"
              />
              <span className="text-foreground">Notify when a task is assigned to a user</span>
            </label>
            <label className="flex items-center gap-3 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={formData.notify_on_mentioned}
                onChange={(e) => setFormData({ ...formData, notify_on_mentioned: e.target.checked })}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/20"
              />
              <span className="text-foreground">Notify when a user is @-mentioned in comments</span>
            </label>
            <label className="flex items-center gap-3 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={formData.notify_on_update}
                onChange={(e) => setFormData({ ...formData, notify_on_update: e.target.checked })}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/20"
              />
              <span className="text-foreground">Notify on task status updates</span>
            </label>
          </div>
        </div>

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={updateSettingsMutation.isPending}
            className="flex items-center gap-2 rounded-md bg-primary px-5 py-2.5 text-sm font-medium text-primary-foreground shadow transition hover:bg-primary/90 disabled:opacity-50"
          >
            <Save className="h-4 w-4" />
            {updateSettingsMutation.isPending ? "Saving..." : "Save Global Settings"}
          </button>
        </div>
      </form>

      {/* Test Email Section */}
      <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
        <div className="mb-4 flex items-center gap-2">
          <Send className="h-5 w-5 text-primary" />
          <h2 className="text-lg font-semibold text-foreground">Send Test Email</h2>
        </div>
        <p className="mb-4 text-xs text-muted-foreground">
          Send a diagnostic test email to verify credentials and endpoint reachability.
        </p>
        <form onSubmit={handleSendTest} className="flex flex-col gap-3 sm:flex-row">
          <input
            type="email"
            value={testRecipient}
            onChange={(e) => setTestRecipient(e.target.value)}
            placeholder="recipient@example.com"
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
          />
          <button
            type="submit"
            disabled={sendTestMutation.isPending || !testRecipient}
            className="flex items-center justify-center gap-2 rounded-md bg-secondary px-5 py-2 text-sm font-medium text-secondary-foreground hover:bg-secondary/80 disabled:opacity-50"
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
            className={`mt-4 flex items-start gap-2 rounded-lg border p-3 text-xs ${
              testResult.success
                ? "border-emerald-500/30 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
                : "border-destructive/30 bg-destructive/10 text-destructive dark:text-red-400"
            }`}
          >
            {testResult.success ? (
              <CheckCircle2 className="h-4 w-4 shrink-0" />
            ) : (
              <XCircle className="h-4 w-4 shrink-0" />
            )}
            <span>{testResult.message}</span>
          </div>
        )}
      </div>

      {/* User Email Overrides Section */}
      <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Users className="h-5 w-5 text-primary" />
            <h2 className="text-lg font-semibold text-foreground">User Email Overrides</h2>
          </div>
          <button
            type="button"
            onClick={() => refetchOverrides()}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loadingOverrides ? "animate-spin" : ""}`} />
            Refresh
          </button>
        </div>

        <p className="mb-4 text-xs text-muted-foreground">
          Map specific user IDs (for accounts whose login username is not an email) to their destination email addresses.
        </p>

        {overrideError && (
          <div className="mb-4 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive dark:text-red-400">
            {overrideError}
          </div>
        )}

        <form onSubmit={handleAddOverride} className="mb-6 flex flex-col gap-3 sm:flex-row">
          <input
            type="text"
            value={newUserId}
            onChange={(e) => setNewUserId(e.target.value)}
            placeholder="User UUID (e.g. 550e8400-e29b-41d4-a716-446655440000)"
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none font-mono text-xs"
          />
          <input
            type="email"
            value={newUserEmail}
            onChange={(e) => setNewUserEmail(e.target.value)}
            placeholder="target@company.com"
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
          />
          <button
            type="submit"
            disabled={saveOverrideMutation.isPending}
            className="flex items-center justify-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
          >
            <Plus className="h-4 w-4" />
            Add Override
          </button>
        </form>

        {overrides.length === 0 ? (
          <p className="py-4 text-center text-xs text-muted-foreground">
            No user email overrides configured. All users with email usernames receive notifications directly.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-border/40 text-muted-foreground">
                  <th className="py-2 font-medium">User ID</th>
                  <th className="py-2 font-medium">Destination Email</th>
                  <th className="py-2 font-medium">Updated</th>
                  <th className="py-2 text-right font-medium">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/20">
                {overrides.map((ov) => (
                  <tr key={ov.user_id} className="hover:bg-muted/30">
                    <td className="py-2.5 font-mono text-foreground">{ov.user_id}</td>
                    <td className="py-2.5 font-mono text-primary">{ov.email}</td>
                    <td className="py-2.5 text-muted-foreground">
                      {new Date(ov.updated_at).toLocaleString()}
                    </td>
                    <td className="py-2.5 text-right">
                      <button
                        type="button"
                        onClick={() => handleDeleteOverride(ov.user_id)}
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

      {/* Global Delivery Audit Log */}
      <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-foreground">Global Delivery Audit Log</h2>
          <button
            type="button"
            onClick={() => refetchLogs()}
            className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loadingLogs ? "animate-spin" : ""}`} />
            Refresh
          </button>
        </div>

        {logs.length === 0 ? (
          <p className="py-6 text-center text-xs text-muted-foreground">
            No email logs recorded yet.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-border/40 text-muted-foreground">
                  <th className="py-2 font-medium">Status</th>
                  <th className="py-2 font-medium">Recipient</th>
                  <th className="py-2 font-medium">Subject</th>
                  <th className="py-2 font-medium">Type</th>
                  <th className="py-2 font-medium">Project</th>
                  <th className="py-2 font-medium">Time</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/20">
                {logs.map((log) => (
                  <tr key={log.id} className="hover:bg-muted/30">
                    <td className="py-2.5">
                      {log.status === "sent" ? (
                        <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-emerald-600 dark:text-emerald-400 font-medium">
                          <CheckCircle2 className="h-3 w-3" /> sent
                        </span>
                      ) : log.status === "skipped" ? (
                        <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-muted-foreground font-medium">
                          skipped
                        </span>
                      ) : (
                        <span
                          className="inline-flex items-center gap-1 rounded-full bg-destructive/10 px-2 py-0.5 text-destructive dark:text-red-400 font-medium"
                          title={log.error_message || "Failed"}
                        >
                          <XCircle className="h-3 w-3" /> failed
                        </span>
                      )}
                    </td>
                    <td className="py-2.5 font-mono text-foreground">{log.recipient_email}</td>
                    <td className="py-2.5 max-w-xs truncate text-foreground">{log.subject}</td>
                    <td className="py-2.5 text-muted-foreground">{log.notification_type}</td>
                    <td className="py-2.5 font-mono text-muted-foreground">
                      {log.project_id ? log.project_id.slice(0, 8) : "—"}
                    </td>
                    <td className="py-2.5 text-muted-foreground">
                      {new Date(log.created_at).toLocaleString()}
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
