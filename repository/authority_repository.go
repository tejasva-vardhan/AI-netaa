package repository

import (
	"database/sql"
	"finalneta/models"
	"finalneta/utils"
	"fmt"
)

// AuthorityRepository handles database operations for authority dashboard
type AuthorityRepository struct {
	db *sql.DB
}

// NewAuthorityRepository creates a new authority repository
func NewAuthorityRepository(db *sql.DB) *AuthorityRepository {
	return &AuthorityRepository{db: db}
}

// ValidateCredentials validates email and password for authority login. Passwords stored as bcrypt hashes.
func (r *AuthorityRepository) ValidateCredentials(email, password string) (int64, error) {
	query := `
		SELECT ac.officer_id, ac.password_hash, o.is_active
		FROM authority_credentials ac
		JOIN officers o ON ac.officer_id = o.officer_id
		WHERE ac.email = ? AND ac.is_active = true
		LIMIT 1
	`

	var officerID int64
	var passwordHash string
	var isActive bool

	err := r.db.QueryRow(query, email).Scan(&officerID, &passwordHash, &isActive)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("invalid credentials")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to validate credentials: %w", err)
	}

	if !isActive {
		return 0, fmt.Errorf("officer account is inactive")
	}

	if err := utils.CheckAuthorityPassword(password, passwordHash); err != nil {
		return 0, fmt.Errorf("invalid credentials")
	}

	// Update last login
	r.db.Exec("UPDATE authority_credentials SET last_login_at = NOW() WHERE officer_id = ?", officerID)

	return officerID, nil
}

// ValidateStaticToken validates static token for pilot authentication
func (r *AuthorityRepository) ValidateStaticToken(token string) (int64, error) {
	query := `
		SELECT ac.officer_id, o.is_active
		FROM authority_credentials ac
		JOIN officers o ON ac.officer_id = o.officer_id
		WHERE ac.static_token = ?
		  AND ac.is_active = true
		  AND (ac.token_expires_at IS NULL OR ac.token_expires_at > NOW())
		LIMIT 1
	`

	var officerID int64
	var isActive bool

	err := r.db.QueryRow(query, token).Scan(&officerID, &isActive)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("invalid static token")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to validate token: %w", err)
	}

	if !isActive {
		return 0, fmt.Errorf("officer account is inactive")
	}

	return officerID, nil
}

