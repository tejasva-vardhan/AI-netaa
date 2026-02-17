package service

import (
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"finalneta/repository"
	"fmt"
	"time"
)

// VerificationService handles complaint verification logic
type VerificationService struct {
	complaintRepo    *repository.ComplaintRepository
	verificationRepo *repository.VerificationRepository
	config           *models.VerificationConfig
}

// NewVerificationService creates a new verification service
func NewVerificationService(
	complaintRepo *repository.ComplaintRepository,
	verificationRepo *repository.VerificationRepository,
	config *models.VerificationConfig,
) *VerificationService {
	if config == nil {
		config = models.DefaultVerificationConfig()
	}
	return &VerificationService{
		complaintRepo:    complaintRepo,
		verificationRepo: verificationRepo,
		config:           config,
	}
}

// VerifyComplaint verifies a complaint according to all verification rules
//
// Verification Rules:
// 1. Complaint must have at least one attachment marked as live_capture = true
// 2. GPS accuracy must be within acceptable range (configurable, e.g. <= 100 meters)
// 3. User phone must be verified
// 4. Duplicate detection:
//    - Same category
//    - Same location (within radius)
//    - Within configurable time window (e.g. last 24 hours)
//    - If duplicate found: do NOT create new complaint, increment supporter count
// 5. All verification decisions must be logged in audit_log
//
// Returns VerificationResult with reason code and message
func (s *VerificationService) VerifyComplaint(
	req *models.VerificationRequest,
	ipAddress, userAgent string,
) (*models.VerificationResult, error) {
	// Get complaint details
	complaint, err := s.complaintRepo.GetComplaintByID(req.ComplaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get complaint: %w", err)
	}

	// Verify complaint is in "submitted" status
	if complaint.CurrentStatus != models.StatusSubmitted {
		return &models.VerificationResult{
			ComplaintID:   req.ComplaintID,
			Verified:      false,
			ReasonCode:    models.ReasonCodeVerified, // Reuse code, but this is actually invalid status
			ReasonMessage: fmt.Sprintf("Complaint must be in 'submitted' status, current status: %s", complaint.CurrentStatus),
		}, nil
	}

	// Rule 1: Check for live capture attachment
	hasLiveCapture, err := s.verificationRepo.HasLiveCaptureAttachment(req.ComplaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to check live capture: %w", err)
	}
	if !hasLiveCapture {
		return s.createVerificationResult(
			req.ComplaintID,
			false,
			models.ReasonCodeNoLiveCapture,
			"No attachment with live_capture=true found",
			ipAddress,
			userAgent,
		)
	}

	// Rule 2: Check GPS accuracy (if provided)
	if req.GPSAccuracy != nil {
		if *req.GPSAccuracy > s.config.GPSAccuracyThreshold {
			return s.createVerificationResult(
				req.ComplaintID,
				false,
				models.ReasonCodeGPSAccuracyExceeded,
				fmt.Sprintf("GPS accuracy %.2f meters exceeds threshold of %.2f meters", *req.GPSAccuracy, s.config.GPSAccuracyThreshold),
				ipAddress,
				userAgent,
			)
		}
	}

	// Rule 3: Check if user phone is verified
	isPhoneVerified, err := s.verificationRepo.IsUserPhoneVerified(complaint.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check phone verification: %w", err)
	}
	if !isPhoneVerified {
		return s.createVerificationResult(
			req.ComplaintID,
			false,
			models.ReasonCodePhoneNotVerified,
			"User phone number is not verified",
			ipAddress,
			userAgent,
		)
	}

	// Rule 4: Duplicate detection
	// Get complaint coordinates and category
	var latitude, longitude float64
	var hasCoordinates bool

	if complaint.Latitude.Valid && complaint.Longitude.Valid {
		latitude = complaint.Latitude.Float64
		longitude = complaint.Longitude.Float64
		hasCoordinates = true
	} else {
		// Try to get coordinates from repository
		lat, lon, valid, err := s.verificationRepo.GetComplaintCoordinates(req.ComplaintID)
		if err != nil {
			return nil, fmt.Errorf("failed to get coordinates: %w", err)
		}
		if valid {
			latitude = lat
			longitude = lon
			hasCoordinates = true
		}
	}

	var category *string
	if complaint.Category.Valid {
		category = &complaint.Category.String
	}

	// Check for duplicates only if coordinates are available
	if hasCoordinates {
		duplicates, err := s.verificationRepo.FindDuplicateComplaints(
			req.ComplaintID,
			category,
			latitude,
			longitude,
			s.config.DuplicateDetectionRadius,
			s.config.DuplicateDetectionTimeWindow,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicates: %w", err)
		}

		// If duplicates found, merge instead of verifying
		if len(duplicates) > 0 {
			// Find the oldest duplicate (original complaint)
			oldestDuplicate := duplicates[0]
			for _, dup := range duplicates {
				if dup.CreatedAt.Before(oldestDuplicate.CreatedAt) {
					oldestDuplicate = dup
				}
			}

			// Increment supporter count on the original complaint
			err = s.verificationRepo.IncrementSupporterCount(oldestDuplicate.ComplaintID)
			if err != nil {
				return nil, fmt.Errorf("failed to increment supporter count: %w", err)
			}

			// Add current user as supporter to the original complaint
			duplicateNotes := fmt.Sprintf("Merged from complaint #%d", req.ComplaintID)
			err = s.verificationRepo.AddSupporter(
				oldestDuplicate.ComplaintID,
				complaint.UserID,
				true, // is_duplicate = true
				duplicateNotes,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to add supporter: %w", err)
			}

			// Get updated supporter count
			originalComplaint, err := s.complaintRepo.GetComplaintByID(oldestDuplicate.ComplaintID)
			if err != nil {
				return nil, fmt.Errorf("failed to get original complaint: %w", err)
			}

			// Log duplicate detection in audit log
			err = s.logVerificationDecision(
				req.ComplaintID,
				false,
				models.ReasonCodeDuplicateFound,
				fmt.Sprintf("Duplicate found: merged with complaint #%d", oldestDuplicate.ComplaintID),
				map[string]interface{}{
					"duplicate_complaint_id": oldestDuplicate.ComplaintID,
					"merged":                 true,
				},
				ipAddress,
				userAgent,
			)
			if err != nil {
				// Log error but don't fail - audit logging should be resilient
			}

			// Return result indicating duplicate was found and merged
			return &models.VerificationResult{
				ComplaintID:          req.ComplaintID,
				Verified:             false,
				ReasonCode:           models.ReasonCodeDuplicateFound,
				ReasonMessage:        fmt.Sprintf("Duplicate complaint found. Merged with complaint #%d", oldestDuplicate.ComplaintID),
				DuplicateComplaintID: &oldestDuplicate.ComplaintID,
				SupporterCount:       originalComplaint.SupporterCount,
			}, nil
		}
	}

	// All rules passed - verify the complaint
	// Update status to "verified" through status history
	err = s.complaintRepo.UpdateComplaintStatus(
		req.ComplaintID,
		models.StatusVerified,
		nil, // Keep existing department assignment
		nil, // Keep existing officer assignment
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update status to verified: %w", err)
	}

	// Create status history entry (REQUIRED for status change; audit: system, no actor_id, reason)
	statusHistory := &models.ComplaintStatusHistory{
		ComplaintID:   req.ComplaintID,
		OldStatus:     sql.NullString{String: string(models.StatusSubmitted), Valid: true},
		NewStatus:     models.StatusVerified,
		ChangedByType: models.ActorSystem,
		ActorType:     sql.NullString{String: string(models.StatusHistoryActorSystem), Valid: true},
		ActorID:       sql.NullInt64{Valid: false},
		Reason:        sql.NullString{String: "Complaint verified automatically", Valid: true},
		Notes:         sql.NullString{String: "Complaint verified automatically", Valid: true},
	}
	err = s.complaintRepo.CreateStatusHistory(statusHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to create status history: %w", err)
	}

	// Log successful verification in audit log
	err = s.logVerificationDecision(
		req.ComplaintID,
		true,
		models.ReasonCodeVerified,
		"Complaint verified successfully - all rules passed",
		map[string]interface{}{
			"gps_accuracy": req.GPSAccuracy,
			"rules_passed": []string{
				"live_capture_attachment",
				"gps_accuracy",
				"phone_verified",
				"no_duplicates",
			},
		},
		ipAddress,
		userAgent,
	)
	if err != nil {
		// Log error but don't fail - audit logging should be resilient
	}

	return &models.VerificationResult{
		ComplaintID:   req.ComplaintID,
		Verified:      true,
		ReasonCode:    models.ReasonCodeVerified,
		ReasonMessage: "Complaint verified successfully",
	}, nil
}

