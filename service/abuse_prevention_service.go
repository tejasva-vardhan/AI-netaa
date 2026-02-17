package service

import (
	"crypto/sha256"
	"encoding/hex"
	"finalneta/repository"
	"fmt"
	"time"
)

// AbusePreventionService handles abuse prevention checks for complaint submission
type AbusePreventionService struct {
	abuseRepo *repository.AbusePreventionRepository
}

// NewAbusePreventionService creates a new abuse prevention service
func NewAbusePreventionService(abuseRepo *repository.AbusePreventionRepository) *AbusePreventionService {
	return &AbusePreventionService{
		abuseRepo: abuseRepo,
	}
}

// AbuseCheckResult represents the result of abuse prevention checks
type AbuseCheckResult struct {
	Allowed   bool
	Reason    string
	ErrorCode string // For internal tracking (not exposed to user)
}

// CheckRateLimit verifies user hasn't exceeded complaint submission rate limit
// Max 3 complaints per user per 24 hours
func (s *AbusePreventionService) CheckRateLimit(userID int64) (*AbuseCheckResult, error) {
	count, err := s.abuseRepo.CountComplaintsByUserInLast24Hours(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	if count >= 3 {
		return &AbuseCheckResult{
			Allowed:   false,
			Reason:    "You have reached the maximum number of complaints allowed per day. Please try again tomorrow.",
			ErrorCode: "RATE_LIMIT_EXCEEDED",
		}, nil
	}

	return &AbuseCheckResult{
		Allowed: true,
	}, nil
}

// CheckDuplicate verifies complaint is not a duplicate submission
// Same user + same issue_summary + same pincode within 30 minutes → reject
func (s *AbusePreventionService) CheckDuplicate(
	userID int64,
	issueSummary string,
	pincode string,
) (*AbuseCheckResult, error) {
	exists, err := s.abuseRepo.HasDuplicateComplaint(userID, issueSummary, pincode, 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplicate: %w", err)
	}

	if exists {
		return &AbuseCheckResult{
			Allowed:   false,
			Reason:    "A similar complaint was recently submitted. Please wait before submitting again.",
			ErrorCode: "DUPLICATE_SUBMISSION",
		}, nil
	}

	return &AbuseCheckResult{
		Allowed: true,
	}, nil
}

// GenerateDeviceFingerprint generates a hash fingerprint from user_id + user_agent + screen_size
// Returns hex-encoded SHA256 hash
func GenerateDeviceFingerprint(userID int64, userAgent string, screenSize string) string {
	// Combine inputs
	input := fmt.Sprintf("%d|%s|%s", userID, userAgent, screenSize)

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(input))

	// Return hex-encoded string
	return hex.EncodeToString(hash[:])
}

// ValidateComplaintSubmission performs all abuse prevention checks before allowing submission
func (s *AbusePreventionService) ValidateComplaintSubmission(
	userID int64,
	issueSummary string,
	pincode string,
) (*AbuseCheckResult, error) {
	// Check 1: Rate limiting
	rateLimitResult, err := s.CheckRateLimit(userID)
	if err != nil {
		return nil, err
	}
	if !rateLimitResult.Allowed {
		return rateLimitResult, nil
	}

	// Check 2: Duplicate detection
	duplicateResult, err := s.CheckDuplicate(userID, issueSummary, pincode)
	if err != nil {
		return nil, err
	}
	if !duplicateResult.Allowed {
		return duplicateResult, nil
	}

	// All checks passed
	return &AbuseCheckResult{
		Allowed: true,
	}, nil
}
