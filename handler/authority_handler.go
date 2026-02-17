package handler

import (
	"encoding/json"
	"finalneta/models"
	"finalneta/service"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

// AuthorityHandler handles HTTP requests for authority dashboard
type AuthorityHandler struct {
	authorityService *service.AuthorityService
}

// NewAuthorityHandler creates a new authority handler
func NewAuthorityHandler(authorityService *service.AuthorityService) *AuthorityHandler {
	return &AuthorityHandler{
		authorityService: authorityService,
	}
}

// GetMyComplaints handles GET /authority/complaints?status=&page=1&page_size=20 (read-only; only complaints assigned to this authority).
func (h *AuthorityHandler) GetMyComplaints(w http.ResponseWriter, r *http.Request) {
	officerID, err := getOfficerIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Officer ID not found in context")
		return
	}
	status := r.URL.Query().Get("status")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	complaints, total, err := h.authorityService.GetComplaintsByOfficerIDPaginated(officerID, status, page, pageSize)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"complaints": complaints,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
	})
}

// UpdateComplaintStatus handles POST /api/v1/authority/complaints/{complaint_id}/status (body: new_status, reason). Assignment and transition validated in service.
func (h *AuthorityHandler) UpdateComplaintStatus(w http.ResponseWriter, r *http.Request) {
	officerID, err := getOfficerIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Officer ID not found in context")
		return
	}
	vars := mux.Vars(r)
	complaintID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Invalid complaint ID")
		return
	}
	var req models.AuthorityUpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Failed to parse request body")
		return
	}
	if req.NewStatus == "" {
		respondWithError(w, http.StatusBadRequest, "Validation error", "New status is required")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Reason is required for status change")
		return
	}
	ipAddress := getClientIP(r)
	userAgent := r.UserAgent()
	response, err := h.authorityService.UpdateComplaintStatus(complaintID, officerID, &req, ipAddress, userAgent)
	if err != nil {
		if err.Error() == "complaint not found" {
			respondWithError(w, http.StatusNotFound, "Not found", err.Error())
			return
		}
		if err.Error() == "complaint not assigned to this authority" {
			respondWithError(w, http.StatusForbidden, "Forbidden", err.Error())
			return
		}
		if strings.Contains(err.Error(), "invalid status transition") || err.Error() == "invalid status transition: closed is system-only" {
			respondWithError(w, http.StatusBadRequest, "Validation error", err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, response)
}

// AddNote handles POST /authority/complaints/{id}/note
// Adds an internal note (not visible to citizen)
func (h *AuthorityHandler) AddNote(w http.ResponseWriter, r *http.Request) {
	// Extract officer_id from context
	officerID, err := getOfficerIDFromContext(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Officer ID not found in context")
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
	var req models.AuthorityAddNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Failed to parse request body")
		return
	}

	// Validate required fields
	if req.NoteText == "" {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Note text is required")
		return
	}

	// Add note via service
	response, err := h.authorityService.AddNote(complaintID, officerID, req.NoteText)
	if err != nil {
		if err.Error() == "complaint not found" {
			respondWithError(w, http.StatusNotFound, "Not found", err.Error())
			return
		}
		if err.Error() == "complaint not assigned to this authority" {
			respondWithError(w, http.StatusForbidden, "Forbidden", err.Error())
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, response)
}

// Helper function to extract officer_id from context
func getOfficerIDFromContext(r *http.Request) (int64, error) {
	officerIDVal := r.Context().Value("officer_id")
	if officerIDVal == nil {
		return 0, fmt.Errorf("officer ID not found in context - authentication required")
	}

	officerID, ok := officerIDVal.(int64)
	if !ok {
		return 0, fmt.Errorf("invalid officer ID type in context")
	}

	return officerID, nil
}
