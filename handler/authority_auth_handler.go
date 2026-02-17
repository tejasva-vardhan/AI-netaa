package handler

import (
	"encoding/json"
	"finalneta/service"
	"finalneta/utils"
	"net/http"
	"os"
)

// AuthorityAuthHandler handles authority authentication (pilot: static token)
type AuthorityAuthHandler struct {
	authorityService *service.AuthorityService
}

// NewAuthorityAuthHandler creates a new authority auth handler
func NewAuthorityAuthHandler(authorityService *service.AuthorityService) *AuthorityAuthHandler {
	return &AuthorityAuthHandler{
		authorityService: authorityService,
	}
}

// LoginRequest represents authority login request
type AuthorityLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// OR
	StaticToken string `json:"static_token,omitempty"` // Pilot: static token option
}

// LoginResponse represents authority login response (token is scoped to authority; do not use on citizen endpoints).
type AuthorityLoginResponse struct {
	Success        bool   `json:"success"`
	Token          string `json:"token"`
	OfficerID      int64  `json:"officer_id"`
	AuthorityLevel int    `json:"authority_level"`
	Message        string `json:"message"`
}

// Login handles POST /authority/login
// PILOT: Supports email+password OR static token
func (h *AuthorityAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req AuthorityLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Failed to parse request body")
		return
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "pilot-secret-key-change-in-production"
	}

	var officerID int64
	var err error
	if req.StaticToken != "" {
		officerID, err = h.authorityService.ValidateStaticToken(req.StaticToken)
	} else {
		if req.Email == "" || req.Password == "" {
			respondWithError(w, http.StatusBadRequest, "Validation error", "Email and password are required")
			return
		}
		officerID, err = h.authorityService.ValidateCredentials(req.Email, req.Password)
	}
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid credentials")
		return
	}

	_, _, authorityLevel, err := h.authorityService.GetOfficerProfile(officerID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", "Failed to load profile")
		return
	}

	token, err := utils.GenerateAuthorityJWT(officerID, authorityLevel, []byte(jwtSecret), 24*7) // 7 days
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", "Failed to generate token")
		return
	}

	respondWithJSON(w, http.StatusOK, AuthorityLoginResponse{
		Success:        true,
		Token:          token,
		OfficerID:      officerID,
		AuthorityLevel: authorityLevel,
		Message:        "Login successful",
	})
}

// Logout handles POST /authority/logout (pilot: client discards token; no server-side invalidation).
func (h *AuthorityAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	respondWithJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "Logged out"})
}

// MeResponse is the response for GET /authority/me (officer profile from token + DB).
type MeResponse struct {
	OfficerID      int64 `json:"officer_id"`
	DepartmentID   int64 `json:"department_id"`
	LocationID     int64 `json:"location_id"`
	AuthorityLevel int   `json:"authority_level"`
}

// Me handles GET /authority/me (requires authority auth; returns officer profile).
func (h *AuthorityAuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	officerIDVal := r.Context().Value("officer_id")
	if officerIDVal == nil {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Officer ID not found")
		return
	}
	officerID, ok := officerIDVal.(int64)
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", "Invalid officer context")
		return
	}
	deptID, locID, level, err := h.authorityService.GetOfficerProfile(officerID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", err.Error())
		return
	}
	respondWithJSON(w, http.StatusOK, MeResponse{
		OfficerID:      officerID,
		DepartmentID:   deptID,
		LocationID:     locID,
		AuthorityLevel: level,
	})
}
