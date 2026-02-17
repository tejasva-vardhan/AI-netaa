package handler

import (
	"encoding/json"
	"finalneta/models"
	"finalneta/service"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// VerificationHandler handles HTTP requests for complaint verification
type VerificationHandler struct {
	service *service.VerificationService
}

// NewVerificationHandler creates a new verification handler
func NewVerificationHandler(svc *service.VerificationService) *VerificationHandler {
	return &VerificationHandler{service: svc}
}

// VerifyComplaint handles POST /api/v1/complaints/{id}/verify
// Verifies a complaint according to all verification rules
func (h *VerificationHandler) VerifyComplaint(w http.ResponseWriter, r *http.Request) {
	// Extract complaint ID from URL
	vars := mux.Vars(r)
	complaintIDStr := vars["id"]
	complaintID, err := strconv.ParseInt(complaintIDStr, 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Invalid complaint ID")
		return
	}

	// Parse request body (optional - GPS accuracy can be provided)
	var req models.VerificationRequest
	req.ComplaintID = complaintID
	
	// Try to decode body if present
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// If body parsing fails, continue with default request (GPS accuracy optional)
			req.ComplaintID = complaintID
		}
	}

	// Extract IP and User-Agent for audit logging
	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()

	// Verify complaint
	result, err := h.service.VerifyComplaint(&req, ipAddress, userAgent)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}

	// Return appropriate status code based on verification result
	if result.Verified {
		respondWithJSON(w, http.StatusOK, result)
	} else {
		// Verification failed - return 200 with result indicating failure reason
		// This allows client to see why verification failed without treating it as an error
		respondWithJSON(w, http.StatusOK, result)
	}
}