// VerifyOfficerExists checks if officer exists and is active
func (r *AuthorityRepository) VerifyOfficerExists(officerID int64) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM officers
		WHERE officer_id = ? AND is_active = true
	`

	var count int
	err := r.db.QueryRow(query, officerID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to verify officer: %w", err)
	}

	return count > 0, nil
}

// GetComplaintsByOfficerIDPaginated returns complaints assigned to officer with optional status filter; total count for pagination (read-only, uses assigned_officer_id only).
func (r *AuthorityRepository) GetComplaintsByOfficerIDPaginated(officerID int64, statusFilter string, limit, offset int) ([]models.Complaint, int64, error) {
	args := []interface{}{officerID}
	countQuery := `SELECT COUNT(*) FROM complaints WHERE assigned_officer_id = ?`
	if statusFilter != "" {
		countQuery += ` AND current_status = ?`
		args = append(args, statusFilter)
	}
	var total int64
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count complaints: %w", err)
	}
	listArgs := []interface{}{officerID}
	listQuery := `
		SELECT complaint_id, complaint_number, user_id, title, description, category,
			location_id, latitude, longitude, assigned_department_id, assigned_officer_id,
			current_status, priority, is_public, public_consent_given, supporter_count,
			resolved_at, closed_at, created_at, updated_at
		FROM complaints
		WHERE assigned_officer_id = ?
	`
	if statusFilter != "" {
		listQuery += ` AND current_status = ?`
		listArgs = append(listArgs, statusFilter)
	}
	listQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	listArgs = append(listArgs, limit, offset)
	rows, err := r.db.Query(listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query complaints: %w", err)
	}
	defer rows.Close()
	var complaints []models.Complaint
	for rows.Next() {
		var c models.Complaint
		var updatedAt sql.NullTime
		err := rows.Scan(
			&c.ComplaintID, &c.ComplaintNumber, &c.UserID, &c.Title, &c.Description, &c.Category,
			&c.LocationID, &c.Latitude, &c.Longitude, &c.AssignedDepartmentID, &c.AssignedOfficerID,
			&c.CurrentStatus, &c.Priority, &c.IsPublic, &c.PublicConsentGiven, &c.SupporterCount,
			&c.ResolvedAt, &c.ClosedAt, &c.CreatedAt, &updatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan complaint: %w", err)
		}
		if updatedAt.Valid {
			c.UpdatedAt = updatedAt
		}
		complaints = append(complaints, c)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating complaints: %w", err)
	}
	return complaints, total, nil
}

// GetOfficerProfile returns department_id, location_id, authority_level for login/me (pilot: authority_level from column if present, else 1).
func (r *AuthorityRepository) GetOfficerProfile(officerID int64) (departmentID, locationID int64, authorityLevel int, err error) {
	query := `SELECT department_id, location_id FROM officers WHERE officer_id = ? AND is_active = true`
	err = r.db.QueryRow(query, officerID).Scan(&departmentID, &locationID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get officer profile: %w", err)
	}
	authorityLevel = 1
	_ = r.db.QueryRow(`SELECT COALESCE(authority_level, 1) FROM officers WHERE officer_id = ?`, officerID).Scan(&authorityLevel)
	return departmentID, locationID, authorityLevel, nil
}

// GetComplaintsByOfficerID retrieves all complaints assigned to an officer
// Returns complaints sorted by created_at DESC
func (r *AuthorityRepository) GetComplaintsByOfficerID(officerID int64) ([]models.Complaint, error) {
	query := `
		SELECT 
			complaint_id, complaint_number, user_id, title, description, category,
			location_id, latitude, longitude, assigned_department_id, assigned_officer_id,
			current_status, priority, is_public, public_consent_given, supporter_count,
			resolved_at, closed_at, created_at, updated_at
		FROM complaints
		WHERE assigned_officer_id = ?
		  AND current_status NOT IN ('closed', 'rejected')
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, officerID)
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

// CreateNote creates an internal note for a complaint
func (r *AuthorityRepository) CreateNote(
	complaintID int64,
	officerID int64,
	noteText string,
) (int64, error) {
	query := `
		INSERT INTO authority_notes (
			complaint_id, officer_id, note_text, is_visible_to_citizen, created_at
		) VALUES (?, ?, ?, FALSE, NOW())
	`

	result, err := r.db.Exec(query, complaintID, officerID, noteText)
	if err != nil {
		return 0, fmt.Errorf("failed to create note: %w", err)
	}

	noteID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get note ID: %w", err)
	}

	return noteID, nil
}

