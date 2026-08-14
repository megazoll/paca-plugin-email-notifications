CREATE TABLE IF NOT EXISTS smtp_settings (
    id VARCHAR(64) PRIMARY KEY,
    project_id VARCHAR(64),
    provider VARCHAR(32) NOT NULL DEFAULT 'yandex_postbox',
    endpoint VARCHAR(512) NOT NULL DEFAULT 'https://postbox.cloud.yandex.net/v2/email/outbound-emails',
    api_key TEXT NOT NULL DEFAULT '',
    from_email VARCHAR(255) NOT NULL,
    from_name VARCHAR(255) NOT NULL DEFAULT 'PACA Notifications',
    notify_on_assign BOOLEAN NOT NULL DEFAULT true,
    notify_on_mention BOOLEAN NOT NULL DEFAULT true,
    notify_on_update BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS email_logs (
    id VARCHAR(64) PRIMARY KEY,
    project_id VARCHAR(64),
    recipient_user_id VARCHAR(64) DEFAULT '',
    recipient_email VARCHAR(255) NOT NULL,
    notification_type VARCHAR(64) NOT NULL,
    subject VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL, -- 'sent', 'failed', 'skipped'
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS user_email_overrides (
    user_id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_smtp_settings_project_id ON smtp_settings(project_id);
CREATE INDEX IF NOT EXISTS idx_email_logs_project_id ON email_logs(project_id);
CREATE INDEX IF NOT EXISTS idx_email_logs_created_at ON email_logs(created_at DESC);
