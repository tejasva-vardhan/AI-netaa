package repository

import (
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"fmt"
	"log"
	"time"
)

// EscalationRepository handles database operations for escalations
type EscalationRepository struct {
	db *sql.DB
}

// NewEscalationRepository creates a new escalation repository
func NewEscalationRepository(db *sql.DB) *EscalationRepository {
	return &EscalationRepository{db: db}
}

// GetActiveEscalationRules retrieves all active escalation rules
func (r *EscalationRepository) GetActiveEscalationRules() ([]models.EscalationRule, error) {
	query := `
		SELECT 
			rule_id, from_department_id, from_location_id,
			to_department_id, to_location_id, escalation_level,
			conditions, is_active, created_at, updated_at
		FROM escalation_rules
		WHERE is_active = true
		ORDER BY escalation_level ASC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query escalation rules: %w", err)
	}
	defer rows.Close()

	var rules []models.EscalationRule
	for rows.Next() {
		var rule models.EscalationRule
		err := rows.Scan(
			&rule.RuleID,
			&rule.FromDepartmentID,
			&rule.FromLocationID,
			&rule.ToDepartmentID,
			&rule.ToLocationID,
			&rule.EscalationLevel,
			&rule.Conditions,
			&rule.IsActive,
			&rule.CreatedAt,
			&rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan escalation rule: %w", err)
		}
		rules = append(rules, rule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating escalation rules: %w", err)
	}

	return rules, nil
}

// GetEscalationCandidates retrieves complaints that may need escalation.
// Only filters by status; SLA timing is applied later in evaluateEscalationConditions.
// (Previously a 24h "stale" filter excluded recent complaints and prevented 2-min pilot escalation.)
func (r *EscalationRepository) GetEscalationCandidates(
	statuses []models.ComplaintStatus,
	_ time.Duration, // unused; kept for API compatibility
) ([]models.EscalationCandidate, error) {
	statusFilter := "AND 1=1"
	args := []interface{}{}

	if len(statuses) > 0 {
		statusFilter = "AND c.current_status IN ("
		for i, status := range statuses {
			if i > 0 {
				statusFilter += ", "
			}
			statusFilter += "?"
			args = append(args, string(status))
		}
		statusFilter += ")"
	}

	query := fmt.Sprintf(`
		SELECT 
			c.complaint_id,
			c.complaint_number,
			c.current_status,
			c.priority,
			c.assigned_department_id,
			c.assigned_officer_id,
			c.location_id,
			c.pincode,
			c.created_at,
			c.updated_at,
			COALESCE(
				(SELECT MAX(created_at) 
				 FROM complaint_status_history 
				 WHERE complaint_id = c.complaint_id),
				c.created_at
			) as last_status_change_at
		FROM complaints c
		WHERE c.current_status NOT IN ('resolved', 'closed', 'rejected')
			%s
		ORDER BY c.created_at ASC
	`, statusFilter)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query escalation candidates: %w", err)
	}
	defer rows.Close()

	var candidates []models.EscalationCandidate
	for rows.Next() {
		var candidate models.EscalationCandidate
		err := rows.Scan(
			&candidate.ComplaintID,
			&candidate.ComplaintNumber,
			&candidate.CurrentStatus,
			&candidate.Priority,
			&candidate.AssignedDepartmentID,
			&candidate.AssignedOfficerID,
			&candidate.LocationID,
			&candidate.Pincode,
			&candidate.CreatedAt,
			&candidate.UpdatedAt,
			&candidate.LastStatusChangeAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan escalation candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating escalation candidates: %w", err)
	}

	return candidates, nil
}

// HasExistingEscalation checks if complaint already has an escalation at the given level
// This ensures idempotency - don't escalate twice at the same level
func (r *EscalationRepository) HasExistingEscalation(
	complaintID int64,
	escalationLevel int,
	withinHours int,
) (bool, error) {
	cutoffTime := time.Now().UTC().Add(-time.Duration(withinHours) * time.Hour)

	query := `
		SELECT COUNT(*) 
		FROM complaint_escalations
		WHERE complaint_id = ?
			AND escalation_level = ?
			AND created_at >= ?
	`

	var count int
	err := r.db.QueryRow(query, complaintID, escalationLevel, cutoffTime).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check existing escalation: %w", err)
	}

	return count > 0, nil
}

// GetLastEscalationLevel gets the highest escalation level for a complaint
func (r *EscalationRepository) GetLastEscalationLevel(complaintID int64) (int, error) {
	query := `
		SELECT COALESCE(MAX(escalation_level), 0)
		FROM complaint_escalations
		WHERE complaint_id = ?
	`

	var level int
	err := r.db.QueryRow(query, complaintID).Scan(&level)
	if err != nil {
		return 0, fmt.Errorf("failed to get last escalation level: %w", err)
	}

	return level, nil
}

// CreateEscalation creates a new escalation record
func (r *EscalationRepository) CreateEscalation(escalation *models.ComplaintEscalation) error {
	query := `
		INSERT INTO complaint_escalations (
			complaint_id, from_department_id, from_officer_id,
			to_department_id, to_officer_id, escalation_level,
			reason, escalated_by_type, escalated_by_user_id,
			escalated_by_officer_id, status_history_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.Exec(
		query,
		escalation.ComplaintID,
		escalation.FromDepartmentID,
		escalation.FromOfficerID,
		escalation.ToDepartmentID,
		escalation.ToOfficerID,
		escalation.EscalationLevel,
		escalation.Reason,
		escalation.EscalatedByType,
		escalation.EscalatedByUserID,
		escalation.EscalatedByOfficerID,
		escalation.StatusHistoryID,
	)
	if err != nil {
		return fmt.Errorf("failed to create escalation: %w", err)
	}

	escalationID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get escalation ID: %w", err)
	}

	escalation.EscalationID = escalationID
	return nil
}

