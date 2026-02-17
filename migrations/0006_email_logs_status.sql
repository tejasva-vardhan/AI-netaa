-- Email pipeline: log delivery status and error message (sent/failed).
-- Run once; if columns exist, alter will fail (safe to ignore on re-run).

ALTER TABLE email_logs
  ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'pending' COMMENT 'sent | failed',
  ADD COLUMN error_message TEXT NULL COMMENT 'Error details when status=failed';
