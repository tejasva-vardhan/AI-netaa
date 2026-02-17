package repository

import (
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ComplaintRepository handles database operations for complaints
type ComplaintRepository struct {
	db *sql.DB
}

// NewComplaintRepository creates a new complaint repository
func NewComplaintRepository(db *sql.DB) *ComplaintRepository {
	return &ComplaintRepository{db: db}
}

// GenerateComplaintNumber generates a unique complaint number
// Format: COMP-YYYYMMDD-{UUID}
func (r *ComplaintRepository) GenerateComplaintNumber() (string, error) {
	datePrefix := time.Now().UTC().Format("20060102")
	uniqueID := uuid.New().String()[:8]
	return fmt.Sprintf("COMP-%s-%s", datePrefix, uniqueID), nil
}

// CreateComplaint creates a new complaint in the database
func (r *ComplaintRepository) CreateComplaint(complaint *models.Complaint) error {
	// ISSUE 4: Final defensive check - verify user exists before insert
	// This is a last-resort check in case handler check was bypassed
	var userExists int
	checkQuery := `SELECT COUNT(*) FROM users WHERE user_id = ?`
	err := r.db.QueryRow(checkQuery, complaint.UserID).Scan(&userExists)
	if err != nil {
		return fmt.Errorf("failed to verify user existence before insert: %w", err)
	}
	if userExists == 0 {
		return fmt.Errorf("user_id %d does not exist in users table - cannot create complaint", complaint.UserID)
	}

	query := `
		INSERT INTO complaints (
			complaint_number, user_id, title, description, category,
			location_id, latitude, longitude, pincode,
			assigned_department_id, assigned_officer_id, current_status, priority, 
			is_public, public_consent_given, supporter_count, device_fingerprint
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		complaint.ComplaintNumber,
		complaint.UserID,
		complaint.Title,
		complaint.Description,
		complaint.Category,
		complaint.LocationID,
		complaint.Latitude,
		complaint.Longitude,
		complaint.Pincode,
		complaint.AssignedDepartmentID,
		complaint.AssignedOfficerID,
		complaint.CurrentStatus,
		complaint.Priority,
		complaint.IsPublic,
		complaint.PublicConsentGiven,
		complaint.SupporterCount,
		complaint.DeviceFingerprint,
	)
	if err != nil && (strings.Contains(err.Error(), "pincode") || strings.Contains(err.Error(), "device_fingerprint") || strings.Contains(err.Error(), "Unknown column")) {
		// Fallback: schema may not have pincode/device_fingerprint (abuse prevention migration not run)
		queryFallback := `
			INSERT INTO complaints (
				complaint_number, user_id, title, description, category,
				location_id, latitude, longitude,
				assigned_department_id, assigned_officer_id, current_status, priority, 
				is_public, public_consent_given, supporter_count
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		result, err = r.db.Exec(
			queryFallback,
			complaint.ComplaintNumber,
			complaint.UserID,
			complaint.Title,
			complaint.Description,
			complaint.Category,
			complaint.LocationID,
			complaint.Latitude,
			complaint.Longitude,
			complaint.AssignedDepartmentID,
			complaint.AssignedOfficerID,
			complaint.CurrentStatus,
			complaint.Priority,
			complaint.IsPublic,
			complaint.PublicConsentGiven,
			complaint.SupporterCount,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to create complaint: %w", err)
	}

	complaintID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get complaint ID: %w", err)
	}

	complaint.ComplaintID = complaintID
	return nil
}

// GetComplaintsByUserID retrieves all complaints for a specific user
func (r *ComplaintRepository) GetComplaintsByUserID(userID int64) ([]models.Complaint, error) {
	query := `
		SELECT 
			complaint_id, complaint_number, user_id, title, description, category,
			location_id, latitude, longitude, assigned_department_id, assigned_officer_id,
			current_status, priority, is_public, public_consent_given, supporter_count,
			resolved_at, closed_at, created_at, updated_at
		FROM complaints
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query complaints: %w", err)
	}
	defer rows.Close()

	var complaints []models.Complaint
	for rows.Next() {
		var complaint models.Complaint
		var updatedAt sql.NullTime

		err := rows.Scan(
			&complaint.ComplaintID, &complaint.ComplaintNumber, &complaint.UserID,
			&complaint.Title, &complaint.Description, &complaint.Category,
			&complaint.LocationID, &complaint.Latitude, &complaint.Longitude,
			&complaint.AssignedDepartmentID, &complaint.AssignedOfficerID,
			&complaint.CurrentStatus, &complaint.Priority,
			&complaint.IsPublic, &complaint.PublicConsentGiven, &complaint.SupporterCount,
			&complaint.ResolvedAt, &complaint.ClosedAt,
			&complaint.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan complaint: %w", err)
		}

		if updatedAt.Valid {
			complaint.UpdatedAt = updatedAt
		}

		complaints = append(complaints, complaint)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating complaints: %w", err)
	}

	return complaints, nil
}

// GetComplaintOwnerID returns the user_id (owner) of the complaint, or error if not found.
func (r *ComplaintRepository) GetComplaintOwnerID(complaintID int64) (int64, error) {
	var userID int64
	err := r.db.QueryRow(`SELECT user_id FROM complaints WHERE complaint_id = ?`, complaintID).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("complaint not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get complaint owner: %w", err)
	}
	return userID, nil
}

// GetComplaintByID retrieves a complaint by its ID
func (r *ComplaintRepository) GetComplaintByID(complaintID int64) (*models.Complaint, error) {
	query := `
		SELECT 
			complaint_id, complaint_number, user_id, title, description,
			category, location_id, latitude, longitude,
			assigned_department_id, assigned_officer_id, current_status,
			priority, is_public, public_consent_given, supporter_count,
			resolved_at, closed_at, created_at, updated_at
		FROM complaints
		WHERE complaint_id = ?
	`

	var complaint models.Complaint
	err := r.db.QueryRow(query, complaintID).Scan(
		&complaint.ComplaintID,
		&complaint.ComplaintNumber,
		&complaint.UserID,
		&complaint.Title,
		&complaint.Description,
		&complaint.Category,
		&complaint.LocationID,
		&complaint.Latitude,
		&complaint.Longitude,
		&complaint.AssignedDepartmentID,
		&complaint.AssignedOfficerID,
		&complaint.CurrentStatus,
		&complaint.Priority,
		&complaint.IsPublic,
		&complaint.PublicConsentGiven,
		&complaint.SupporterCount,
		&complaint.ResolvedAt,
		&complaint.ClosedAt,
		&complaint.CreatedAt,
		&complaint.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("complaint not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get complaint: %w", err)
	}

	return &complaint, nil
}

// GetComplaintByNumber retrieves a complaint by its complaint number
func (r *ComplaintRepository) GetComplaintByNumber(complaintNumber string) (*models.Complaint, error) {
	query := `
		SELECT 
			complaint_id, complaint_number, user_id, title, description,
			category, location_id, latitude, longitude,
			assigned_department_id, assigned_officer_id, current_status,
			priority, is_public, public_consent_given, supporter_count,
			resolved_at, closed_at, created_at, updated_at
		FROM complaints
		WHERE complaint_number = ?
	`

	var complaint models.Complaint
	err := r.db.QueryRow(query, complaintNumber).Scan(
		&complaint.ComplaintID,
		&complaint.ComplaintNumber,
		&complaint.UserID,
		&complaint.Title,
		&complaint.Description,
		&complaint.Category,
		&complaint.LocationID,
		&complaint.Latitude,
		&complaint.Longitude,
		&complaint.AssignedDepartmentID,
		&complaint.AssignedOfficerID,
		&complaint.CurrentStatus,
		&complaint.Priority,
		&complaint.IsPublic,
		&complaint.PublicConsentGiven,
		&complaint.SupporterCount,
		&complaint.ResolvedAt,
		&complaint.ClosedAt,
		&complaint.CreatedAt,
		&complaint.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("complaint not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get complaint: %w", err)
	}

	return &complaint, nil
}

// UpdateComplaintStatus updates the status and related fields of a complaint
func (r *ComplaintRepository) UpdateComplaintStatus(
	complaintID int64,
	newStatus models.ComplaintStatus,
	assignedDepartmentID *int64,
	assignedOfficerID *int64,
) error {
	query := `
		UPDATE complaints
		SET current_status = ?,
			assigned_department_id = ?,
			assigned_officer_id = ?,
			updated_at = NOW()
		WHERE complaint_id = ?
	`

	_, err := r.db.Exec(
		query,
		newStatus,
		assignedDepartmentID,
		assignedOfficerID,
		complaintID,
	)
	if err != nil {
		return fmt.Errorf("failed to update complaint status: %w", err)
	}

	// Update resolved_at or closed_at based on status
	if newStatus == models.StatusResolved {
		_, err = r.db.Exec(
			"UPDATE complaints SET resolved_at = NOW() WHERE complaint_id = ? AND resolved_at IS NULL",
			complaintID,
		)
		if err != nil {
			return fmt.Errorf("failed to update resolved_at: %w", err)
		}
	}

	if newStatus == models.StatusClosed {
		_, err = r.db.Exec(
			"UPDATE complaints SET closed_at = NOW() WHERE complaint_id = ? AND closed_at IS NULL",
			complaintID,
		)
		if err != nil {
			return fmt.Errorf("failed to update closed_at: %w", err)
		}
	}

	return nil
}

// UpdateComplaintEscalationLevel sets complaints.current_escalation_level (0=L1, 1=L2, 2=L3).
// Call after creating an escalation record so the complaint reflects the new level.
func (r *ComplaintRepository) UpdateComplaintEscalationLevel(complaintID int64, level int) error {
	_, err := r.db.Exec(
		`UPDATE complaints SET current_escalation_level = ?, updated_at = NOW() WHERE complaint_id = ?`,
		level,
		complaintID,
	)
	if err != nil {
		return fmt.Errorf("failed to update complaint escalation level: %w", err)
	}
	return nil
}

// UpdateComplaintStatusWithTimestamps updates complaint status with resolved_at/closed_at timestamps
func (r *ComplaintRepository) UpdateComplaintStatusWithTimestamps(
	complaintID int64,
	newStatus models.ComplaintStatus,
	resolvedAt *time.Time,
	closedAt *time.Time,
) error {
	query := `
		UPDATE complaints
		SET current_status = ?,
			resolved_at = ?,
			closed_at = ?,
			updated_at = NOW()
		WHERE complaint_id = ?
	`

	_, err := r.db.Exec(
		query,
		newStatus,
		resolvedAt,
		closedAt,
		complaintID,
	)
	if err != nil {
		return fmt.Errorf("failed to update complaint status: %w", err)
	}

	return nil
}

// auditActorType maps ActorType to status_history actor_type enum ('system','authority','user').
func auditActorType(t models.ActorType) string {
	switch t {
	case models.ActorUser:
		return "user"
	case models.ActorOfficer, models.ActorAdmin:
		return "authority"
	case models.ActorSystem:
		return "system"
	default:
		return "system"
	}
}

// CreateStatusHistory creates a new status history entry (immutable). Always writes actor_type, actor_id, reason when available.
func (r *ComplaintRepository) CreateStatusHistory(history *models.ComplaintStatusHistory) error {
	var actorType string
	if history.ActorType.Valid && history.ActorType.String != "" {
		actorType = history.ActorType.String
	} else {
		actorType = auditActorType(history.ChangedByType)
	}
	var actorID sql.NullInt64
	if history.ActorID.Valid {
		actorID = history.ActorID
	} else if history.ChangedByUserID.Valid {
		actorID = history.ChangedByUserID
	} else if history.ChangedByOfficerID.Valid {
		actorID = history.ChangedByOfficerID
	}
	reason := history.Reason
	if !reason.Valid && history.Notes.Valid {
		reason = history.Notes
	}

	query := `
		INSERT INTO complaint_status_history (
			complaint_id, old_status, new_status, changed_by_type,
			changed_by_user_id, changed_by_officer_id,
			assigned_department_id, assigned_officer_id, notes,
			actor_type, actor_id, reason
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		history.ComplaintID,
		history.OldStatus,
		history.NewStatus,
		history.ChangedByType,
		history.ChangedByUserID,
		history.ChangedByOfficerID,
		history.AssignedDepartmentID,
		history.AssignedOfficerID,
		history.Notes,
		actorType,
		actorID,
		reason,
	)
	if err != nil {
		return fmt.Errorf("failed to create status history: %w", err)
	}

	historyID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get history ID: %w", err)
	}

	history.HistoryID = historyID
	return nil
}

