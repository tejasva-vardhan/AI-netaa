package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// AbusePreventionRepository handles database operations for abuse prevention
type AbusePreventionRepository struct {
	db *sql.DB
}

// NewAbusePreventionRepository creates a new abuse prevention repository
func NewAbusePreventionRepository(db *sql.DB) *AbusePreventionRepository {
	return &AbusePreventionRepository{db: db}
}

// CountComplaintsByUserInLast24Hours counts complaints submitted by user in last 24 hours
func (r *AbusePreventionRepository) CountComplaintsByUserInLast24Hours(userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM complaints
		WHERE user_id = ?
		  AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		  AND current_status != 'rejected'
	`

	var count int
	err := r.db.QueryRow(query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count complaints: %w", err)
	}

	return count, nil
}

// HasDuplicateComplaint checks if user has submitted similar complaint recently
// Same user + same issue_summary (title) + same pincode within time window.
// If pincode column is missing (migration not run), falls back to user_id + title only.
func (r *AbusePreventionRepository) HasDuplicateComplaint(
	userID int64,
	issueSummary string,
	pincode string,
	withinDuration time.Duration,
) (bool, error) {
	cutoffTime := time.Now().UTC().Add(-withinDuration)

	queryWithPincode := `
		SELECT COUNT(*) > 0
		FROM complaints
		WHERE user_id = ?
		  AND title = ?
		  AND (pincode = ? OR (pincode IS NULL AND ? = ''))
		  AND created_at >= ?
		  AND current_status != 'rejected'
	`
	var exists bool
	err := r.db.QueryRow(queryWithPincode, userID, issueSummary, pincode, pincode, cutoffTime).Scan(&exists)
	if err != nil {
		// If pincode column does not exist (migration not run), fall back to check without pincode
		if strings.Contains(err.Error(), "pincode") || strings.Contains(err.Error(), "Unknown column") {
			queryWithoutPincode := `
				SELECT COUNT(*) > 0
				FROM complaints
				WHERE user_id = ?
				  AND title = ?
				  AND created_at >= ?
				  AND current_status != 'rejected'
			`
			err = r.db.QueryRow(queryWithoutPincode, userID, issueSummary, cutoffTime).Scan(&exists)
		}
		if err != nil {
			return false, fmt.Errorf("failed to check duplicate: %w", err)
		}
	}
	return exists, nil
}
