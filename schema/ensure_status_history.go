// Package schema: ensure complaint_status_history table and required columns exist (auto-migration at startup).

package schema

import (
	"database/sql"
	"log"
)

const tableComplaintStatusHistory = "complaint_status_history"

// EnsureComplaintStatusHistory ensures the complaint_status_history table exists and has required columns
// (actor_type, actor_id, reason). Creates the table if missing; adds only missing columns if table exists.
// Does not drop or recreate the table; does not remove existing data.
func EnsureComplaintStatusHistory(db *sql.DB) {
	exists, err := tableExists(db, tableComplaintStatusHistory)
	if err != nil {
		log.Fatalf("[SCHEMA] Failed to check if table %s exists: %v", tableComplaintStatusHistory, err)
	}
	if !exists {
		createComplaintStatusHistoryTable(db)
		return
	}
	// Table exists: add any missing columns
	ensureColumn(db, tableComplaintStatusHistory, "actor_type", "VARCHAR(50) NULL COMMENT 'Audit: who made the change (system, authority, user)'")
	ensureColumn(db, tableComplaintStatusHistory, "actor_id", "BIGINT NULL COMMENT 'Audit: user_id or officer_id; NULL for system'")
	ensureColumn(db, tableComplaintStatusHistory, "reason", "TEXT NULL COMMENT 'Audit: reason for change when available'")
	log.Println("[SCHEMA] Schema check passed")
}

func tableExists(db *sql.DB, table string) (bool, error) {
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`,
		table,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func ensureColumn(db *sql.DB, table, column, spec string) {
	exists, err := columnExists(db, table, column)
	if err != nil {
		log.Fatalf("[SCHEMA] Failed to check column %s.%s: %v", table, column, err)
	}
	if exists {
		return
	}
	// MySQL does not support ADD COLUMN IF NOT EXISTS; we checked above so safe to add
	query := "ALTER TABLE " + table + " ADD COLUMN " + column + " " + spec
	if _, err := db.Exec(query); err != nil {
		log.Fatalf("[SCHEMA] Failed to add column %s.%s: %v", table, column, err)
	}
	log.Printf("[SCHEMA] Added missing column: %s", column)
}

func createComplaintStatusHistoryTable(db *sql.DB) {
	// Full table definition including required audit columns (matches database_schema.sql).
	// Requires complaints and users tables to exist (FKs). Fails fast if they do not.
	query := `
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
    actor_type VARCHAR(50) NULL COMMENT 'Audit: who made the change',
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := db.Exec(query); err != nil {
		log.Fatalf("[SCHEMA] Failed to create table %s: %v", tableComplaintStatusHistory, err)
	}
	log.Printf("[SCHEMA] Created table %s with required columns", tableComplaintStatusHistory)
	log.Println("[SCHEMA] Schema check passed")
}