// GetLastReminderTime gets the last reminder time for a complaint (if any)
// Checks audit_log for reminder actions
func (r *EscalationRepository) GetLastReminderTime(complaintID int64) (*time.Time, error) {
	query := `
		SELECT MAX(created_at)
		FROM audit_log
		WHERE entity_type = 'complaint'
			AND entity_id = ?
			AND action = 'reminder'
	`

	var lastReminder sql.NullTime
	err := r.db.QueryRow(query, complaintID).Scan(&lastReminder)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get last reminder time: %w", err)
	}

	if !lastReminder.Valid {
		return nil, nil
	}

	return &lastReminder.Time, nil
}

// ParseEscalationConditions parses JSON conditions from escalation rule
func ParseEscalationConditions(conditionsJSON sql.NullString) (*models.EscalationConditions, error) {
	if !conditionsJSON.Valid || conditionsJSON.String == "" {
		return nil, nil
	}

	var conditions models.EscalationConditions
	err := json.Unmarshal([]byte(conditionsJSON.String), &conditions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse escalation conditions: %w", err)
	}

	return &conditions, nil
}

// FindAuthorityByDepartmentPincodeLevel finds authority (officer) for escalation.
// Tries (1) employee_id pattern L{level}. [PILOT] Then (2) any active officer in department+location.
func (r *EscalationRepository) FindAuthorityByDepartmentPincodeLevel(
	departmentID int64,
	locationID int64,
	escalationLevel int, // CURRENT level before escalation (0=L1, 1=L2)
) (*int64, error) {
	targetAuthorityLevel := escalationLevel + 1
	log.Printf("[ESCALATION_DEBUG] FindAuthorityByDepartmentPincodeLevel: department_id=%d location_id=%d current_level=%d target_authority_level=%d",
		departmentID, locationID, escalationLevel, targetAuthorityLevel)

	// 1) Prefer officer matching employee_id pattern (e.g. PHED-L2-001)
	pattern1 := fmt.Sprintf("%%-L%d-%%", targetAuthorityLevel)
	pattern2 := fmt.Sprintf("%%-L%d%%", targetAuthorityLevel)
	queryPattern := `
		SELECT officer_id FROM officers
		WHERE department_id = ? AND location_id = ? AND is_active = true
		AND (employee_id LIKE ? OR employee_id LIKE ?)
		LIMIT 1
	`
	log.Printf("[ESCALATION_DEBUG] Authority SQL (pattern): SELECT officer_id FROM officers WHERE department_id=? AND location_id=? AND is_active=true AND (employee_id LIKE ? OR employee_id LIKE ?) LIMIT 1")
	var officerID sql.NullInt64
	err := r.db.QueryRow(queryPattern, departmentID, locationID, pattern1, pattern2).Scan(&officerID)
	if err == nil && officerID.Valid {
		log.Printf("[ESCALATION_DEBUG] Authority lookup (pattern): rows_returned=1 officer_id=%d", officerID.Int64)
		return &officerID.Int64, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to find authority (pattern): %w", err)
	}
	log.Printf("[ESCALATION_DEBUG] Authority lookup (pattern): rows_returned=0, trying any active officer [PILOT]")

	// [PILOT] Fallback: any active officer in department + location
	queryAny := `
		SELECT officer_id FROM officers
		WHERE department_id = ? AND location_id = ? AND is_active = true
		LIMIT 1
	`
	err = r.db.QueryRow(queryAny, departmentID, locationID).Scan(&officerID)
	if err == sql.ErrNoRows {
		log.Printf("[ESCALATION_DEBUG] Authority lookup (any): rows_returned=0")
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find authority (any): %w", err)
	}
	if !officerID.Valid {
		log.Printf("[ESCALATION_DEBUG] Authority lookup (any): rows_returned=1 but officer_id NULL")
		return nil, nil
	}
	log.Printf("[ESCALATION_DEBUG] Authority lookup (any): rows_returned=1 officer_id=%d", officerID.Int64)
	return &officerID.Int64, nil
}
