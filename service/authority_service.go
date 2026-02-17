package service

import (
	"database/sql"
	"finalneta/models"
	"finalneta/repository"
	"fmt"
	"time"
)

// AuthorityService handles business logic for authority dashboard
type AuthorityService struct {
	complaintRepo      *repository.ComplaintRepository
	authorityRepo      *repository.AuthorityRepository
	emailShadowService *EmailShadowService // optional; pilot email shadow mode
	pilotMetricsService *PilotMetricsService // optional; pilot metrics
}

// NewAuthorityService creates a new authority service
func NewAuthorityService(
	complaintRepo *repository.ComplaintRepository,
	authorityRepo *repository.AuthorityRepository,
	emailShadowService *EmailShadowService,
	pilotMetricsService *PilotMetricsService,
) *AuthorityService {
	return &AuthorityService{
		complaintRepo:      complaintRepo,
		authorityRepo:      authorityRepo,
		emailShadowService: emailShadowService,
		pilotMetricsService: pilotMetricsService,
	}
}

// ValidateCredentials validates email and password for authority login
func (s *AuthorityService) ValidateCredentials(email, password string) (int64, error) {
	return s.authorityRepo.ValidateCredentials(email, password)
}

// ValidateStaticToken validates static token for pilot authentication
func (s *AuthorityService) ValidateStaticToken(token string) (int64, error) {
	return s.authorityRepo.ValidateStaticToken(token)
}

// VerifyOfficerExists checks if officer exists and is active
func (s *AuthorityService) VerifyOfficerExists(officerID int64) (bool, error) {
	return s.authorityRepo.VerifyOfficerExists(officerID)
}

// GetOfficerProfile returns department_id, location_id, authority_level for the officer (for login JWT and /me).
func (s *AuthorityService) GetOfficerProfile(officerID int64) (departmentID, locationID int64, authorityLevel int, err error) {
	return s.authorityRepo.GetOfficerProfile(officerID)
}

// GetComplaintsByOfficerID retrieves all complaints assigned to an officer (legacy; prefer paginated).
func (s *AuthorityService) GetComplaintsByOfficerID(officerID int64) ([]models.ComplaintSummary, error) {
	complaints, err := s.authorityRepo.GetComplaintsByOfficerID(officerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get complaints: %w", err)
	}
	summaries := make([]models.ComplaintSummary, 0, len(complaints))
	for _, c := range complaints {
		summaries = append(summaries, models.ComplaintSummary{
			ComplaintID:     c.ComplaintID,
			ComplaintNumber: c.ComplaintNumber,
			Title:           c.Title,
			CurrentStatus:   string(c.CurrentStatus),
			Priority:        string(c.Priority),
			CreatedAt:       c.CreatedAt,
			SupporterCount:  c.SupporterCount,
		})
	}
	return summaries, nil
}

