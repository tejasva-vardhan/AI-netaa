package service

import (
	"database/sql"
	"finalneta/models"
	"finalneta/repository"
	"finalneta/utils"
	"fmt"
	"time"
)

// EvidenceService handles evidence integrity verification logic
type EvidenceService struct {
	evidenceRepo *repository.EvidenceRepository
}

// NewEvidenceService creates a new evidence service
func NewEvidenceService(evidenceRepo *repository.EvidenceRepository) *EvidenceService {
	return &EvidenceService{
		evidenceRepo: evidenceRepo,
	}
}

// CreateEvidenceRecord creates an evidence integrity record for an attachment.
// Hash is generated from RAW image bytes + latitude + longitude + server-generated captured_at.
// capturedAt MUST be server time (e.g. time.Now()) — do not accept client-provided timestamps.
//
// complaint_evidence is WRITE-ONCE: no updates to evidence_hash, latitude, longitude, or
// captured_at after insert. This service does not expose any update method; immutability
// is enforced at the service layer.
func (s *EvidenceService) CreateEvidenceRecord(
	attachmentID int64,
	complaintID int64,
	imageBytes []byte,
	latitude *float64,
	longitude *float64,
	capturedAt time.Time,
) (*models.ComplaintEvidence, error) {
	// Default latitude/longitude if not provided
	lat := 0.0
	lon := 0.0
	if latitude != nil {
		lat = *latitude
	}
	if longitude != nil {
		lon = *longitude
	}

	// Generate evidence hash (server-side, immutable)
	evidenceHash := utils.GenerateEvidenceHash(imageBytes, lat, lon, capturedAt)

	// Create evidence record
	evidence := &models.ComplaintEvidence{
		AttachmentID: attachmentID,
		ComplaintID:   complaintID,
		EvidenceHash:  evidenceHash,
		CapturedAt:    capturedAt, // Server timestamp
	}

	// Set latitude/longitude if provided
	if latitude != nil {
		evidence.Latitude = sql.NullFloat64{Float64: *latitude, Valid: true}
	}
	if longitude != nil {
		evidence.Longitude = sql.NullFloat64{Float64: *longitude, Valid: true}
	}

	// Store in database
	err := s.evidenceRepo.CreateEvidence(evidence)
	if err != nil {
		return nil, fmt.Errorf("failed to create evidence record: %w", err)
	}

	return evidence, nil
}

// GetEvidenceByAttachmentID retrieves evidence record for an attachment
func (s *EvidenceService) GetEvidenceByAttachmentID(attachmentID int64) (*models.ComplaintEvidence, error) {
	return s.evidenceRepo.GetEvidenceByAttachmentID(attachmentID)
}

// GetEvidenceByComplaintID retrieves all evidence records for a complaint
func (s *EvidenceService) GetEvidenceByComplaintID(complaintID int64) ([]models.ComplaintEvidence, error) {
	return s.evidenceRepo.GetEvidenceByComplaintID(complaintID)
}
