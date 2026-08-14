export type EmailProviderType =
  | "yandex_postbox"
  | "resend"
  | "sendgrid"
  | "mailgun"
  | "postmark"
  | "brevo"
  | "webhook";

export interface SMTPSettings {
  id: string;
  project_id?: string;
  provider: EmailProviderType;
  endpoint: string;
  api_key?: string;
  from_email: string;
  from_name: string;
  notify_on_assigned: boolean;
  notify_on_mentioned: boolean;
  notify_on_update: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpdateSettingsInput {
  provider?: EmailProviderType;
  endpoint?: string;
  api_key?: string;
  from_email?: string;
  from_name?: string;
  notify_on_assigned?: boolean;
  notify_on_mentioned?: boolean;
  notify_on_update?: boolean;
}

export interface EmailLog {
  id: string;
  project_id?: string;
  recipient_user_id: string;
  recipient_email: string;
  notification_type: string;
  subject: string;
  status: "sent" | "failed" | "skipped";
  error_message?: string;
  created_at: string;
}

export interface UserEmailOverride {
  user_id: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface TestEmailInput {
  to_email: string;
  provider?: EmailProviderType;
  endpoint?: string;
  api_key?: string;
  from_email?: string;
  from_name?: string;
}
