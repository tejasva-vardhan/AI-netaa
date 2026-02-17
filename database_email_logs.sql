-- Email shadow mode (pilot): log all authority emails and send only to pilot inbox
-- Table: email_logs
-- Authority abstraction: department_id + level (L1/L2/L3), not officer-based

CREATE TABLE IF NOT EXISTS email_logs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_id BIGINT NOT NULL COMMENT 'Related complaint',
    email_type ENUM('assignment','escalation','resolution') NOT NULL COMMENT 'Type of email',
    intended_authority_id BIGINT NULL COMMENT 'Authority identifier (for authority abstraction, not officer)',
    intended_level VARCHAR(10) NULL COMMENT 'L1/L2/L3 escalation level',
    department_id BIGINT NOT NULL COMMENT 'Department ID (explicit authority department)',
    sent_to_email VARCHAR(255) NOT NULL COMMENT 'Actual recipient (pilot: aineta502@gmail.com)',
    subject VARCHAR(500) NOT NULL COMMENT 'Email subject',
    body TEXT NOT NULL COMMENT 'Email body (full content as if real send)',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'When logged',

    INDEX idx_complaint_id (complaint_id),
    INDEX idx_email_type (email_type),
    INDEX idx_department_id (department_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
