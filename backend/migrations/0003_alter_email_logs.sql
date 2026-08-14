-- 0003_alter_email_logs.sql
-- Ensure recipient_user_id in email_logs is nullable with default empty string
ALTER TABLE email_logs ALTER COLUMN recipient_user_id DROP NOT NULL;
ALTER TABLE email_logs ALTER COLUMN recipient_user_id SET DEFAULT '';
