-- Notification System Tables
-- These tables support the notification system with status tracking and retry logic

-- notifications_log: Main table for queued and sent notifications
CREATE TABLE IF NOT EXISTS notifications_log (
    notification_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    entity_type VARCHAR(100) NOT NULL COMMENT 'Entity type (e.g., complaint, user)',
    entity_id BIGINT NOT NULL COMMENT 'Entity ID',
    channel ENUM('email', 'sms', 'whatsapp') NOT NULL COMMENT 'Notification channel',
    recipient VARCHAR(255) NOT NULL COMMENT 'Recipient (email, phone number, etc.)',
    subject VARCHAR(500) NULL COMMENT 'Email subject (for email channel)',
    body TEXT NOT NULL COMMENT 'Notification body',
    template_id VARCHAR(100) NULL COMMENT 'Template identifier',
    template_data JSON NULL COMMENT 'Template data (JSON)',
    status ENUM('pending', 'sent', 'failed', 'retrying') NOT NULL DEFAULT 'pending' COMMENT 'Notification status',
    priority ENUM('low', 'normal', 'high', 'urgent') NOT NULL DEFAULT 'normal' COMMENT 'Notification priority',
    retry_count INT NOT NULL DEFAULT 0 COMMENT 'Number of retry attempts',
    max_retries INT NOT NULL DEFAULT 3 COMMENT 'Maximum retry attempts',
    next_retry_at TIMESTAMP NULL COMMENT 'Next retry time (for retrying status)',
    sent_at TIMESTAMP NULL COMMENT 'When notification was successfully sent',
    failed_at TIMESTAMP NULL COMMENT 'When notification was marked as failed',
    error_message TEXT NULL COMMENT 'Error message if failed',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'When notification was created',
    updated_at TIMESTAMP NULL COMMENT 'Last update timestamp',
    
    -- Indexes for efficient querying
    INDEX idx_status_retry (status, next_retry_at),
    INDEX idx_entity (entity_type, entity_id),
    INDEX idx_priority_status (priority, status),
    INDEX idx_channel_status (channel, status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- notification_attempts_log: Log of all notification send attempts
CREATE TABLE IF NOT EXISTS notification_attempts_log (
    log_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    notification_id BIGINT NOT NULL COMMENT 'Reference to notifications_log',
    attempt_number INT NOT NULL COMMENT 'Attempt number (1, 2, 3, ...)',
    status ENUM('pending', 'sent', 'failed', 'retrying') NOT NULL COMMENT 'Status of this attempt',
    error_message TEXT NULL COMMENT 'Error message if attempt failed',
    response_data JSON NULL COMMENT 'Response data from sender (JSON)',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'When attempt was made',
    
    FOREIGN KEY (notification_id) REFERENCES notifications_log(notification_id) ON DELETE CASCADE,
    INDEX idx_notification (notification_id, attempt_number),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Example queries:

-- Get pending notifications ready to send
-- SELECT * FROM notifications_log 
-- WHERE status IN ('pending', 'retrying') 
--   AND (next_retry_at IS NULL OR next_retry_at <= NOW())
-- ORDER BY priority, created_at ASC
-- LIMIT 100;

-- Get notification statistics
-- SELECT 
--     status,
--     COUNT(*) as count,
--     AVG(retry_count) as avg_retries
-- FROM notifications_log
-- GROUP BY status;

-- Get failed notifications
-- SELECT * FROM notifications_log
-- WHERE status = 'failed'
-- ORDER BY failed_at DESC;
