package repository

import (
	"database/sql"
	"finalneta/models"
	"fmt"
)

// EvidenceRepository handles database operations for evidence integrity.
// complaint_evidence is write-once: no updates to evidence_hash, latitude, longitude, captured_at.
type EvidenceRepository struct {
	db *sql.DB
}

// NewEvidenceRepository creates a new evidence repository
func NewEvidenceRepository(db *sql.DB) *EvidenceRepository {
	return &EvidenceRepository{db: db}
}

// CreateEvidence creates a new evidence integrity record
func (r *EvidenceRepository) CreateEvidence(evidence *models.ComplaintEvidence) error {
	query := `
		INSERT INTO complaint_evidence (
			attachment_id, complaint_id, evidence_hash, captured_at,
			latitude, longitude, created_at
		) VALUES (?, ?, ?, ?, ?, ?, NOW())
	`

	result, err := r.db.Exec(
		query,
		evidence.AttachmentID,
		evidence.ComplaintID,
		evidence.EvidenceHash,
		evidence.CapturedAt,
		evidence.Latitude,
		evidence.Longitude,
	)
	if err != nil {
		return fmt.Errorf("failed to create evidence record: %w", err)
	}

	evidenceID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get evidence ID: %w", err)
	}

	evidence.EvidenceID = evidenceID
	return nil
}

// GetEvidenceByAttachmentID retrieves evidence record for an attachment
func (r *EvidenceRepository) GetEvidenceByAttachmentID(attachmentID int64) (*models.ComplaintEvidence, error) {
	query := `
		SELECT 
			evidence_id, attachment_id, complaint_id, evidence_hash,
			captured_at, latitude, longitude, created_at
		FROM complaint_evidence
		WHERE attachment_id = ?
		LIMIT 1
	`

	var evidence models.ComplaintEvidence
	err := r.db.QueryRow(query, attachmentID).Scan(
		&evidence.EvidenceID,
		&evidence.AttachmentID,
		&evidence.ComplaintID,
		&evidence.EvidenceHash,
		&evidence.CapturedAt,
		&evidence.Latitude,
		&evidence.Longitude,
		&evidence.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("evidence not found for attachment %d", attachmentID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence: %w", err)
	}

	return &evidence, nil
}

// GetEvidenceByComplaintID retrieves all evidence records for a complaint
func (r *EvidenceRepository) GetEvidenceByComplaintID(complaintID int64) ([]models.ComplaintEvidence, error) {
	query := `
		SELECT 
			evidence_id, attachment_id, complaint_id, evidence_hash,
			captured_at, latitude, longitude, created_at
		FROM complaint_evidence
		WHERE complaint_id = ?
		ORDER BY captured_at ASC
	`

	rows, err := r.db.Query(query, complaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to query evidence: %w", err)
	}
	defer rows.Close()

	var evidenceList []models.ComplaintEvidence
	for rows.Next() {
		var evidence models.ComplaintEvidence
		err := rows.Scan(
			&evidence.EvidenceID,
			&evidence.AttachmentID,
			&evidence.ComplaintID,
			&evidence.EvidenceHash,
			&evidence.CapturedAt,
			&evidence.Latitude,
			&evidence.Longitude,
			&evidence.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan evidence: %w", err)
		}
		evidenceList = append(evidenceList, evidence)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating evidence: %w", err)
	}

	return evidenceList, nil
}
