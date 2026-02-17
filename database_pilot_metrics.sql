-- Pilot Metrics Events Table
-- Used for pilot evaluation and demos (backend-only, no dashboards yet)
-- Tracks key metrics: complaints created, time to first action, escalations, resolutions, chat dropoffs

CREATE TABLE IF NOT EXISTS pilot_metrics_events (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    event_type VARCHAR(50) NOT NULL COMMENT 'Event type: complaint_created, first_authority_action, escalation_triggered, complaint_resolved, chat_abandoned',
    complaint_id BIGINT NULL COMMENT 'Related complaint ID (nullable)',
    user_id BIGINT NULL COMMENT 'Related user ID (nullable)',
    metadata JSON NULL COMMENT 'Additional event data (JSON)',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Event timestamp',
    
    INDEX idx_event_type (event_type),
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at),
    INDEX idx_event_type_created_at (event_type, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
