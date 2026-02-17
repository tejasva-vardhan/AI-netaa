package repository

import (
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"fmt"
)

// PilotMetricsRepository handles database operations for pilot metrics events
type PilotMetricsRepository struct {
	db *sql.DB
}

// NewPilotMetricsRepository creates a new pilot metrics repository
func NewPilotMetricsRepository(db *sql.DB) *PilotMetricsRepository {
	return &PilotMetricsRepository{db: db}
}

// CreateEvent creates a new pilot metrics event
func (r *PilotMetricsRepository) CreateEvent(event *models.PilotMetricsEvent) error {
	// Serialize metadata to JSON if provided
	var metadataJSON sql.NullString
	if event.Metadata.Valid && event.Metadata.String != "" {
		metadataJSON = event.Metadata
	}

	query := `
		INSERT INTO pilot_metrics_events (
			event_type, complaint_id, user_id, metadata, created_at
		) VALUES (?, ?, ?, ?, NOW())
	`

	result, err := r.db.Exec(
		query,
		string(event.EventType),
		event.ComplaintID,
		event.UserID,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create pilot metrics event: %w", err)
	}

	eventID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get event ID: %w", err)
	}

	event.ID = eventID
	return nil
}

// CreateEventWithMetadata creates a new pilot metrics event with metadata object
func (r *PilotMetricsRepository) CreateEventWithMetadata(
	eventType models.PilotMetricsEventType,
	complaintID *int64,
	userID *int64,
	metadata map[string]interface{},
) error {
	var metadataJSON sql.NullString
	if len(metadata) > 0 {
		jsonBytes, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = sql.NullString{String: string(jsonBytes), Valid: true}
	}

	var complaintIDNull sql.NullInt64
	if complaintID != nil {
		complaintIDNull = sql.NullInt64{Int64: *complaintID, Valid: true}
	}

	var userIDNull sql.NullInt64
	if userID != nil {
		userIDNull = sql.NullInt64{Int64: *userID, Valid: true}
	}

	event := &models.PilotMetricsEvent{
		EventType:   eventType,
		ComplaintID: complaintIDNull,
		UserID:      userIDNull,
		Metadata:    metadataJSON,
	}

	return r.CreateEvent(event)
}
