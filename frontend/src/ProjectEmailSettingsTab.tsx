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
  Bell,
  Info,
  ShieldCheck,
} from "lucide-react";
import {
  useProjectEmailSettings,
  useUpdateProjectEmailSettings,
  useProjectEmailLogs,
  useSendProjectTestEmail,
} from "./api";
import type { EmailProviderType, UpdateSettingsInput } from "./types";

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
    provider: "yandex_postbox",
    endpoint: "https://postbox.cloud.yandex.net/v2/email/outbound-emails",
    api_key: "",
    from_email: "",
    from_name: "PACA Notifications",
    notify_on_assigned: true,
    notify_on_mentioned: true,
    notify_on_update: false,
  });

  const [useCustomProjectProvider, setUseCustomProjectProvider] = useState(false);
  const [testRecipient, setTestRecipient] = useState("");
  const [saveSuccessMessage, setSaveSuccessMessage] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

  useEffect(() => {
    if (settings) {
      setFormData({
        provider: settings.provider || "yandex_postbox",
        endpoint: settings.endpoint || "https://postbox.cloud.yandex.net/v2/email/outbound-emails",
        api_key: settings.api_key || "",
        from_email: settings.from_email || "",
        from_name: settings.from_name || "PACA Notifications",
        notify_on_assigned: settings.notify_on_assigned,
        notify_on_mentioned: settings.notify_on_mentioned,
        notify_on_update: settings.notify_on_update,
      });
      setUseCustomProjectProvider(Boolean(settings.api_key && settings.api_key.trim().length > 0));
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
      const payload: UpdateSettingsInput = {
        notify_on_assigned: formData.notify_on_assigned,
        notify_on_mentioned: formData.notify_on_mentioned,
        notify_on_update: formData.notify_on_update,
      };

      if (useCustomProjectProvider) {
        payload.provider = formData.provider;
        payload.endpoint = formData.endpoint;
        payload.api_key = formData.api_key;
        payload.from_email = formData.from_email;
        payload.from_name = formData.from_name;
      } else {
        payload.api_key = "";
      }

      await updateSettingsMutation.mutateAsync(payload);
      setSaveSuccessMessage("Settings saved successfully!");
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
        provider: useCustomProjectProvider ? formData.provider : undefined,
        endpoint: useCustomProjectProvider ? formData.endpoint : undefined,
        api_key: useCustomProjectProvider ? formData.api_key : undefined,
        from_email: useCustomProjectProvider ? formData.from_email : undefined,
        from_name: useCustomProjectProvider ? formData.from_name : undefined,
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

  if (loadingSettings) {
    return (
      <div className="flex h-64 items-center justify-center">
        <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-4xl space-y-8 p-6">
      {/* Header */}
      <div className="border-b border-border/40 pb-5">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
            <Mail className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-xl font-semibold tracking-tight text-foreground">
              Email Notifications
            </h2>
            <p className="text-sm text-muted-foreground">
              Automatically duplicates Paca notifications to user email addresses via Yandex Cloud Postbox, Resend, SendGrid, or Webhooks.
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

      {/* Info Callout */}
      <div className="flex gap-3 rounded-lg border border-border/60 bg-muted/40 p-4 text-sm text-muted-foreground">
        <Info className="h-5 w-5 shrink-0 text-primary" />
        <div>
          <span className="font-medium text-foreground">Username Email Policy: </span>
          When an in-app notification is triggered, Paca inspects the recipient's username. If it is a valid email address (e.g. <code className="text-xs bg-muted px-1 py-0.5 rounded">alice@company.com</code>) or has an admin override, an email is dispatched. Non-email usernames are skipped safely.
        </div>
      </div>

      <form onSubmit={handleSave} className="space-y-6">
        {/* Notification Triggers */}
        <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
          <div className="mb-4 flex items-center gap-2">
            <Bell className="h-4 w-4 text-primary" />
            <h3 className="text-base font-semibold text-foreground">Notification Triggers</h3>
          </div>
          <div className="space-y-3">
            <label className="flex items-center gap-3 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={formData.notify_on_assigned}
                onChange={(e) => setFormData({ ...formData, notify_on_assigned: e.target.checked })}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/20"
              />
              <span className="text-foreground">Notify when a task is assigned to the user</span>
            </label>
            <label className="flex items-center gap-3 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={formData.notify_on_mentioned}
                onChange={(e) => setFormData({ ...formData, notify_on_mentioned: e.target.checked })}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/20"
              />
              <span className="text-foreground">Notify when the user is @-mentioned in task comments</span>
            </label>
            <label className="flex items-center gap-3 text-sm cursor-pointer">
              <input
                type="checkbox"
                checked={formData.notify_on_update}
                onChange={(e) => setFormData({ ...formData, notify_on_update: e.target.checked })}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/20"
              />
              <span className="text-foreground">Notify on task status and priority updates</span>
            </label>
          </div>
        </div>

        {/* Project Custom Provider */}
        <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
          <div className="mb-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-primary" />
              <h3 className="text-base font-semibold text-foreground">Email Service Provider</h3>
            </div>
            <label className="flex items-center gap-2 text-xs font-medium text-muted-foreground cursor-pointer">
              <input
                type="checkbox"
                checked={useCustomProjectProvider}
                onChange={(e) => setUseCustomProjectProvider(e.target.checked)}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/20"
              />
              <span>Use custom credentials for this project</span>
            </label>
          </div>

          {useCustomProjectProvider ? (
            <div className="space-y-4 pt-2">
              <div>
                <label className="mb-1.5 block text-xs font-medium text-foreground">
                  Provider
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
                    Sender Email Address
                  </label>
                  <input
                    type="email"
                    value={formData.from_email || ""}
                    onChange={(e) => setFormData({ ...formData, from_email: e.target.value })}
                    placeholder="notifications@yourdomain.com"
                    className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground focus:border-primary focus:outline-none"
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-xs font-medium text-foreground">
                    Sender Name
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
          ) : (
            <p className="text-xs text-muted-foreground">
              This project inherits global instance email configuration configured by the administrator in Admin Settings.
            </p>
          )}
        </div>

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={updateSettingsMutation.isPending}
            className="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground shadow transition hover:bg-primary/90 disabled:opacity-50"
          >
            <Save className="h-4 w-4" />
            {updateSettingsMutation.isPending ? "Saving..." : "Save Changes"}
          </button>
        </div>
      </form>

      {/* Test Email Section */}
      <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
        <div className="mb-4 flex items-center gap-2">
          <Send className="h-4 w-4 text-primary" />
          <h3 className="text-base font-semibold text-foreground">Send Test Email</h3>
        </div>
        <p className="mb-4 text-xs text-muted-foreground">
          Verify email delivery by sending a diagnostic test message using the configured provider.
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
            className="flex items-center justify-center gap-2 rounded-md bg-secondary px-4 py-2 text-sm font-medium text-secondary-foreground hover:bg-secondary/80 disabled:opacity-50"
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

      {/* Audit Log Table */}
      <div className="rounded-xl border border-border/60 bg-card p-6 shadow-sm">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-base font-semibold text-foreground">Delivery Audit Log</h3>
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
            No emails have been dispatched for this project yet.
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
