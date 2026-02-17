package repository

import (
	"database/sql"
	"finalneta/models"
	"fmt"
	"math"
	"time"
)

// VerificationRepository handles database operations for verification
type VerificationRepository struct {
	db *sql.DB
}

// NewVerificationRepository creates a new verification repository
func NewVerificationRepository(db *sql.DB) *VerificationRepository {
	return &VerificationRepository{db: db}
}

// IsUserPhoneVerified checks if a user's phone is verified
func (r *VerificationRepository) IsUserPhoneVerified(userID int64) (bool, error) {
	query := `
		SELECT phone_verified_at
		FROM users
		WHERE user_id = ? AND phone_verified_at IS NOT NULL
	`

	var verifiedAt sql.NullTime
	err := r.db.QueryRow(query, userID).Scan(&verifiedAt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check phone verification: %w", err)
	}

	return verifiedAt.Valid, nil
}

// HasLiveCaptureAttachment checks if complaint has at least one attachment with live_capture = true
//
// IMPORTANT: Since live_capture field is not in the database schema (per requirements),
// this implementation checks if at least one attachment exists.
//
// In production, you should either:
// 1. Add a `live_capture` BOOLEAN field to `complaint_attachments` table, OR
// 2. Store it in a metadata JSON field and query: WHERE JSON_EXTRACT(metadata, '$.live_capture') = true
//
// Current implementation: Assumes at least one attachment means live capture exists.
// This is a temporary workaround - update when schema allows.
func (r *VerificationRepository) HasLiveCaptureAttachment(complaintID int64) (bool, error) {
	// TODO: Update query when live_capture field is added to schema:
	// SELECT COUNT(*) FROM complaint_attachments 
	// WHERE complaint_id = ? AND live_capture = true
	
	query := `
		SELECT COUNT(*) 
		FROM complaint_attachments 
		WHERE complaint_id = ?
	`

	var count int
	err := r.db.QueryRow(query, complaintID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check live capture attachment: %w", err)
	}

	// For now, assume at least one attachment means live capture exists
	// In production, add: AND live_capture = true
	return count > 0, nil
}

// FindDuplicateComplaints finds duplicate complaints based on:
// - Same category (if category is not null)
// - Same location (within radius)
// - Within time window
func (r *VerificationRepository) FindDuplicateComplaints(
	complaintID int64,
	category *string,
	latitude, longitude float64,
	radiusMeters float64,
	timeWindow time.Duration,
) ([]models.DuplicateComplaint, error) {
	// Calculate time window start
				timeWindowStart := time.Now().UTC().Add(-timeWindow)

	// Build query based on whether category is provided
	var query string
	var args []interface{}

	if category != nil && *category != "" {
		// Query with category match
		query = `
			SELECT 
				complaint_id, category, latitude, longitude, created_at, user_id
			FROM complaints
			WHERE complaint_id != ?
				AND category = ?
				AND latitude IS NOT NULL
				AND longitude IS NOT NULL
				AND created_at >= ?
				AND current_status NOT IN ('rejected', 'closed')
		`
		args = []interface{}{complaintID, *category, timeWindowStart}
	} else {
		// Query without category (category is null or empty)
		query = `
			SELECT 
				complaint_id, category, latitude, longitude, created_at, user_id
			FROM complaints
			WHERE complaint_id != ?
				AND (category IS NULL OR category = '')
				AND latitude IS NOT NULL
				AND longitude IS NOT NULL
				AND created_at >= ?
				AND current_status NOT IN ('rejected', 'closed')
		`
		args = []interface{}{complaintID, timeWindowStart}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query duplicate complaints: %w", err)
	}
	defer rows.Close()

	var duplicates []models.DuplicateComplaint
	for rows.Next() {
		var dup models.DuplicateComplaint
		var lat, lon sql.NullFloat64
		var cat sql.NullString

		err := rows.Scan(&dup.ComplaintID, &cat, &lat, &lon, &dup.CreatedAt, &dup.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan duplicate complaint: %w", err)
		}

		if !lat.Valid || !lon.Valid {
			continue // Skip if coordinates are missing
		}

		dup.Latitude = lat.Float64
		dup.Longitude = lon.Float64
		if cat.Valid {
			dup.Category = cat.String
		}

		// Calculate distance using Haversine formula
		distance := calculateDistance(latitude, longitude, dup.Latitude, dup.Longitude)
		
		// Only include if within radius
		if distance <= radiusMeters {
			duplicates = append(duplicates, dup)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating duplicate complaints: %w", err)
	}

	return duplicates, nil
}

// IncrementSupporterCount increments the supporter count for a complaint
func (r *VerificationRepository) IncrementSupporterCount(complaintID int64) error {
	query := `
		UPDATE complaints
		SET supporter_count = supporter_count + 1
		WHERE complaint_id = ?
	`

	_, err := r.db.Exec(query, complaintID)
	if err != nil {
		return fmt.Errorf("failed to increment supporter count: %w", err)
	}

	return nil
}

// AddSupporter adds a user as a supporter for a complaint
func (r *VerificationRepository) AddSupporter(complaintID, userID int64, isDuplicate bool, duplicateNotes string) error {
	query := `
		INSERT INTO complaint_supporters (
			complaint_id, user_id, is_duplicate, duplicate_notes
		) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			is_duplicate = VALUES(is_duplicate),
			duplicate_notes = VALUES(duplicate_notes)
	`

	_, err := r.db.Exec(query, complaintID, userID, isDuplicate, duplicateNotes)
	if err != nil {
		return fmt.Errorf("failed to add supporter: %w", err)
	}

	return nil
}

// GetComplaintCoordinates retrieves latitude and longitude for a complaint
func (r *VerificationRepository) GetComplaintCoordinates(complaintID int64) (float64, float64, bool, error) {
	query := `
		SELECT latitude, longitude
		FROM complaints
		WHERE complaint_id = ?
	`

	var lat, lon sql.NullFloat64
	err := r.db.QueryRow(query, complaintID).Scan(&lat, &lon)
	if err == sql.ErrNoRows {
		return 0, 0, false, fmt.Errorf("complaint not found")
	}
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to get coordinates: %w", err)
	}

	if !lat.Valid || !lon.Valid {
		return 0, 0, false, nil
	}

	return lat.Float64, lon.Float64, true, nil
}

// GetComplaintCategory retrieves category for a complaint
func (r *VerificationRepository) GetComplaintCategory(complaintID int64) (*string, error) {
	query := `
		SELECT category
		FROM complaints
		WHERE complaint_id = ?
	`

	var category sql.NullString
	err := r.db.QueryRow(query, complaintID).Scan(&category)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("complaint not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	if !category.Valid {
		return nil, nil
	}

	cat := category.String
	return &cat, nil
}

// calculateDistance calculates the distance between two coordinates using Haversine formula
// Returns distance in meters
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000 // Earth radius in meters

	// Convert to radians
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLon := (lon2 - lon1) * math.Pi / 180

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadiusMeters * c
	return distance
}
