package repository

import (
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"fmt"
	"time"
)

// NotificationRepository handles database operations for notifications
type NotificationRepository struct {
	db *sql.DB
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// CreateNotification creates a new notification record
func (r *NotificationRepository) CreateNotification(notification *models.Notification) error {
	query := `
		INSERT INTO notifications_log (
			entity_type, entity_id, channel, recipient,
			subject, body, template_id, template_data,
			status, priority, retry_count, max_retries,
			next_retry_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var templateDataJSON sql.NullString
	if notification.TemplateData.Valid && notification.TemplateData.String != "" {
		templateDataJSON = notification.TemplateData
	} else if notification.TemplateID.Valid {
		// If template_id is provided but no template_data, create empty JSON
		templateDataJSON = sql.NullString{String: "{}", Valid: true}
	}

	result, err := r.db.Exec(
		query,
		notification.EntityType,
		notification.EntityID,
		notification.Channel,
		notification.Recipient,
		notification.Subject,
		notification.Body,
		notification.TemplateID,
		templateDataJSON,
		notification.Status,
		notification.Priority,
		notification.RetryCount,
		notification.MaxRetries,
		notification.NextRetryAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	notificationID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get notification ID: %w", err)
	}

	notification.NotificationID = notificationID
	return nil
}

// GetPendingNotifications retrieves pending notifications ready to be sent
func (r *NotificationRepository) GetPendingNotifications(limit int) ([]models.Notification, error) {
	query := `
		SELECT 
			notification_id, entity_type, entity_id, channel, recipient,
			subject, body, template_id, template_data,
			status, priority, retry_count, max_retries,
			next_retry_at, sent_at, failed_at, error_message,
			created_at, updated_at
		FROM notifications_log
		WHERE status IN ('pending', 'retrying')
			AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY 
			CASE priority
				WHEN 'urgent' THEN 1
				WHEN 'high' THEN 2
				WHEN 'normal' THEN 3
				WHEN 'low' THEN 4
			END,
			created_at ASC
		LIMIT ?
	`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending notifications: %w", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notification models.Notification
		err := rows.Scan(
			&notification.NotificationID,
			&notification.EntityType,
			&notification.EntityID,
			&notification.Channel,
			&notification.Recipient,
			&notification.Subject,
			&notification.Body,
			&notification.TemplateID,
			&notification.TemplateData,
			&notification.Status,
			&notification.Priority,
			&notification.RetryCount,
			&notification.MaxRetries,
			&notification.NextRetryAt,
			&notification.SentAt,
			&notification.FailedAt,
			&notification.ErrorMessage,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notifications: %w", err)
	}

	return notifications, nil
}

// UpdateNotificationStatus updates notification status and related fields
func (r *NotificationRepository) UpdateNotificationStatus(
	notificationID int64,
	status models.NotificationStatus,
	errorMessage *string,
) error {
	var query string
	var args []interface{}

	switch status {
	case models.NotificationStatusSent:
		query = `
			UPDATE notifications_log
			SET status = ?,
				sent_at = NOW(),
				updated_at = NOW(),
				error_message = NULL
			WHERE notification_id = ?
		`
		args = []interface{}{status, notificationID}
	case models.NotificationStatusFailed:
		query = `
			UPDATE notifications_log
			SET status = ?,
				failed_at = NOW(),
				updated_at = NOW(),
				error_message = ?
			WHERE notification_id = ?
		`
		if errorMessage != nil {
			args = []interface{}{status, *errorMessage, notificationID}
		} else {
			args = []interface{}{status, sql.NullString{}, notificationID}
		}
	case models.NotificationStatusRetrying:
		query = `
			UPDATE notifications_log
			SET status = ?,
				retry_count = retry_count + 1,
				updated_at = NOW(),
				error_message = ?
			WHERE notification_id = ?
		`
		if errorMessage != nil {
			args = []interface{}{status, *errorMessage, notificationID}
		} else {
			args = []interface{}{status, sql.NullString{}, notificationID}
		}
	default:
		query = `
			UPDATE notifications_log
			SET status = ?,
				updated_at = NOW()
			WHERE notification_id = ?
		`
		args = []interface{}{status, notificationID}
	}

	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	return nil
}

// ScheduleRetry schedules a retry for a failed notification
func (r *NotificationRepository) ScheduleRetry(
	notificationID int64,
	nextRetryAt time.Time,
	errorMessage string,
) error {
	query := `
		UPDATE notifications_log
		SET status = 'retrying',
			retry_count = retry_count + 1,
			next_retry_at = ?,
			error_message = ?,
			updated_at = NOW()
		WHERE notification_id = ?
	`

	_, err := r.db.Exec(query, nextRetryAt, errorMessage, notificationID)
	if err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	return nil
}

// CreateNotificationAttemptLog creates a log entry for a notification attempt
func (r *NotificationRepository) CreateNotificationAttemptLog(log *models.NotificationLog) error {
	query := `
		INSERT INTO notification_attempts_log (
			notification_id, attempt_number, status,
			error_message, response_data
		) VALUES (?, ?, ?, ?, ?)
	`

	var responseDataJSON sql.NullString
	if log.ResponseData.Valid {
		responseDataJSON = log.ResponseData
	}

	result, err := r.db.Exec(
		query,
		log.NotificationID,
		log.AttemptNumber,
		log.Status,
		log.ErrorMessage,
		responseDataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification attempt log: %w", err)
	}

	logID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get log ID: %w", err)
	}

	log.LogID = logID
	return nil
}

// SerializeTemplateData converts template data map to JSON string
func SerializeTemplateData(data map[string]interface{}) (string, error) {
	if data == nil || len(data) == 0 {
		return "{}", nil
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
