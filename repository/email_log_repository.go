package repository

import (
	"database/sql"
	"finalneta/models"
	"fmt"
)

// EmailLogRepository handles email_logs table (pilot shadow mode)
type EmailLogRepository struct {
	db *sql.DB
}

// NewEmailLogRepository creates a new email log repository
func NewEmailLogRepository(db *sql.DB) *EmailLogRepository {
	return &EmailLogRepository{db: db}
}

// Create inserts an email log record (status = pending until send completes).
func (r *EmailLogRepository) Create(log *models.EmailLog) error {
	status := log.Status
	if status == "" {
		status = "pending"
	}
	query := `
		INSERT INTO email_logs (
			complaint_id, email_type, intended_authority_id, intended_level,
			department_id, sent_to_email, subject, body, status, error_message, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
	`
	result, err := r.db.Exec(
		query,
		log.ComplaintID,
		log.EmailType,
		log.IntendedAuthorityID,
		log.IntendedLevel,
		log.DepartmentID,
		log.SentToEmail,
		log.Subject,
		log.Body,
		status,
		log.ErrorMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to create email log: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get email log id: %w", err)
	}
	log.ID = id
	return nil
}

// UpdateStatus sets status and optional error_message for an email log (after send attempt).
func (r *EmailLogRepository) UpdateStatus(id int64, status string, errorMessage string) error {
	var errMsg interface{}
	if errorMessage != "" {
		errMsg = errorMessage
	} else {
		errMsg = nil
	}
	_, err := r.db.Exec(
		`UPDATE email_logs SET status = ?, error_message = ? WHERE id = ?`,
		status, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update email log status: %w", err)
	}
	return nil
}
