package service

import (
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"finalneta/repository"
	"fmt"
	"log"
)

// ComplaintService handles business logic for complaints
type ComplaintService struct {
	repo              *repository.ComplaintRepository
	departmentRepo    *repository.DepartmentRepository
	emailShadowService *EmailShadowService // optional; pilot email shadow mode
	pilotMetricsService *PilotMetricsService // optional; pilot metrics
}

// NewComplaintService creates a new complaint service
func NewComplaintService(
	repo *repository.ComplaintRepository,
	departmentRepo *repository.DepartmentRepository,
	emailShadowService *EmailShadowService,
	pilotMetricsService *PilotMetricsService,
) *ComplaintService {
	return &ComplaintService{
		repo:               repo,
		departmentRepo:     departmentRepo,
		emailShadowService: emailShadowService,
		pilotMetricsService: pilotMetricsService,
	}
}

// CreateComplaint creates a new complaint with proper lifecycle initialization
//
// Lifecycle Rules:
// 1. New complaints start with status 'draft' or 'submitted' based on completion
// 2. Initial status history entry is created
// 3. All attachments are linked to the complaint
// 4. Audit log entry is created for creation action
func (s *ComplaintService) CreateComplaint(
	req *models.CreateComplaintRequest,
	userID int64,
	ipAddress, userAgent string,
) (*models.CreateComplaintResponse, error) {
	// Generate unique complaint number
	complaintNumber, err := s.repo.GenerateComplaintNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate complaint number: %w", err)
	}

	// Determine initial status
	// If all required fields are present, start as 'submitted', otherwise 'draft'
	initialStatus := models.StatusSubmitted
	if req.Title == "" || req.Description == "" {
		initialStatus = models.StatusDraft
	}

	// Set priority (default to medium if not provided)
	priority := models.PriorityMedium
	if req.Priority != nil {
		switch *req.Priority {
		case "low":
			priority = models.PriorityLow
		case "medium":
			priority = models.PriorityMedium
		case "high":
			priority = models.PriorityHigh
		case "urgent":
			priority = models.PriorityUrgent
		}
	}

	// Create complaint entity
	complaint := &models.Complaint{
		ComplaintNumber:    complaintNumber,
		UserID:             userID,
		Title:              req.Title,
		Description:        req.Description,
		LocationID:         req.LocationID,
		CurrentStatus:      initialStatus,
		Priority:           priority,
		IsPublic:           req.PublicConsentGiven,
		PublicConsentGiven: req.PublicConsentGiven,
		SupporterCount:     0,
	}

	// Handle optional fields
	if req.Category != nil {
		complaint.Category = sql.NullString{String: *req.Category, Valid: true}
	}
	if req.Latitude != nil {
		complaint.Latitude = sql.NullFloat64{Float64: *req.Latitude, Valid: true}
	}
	if req.Longitude != nil {
		complaint.Longitude = sql.NullFloat64{Float64: *req.Longitude, Valid: true}
	}
	if req.Pincode != nil && *req.Pincode != "" {
		complaint.Pincode = sql.NullString{String: *req.Pincode, Valid: true}
	}
	if req.DeviceFingerprint != nil && *req.DeviceFingerprint != "" {
		complaint.DeviceFingerprint = sql.NullString{String: *req.DeviceFingerprint, Valid: true}
	}

	// Auto-assign department based on category (ALWAYS assign, fallback to default)
	// Default department: ID 7 (District Collector Office)
	const defaultDepartmentID int64 = 7
	var assignedDeptID int64 = defaultDepartmentID
	var priorityOverride *string

	// Try to get department from category mapping if category provided
	if req.Category != nil && *req.Category != "" && s.departmentRepo != nil {
		deptID, prioOverride, err := s.departmentRepo.GetDepartmentByCategoryAndLocation(
			*req.Category,
			req.LocationID,
		)
		if err == nil && deptID != nil {
			assignedDeptID = *deptID
			priorityOverride = prioOverride
		}
		// If mapping fails, assignedDeptID remains defaultDepartmentID (fallback)
	}

	// ALWAYS assign department (either from category mapping or default)
	complaint.AssignedDepartmentID = sql.NullInt64{Int64: assignedDeptID, Valid: true}
	log.Printf("[complaint] Assigned to department ID=%d", assignedDeptID)

	// Override priority if category mapping specifies it
	if priorityOverride != nil {
		switch *priorityOverride {
		case "low":
			priority = models.PriorityLow
		case "medium":
			priority = models.PriorityMedium
		case "high":
			priority = models.PriorityHigh
		case "urgent":
			priority = models.PriorityUrgent
		}
	}

	// Try to find an officer for this department and location (optional)
	if s.departmentRepo != nil {
		officerID, err := s.departmentRepo.FindOfficerForDepartment(assignedDeptID, req.LocationID)
		if err == nil && officerID != nil {
			complaint.AssignedOfficerID = sql.NullInt64{Int64: *officerID, Valid: true}
		}
	}

	// Create complaint in database
	log.Printf("[complaint] Creating complaint with category=%v, location_id=%d", req.Category, req.LocationID)
	err = s.repo.CreateComplaint(complaint)
	if err != nil {
		return nil, fmt.Errorf("failed to create complaint: %w", err)
	}
	log.Printf("[complaint] Complaint created with ID=%d, number=%s", complaint.ComplaintID, complaint.ComplaintNumber)

	// Create initial status history entry (submission audit: user, actor_id, reason)
	statusHistory := &models.ComplaintStatusHistory{
		ComplaintID:          complaint.ComplaintID,
		OldStatus:            sql.NullString{Valid: false}, // No old status for creation
		NewStatus:            initialStatus,
		ChangedByType:        models.ActorUser,
		ChangedByUserID:      sql.NullInt64{Int64: userID, Valid: true},
		ActorType:            sql.NullString{String: string(models.StatusHistoryActorUser), Valid: true},
		ActorID:              sql.NullInt64{Int64: userID, Valid: true},
		Reason:               sql.NullString{String: "Complaint created", Valid: true},
		Notes:                sql.NullString{String: "Complaint created", Valid: true},
	}
	err = s.repo.CreateStatusHistory(statusHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to create initial status history: %w", err)
	}

	// Create attachments if provided
	for _, url := range req.AttachmentURLs {
		attachment := &models.ComplaintAttachment{
			ComplaintID:      complaint.ComplaintID,
			FileName:         extractFileName(url),
			FilePath:         url,
			IsPublic:         req.PublicConsentGiven,
			UploadedByUserID: sql.NullInt64{Int64: userID, Valid: true},
		}
		err = s.repo.CreateAttachment(attachment)
		if err != nil {
			// Log error but don't fail the entire operation
			// In production, consider using a transaction or retry mechanism
			continue
		}
	}

	// Pilot: send assignment email to shadow inbox only (async, non-blocking)
	// Authority abstraction: department_id only, not officer-based
	// CRITICAL: Email MUST be sent after successful assignment
	if complaint.AssignedDepartmentID.Valid {
		deptID := complaint.AssignedDepartmentID.Int64
		
		// Get real department name from repository
		var deptName string
		if s.departmentRepo != nil {
			name, err := s.departmentRepo.GetDepartmentName(deptID)
			if err != nil {
				log.Printf("[complaint] Warning: Failed to get department name for ID=%d: %v", deptID, err)
				deptName = fmt.Sprintf("Department %d", deptID) // Fallback
			} else {
				deptName = name
			}
		} else {
			deptName = fmt.Sprintf("Department %d", deptID) // Fallback if repo not available
		}
		
		log.Printf("[complaint] Sending assignment email for complaint ID=%d to department ID=%d (%s)", 
			complaint.ComplaintID, deptID, deptName)
		
		if s.emailShadowService != nil {
			s.emailShadowService.SendAssignmentEmailAsync(complaint.ComplaintID, complaint.ComplaintNumber, deptID, deptName)
			log.Printf("[complaint] Assignment email queued for complaint ID=%d", complaint.ComplaintID)
		} else {
			log.Printf("[complaint] ERROR: emailShadowService is nil - email NOT sent for complaint ID=%d", complaint.ComplaintID)
		}
	} else {
		log.Printf("[complaint] ERROR: AssignedDepartmentID is not valid - email NOT sent for complaint ID=%d", complaint.ComplaintID)
	}

	// Emit pilot metrics event: complaint_created
	if s.pilotMetricsService != nil {
		metadata := map[string]interface{}{
			"complaint_number": complaint.ComplaintNumber,
			"status":           string(initialStatus),
			"priority":         string(priority),
			"has_category":     req.Category != nil && *req.Category != "",
			"attachment_count": len(req.AttachmentURLs),
		}
		if complaint.AssignedDepartmentID.Valid {
			metadata["assigned_department_id"] = complaint.AssignedDepartmentID.Int64
		}
		s.pilotMetricsService.EmitComplaintCreated(complaint.ComplaintID, userID, metadata)
	}

	// Create audit log entry
	auditData := map[string]interface{}{
		"complaint_id":     complaint.ComplaintID,
		"complaint_number": complaintNumber,
		"title":            req.Title,
		"status":           string(initialStatus),
	}
	newValuesJSON, _ := json.Marshal(auditData)

	auditLog := &models.AuditLog{
		EntityType:     "complaint",
		EntityID:       complaint.ComplaintID,
		Action:         "create",
		ActionByType:   models.ActorUser,
		ActionByUserID: sql.NullInt64{Int64: userID, Valid: true},
		NewValues:      sql.NullString{String: string(newValuesJSON), Valid: true},
		IPAddress:      sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:      sql.NullString{String: userAgent, Valid: userAgent != ""},
	}
	err = s.repo.CreateAuditLog(auditLog)
	if err != nil {
		// Log error but don't fail the operation
		// Audit logging should be resilient
	}

	// Build response with assigned department ID for admin visibility
	response := &models.CreateComplaintResponse{
		ComplaintID:     complaint.ComplaintID,
		ComplaintNumber: complaintNumber,
		Status:          string(initialStatus),
		Message:         "Complaint created successfully",
	}
	
	// Include assigned_department_id in response for admin visibility
	if complaint.AssignedDepartmentID.Valid {
		deptID := complaint.AssignedDepartmentID.Int64
		response.AssignedDepartmentID = &deptID
		log.Printf("[complaint] Response includes assigned_department_id=%d", deptID)
	}
	
	return response, nil
}

