-- Core Database Schema for Public Accountability System
-- This file contains CREATE TABLE statements for core tables
-- Notification tables are defined separately in database_notifications.sql

-- 1. users: Stores citizen/user information with phone-based verification
CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    phone_number VARCHAR(15) UNIQUE NOT NULL COMMENT 'Phone number (E.164 format)',
    phone_verified_at TIMESTAMP NULL COMMENT 'When phone verification completed',
    phone_verification_code VARCHAR(10) NULL COMMENT 'Temporary OTP (hashed)',
    phone_verification_expires_at TIMESTAMP NULL COMMENT 'OTP expiration time',
    full_name VARCHAR(255) NULL COMMENT 'User''s full name (optional)',
    email VARCHAR(255) NULL COMMENT 'Email address (optional)',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Account creation time',
    last_active_at TIMESTAMP NULL COMMENT 'Last activity timestamp',
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Account suspension flag',
    blocked_reason TEXT NULL COMMENT 'Reason for blocking',
    blocked_at TIMESTAMP NULL COMMENT 'When account was blocked',
    
    INDEX idx_phone_number (phone_number),
    INDEX idx_phone_verified (phone_verified_at),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 2. complaints: Main complaint records filed by citizens
CREATE TABLE IF NOT EXISTS complaints (
    complaint_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_number VARCHAR(50) UNIQUE NOT NULL COMMENT 'Public-facing complaint number',
    user_id BIGINT NOT NULL COMMENT 'Complainant',
    title VARCHAR(500) NOT NULL COMMENT 'Complaint title',
    description TEXT NOT NULL COMMENT 'Detailed description',
    category VARCHAR(100) NULL COMMENT 'Complaint category',
    location_id BIGINT NOT NULL COMMENT 'Complaint location',
    latitude DECIMAL(10, 8) NULL COMMENT 'Specific coordinates',
    longitude DECIMAL(11, 8) NULL COMMENT 'Specific coordinates',
    assigned_department_id BIGINT NULL COMMENT 'Initially assigned department',
    assigned_officer_id BIGINT NULL COMMENT 'Currently assigned officer',
    current_status ENUM('draft', 'submitted', 'verified', 'under_review', 'in_progress', 'resolved', 'rejected', 'closed', 'escalated') NOT NULL DEFAULT 'draft' COMMENT 'Current status',
    priority ENUM('low', 'medium', 'high', 'urgent') NOT NULL DEFAULT 'medium' COMMENT 'Priority level',
    is_public BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Public visibility flag',
    public_consent_given BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'User consent for public disclosure',
    supporter_count INT NOT NULL DEFAULT 0 COMMENT 'Count of supporting users',
    resolved_at TIMESTAMP NULL COMMENT 'Resolution timestamp',
    closed_at TIMESTAMP NULL COMMENT 'Closure timestamp',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Complaint creation',
    updated_at TIMESTAMP NULL COMMENT 'Last update (for non-audit changes)',
    
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE RESTRICT,
    INDEX idx_user_id (user_id),
    INDEX idx_complaint_number (complaint_number),
    INDEX idx_location_id (location_id),
    INDEX idx_status (current_status),
    INDEX idx_assigned_department (assigned_department_id),
    INDEX idx_assigned_officer (assigned_officer_id),
    INDEX idx_created_at (created_at),
    INDEX idx_status_created (current_status, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 3. complaint_status_history: Immutable timeline of complaint status changes (append-only)
-- Audit: actor_type, actor_id, reason (old rows may have NULL actor_id/actor_type/reason)
CREATE TABLE IF NOT EXISTS complaint_status_history (
    history_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_id BIGINT NOT NULL COMMENT 'Related complaint',
    old_status ENUM('draft', 'submitted', 'verified', 'under_review', 'in_progress', 'resolved', 'rejected', 'closed', 'escalated') NULL COMMENT 'Previous status',
    new_status ENUM('draft', 'submitted', 'verified', 'under_review', 'in_progress', 'resolved', 'rejected', 'closed', 'escalated') NOT NULL COMMENT 'New status',
    changed_by_type ENUM('user', 'officer', 'system', 'admin') NOT NULL COMMENT 'Who made the change',
    changed_by_user_id BIGINT NULL COMMENT 'User who changed (if applicable)',
    changed_by_officer_id BIGINT NULL COMMENT 'Officer who changed (if applicable)',
    assigned_department_id BIGINT NULL COMMENT 'Department at time of change',
    assigned_officer_id BIGINT NULL COMMENT 'Officer at time of change',
    notes TEXT NULL COMMENT 'Status change notes/comments',
    actor_type ENUM('system','authority','user') NULL COMMENT 'Audit: who made the change',
    actor_id BIGINT NULL COMMENT 'Audit: user_id or officer_id; NULL for system',
    reason TEXT NULL COMMENT 'Audit: reason for change when available',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Change timestamp',
    
    FOREIGN KEY (complaint_id) REFERENCES complaints(complaint_id) ON DELETE CASCADE,
    FOREIGN KEY (changed_by_user_id) REFERENCES users(user_id) ON DELETE SET NULL,
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_complaint_created (complaint_id, created_at DESC),
    INDEX idx_changed_by_user (changed_by_user_id),
    INDEX idx_changed_by_officer (changed_by_officer_id),
    INDEX idx_status_history_actor (actor_type, actor_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 4. complaint_attachments: Stores file attachments (photos, documents) for complaints
CREATE TABLE IF NOT EXISTS complaint_attachments (
    attachment_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_id BIGINT NOT NULL COMMENT 'Related complaint',
    file_name VARCHAR(255) NOT NULL COMMENT 'Original filename',
    file_path VARCHAR(1000) NOT NULL COMMENT 'Storage path/URL',
    file_type VARCHAR(100) NULL COMMENT 'MIME type',
    file_size BIGINT NULL COMMENT 'Size in bytes',
    uploaded_by_user_id BIGINT NULL COMMENT 'Uploader',
    is_public BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Public visibility',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Upload timestamp',
    
    FOREIGN KEY (complaint_id) REFERENCES complaints(complaint_id) ON DELETE CASCADE,
    FOREIGN KEY (uploaded_by_user_id) REFERENCES users(user_id) ON DELETE SET NULL,
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_uploaded_by (uploaded_by_user_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 5. complaint_supporters: Tracks users who support/duplicate a complaint
CREATE TABLE IF NOT EXISTS complaint_supporters (
    support_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_id BIGINT NOT NULL COMMENT 'Supported complaint',
    user_id BIGINT NOT NULL COMMENT 'Supporting user',
    is_duplicate BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'Marks as duplicate complaint',
    duplicate_notes TEXT NULL COMMENT 'Notes if marking as duplicate',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Support timestamp',
    
    FOREIGN KEY (complaint_id) REFERENCES complaints(complaint_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE,
    UNIQUE KEY uk_complaint_user (complaint_id, user_id),
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 6. officers: Government officers assigned to handle complaints
CREATE TABLE IF NOT EXISTS officers (
    officer_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    employee_id VARCHAR(100) UNIQUE NULL COMMENT 'Official employee ID',
    full_name VARCHAR(255) NOT NULL COMMENT 'Officer''s name',
    designation VARCHAR(255) NULL COMMENT 'Job title/designation',
    email VARCHAR(255) NULL COMMENT 'Official email',
    phone_number VARCHAR(15) NULL COMMENT 'Contact number',
    department_id BIGINT NOT NULL COMMENT 'Assigned department',
    location_id BIGINT NOT NULL COMMENT 'Jurisdiction location',
    is_active BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Active status',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Record creation',
    updated_at TIMESTAMP NULL COMMENT 'Last update',
    
    INDEX idx_department_id (department_id),
    INDEX idx_location_id (location_id),
    INDEX idx_department_location (department_id, location_id),
    INDEX idx_is_active (is_active),
    INDEX idx_employee_id (employee_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7. escalation_rules: Defines escalation hierarchy and rules
CREATE TABLE IF NOT EXISTS escalation_rules (
    rule_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    from_department_id BIGINT NULL COMMENT 'Source department (NULL = any)',
    from_location_id BIGINT NULL COMMENT 'Source location (NULL = any)',
    to_department_id BIGINT NOT NULL COMMENT 'Target department',
    to_location_id BIGINT NULL COMMENT 'Target location (NULL = same)',
    escalation_level INT NOT NULL COMMENT 'Level in hierarchy',
    conditions JSON NULL COMMENT 'Conditions for escalation (time-based, status-based, etc.)',
    is_active BOOLEAN NOT NULL DEFAULT TRUE COMMENT 'Active status',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Rule creation',
    updated_at TIMESTAMP NULL COMMENT 'Last update',
    
    INDEX idx_from_department (from_department_id),
    INDEX idx_to_department (to_department_id),
    INDEX idx_escalation_level (escalation_level),
    INDEX idx_is_active (is_active),
    INDEX idx_level_active (escalation_level, is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 8. complaint_escalations: Tracks escalation hierarchy and escalation events
CREATE TABLE IF NOT EXISTS complaint_escalations (
    escalation_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_id BIGINT NOT NULL COMMENT 'Escalated complaint',
    from_department_id BIGINT NULL COMMENT 'Department escalated from',
    from_officer_id BIGINT NULL COMMENT 'Officer escalated from',
    to_department_id BIGINT NOT NULL COMMENT 'Department escalated to',
    to_officer_id BIGINT NULL COMMENT 'Officer escalated to',
    escalation_level INT NOT NULL COMMENT 'Level in hierarchy (1, 2, 3...)',
    reason TEXT NULL COMMENT 'Escalation reason',
    escalated_by_type ENUM('user', 'officer', 'system', 'admin') NOT NULL COMMENT 'Who escalated',
    escalated_by_user_id BIGINT NULL COMMENT 'User who escalated (if applicable)',
    escalated_by_officer_id BIGINT NULL COMMENT 'Officer who escalated (if applicable)',
    status_history_id BIGINT NULL COMMENT 'Related status change',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Escalation timestamp',
    
    FOREIGN KEY (complaint_id) REFERENCES complaints(complaint_id) ON DELETE CASCADE,
    FOREIGN KEY (status_history_id) REFERENCES complaint_status_history(history_id) ON DELETE SET NULL,
    FOREIGN KEY (escalated_by_user_id) REFERENCES users(user_id) ON DELETE SET NULL,
    INDEX idx_complaint_id (complaint_id),
    INDEX idx_complaint_level (complaint_id, escalation_level),
    INDEX idx_to_department (to_department_id),
    INDEX idx_escalation_level (escalation_level),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 9. audit_log: Immutable append-only audit trail for all critical actions
CREATE TABLE IF NOT EXISTS audit_log (
    audit_id BIGINT PRIMARY KEY AUTO_INCREMENT,
    entity_type VARCHAR(100) NOT NULL COMMENT 'Entity type (e.g., ''complaint'', ''user'', ''officer'')',
    entity_id BIGINT NOT NULL COMMENT 'Entity identifier',
    action VARCHAR(100) NOT NULL COMMENT 'Action performed (e.g., ''create'', ''update'', ''delete'', ''status_change'')',
    action_by_type ENUM('user', 'officer', 'system', 'admin') NOT NULL COMMENT 'Actor type',
    action_by_user_id BIGINT NULL COMMENT 'User actor',
    action_by_officer_id BIGINT NULL COMMENT 'Officer actor',
    old_values JSON NULL COMMENT 'Previous state (JSON snapshot)',
    new_values JSON NULL COMMENT 'New state (JSON snapshot)',
    changes JSON NULL COMMENT 'Diff of changes',
    ip_address VARCHAR(45) NULL COMMENT 'IP address',
    user_agent VARCHAR(500) NULL COMMENT 'Browser/device info',
    metadata JSON NULL COMMENT 'Additional context',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Audit timestamp',
    
    FOREIGN KEY (action_by_user_id) REFERENCES users(user_id) ON DELETE SET NULL,
    INDEX idx_entity (entity_type, entity_id, created_at DESC),
    INDEX idx_action_by_user (action_by_type, action_by_user_id, created_at DESC),
    INDEX idx_action_by_officer (action_by_type, action_by_officer_id, created_at DESC),
    INDEX idx_created_at (created_at),
    INDEX idx_entity_type (entity_type),
    INDEX idx_action (action)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
