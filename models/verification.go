package models

import "time"

// VerificationReasonCode represents the reason for verification failure
type VerificationReasonCode string

const (
	// ReasonCodeNoLiveCapture - No attachment marked as live_capture = true
	ReasonCodeNoLiveCapture VerificationReasonCode = "NO_LIVE_CAPTURE"
	
	// ReasonCodeGPSAccuracyExceeded - GPS accuracy exceeds acceptable range
	ReasonCodeGPSAccuracyExceeded VerificationReasonCode = "GPS_ACCURACY_EXCEEDED"
	
	// ReasonCodePhoneNotVerified - User phone number is not verified
	ReasonCodePhoneNotVerified VerificationReasonCode = "PHONE_NOT_VERIFIED"
	
	// ReasonCodeDuplicateFound - Duplicate complaint found (merged)
	ReasonCodeDuplicateFound VerificationReasonCode = "DUPLICATE_FOUND"
	
	// ReasonCodeVerified - Verification successful
	ReasonCodeVerified VerificationReasonCode = "VERIFIED"
)

// VerificationResult represents the result of verification
type VerificationResult struct {
	ComplaintID      int64                  `json:"complaint_id"`
	Verified         bool                   `json:"verified"`
	ReasonCode       VerificationReasonCode `json:"reason_code"`
	ReasonMessage    string                 `json:"reason_message"`
	DuplicateComplaintID *int64             `json:"duplicate_complaint_id,omitempty"` // If duplicate found
	SupporterCount   int                    `json:"supporter_count,omitempty"`         // Updated supporter count if merged
}

// VerificationRequest represents the request to verify a complaint
type VerificationRequest struct {
	ComplaintID      int64   `json:"complaint_id"`
	GPSAccuracy      *float64 `json:"gps_accuracy,omitempty"` // GPS accuracy in meters
}

// VerificationConfig holds configuration for verification rules
type VerificationConfig struct {
	// GPSAccuracyThreshold - Maximum acceptable GPS accuracy in meters
	GPSAccuracyThreshold float64
	
	// DuplicateDetectionRadius - Radius in meters for duplicate detection
	DuplicateDetectionRadius float64
	
	// DuplicateDetectionTimeWindow - Time window in hours for duplicate detection
	DuplicateDetectionTimeWindow time.Duration
}

// DefaultVerificationConfig returns default verification configuration
func DefaultVerificationConfig() *VerificationConfig {
	return &VerificationConfig{
		GPSAccuracyThreshold:         100.0, // 100 meters
		DuplicateDetectionRadius:     50.0,  // 50 meters
		DuplicateDetectionTimeWindow: 24 * time.Hour, // 24 hours
	}
}

// User represents user information needed for verification
type User struct {
	UserID          int64       `db:"user_id"`
	PhoneVerifiedAt *time.Time  `db:"phone_verified_at"`
}

// DuplicateComplaint represents a potential duplicate complaint
type DuplicateComplaint struct {
	ComplaintID int64     `db:"complaint_id"`
	Category    string    `db:"category"`
	Latitude    float64   `db:"latitude"`
	Longitude   float64   `db:"longitude"`
	CreatedAt   time.Time `db:"created_at"`
	UserID      int64     `db:"user_id"`
}