// GetNotesByComplaintID retrieves all notes for a complaint (authority view)
func (r *AuthorityRepository) GetNotesByComplaintID(complaintID int64) ([]models.AuthorityNote, error) {
	query := `
		SELECT note_id, complaint_id, officer_id, note_text, created_at
		FROM authority_notes
		WHERE complaint_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}
	defer rows.Close()

	var notes []models.AuthorityNote
	for rows.Next() {
		var note models.AuthorityNote
		err := rows.Scan(
			&note.NoteID,
			&note.ComplaintID,
			&note.OfficerID,
			&note.NoteText,
			&note.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, note)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating notes: %w", err)
	}

	return notes, nil
}

// OfficerListItem is used by admin list; no schema change.
type OfficerListItem struct {
	OfficerID      int64
	FullName       string
	DepartmentID   int64
	LocationID     int64
	AuthorityLevel int
	IsActive       bool
	Email          string
}

// ListOfficers returns all officers with login email (for admin GET /authorities).
func (r *AuthorityRepository) ListOfficers() ([]OfficerListItem, error) {
	query := `
		SELECT o.officer_id, o.full_name, o.department_id, o.location_id,
		       COALESCE(o.authority_level, 1), o.is_active,
		       COALESCE(ac.email, '')
		FROM officers o
		LEFT JOIN authority_credentials ac ON ac.officer_id = o.officer_id AND ac.is_active = true
		ORDER BY o.officer_id
	`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list officers: %w", err)
	}
	defer rows.Close()
	var list []OfficerListItem
	for rows.Next() {
		var row OfficerListItem
		err := rows.Scan(&row.OfficerID, &row.FullName, &row.DepartmentID, &row.LocationID,
			&row.AuthorityLevel, &row.IsActive, &row.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to scan officer: %w", err)
		}
		list = append(list, row)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating officers: %w", err)
	}
	return list, nil
}

// CreateOfficer inserts officer and one credential row. Password is hashed (bcrypt); no plaintext storage.
func (r *AuthorityRepository) CreateOfficer(fullName, designation, email, password string, departmentID, locationID int64, authorityLevel int, isActive bool) (int64, error) {
	hashed, err := utils.HashAuthorityPassword(password)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	authLevel := authorityLevel
	if authLevel < 1 || authLevel > 3 {
		authLevel = 1
	}
	res, err := r.db.Exec(`
		INSERT INTO officers (full_name, designation, email, department_id, location_id, authority_level, is_active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW())`,
		fullName, designation, email, departmentID, locationID, authLevel, isActive)
	if err != nil {
		return 0, fmt.Errorf("failed to create officer: %w", err)
	}
	officerID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get officer id: %w", err)
	}
	_, err = r.db.Exec(`
		INSERT INTO authority_credentials (officer_id, email, password_hash, is_active, created_at)
		VALUES (?, ?, ?, ?, NOW())`,
		officerID, email, hashed, isActive)
	if err != nil {
		return 0, fmt.Errorf("failed to create authority credential: %w", err)
	}
	return officerID, nil
}

// GetOfficerByID returns one officer for admin update; error if not found.
func (r *AuthorityRepository) GetOfficerByID(officerID int64) (fullName string, departmentID, locationID int64, authorityLevel int, isActive bool, err error) {
	err = r.db.QueryRow(`
		SELECT full_name, department_id, location_id, COALESCE(authority_level, 1), is_active
		FROM officers WHERE officer_id = ?`, officerID).
		Scan(&fullName, &departmentID, &locationID, &authorityLevel, &isActive)
	if err != nil {
		return "", 0, 0, 0, false, fmt.Errorf("officer not found: %w", err)
	}
	return fullName, departmentID, locationID, authorityLevel, isActive, nil
}

// UpdateOfficer updates only non-nil fields (department_id, location_id, authority_level, is_active). No escalation rule changes.
func (r *AuthorityRepository) UpdateOfficer(officerID int64, departmentID, locationID *int64, authorityLevel *int, isActive *bool) error {
	row := r.db.QueryRow(`SELECT department_id, location_id, COALESCE(authority_level,1), is_active FROM officers WHERE officer_id = ?`, officerID)
	var dept, loc int64
	var level int
	var active bool
	if err := row.Scan(&dept, &loc, &level, &active); err != nil {
		return fmt.Errorf("officer not found: %w", err)
	}
	if departmentID != nil {
		dept = *departmentID
	}
	if locationID != nil {
		loc = *locationID
	}
	if authorityLevel != nil {
		level = *authorityLevel
		if level < 1 {
			level = 1
		}
		if level > 3 {
			level = 3
		}
	}
	if isActive != nil {
		active = *isActive
	}
	_, err := r.db.Exec(`
		UPDATE officers SET department_id = ?, location_id = ?, authority_level = ?, is_active = ?, updated_at = NOW()
		WHERE officer_id = ?`, dept, loc, level, active, officerID)
	if err != nil {
		return fmt.Errorf("failed to update officer: %w", err)
	}
	return nil
}