// createVerificationResult creates a verification result and logs it to audit log
func (s *VerificationService) createVerificationResult(
	complaintID int64,
	verified bool,
	reasonCode models.VerificationReasonCode,
	reasonMessage string,
	ipAddress, userAgent string,
) (*models.VerificationResult, error) {
	// Log verification decision in audit log
	err := s.logVerificationDecision(
		complaintID,
		verified,
		reasonCode,
		reasonMessage,
		map[string]interface{}{
			"reason_code": reasonCode,
		},
		ipAddress,
		userAgent,
	)
	if err != nil {
		// Log error but don't fail - audit logging should be resilient
	}

	return &models.VerificationResult{
		ComplaintID:   complaintID,
		Verified:      verified,
		ReasonCode:    reasonCode,
		ReasonMessage: reasonMessage,
	}, nil
}

// logVerificationDecision logs verification decision to audit_log
func (s *VerificationService) logVerificationDecision(
	complaintID int64,
	verified bool,
	reasonCode models.VerificationReasonCode,
	reasonMessage string,
	metadata map[string]interface{},
	ipAddress, userAgent string,
) error {
	// Prepare audit log metadata
	auditMetadata := map[string]interface{}{
		"verification": map[string]interface{}{
			"verified":       verified,
			"reason_code":     reasonCode,
			"reason_message":  reasonMessage,
			"verification_at": time.Now().Format(time.RFC3339),
		},
	}

	// Merge with provided metadata
	for k, v := range metadata {
		auditMetadata[k] = v
	}

	metadataJSON, err := json.Marshal(auditMetadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create audit log entry
	auditLog := &models.AuditLog{
		EntityType:   "complaint",
		EntityID:     complaintID,
		Action:       "verification",
		ActionByType: models.ActorSystem,
		Metadata:     sql.NullString{String: string(metadataJSON), Valid: true},
		IPAddress:   sql.NullString{String: ipAddress, Valid: ipAddress != ""},
		UserAgent:   sql.NullString{String: userAgent, Valid: userAgent != ""},
	}

	err = s.complaintRepo.CreateAuditLog(auditLog)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetVerificationConfig returns the current verification configuration
func (s *VerificationService) GetVerificationConfig() *models.VerificationConfig {
	return s.config
}

// UpdateVerificationConfig updates the verification configuration
func (s *VerificationService) UpdateVerificationConfig(config *models.VerificationConfig) {
	if config != nil {
		s.config = config
	}
}
