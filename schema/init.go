// Package schema: safe database initialization — create only missing tables, never drop or overwrite.

package schema

import (
	"database/sql"
	"log"
)

const (
	tableUsers     = "users"
	tableComplaints = "complaints"
)

// InitializeDatabase ensures core tables exist. Checks INFORMATION_SCHEMA.TABLES; creates only missing
// tables in order: users → complaints → complaint_status_history. Then runs EnsureComplaintStatusHistory
// to add any missing columns. Does not drop or recreate tables; does not remove data.
func InitializeDatabase(db *sql.DB) {
	// 1. users
	if exists, err := tableExists(db, tableUsers); err != nil {
		log.Fatalf("[SCHEMA] Failed to check if table %s exists: %v", tableUsers, err)
	} else if exists {
		log.Println("[SCHEMA] users table exists")
	} else {
		createUsersTable(db)
		log.Println("[SCHEMA] created users table")
	}

	// 2. complaints (depends on users)
	if exists, err := tableExists(db, tableComplaints); err != nil {
		log.Fatalf("[SCHEMA] Failed to check if table %s exists: %v", tableComplaints, err)
	} else if exists {
		log.Println("[SCHEMA] complaints table exists")
	} else {
		createComplaintsTable(db)
		log.Println("[SCHEMA] created complaints table")
	}

	// 3. complaint_status_history (depends on complaints)
	if exists, err := tableExists(db, tableComplaintStatusHistory); err != nil {
		log.Fatalf("[SCHEMA] Failed to check if table complaint_status_history exists: %v", err)
	} else if exists {
		log.Println("[SCHEMA] complaint_status_history exists")
	} else {
		createComplaintStatusHistoryTable(db)
		log.Println("[SCHEMA] created complaint_status_history")
	}

	// Fix missing columns on complaint_status_history (actor_type, actor_id, reason)
	EnsureComplaintStatusHistory(db)

	// 4. escalation_rules (minimal safe init if missing)
	if exists, err := tableExists(db, "escalation_rules"); err != nil {
		log.Fatalf("[SCHEMA] Failed to check if table escalation_rules exists: %v", err)
	} else if !exists {
		createEscalationRulesTable(db)
		log.Println("[SCHEMA] created escalation_rules table")
	}

	// 5. notifications_log (minimal safe init if missing)
	if exists, err := tableExists(db, "notifications_log"); err != nil {
		log.Fatalf("[SCHEMA] Failed to check if table notifications_log exists: %v", err)
	} else if !exists {
		createNotificationsLogTable(db)
		log.Println("[SCHEMA] created notifications_log table")
	}
}

func createEscalationRulesTable(db *sql.DB) {
	q := `
CREATE TABLE IF NOT EXISTS escalation_rules (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    complaint_type VARCHAR(255) NULL,
    escalation_time INT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := db.Exec(q); err != nil {
		log.Fatalf("[SCHEMA] Failed to create table escalation_rules: %v", err)
	}
}

func createNotificationsLogTable(db *sql.DB) {
	q := `
CREATE TABLE IF NOT EXISTS notifications_log (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NULL,
    message TEXT NULL,
    status VARCHAR(50) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := db.Exec(q); err != nil {
		log.Fatalf("[SCHEMA] Failed to create table notifications_log: %v", err)
	}
}

func createUsersTable(db *sql.DB) {
	q := `
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := db.Exec(q); err != nil {
		log.Fatalf("[SCHEMA] Failed to create table %s: %v", tableUsers, err)
	}
}

func createComplaintsTable(db *sql.DB) {
	q := `
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := db.Exec(q); err != nil {
		log.Fatalf("[SCHEMA] Failed to create table %s: %v", tableComplaints, err)
	}
}
