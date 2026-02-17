package handler

import (
	"encoding/json"
	"fmt"
	"finalneta/models"
	"finalneta/repository"
	"finalneta/service"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// ComplaintHandler handles HTTP requests for complaints
type ComplaintHandler struct {
	service               *service.ComplaintService
	userService           *service.UserService
	abusePreventionService *service.AbusePreventionService
	complaintRepo          *repository.ComplaintRepository
	voiceNoteRepo          *repository.VoiceNoteRepository
	uploadBasePath         string
}

// NewComplaintHandler creates a new complaint handler
func NewComplaintHandler(
	svc *service.ComplaintService,
	userService *service.UserService,
	abusePreventionService *service.AbusePreventionService,
	complaintRepo *repository.ComplaintRepository,
	voiceNoteRepo *repository.VoiceNoteRepository,
) *ComplaintHandler {
	basePath := os.Getenv("UPLOAD_BASE_PATH")
	if basePath == "" {
		basePath = "uploads"
	}
	return &ComplaintHandler{
		service:               svc,
		userService:           userService,
		abusePreventionService: abusePreventionService,
		complaintRepo:          complaintRepo,
		voiceNoteRepo:          voiceNoteRepo,
		uploadBasePath:         basePath,
	}
}

// CreateComplaint handles POST /api/complaints
// Creates a new complaint with proper lifecycle initialization
func (h *ComplaintHandler) CreateComplaint(w http.ResponseWriter, r *http.Request) {
	// Extract user_id from context (assumed to be set by auth middleware)
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "User ID not found in context")
		return
	}

	// User_id is already validated by auth middleware (JWT token validation)
	// Auth middleware ensures:
	// 1. Token is valid
	// 2. User exists
	// 3. Phone is verified
	// So we can proceed directly to complaint creation

	// Parse request body
	var req models.CreateComplaintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Failed to parse request body")
		return
	}

	// Basic validation
	if req.Title == "" {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Title is required")
		return
	}
	if req.Description == "" {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Description is required")
		return
	}
	if req.LocationID == 0 {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Location ID is required")
		return
	}
	// Require GPS coordinates (live proof requirement)
	if req.Latitude == nil || req.Longitude == nil {
		respondWithError(w, http.StatusBadRequest, "Validation error", "GPS coordinates (latitude and longitude) are required for live proof")
		return
	}
	// Require at least one attachment (live photo proof requirement)
	if len(req.AttachmentURLs) == 0 {
		respondWithError(w, http.StatusBadRequest, "Validation error", "At least one photo attachment is required for live proof")
		return
	}

	// Extract IP and User-Agent for audit logging
	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()
	
	// Extract screen size from header (frontend sends X-Screen-Size)
	screenSize := r.Header.Get("X-Screen-Size")
	if screenSize == "" {
		screenSize = "unknown" // Default if not provided
	}

	// ABUSE PREVENTION: Validate submission before creating complaint
	if h.abusePreventionService != nil {
		// Get pincode from request (required for duplicate detection)
		pincode := ""
		if req.Pincode != nil {
			pincode = *req.Pincode
		}
		
		// Perform abuse prevention checks
		abuseCheck, err := h.abusePreventionService.ValidateComplaintSubmission(
			userID,
			req.Title, // issue_summary (title)
			pincode,
		)
		if err != nil {
			log.Printf("[complaint] ValidateComplaintSubmission error: %v", err)
			respondWithError(w, http.StatusInternalServerError, "Internal error", "Failed to validate submission")
			return
		}
		
		if !abuseCheck.Allowed {
			// Return generic error message (no accusation)
			respondWithError(w, http.StatusTooManyRequests, "Submission limit", abuseCheck.Reason)
			return
		}
		
		// Generate device fingerprint
		deviceFingerprint := service.GenerateDeviceFingerprint(userID, userAgent, screenSize)
		req.DeviceFingerprint = &deviceFingerprint
	}

	// Create complaint
	response, err := h.service.CreateComplaint(&req, userID, ipAddress, userAgent)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, response)
}

