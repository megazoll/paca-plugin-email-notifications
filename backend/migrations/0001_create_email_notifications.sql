CREATE TABLE IF NOT EXISTS smtp_settings (
    id UUID PRIMARY KEY,
    scope TEXT NOT NULL DEFAULT 'global',
    project_id UUID,
    enabled BOOLEAN NOT NULL DEFAULT true,
    host TEXT NOT NULL DEFAULT '',
    port INT NOT NULL DEFAULT 587,
    username TEXT NOT NULL DEFAULT '',
    password TEXT NOT NULL DEFAULT '',
    from_email TEXT NOT NULL DEFAULT 'notifications@paca.local',
    from_name TEXT NOT NULL DEFAULT 'Paca',
    security TEXT NOT NULL DEFAULT 'starttls',
    webhook_url TEXT NOT NULL DEFAULT '',
    webhook_api_key TEXT NOT NULL DEFAULT '',
    notify_on_assigned BOOLEAN NOT NULL DEFAULT true,
    notify_on_mentioned BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS email_logs (
    id UUID PRIMARY KEY,
    project_id UUID,
    notification_id UUID,
    recipient_user_id UUID,
    recipient_email TEXT NOT NULL,
    notification_type TEXT NOT NULL,
    subject TEXT NOT NULL,
    body_text TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_email_logs_project_id ON email_logs(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_logs_recipient ON email_logs(recipient_email, created_at DESC);

CREATE TABLE IF NOT EXISTS user_email_overrides (
    user_id UUID PRIMARY KEY,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