// PublicComplaintData is whitelisted for public case page. Complaints table only; no joins; complaint_id never exposed.
type PublicComplaintData struct {
	ComplaintNumber string
	LocationID      int64
	DepartmentID    int64 // 0 if NULL
	CurrentStatus   string
	CreatedAt       time.Time
}

// GetPublicComplaintByNumber fetches by complaint_number (shareable identifier). Returns data + complaintID for internal timeline fetch only; complaint_id never exposed.
func (r *ComplaintRepository) GetPublicComplaintByNumber(complaintNumber string) (*PublicComplaintData, int64, error) {
	query := `
		SELECT complaint_id, complaint_number, location_id, COALESCE(assigned_department_id, 0), current_status, created_at
		FROM complaints
		WHERE complaint_number = ?
	`
	var complaintID int64
	var data PublicComplaintData
	err := r.db.QueryRow(query, complaintNumber).Scan(
		&complaintID,
		&data.ComplaintNumber,
		&data.LocationID,
		&data.DepartmentID,
		&data.CurrentStatus,
		&data.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get public complaint: %w", err)
	}
	return &data, complaintID, nil
}

// GetStatusHistory retrieves the status timeline for a complaint (ordered by created_at DESC)
func (r *ComplaintRepository) GetStatusHistory(complaintID int64) ([]models.ComplaintStatusHistory, error) {
	query := `
		SELECT 
			history_id, complaint_id, old_status, new_status,
			changed_by_type, changed_by_user_id, changed_by_officer_id,
			assigned_department_id, assigned_officer_id, notes,
			actor_type, actor_id, reason,
			created_at
		FROM complaint_status_history
		WHERE complaint_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to query status history: %w", err)
	}
	defer rows.Close()

	var history []models.ComplaintStatusHistory
	for rows.Next() {
		var h models.ComplaintStatusHistory
		err := rows.Scan(
			&h.HistoryID,
			&h.ComplaintID,
			&h.OldStatus,
			&h.NewStatus,
			&h.ChangedByType,
			&h.ChangedByUserID,
			&h.ChangedByOfficerID,
			&h.AssignedDepartmentID,
			&h.AssignedOfficerID,
			&h.Notes,
			&h.ActorType,
			&h.ActorID,
			&h.Reason,
			&h.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan status history: %w", err)
		}
		history = append(history, h)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating status history: %w", err)
	}

	return history, nil
}

// CreateAttachment creates a new attachment record
func (r *ComplaintRepository) CreateAttachment(attachment *models.ComplaintAttachment) error {
	query := `
		INSERT INTO complaint_attachments (
			complaint_id, file_name, file_path, file_type,
			file_size, uploaded_by_user_id, is_public
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		attachment.ComplaintID,
		attachment.FileName,
		attachment.FilePath,
		attachment.FileType,
		attachment.FileSize,
		attachment.UploadedByUserID,
		attachment.IsPublic,
	)
	if err != nil {
		return fmt.Errorf("failed to create attachment: %w", err)
	}

	attachmentID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get attachment ID: %w", err)
	}

	attachment.AttachmentID = attachmentID
	return nil
}