// GetComplaintByID handles GET /api/complaints/{id}
// Retrieves complaint details (citizen view)
func (h *ComplaintHandler) GetComplaintByID(w http.ResponseWriter, r *http.Request) {
	// Extract user_id from context
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "User ID not found in context")
		return
	}

	// Extract complaint ID from URL
	vars := mux.Vars(r)
	complaintIDStr := vars["id"]
	complaintID, err := strconv.ParseInt(complaintIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Invalid complaint ID")
		return
	}

	// Get complaint
	response, err := h.service.GetComplaintByID(complaintID, userID)
	if err != nil {
		if err.Error() == "complaint not found" || err.Error() == "complaint not found or access denied" {
			respondWithError(w, http.StatusNotFound, "Not found", err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// GetStatusTimeline handles GET /api/complaints/{id}/timeline
// Retrieves the complete status timeline for a complaint
func (h *ComplaintHandler) GetStatusTimeline(w http.ResponseWriter, r *http.Request) {
	// Extract user_id from context
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "User ID not found in context")
		return
	}

	// Extract complaint ID from URL
	vars := mux.Vars(r)
	complaintIDStr := vars["id"]
	complaintID, err := strconv.ParseInt(complaintIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Invalid complaint ID")
		return
	}

	// Get timeline
	response, err := h.service.GetStatusTimeline(complaintID, userID)
	if err != nil {
		if err.Error() == "complaint not found" || err.Error() == "complaint not found or access denied" {
			respondWithError(w, http.StatusNotFound, "Not found", err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// UploadVoice handles POST /api/v1/complaints/{id}/voice
// Citizen JWT only; only complaint owner can upload. One voice per complaint (overwrite allowed). Voice is not public.
func (h *ComplaintHandler) UploadVoice(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "User ID not found in context")
		return
	}
	vars := mux.Vars(r)
	complaintIDStr := vars["id"]
	complaintID, err := strconv.ParseInt(complaintIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Invalid complaint ID")
		return
	}
	ownerID, err := h.complaintRepo.GetComplaintOwnerID(complaintID)
	if err != nil {
		if err.Error() == "complaint not found" {
			respondWithError(w, http.StatusNotFound, "Not found", "Complaint not found")
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}
	if ownerID != userID {
		respondWithError(w, http.StatusForbidden, "Forbidden", "Only the complaint owner can upload a voice note")
		return
	}
	contentType := r.Header.Get("Content-Type")
	var ext string
	switch {
	case strings.Contains(contentType, "audio/webm"):
		ext = "webm"
	case strings.Contains(contentType, "audio/wav"):
		ext = "wav"
	default:
		ext = "webm"
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Empty voice data")
		return
	}
	voiceDir := filepath.Join(h.uploadBasePath, "voices")
	if err := os.MkdirAll(voiceDir, 0755); err != nil {
		log.Printf("[voice] failed to create directory %s: %v", voiceDir, err)
		respondWithError(w, http.StatusInternalServerError, "Internal error", "Failed to save voice note")
		return
	}
	filename := fmt.Sprintf("%d.%s", complaintID, ext)
	filePath := filepath.Join(voiceDir, filename)
	if err := os.WriteFile(filePath, body, 0644); err != nil {
		log.Printf("[voice] failed to write file %s: %v", filePath, err)
		respondWithError(w, http.StatusInternalServerError, "Internal error", "Failed to save voice note")
		return
	}
	relativePath := filepath.Join("voices", filename)
	mimeType := "audio/webm"
	if ext == "wav" {
		mimeType = "audio/wav"
	}
	note := &models.ComplaintVoiceNote{
		ComplaintID: complaintID,
		FilePath:    relativePath,
		MimeType:    mimeType,
	}
	if err := h.voiceNoteRepo.CreateOrUpdate(note); err != nil {
		log.Printf("[voice] failed to save voice note record: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Internal error", "Failed to save voice note")
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":      "Voice note attached",
		"complaint_id": complaintID,
	})
}

// GetUserComplaints handles GET /api/v1/complaints
// Retrieves all complaints for the authenticated user
func (h *ComplaintHandler) GetUserComplaints(w http.ResponseWriter, r *http.Request) {
	// Extract user_id from context (set by auth middleware)
	userID, err := getUserIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "User ID not found in context")
		return
	}

	// Get user's complaints
	complaints, err := h.service.GetUserComplaints(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, complaints)
}

// UpdateComplaintStatus handles PATCH /api/complaints/{id}/status
// Updates complaint status (internal use only - requires officer/admin authentication)
func (h *ComplaintHandler) UpdateComplaintStatus(w http.ResponseWriter, r *http.Request) {
	// Extract actor information from context
	// In production, this would come from authentication middleware
	actorType, actorUserID, actorOfficerID, err := getActorFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Actor information not found in context")
		return
	}

	// Extract complaint ID from URL
	vars := mux.Vars(r)
	complaintIDStr := vars["id"]
	complaintID, err := strconv.ParseInt(complaintIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Invalid complaint ID")
		return
	}

	// Parse request body
	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Failed to parse request body")
		return
	}

	// Basic validation
	if req.NewStatus == "" {
		respondWithError(w, http.StatusBadRequest, "Validation error", "New status is required")
		return
	}

	// Extract IP and User-Agent for audit logging
	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()

	// Update status
	response, err := h.service.UpdateComplaintStatus(
		complaintID,
		&req,
		actorType,
		actorUserID,
		actorOfficerID,
		ipAddress,
		userAgent,
	)
	if err != nil {
		if err.Error() == "complaint not found" {
			respondWithError(w, http.StatusNotFound, "Not found", err.Error())
			return
		}
		if err.Error() == "invalid status transition" {
			respondWithError(w, http.StatusBadRequest, "Validation error", err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, response)
}

// Helper functions

// getUserIDFromContext extracts user_id from request context
// Set by authentication middleware after JWT validation
func getUserIDFromContext(r *http.Request) (int64, error) {
	// Get user_id from context (set by auth middleware)
	userIDVal := r.Context().Value("user_id")
	if userIDVal == nil {
		return 0, fmt.Errorf("user ID not found in context - authentication required")
	}

	userID, ok := userIDVal.(int64)
	if !ok {
		return 0, fmt.Errorf("invalid user ID type in context")
	}

	return userID, nil
}

// getActorFromContext extracts actor information from request context
// In production, this would be set by authentication middleware
func getActorFromContext(r *http.Request) (models.ActorType, *int64, *int64, error) {
	actorTypeStr := r.Header.Get("X-Actor-Type")
	if actorTypeStr == "" {
		actorTypeStr = "officer" // Default for internal endpoints
	}

	actorType := models.ActorType(actorTypeStr)
	var actorUserID *int64
	var actorOfficerID *int64

	if actorType == models.ActorUser {
		userIDHeader := r.Header.Get("X-User-ID")
		if userIDHeader != "" {
			userID, err := strconv.ParseInt(userIDHeader, 10, 64)
			if err == nil {
				actorUserID = &userID
			}
		}
	} else if actorType == models.ActorOfficer {
		officerIDHeader := r.Header.Get("X-Officer-ID")
		if officerIDHeader != "" {
			officerID, err := strconv.ParseInt(officerIDHeader, 10, 64)
			if err == nil {
				actorOfficerID = &officerID
			}
		}
	}

	return actorType, actorUserID, actorOfficerID, nil
}

// getClientIP extracts client IP address from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}
	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	// Fallback to RemoteAddr
	return r.RemoteAddr
}

// respondWithJSON sends a JSON response
func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// respondWithError sends an error response
func respondWithError(w http.ResponseWriter, statusCode int, errorType, message string) {
	response := models.ErrorResponse{
		Error:   errorType,
		Message: message,
		Code:    statusCode,
	}
	respondWithJSON(w, statusCode, response)
}
