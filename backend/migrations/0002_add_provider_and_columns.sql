-- Ensure all columns exist even if 0001 was already applied under an older version
ALTER TABLE smtp_settings ADD COLUMN IF NOT EXISTS provider VARCHAR(32) NOT NULL DEFAULT 'yandex_postbox';
ALTER TABLE smtp_settings ADD COLUMN IF NOT EXISTS endpoint VARCHAR(512) NOT NULL DEFAULT 'https://postbox.cloud.yandex.net/v2/email/outbound-emails';
ALTER TABLE smtp_settings ADD COLUMN IF NOT EXISTS api_key TEXT NOT NULL DEFAULT '';
ALTER TABLE smtp_settings ADD COLUMN IF NOT EXISTS notify_on_assign BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE smtp_settings ADD COLUMN IF NOT EXISTS notify_on_mention BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE smtp_settings ADD COLUMN IF NOT EXISTS notify_on_update BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE smtp_settings ADD COLUMN IF NOT EXISTS from_email VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE smtp_settings ADD COLUMN IF NOT EXISTS from_name VARCHAR(255) NOT NULL DEFAULT 'PACA Notifications';

CREATE TABLE IF NOT EXISTS user_email_overrides (
    user_id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