// GetComplaintsByOfficerIDPaginated returns paginated complaints assigned to officer with optional status filter (read-only).
func (s *AuthorityService) GetComplaintsByOfficerIDPaginated(officerID int64, statusFilter string, page, pageSize int) ([]models.ComplaintSummary, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	complaints, total, err := s.authorityRepo.GetComplaintsByOfficerIDPaginated(officerID, statusFilter, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	summaries := make([]models.ComplaintSummary, 0, len(complaints))
	for _, c := range complaints {
		summaries = append(summaries, models.ComplaintSummary{
			ComplaintID:     c.ComplaintID,
			ComplaintNumber: c.ComplaintNumber,
			Title:           c.Title,
			CurrentStatus:   string(c.CurrentStatus),
			Priority:        string(c.Priority),
			CreatedAt:       c.CreatedAt,
			SupporterCount:  c.SupporterCount,
		})
	}
	return summaries, total, nil
}

// UpdateComplaintStatus updates complaint status with authority validation
// Enforces valid status transitions: under_review → in_progress → resolved → closed
func (s *AuthorityService) UpdateComplaintStatus(
	complaintID int64,
	officerID int64,
	req *models.AuthorityUpdateStatusRequest,
	ipAddress, userAgent string,
) (*models.UpdateStatusResponse, error) {
	// Step 1: Get complaint and verify assignment
	complaint, err := s.complaintRepo.GetComplaintByID(complaintID)
	if err != nil {
		return nil, fmt.Errorf("complaint not found")
	}

	// Verify complaint is assigned to this officer
	if !complaint.AssignedOfficerID.Valid || complaint.AssignedOfficerID.Int64 != officerID {
		return nil, fmt.Errorf("complaint not assigned to this authority")
	}

	// Step 2: Validate status transition (Authority only: submitted→under_review, under_review→in_progress, in_progress→resolved; closed is system-only).
	newStatus := models.ComplaintStatus(req.NewStatus)
	oldStatus := complaint.CurrentStatus
	if newStatus == models.StatusClosed {
		return nil, fmt.Errorf("invalid status transition: closed is system-only")
	}
	validTransitions := map[models.ComplaintStatus][]models.ComplaintStatus{
		models.StatusSubmitted:   {models.StatusUnderReview},
		models.StatusUnderReview: {models.StatusInProgress},
		models.StatusInProgress:  {models.StatusResolved},
		models.StatusEscalated:   {models.StatusUnderReview, models.StatusInProgress},
	}
	allowedStatuses, ok := validTransitions[oldStatus]
	if !ok {
		return nil, fmt.Errorf("invalid status transition: cannot change from %s", oldStatus)
	}
	validTransition := false
	for _, allowed := range allowedStatuses {
		if newStatus == allowed {
			validTransition = true
			break
		}
	}
	if !validTransition {
		return nil, fmt.Errorf("invalid status transition: cannot change from %s to %s", oldStatus, newStatus)
	}

	// Step 3: Update complaint status via status history (REQUIRED; audit: authority, actor_id, reason)
	statusHistory := &models.ComplaintStatusHistory{
		ComplaintID:          complaintID,
		OldStatus:            sql.NullString{String: string(oldStatus), Valid: true},
		NewStatus:            newStatus,
		ChangedByType:        models.ActorOfficer,
		ChangedByOfficerID:   sql.NullInt64{Int64: officerID, Valid: true},
		ActorType:            sql.NullString{String: string(models.StatusHistoryActorAuthority), Valid: true},
		ActorID:              sql.NullInt64{Int64: officerID, Valid: true},
		Reason:               sql.NullString{String: req.Reason, Valid: true},
		AssignedDepartmentID:  complaint.AssignedDepartmentID,
		AssignedOfficerID:    complaint.AssignedOfficerID,
		Notes:                sql.NullString{String: req.Reason, Valid: true},
	}

	// resolved_at set only when new_status = resolved; closed is system-only (never set here).
	var resolvedAt *time.Time
	if newStatus == models.StatusResolved {
		now := time.Now().UTC()
		resolvedAt = &now
	}
	err = s.complaintRepo.UpdateComplaintStatusWithTimestamps(
		complaintID,
		newStatus,
		resolvedAt,
		nil, // closed_at only set by system
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update complaint status: %w", err)
	}

	// Create status history entry
	err = s.complaintRepo.CreateStatusHistory(statusHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to create status history: %w", err)
	}

	// Check if this is the first authority action (for metrics)
	isFirstAuthorityAction := false
	if s.pilotMetricsService != nil {
		history, err := s.complaintRepo.GetStatusHistory(complaintID)
		if err == nil {
			// Count authority actions (excluding the one we just created)
			authorityActionCount := 0
			for _, h := range history {
				if h.ActorType.Valid && h.ActorType.String == string(models.StatusHistoryActorAuthority) {
					authorityActionCount++
				}
			}
			// If this is the first authority action (count = 1, which is the one we just created)
			isFirstAuthorityAction = authorityActionCount == 1
		}
	}

	// Step 4: Log to audit_log
	auditLog := &models.AuditLog{
		EntityType:      "complaint",
		EntityID:         complaintID,
		Action:           "status_update",
		ActionByType:     models.ActorOfficer,
		ActionByOfficerID: sql.NullInt64{Int64: officerID, Valid: true},
		IPAddress:        sql.NullString{String: ipAddress, Valid: true},
		UserAgent:        sql.NullString{String: userAgent, Valid: true},
	}

	err = s.complaintRepo.CreateAuditLog(auditLog)
	if err != nil {
		// Log error but don't fail - audit logging should be resilient
	}

	// Emit pilot metrics: first_authority_action
	if s.pilotMetricsService != nil && isFirstAuthorityAction {
		metadata := map[string]interface{}{
			"old_status": string(oldStatus),
			"new_status": string(newStatus),
			"officer_id": officerID,
		}
		s.pilotMetricsService.EmitFirstAuthorityAction(complaintID, complaint.UserID, complaint.CreatedAt, metadata)
	}

	// Emit pilot metrics: complaint_resolved (when status becomes resolved or closed)
	if s.pilotMetricsService != nil && (newStatus == models.StatusResolved || newStatus == models.StatusClosed) {
		metadata := map[string]interface{}{
			"old_status": string(oldStatus),
			"new_status": string(newStatus),
			"officer_id": officerID,
		}
		s.pilotMetricsService.EmitComplaintResolved(complaintID, complaint.UserID, complaint.CreatedAt, string(newStatus), metadata)
	}

	// Pilot: send resolution/closure email to shadow inbox only (async, non-blocking)
	// Authority abstraction: department_id only, not officer-based
	if s.emailShadowService != nil && (newStatus == models.StatusResolved || newStatus == models.StatusClosed) {
		if complaint.AssignedDepartmentID.Valid {
			deptID := complaint.AssignedDepartmentID.Int64
			deptName := fmt.Sprintf("Department %d", deptID)
			s.emailShadowService.SendResolutionEmailAsync(
				complaintID,
				complaint.ComplaintNumber,
				deptID,
				deptName,
				req.NewStatus,
				req.Reason,
			)
		}
	}

	return &models.UpdateStatusResponse{
		ComplaintID:     complaintID,
		ComplaintNumber: complaint.ComplaintNumber,
		OldStatus:       string(oldStatus),
		NewStatus:       string(newStatus),
		Message:         "Status updated successfully",
	}, nil
}

// AddNote adds an internal note to a complaint
func (s *AuthorityService) AddNote(
	complaintID int64,
	officerID int64,
	noteText string,
) (*models.AuthorityNoteResponse, error) {
	// Step 1: Verify complaint exists and is assigned to this officer
	complaint, err := s.complaintRepo.GetComplaintByID(complaintID)
	if err != nil {
		return nil, fmt.Errorf("complaint not found")
	}

	if !complaint.AssignedOfficerID.Valid || complaint.AssignedOfficerID.Int64 != officerID {
		return nil, fmt.Errorf("complaint not assigned to this authority")
	}

	// Step 2: Create note
	noteID, err := s.authorityRepo.CreateNote(complaintID, officerID, noteText)
	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	// Step 3: Log to audit_log
	auditLog := &models.AuditLog{
		EntityType:      "complaint",
		EntityID:         complaintID,
		Action:           "add_note",
		ActionByType:     models.ActorOfficer,
		ActionByOfficerID: sql.NullInt64{Int64: officerID, Valid: true},
	}

	err = s.complaintRepo.CreateAuditLog(auditLog)
	if err != nil {
		// Log error but don't fail
	}

	return &models.AuthorityNoteResponse{
		NoteID:      noteID,
		ComplaintID: complaintID,
		Message:     "Note added successfully",
	}, nil
}
