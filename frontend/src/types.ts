export interface SMTPSettings {
  id: string;
  scope: "project" | "global";
  project_id?: string;
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  password?: string;
  from_email: string;
  from_name: string;
  security: "starttls" | "tls" | "none";
  webhook_url?: string;
  webhook_api_key?: string;
  notify_on_assigned: boolean;
  notify_on_mentioned: boolean;
  created_at: string;
  updated_at: string;
}

export interface UpdateSettingsInput {
  enabled?: boolean;
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  from_email?: string;
  from_name?: string;
  security?: "starttls" | "tls" | "none";
  webhook_url?: string;
  webhook_api_key?: string;
  notify_on_assigned?: boolean;
  notify_on_mentioned?: boolean;
}

export interface EmailLog {
  id: string;
  project_id?: string;
  notification_id?: string;
  recipient_user_id?: string;
  recipient_email: string;
  notification_type: string;
  subject: string;
  body_text: string;
  status: "sent" | "failed";
  error_message?: string;
  created_at: string;
}

export interface UserEmailOverride {
  user_id: string;
  email: string;
  created_at: string;
  updated_at: string;
}