// GetUserComplaints retrieves all complaints for a specific user
func (s *ComplaintService) GetUserComplaints(userID int64) ([]models.ComplaintSummary, error) {
	complaints, err := s.repo.GetComplaintsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user complaints: %w", err)
	}

	// Convert to summary format
	summaries := make([]models.ComplaintSummary, 0, len(complaints))
	for _, complaint := range complaints {
		summary := models.ComplaintSummary{
			ComplaintID:     complaint.ComplaintID,
			ComplaintNumber: complaint.ComplaintNumber,
			Title:           complaint.Title,
			CurrentStatus:   string(complaint.CurrentStatus),
			Priority:        string(complaint.Priority),
			CreatedAt:       complaint.CreatedAt,
			SupporterCount:  complaint.SupporterCount,
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// GetComplaintByID retrieves a complaint with full details (citizen view)
func (s *ComplaintService) GetComplaintByID(complaintID int64, requestingUserID int64) (*models.ComplaintDetailResponse, error) {
	complaint, err := s.repo.GetComplaintByID(complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get complaint: %w", err)
	}

	// Check if user has access (owner or public complaint)
	if complaint.UserID != requestingUserID && !complaint.IsPublic {
		return nil, fmt.Errorf("complaint not found or access denied")
	}

	// Get attachments
	attachments, err := s.repo.GetAttachmentsByComplaintID(complaintID)
	if err != nil {
		// Log error but continue
		attachments = []models.ComplaintAttachment{}
	}

	// Build response
	response := &models.ComplaintDetailResponse{
		ComplaintID:     complaint.ComplaintID,
		ComplaintNumber: complaint.ComplaintNumber,
		Title:           complaint.Title,
		Description:     complaint.Description,
		LocationID:      complaint.LocationID,
		CurrentStatus:   string(complaint.CurrentStatus),
		Priority:        string(complaint.Priority),
		IsPublic:        complaint.IsPublic,
		SupporterCount:  complaint.SupporterCount,
		CreatedAt:       complaint.CreatedAt,
	}

	// Handle nullable fields
	if complaint.Category.Valid {
		response.Category = &complaint.Category.String
	}
	if complaint.Latitude.Valid {
		response.Latitude = &complaint.Latitude.Float64
	}
	if complaint.Longitude.Valid {
		response.Longitude = &complaint.Longitude.Float64
	}
	if complaint.AssignedDepartmentID.Valid {
		response.AssignedDepartmentID = &complaint.AssignedDepartmentID.Int64
	}
	if complaint.AssignedOfficerID.Valid {
		response.AssignedOfficerID = &complaint.AssignedOfficerID.Int64
	}
	if complaint.ResolvedAt.Valid {
		response.ResolvedAt = &complaint.ResolvedAt.Time
	}
	if complaint.ClosedAt.Valid {
		response.ClosedAt = &complaint.ClosedAt.Time
	}

	// Convert attachments
	response.Attachments = make([]models.AttachmentInfo, 0, len(attachments))
	for _, att := range attachments {
		// Only include attachments if complaint is public or user is owner
		if complaint.IsPublic || complaint.UserID == requestingUserID {
			attInfo := models.AttachmentInfo{
				AttachmentID: att.AttachmentID,
				FileName:     att.FileName,
				FilePath:     att.FilePath,
				IsPublic:     att.IsPublic,
			}
			if att.FileType.Valid {
				attInfo.FileType = &att.FileType.String
			}
			if att.FileSize.Valid {
				attInfo.FileSize = &att.FileSize.Int64
			}
			response.Attachments = append(response.Attachments, attInfo)
		}
	}

	return response, nil
}

// GetStatusTimeline retrieves the complete status timeline for a complaint
func (s *ComplaintService) GetStatusTimeline(complaintID int64, requestingUserID int64) (*models.StatusTimelineResponse, error) {
	// Verify complaint exists and user has access
	complaint, err := s.repo.GetComplaintByID(complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get complaint: %w", err)
	}

	// Check access
	if complaint.UserID != requestingUserID && !complaint.IsPublic {
		return nil, fmt.Errorf("complaint not found or access denied")
	}

	// Get status history
	history, err := s.repo.GetStatusHistory(complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status history: %w", err)
	}

	// Build timeline response
	timeline := make([]models.StatusTimelineEntry, 0, len(history))
	for _, h := range history {
		entry := models.StatusTimelineEntry{
			HistoryID:     h.HistoryID,
			NewStatus:     string(h.NewStatus),
			ChangedByType: string(h.ChangedByType),
			CreatedAt:     h.CreatedAt,
		}

		if h.OldStatus.Valid {
			entry.OldStatus = &h.OldStatus.String
		}
		if h.ChangedByUserID.Valid {
			entry.ChangedByUserID = &h.ChangedByUserID.Int64
		}
		if h.ChangedByOfficerID.Valid {
			entry.ChangedByOfficerID = &h.ChangedByOfficerID.Int64
		}
		if h.AssignedDepartmentID.Valid {
			entry.AssignedDepartmentID = &h.AssignedDepartmentID.Int64
		}
		if h.AssignedOfficerID.Valid {
			entry.AssignedOfficerID = &h.AssignedOfficerID.Int64
		}
		if h.Notes.Valid {
			entry.Notes = &h.Notes.String
		}

		timeline = append(timeline, entry)
	}

	return &models.StatusTimelineResponse{
		ComplaintID:     complaintID,
		ComplaintNumber: complaint.ComplaintNumber,
		Timeline:        timeline,
	}, nil
}

// UpdateComplaintStatus updates the status of a complaint (internal use only)
//
// Lifecycle Rules:
// 1. Status transitions must be valid (enforced by business logic)
// 2. Every status change MUST create a status history entry
// 3. Every status change MUST create an audit log entry
// 4. resolved_at is set when status becomes 'resolved'
// 5. closed_at is set when status becomes 'closed'
// 6. Assignment changes are tracked in status history
func (s *ComplaintService) UpdateComplaintStatus(
	complaintID int64,
	req *models.UpdateStatusRequest,
	actorType models.ActorType,
	actorUserID *int64,
	actorOfficerID *int64,
	ipAddress, userAgent string,
) (*models.UpdateStatusResponse, error) {
	// Get current complaint state
	complaint, err := s.repo.GetComplaintByID(complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get complaint: %w", err)
	}

	oldStatus := complaint.CurrentStatus
	newStatus := models.ComplaintStatus(req.NewStatus)

	// Validate status transition
	// Note: In production, implement a state machine with allowed transitions
	if !isValidStatusTransition(oldStatus, newStatus) {
		return nil, fmt.Errorf("invalid status transition from %s to %s", oldStatus, newStatus)
	}

	// Prepare assignment updates
	var assignedDeptID *int64
	var assignedOfficerID *int64

	if req.AssignedDepartmentID != nil {
		assignedDeptID = req.AssignedDepartmentID
	} else if complaint.AssignedDepartmentID.Valid {
		assignedDeptID = &complaint.AssignedDepartmentID.Int64
	}

	if req.AssignedOfficerID != nil {
		assignedOfficerID = req.AssignedOfficerID
	} else if complaint.AssignedOfficerID.Valid {
		assignedOfficerID = &complaint.AssignedOfficerID.Int64
	}

	// Update complaint status
	err = s.repo.UpdateComplaintStatus(complaintID, newStatus, assignedDeptID, assignedOfficerID)
	if err != nil {
		return nil, fmt.Errorf("failed to update complaint status: %w", err)
	}

	// Create status history entry (REQUIRED for every status change; audit: actor_type, actor_id, reason)
	statusHistory := &models.ComplaintStatusHistory{
		ComplaintID:   complaintID,
		NewStatus:     newStatus,
		ChangedByType: actorType,
		Notes:         sql.NullString{Valid: false},
	}

	// Old status is always the current status before update
	statusHistory.OldStatus = sql.NullString{String: string(oldStatus), Valid: true}

	switch actorType {
	case models.ActorUser:
		statusHistory.ActorType = sql.NullString{String: string(models.StatusHistoryActorUser), Valid: true}
		if actorUserID != nil {
			statusHistory.ChangedByUserID = sql.NullInt64{Int64: *actorUserID, Valid: true}
			statusHistory.ActorID = sql.NullInt64{Int64: *actorUserID, Valid: true}
		}
	case models.ActorOfficer, models.ActorAdmin:
		statusHistory.ActorType = sql.NullString{String: string(models.StatusHistoryActorAuthority), Valid: true}
		if actorOfficerID != nil {
			statusHistory.ChangedByOfficerID = sql.NullInt64{Int64: *actorOfficerID, Valid: true}
			statusHistory.ActorID = sql.NullInt64{Int64: *actorOfficerID, Valid: true}
		}
	default:
		statusHistory.ActorType = sql.NullString{String: string(models.StatusHistoryActorSystem), Valid: true}
		if actorUserID != nil {
			statusHistory.ChangedByUserID = sql.NullInt64{Int64: *actorUserID, Valid: true}
		}
		if actorOfficerID != nil {
			statusHistory.ChangedByOfficerID = sql.NullInt64{Int64: *actorOfficerID, Valid: true}
		}
	}
	if assignedDeptID != nil {
		statusHistory.AssignedDepartmentID = sql.NullInt64{Int64: *assignedDeptID, Valid: true}
	}
	if assignedOfficerID != nil {
		statusHistory.AssignedOfficerID = sql.NullInt64{Int64: *assignedOfficerID, Valid: true}
	}
	if req.Notes != nil {
		statusHistory.Notes = sql.NullString{String: *req.Notes, Valid: true}
		statusHistory.Reason = sql.NullString{String: *req.Notes, Valid: true}
	}

	err = s.repo.CreateStatusHistory(statusHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to create status history: %w", err)
	}

	// Create audit log entry (REQUIRED for every action)
	oldValues := map[string]interface{}{
		"status":                string(oldStatus),
		"assigned_department_id": nil,
		"assigned_officer_id":    nil,
	}
	if complaint.AssignedDepartmentID.Valid {
		oldValues["assigned_department_id"] = complaint.AssignedDepartmentID.Int64
	}
	if complaint.AssignedOfficerID.Valid {
		oldValues["assigned_officer_id"] = complaint.AssignedOfficerID.Int64
	}

	newValues := map[string]interface{}{
		"status":                string(newStatus),
		"assigned_department_id": assignedDeptID,
		"assigned_officer_id":    assignedOfficerID,
	}

	changes := map[string]interface{}{
		"status": map[string]interface{}{
			"old": string(oldStatus),
			"new": string(newStatus),
		},
	}

	oldValuesJSON, _ := json.Marshal(oldValues)
	newValuesJSON, _ := json.Marshal(newValues)
	changesJSON, _ := json.Marshal(changes)

	auditLog := &models.AuditLog{
		EntityType:   "complaint",
		EntityID:     complaintID,
		Action:       "status_change",
		ActionByType: actorType,
		OldValues:    sql.NullString{String: string(oldValuesJSON), Valid: true},
		NewValues:    sql.NullString{String: string(newValuesJSON), Valid: true},
		Changes:      sql.NullString{String: string(changesJSON), Valid: true},
		IPAddress:    sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:    sql.NullString{String: userAgent, Valid: userAgent != ""},
	}

	if actorUserID != nil {
		auditLog.ActionByUserID = sql.NullInt64{Int64: *actorUserID, Valid: true}
	}
	if actorOfficerID != nil {
		auditLog.ActionByOfficerID = sql.NullInt64{Int64: *actorOfficerID, Valid: true}
	}

	err = s.repo.CreateAuditLog(auditLog)
	if err != nil {
		// Log error but don't fail the operation
		// Audit logging should be resilient
	}

	return &models.UpdateStatusResponse{
		ComplaintID:     complaintID,
		ComplaintNumber: complaint.ComplaintNumber,
		OldStatus:       string(oldStatus),
		NewStatus:       string(newStatus),
		Message:         "Status updated successfully",
	}, nil
}

// isValidStatusTransition validates if a status transition is allowed
// This is a simplified version - in production, implement a proper state machine
func isValidStatusTransition(oldStatus, newStatus models.ComplaintStatus) bool {
	// Define allowed transitions
	allowedTransitions := map[models.ComplaintStatus][]models.ComplaintStatus{
		models.StatusDraft:        {models.StatusSubmitted, models.StatusDraft},
		models.StatusSubmitted:     {models.StatusVerified, models.StatusUnderReview, models.StatusRejected, models.StatusDraft},
		models.StatusVerified:      {models.StatusUnderReview, models.StatusInProgress, models.StatusRejected},
		models.StatusUnderReview:   {models.StatusInProgress, models.StatusRejected, models.StatusEscalated},
		models.StatusInProgress:    {models.StatusResolved, models.StatusRejected, models.StatusEscalated},
		models.StatusResolved:      {models.StatusClosed},
		models.StatusRejected:      {models.StatusClosed, models.StatusUnderReview}, // Can be reopened
		models.StatusEscalated:     {models.StatusUnderReview, models.StatusInProgress},
		models.StatusClosed:        {}, // Terminal state
	}

	allowed, exists := allowedTransitions[oldStatus]
	if !exists {
		return false
	}

	for _, status := range allowed {
		if status == newStatus {
			return true
		}
	}

	return false
}

// extractFileName extracts filename from URL (simple implementation)
func extractFileName(url string) string {
	// Simple implementation - in production, use proper URL parsing
	lastSlash := -1
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			lastSlash = i
			break
		}
	}
	if lastSlash >= 0 && lastSlash < len(url)-1 {
		return url[lastSlash+1:]
	}
	return "attachment"
}
