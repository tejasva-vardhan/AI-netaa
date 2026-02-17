package handler

import (
	"finalneta/repository"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// PublicHandler serves read-only public case data. No auth; whitelisted fields only; no PII.
type PublicHandler struct {
	complaintRepo *repository.ComplaintRepository
}

// NewPublicHandler creates a public handler. Does not touch escalation or existing APIs.
func NewPublicHandler(complaintRepo *repository.ComplaintRepository) *PublicHandler {
	return &PublicHandler{complaintRepo: complaintRepo}
}

// GetPublicComplaintByNumber returns public-safe complaint + timeline. GET /api/v1/public/complaints/by-number/{complaint_number}. No auth; complaint_id never exposed.
func (h *PublicHandler) GetPublicComplaintByNumber(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	complaintNumber, ok := vars["complaint_number"]
	if !ok || complaintNumber == "" {
		respondWithError(w, http.StatusBadRequest, "Bad Request", "complaint_number required")
		return
	}

	data, complaintID, err := h.complaintRepo.GetPublicComplaintByNumber(complaintNumber)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	if data == nil {
		respondWithError(w, http.StatusNotFound, "Not Found", "Complaint not found")
		return
	}

	// Timeline from complaint_status_history; complaint_id used only internally (never in response)
	history, err := h.complaintRepo.GetStatusHistory(complaintID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	timeline := make([]publicTimelineEntry, 0, len(history))
	for _, h := range history {
		actorType := ""
		if h.ActorType.Valid {
			actorType = h.ActorType.String
		} else {
			actorType = string(h.ChangedByType)
		}
		oldStatus := ""
		if h.OldStatus.Valid {
			oldStatus = h.OldStatus.String
		}
		timeline = append(timeline, publicTimelineEntry{
			CreatedAt:  h.CreatedAt.Format(time.RFC3339),
			OldStatus:  oldStatus,
			NewStatus:  string(h.NewStatus),
			ActorType:  actorType,
		})
	}

	respondWithJSON(w, http.StatusOK, publicComplaintResponse{
		ComplaintNumber: data.ComplaintNumber,
		LocationID:      data.LocationID,
		DepartmentID:    data.DepartmentID,
		CurrentStatus:   data.CurrentStatus,
		CreatedAt:       data.CreatedAt.Format(time.RFC3339),
		Timeline:        timeline,
	})
}

// publicComplaintResponse: whitelist only. No complaint_id; no PII, GPS, notes, actor_id.
type publicComplaintResponse struct {
	ComplaintNumber string                `json:"complaint_number"`
	LocationID      int64                 `json:"location_id"`
	DepartmentID    int64                 `json:"department_id"`
	CurrentStatus   string                `json:"current_status"`
	CreatedAt       string                `json:"created_at"`
	Timeline        []publicTimelineEntry `json:"timeline"`
}

type publicTimelineEntry struct {
	CreatedAt string `json:"created_at"`
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
	ActorType string `json:"actor_type"`
}