// GetAttachmentsByComplaintID retrieves all attachments for a complaint
func (r *ComplaintRepository) GetAttachmentsByComplaintID(complaintID int64) ([]models.ComplaintAttachment, error) {
	query := `
		SELECT 
			attachment_id, complaint_id, file_name, file_path,
			file_type, file_size, uploaded_by_user_id, is_public, created_at
		FROM complaint_attachments
		WHERE complaint_id = ?
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(query, complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to query attachments: %w", err)
	}
	defer rows.Close()

	var attachments []models.ComplaintAttachment
	for rows.Next() {
		var a models.ComplaintAttachment
		err := rows.Scan(
			&a.AttachmentID,
			&a.ComplaintID,
			&a.FileName,
			&a.FilePath,
			&a.FileType,
			&a.FileSize,
			&a.UploadedByUserID,
			&a.IsPublic,
			&a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}
		attachments = append(attachments, a)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attachments: %w", err)
	}

	return attachments, nil
}

// CreateAuditLog creates a new audit log entry (immutable)
func (r *ComplaintRepository) CreateAuditLog(audit *models.AuditLog) error {
	query := `
		INSERT INTO audit_log (
			entity_type, entity_id, action, action_by_type,
			action_by_user_id, action_by_officer_id,
			old_values, new_values, changes,
			ip_address, user_agent, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		audit.EntityType,
		audit.EntityID,
		audit.Action,
		audit.ActionByType,
		audit.ActionByUserID,
		audit.ActionByOfficerID,
		audit.OldValues,
		audit.NewValues,
		audit.Changes,
		audit.IPAddress,
		audit.UserAgent,
		audit.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	auditID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get audit ID: %w", err)
	}

	audit.AuditID = auditID
	return nil
}

// SerializeToJSON converts a struct to JSON string for audit log storage
func SerializeToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
