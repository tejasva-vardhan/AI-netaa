package handler

import (
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"finalneta/repository"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// AdminHandler provides admin-only endpoints for operating the pilot (authorities CRUD). No UI; no citizen impact.
type AdminHandler struct {
	authorityRepo *repository.AuthorityRepository
	complaintRepo *repository.ComplaintRepository
}

// NewAdminHandler creates an admin handler. complaintRepo used only for audit_log.
func NewAdminHandler(authorityRepo *repository.AuthorityRepository, complaintRepo *repository.ComplaintRepository) *AdminHandler {
	return &AdminHandler{authorityRepo: authorityRepo, complaintRepo: complaintRepo}
}

// GetAuthorities returns all authorities (officers) for admin. GET /api/v1/admin/authorities
func (h *AdminHandler) GetAuthorities(w http.ResponseWriter, r *http.Request) {
	list, err := h.authorityRepo.ListOfficers()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	// Map to JSON-friendly slice
	out := make([]adminAuthorityResponse, 0, len(list))
	for _, row := range list {
		levelStr := "L1"
		if row.AuthorityLevel == 2 {
			levelStr = "L2"
		} else if row.AuthorityLevel == 3 {
			levelStr = "L3"
		}
		out = append(out, adminAuthorityResponse{
			OfficerID:      row.OfficerID,
			FullName:       row.FullName,
			DepartmentID:   row.DepartmentID,
			LocationID:     row.LocationID,
			AuthorityLevel: levelStr,
			IsActive:       row.IsActive,
			Email:          row.Email,
		})
	}
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"authorities": out})
}

// CreateAuthority creates an officer and credential. POST /api/v1/admin/authorities. Audited.
func (h *AdminHandler) CreateAuthority(w http.ResponseWriter, r *http.Request) {
	var req adminCreateAuthorityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Bad Request", "Invalid JSON body")
		return
	}
	if req.FullName == "" || req.Email == "" || req.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Bad Request", "full_name, email, and password are required")
		return
	}
	if req.DepartmentID == 0 || req.LocationID == 0 {
		respondWithError(w, http.StatusBadRequest, "Bad Request", "department_id and location_id are required")
		return
	}
	level := levelStringToInt(req.AuthorityLevel)
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	officerID, err := h.authorityRepo.CreateOfficer(
		req.FullName, req.Designation, req.Email, req.Password,
		req.DepartmentID, req.LocationID, level, isActive,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	// Audit: admin action on entity_type=officer
	auditData := map[string]interface{}{
		"officer_id": officerID, "full_name": req.FullName,
		"department_id": req.DepartmentID, "location_id": req.LocationID,
		"authority_level": level, "is_active": isActive,
	}
	newVal, _ := json.Marshal(auditData)
	auditLog := &models.AuditLog{
		EntityType:     "officer",
		EntityID:       officerID,
		Action:         "create",
		ActionByType:   models.ActorAdmin,
		NewValues:      sql.NullString{String: string(newVal), Valid: true},
		IPAddress:      sql.NullString{String: r.RemoteAddr, Valid: true},
		UserAgent:      sql.NullString{String: r.UserAgent(), Valid: true},
	}
	_ = h.complaintRepo.CreateAuditLog(auditLog)

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"officer_id": officerID,
		"message":    "Authority created",
	})
}

// UpdateAuthority updates department_id, location_id, authority_level, is_active. PUT /api/v1/admin/authorities/{officer_id}. Audited.
func (h *AdminHandler) UpdateAuthority(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	officerIDStr, ok := vars["officer_id"]
	if !ok {
		respondWithError(w, http.StatusBadRequest, "Bad Request", "officer_id required")
		return
	}
	officerID, err := strconv.ParseInt(officerIDStr, 10, 64)
	if err != nil || officerID <= 0 {
		respondWithError(w, http.StatusBadRequest, "Bad Request", "Invalid officer_id")
		return
	}
	// Load current for audit old_values
	oldName, oldDept, oldLoc, oldLevel, oldActive, err := h.authorityRepo.GetOfficerByID(officerID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Not Found", "Officer not found")
		return
	}
	var req adminUpdateAuthorityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Bad Request", "Invalid JSON body")
		return
	}
	var dept, loc *int64
	var level *int
	var active *bool
	if req.DepartmentID != nil {
		dept = req.DepartmentID
	}
	if req.LocationID != nil {
		loc = req.LocationID
	}
	if req.AuthorityLevel != nil {
		l := levelStringToInt(*req.AuthorityLevel)
		level = &l
	}
	if req.IsActive != nil {
		active = req.IsActive
	}
	if dept == nil && loc == nil && level == nil && active == nil {
		respondWithError(w, http.StatusBadRequest, "Bad Request", "Provide at least one of department_id, location_id, authority_level, is_active")
		return
	}
	if err := h.authorityRepo.UpdateOfficer(officerID, dept, loc, level, active); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}
	// Audit: admin update
	oldVal, _ := json.Marshal(map[string]interface{}{
		"full_name": oldName, "department_id": oldDept, "location_id": oldLoc,
		"authority_level": oldLevel, "is_active": oldActive,
	})
	_, newDept, newLoc, newLevel, newActive, _ := h.authorityRepo.GetOfficerByID(officerID)
	newVal, _ := json.Marshal(map[string]interface{}{
		"full_name": oldName, "department_id": newDept, "location_id": newLoc,
		"authority_level": newLevel, "is_active": newActive,
	})
	auditLog := &models.AuditLog{
		EntityType:   "officer",
		EntityID:     officerID,
		Action:       "update",
		ActionByType: models.ActorAdmin,
		OldValues:    sql.NullString{String: string(oldVal), Valid: true},
		NewValues:    sql.NullString{String: string(newVal), Valid: true},
		IPAddress:    sql.NullString{String: r.RemoteAddr, Valid: true},
		UserAgent:    sql.NullString{String: r.UserAgent(), Valid: true},
	}
	_ = h.complaintRepo.CreateAuditLog(auditLog)

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "Authority updated"})
}

func levelStringToInt(s string) int {
	switch s {
	case "L2":
		return 2
	case "L3":
		return 3
	default:
		return 1
	}
}

type adminAuthorityResponse struct {
	OfficerID      int64  `json:"officer_id"`
	FullName       string `json:"full_name"`
	DepartmentID   int64  `json:"department_id"`
	LocationID     int64  `json:"location_id"`
	AuthorityLevel string `json:"authority_level"`
	IsActive       bool   `json:"is_active"`
	Email          string `json:"email,omitempty"`
}

type adminCreateAuthorityRequest struct {
	FullName       string  `json:"full_name"`
	Designation    string  `json:"designation"`
	Email          string  `json:"email"`
	Password       string  `json:"password"`
	DepartmentID   int64   `json:"department_id"`
	LocationID     int64   `json:"location_id"`
	AuthorityLevel string  `json:"authority_level"`
	IsActive       *bool   `json:"is_active"`
}

type adminUpdateAuthorityRequest struct {
	DepartmentID   *int64  `json:"department_id"`
	LocationID     *int64  `json:"location_id"`
	AuthorityLevel *string `json:"authority_level"`
	IsActive       *bool   `json:"is_active"`
}