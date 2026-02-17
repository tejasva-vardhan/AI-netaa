package repository

import (
	"database/sql"
	"finalneta/models"
	"fmt"
)

// VoiceNoteRepository handles complaint_voice_notes table.
type VoiceNoteRepository struct {
	db *sql.DB
}

// NewVoiceNoteRepository creates a new voice note repository.
func NewVoiceNoteRepository(db *sql.DB) *VoiceNoteRepository {
	return &VoiceNoteRepository{db: db}
}

// GetByComplaintID returns the voice note for a complaint, if any.
func (r *VoiceNoteRepository) GetByComplaintID(complaintID int64) (*models.ComplaintVoiceNote, error) {
	query := `SELECT id, complaint_id, file_path, mime_type, duration_seconds, created_at
		FROM complaint_voice_notes WHERE complaint_id = ?`
	var v models.ComplaintVoiceNote
	var dur sql.NullInt64
	err := r.db.QueryRow(query, complaintID).Scan(
		&v.ID, &v.ComplaintID, &v.FilePath, &v.MimeType, &dur, &v.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get voice note: %w", err)
	}
	if dur.Valid {
		d := int(dur.Int64)
		v.DurationSeconds = &d
	}
	return &v, nil
}

// CreateOrUpdate saves or overwrites the single voice note for a complaint (one per complaint).
func (r *VoiceNoteRepository) CreateOrUpdate(v *models.ComplaintVoiceNote) error {
	// Upsert: if row exists for complaint_id, update; else insert.
	existing, err := r.GetByComplaintID(v.ComplaintID)
	if err != nil {
		return err
	}
	if existing != nil {
		_, err = r.db.Exec(
			`UPDATE complaint_voice_notes SET file_path = ?, mime_type = ?, duration_seconds = ? WHERE complaint_id = ?`,
			v.FilePath, v.MimeType, v.DurationSeconds, v.ComplaintID,
		)
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO complaint_voice_notes (complaint_id, file_path, mime_type, duration_seconds) VALUES (?, ?, ?, ?)`,
		v.ComplaintID, v.FilePath, v.MimeType, v.DurationSeconds,
	)
	return err
}
