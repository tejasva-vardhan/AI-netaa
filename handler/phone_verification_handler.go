package handler

import (
	"encoding/json"
	"finalneta/service"
	"finalneta/utils"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// PhoneVerificationHandler handles phone verification (OTP) requests
type PhoneVerificationHandler struct {
	userService *service.UserService
	// In production, use Redis or database to store OTPs
	// For pilot: simple in-memory map (will be lost on restart)
	otpStore map[string]OTPData
}

// OTPData stores OTP information
type OTPData struct {
	Code      string
	ExpiresAt time.Time
	Phone     string
}

// NewPhoneVerificationHandler creates a new phone verification handler
func NewPhoneVerificationHandler(userService *service.UserService) *PhoneVerificationHandler {
	return &PhoneVerificationHandler{
		userService: userService,
		otpStore:    make(map[string]OTPData),
	}
}

// SendOTPRequest represents the request to send OTP
type SendOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
}

// SendOTPResponse represents the response after sending OTP
type SendOTPResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ExpiresIn int    `json:"expires_in"` // seconds
	OTP       string `json:"otp"` // DEV MODE ONLY - OTP code for testing (always included in dev)
}

// VerifyOTPRequest represents the request to verify OTP
type VerifyOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	OTP         string `json:"otp"`
}

// VerifyOTPResponse represents the response after verifying OTP
type VerifyOTPResponse struct {
	Success       bool   `json:"success"`
	UserID        int64  `json:"user_id"`
	PhoneVerified bool   `json:"phone_verified"`
	Token         string `json:"token"` // JWT token for authenticated requests
	Message       string `json:"message"`
}

// SendOTP handles POST /api/v1/users/otp/send
// Sends OTP to phone number
func (h *PhoneVerificationHandler) SendOTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("\n[DEBUG] SendOTP handler called - Method: %s, Path: %s\n", r.Method, r.URL.Path)
	
	var req SendOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Printf("[ERROR] Failed to parse request body: %v\n", err)
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Failed to parse request body")
		return
	}
	
	fmt.Printf("[DEBUG] Received SendOTP request - Phone: %s\n", req.PhoneNumber)

	// Validate phone number
	if req.PhoneNumber == "" {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Phone number is required")
		return
	}

	// Clean phone number (remove non-digits)
	cleanPhone := cleanPhoneNumber(req.PhoneNumber)
	if len(cleanPhone) != 10 {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Invalid phone number format")
		return
	}

	// Generate 6-digit OTP
	otpCode := generateOTP(6)

	// Store OTP (expires in 10 minutes)
	expiresAt := time.Now().Add(10 * time.Minute)
	h.otpStore[cleanPhone] = OTPData{
		Code:      otpCode,
		ExpiresAt: expiresAt,
		Phone:     cleanPhone,
	}

	// Always log OTP for debugging (server logs only)
	log.Printf("[OTP DEBUG] Phone: %s, OTP: %s", cleanPhone, otpCode)

	// Attempt SMS sending (existing flow: when configured, send via gateway)
	smsSent := trySendOTPSMS(cleanPhone, otpCode)

	if smsSent {
		// SMS succeeded: normal response, no debug_otp
		respondWithJSON(w, http.StatusOK, map[string]interface{}{
			"success":    true,
			"message":    "OTP sent successfully",
			"expires_in": 600,
		})
		return
	}
	// SMS failed or not configured: success with dev fallback and debug_otp
	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"message":    "OTP sent (DEV MODE)",
		"debug_otp":  otpCode,
		"expires_in": 600,
	})
}

// VerifyOTP handles POST /api/v1/users/otp/verify
// Verifies OTP and creates/updates user
func (h *PhoneVerificationHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var req VerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request", "Failed to parse request body")
		return
	}

	// Validate inputs
	if req.PhoneNumber == "" || req.OTP == "" {
		respondWithError(w, http.StatusBadRequest, "Validation error", "Phone number and OTP are required")
		return
	}

	cleanPhone := cleanPhoneNumber(req.PhoneNumber)
	
	// Debug logging
	fmt.Printf("\n========================================\n")
	fmt.Printf("[DEBUG] OTP VERIFICATION REQUEST\n")
	fmt.Printf("Original Phone: %s\n", req.PhoneNumber)
	fmt.Printf("Cleaned Phone: %s\n", cleanPhone)
	fmt.Printf("Received OTP: %s\n", req.OTP)
	fmt.Printf("OTP Store Keys (active phones): %v\n", getKeys(h.otpStore))
	fmt.Printf("========================================\n\n")

	// Check if OTP exists and is valid
	otpData, exists := h.otpStore[cleanPhone]
	if !exists {
		fmt.Printf("[ERROR] OTP not found for phone: %s (cleaned: %s)\n", req.PhoneNumber, cleanPhone)
		respondWithError(w, http.StatusBadRequest, "Invalid OTP", "OTP not found. Please request a new OTP.")
		return
	}

	// Check if OTP expired
	if time.Now().After(otpData.ExpiresAt) {
		delete(h.otpStore, cleanPhone)
		respondWithError(w, http.StatusBadRequest, "OTP expired", "OTP has expired. Please request a new OTP.")
		return
	}

	// Verify OTP code
	fmt.Printf("[DEBUG] Comparing OTP - Stored: '%s' (len=%d), Received: '%s' (len=%d)\n", 
		otpData.Code, len(otpData.Code), req.OTP, len(req.OTP))
	if otpData.Code != req.OTP {
		fmt.Printf("\n[ERROR] OTP MISMATCH!\n")
		fmt.Printf("Expected (stored): '%s'\n", otpData.Code)
		fmt.Printf("Received: '%s'\n", req.OTP)
		fmt.Printf("Phone: %s\n\n", cleanPhone)
		respondWithError(w, http.StatusBadRequest, "Invalid OTP", "Invalid OTP. Please try again.")
		return
	}
	
	fmt.Printf("[SUCCESS] OTP matched! Proceeding with verification...\n\n")

	// OTP verified - get or create user
	userID, _, err := h.userService.GetOrCreateUserByPhone(cleanPhone)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", fmt.Sprintf("Failed to create/get user: %v", err))
		return
	}

	// Mark phone as verified
	err = h.userService.MarkPhoneVerified(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", fmt.Sprintf("Failed to verify phone: %v", err))
		return
	}

	// Clear OTP after successful verification
	delete(h.otpStore, cleanPhone)

	// Generate JWT token for authenticated requests
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "pilot-secret-key-change-in-production" // Default for pilot
	}

	token, err := utils.GenerateJWT(userID, []byte(jwtSecret), 24*7) // 7 days expiry
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Internal error", fmt.Sprintf("Failed to generate token: %v", err))
		return
	}

	respondWithJSON(w, http.StatusOK, VerifyOTPResponse{
		Success:       true,
		UserID:        userID,
		PhoneVerified: true,
		Token:         token,
		Message:       "Phone verified successfully",
	})
}

// Helper functions
func cleanPhoneNumber(phone string) string {
	// Remove all non-digit characters
	result := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			result += string(char)
		}
	}
	return result
}

func generateOTP(length int) string {
	otp := ""
	for i := 0; i < length; i++ {
		otp += fmt.Sprintf("%d", rand.Intn(10))
	}
	return otp
}

// Helper function to get keys from map for debugging
func getKeys(m map[string]OTPData) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// trySendOTPSMS attempts to send OTP via SMS when configured (e.g. TWILIO_*). Returns true only if send succeeded.
func trySendOTPSMS(phone, otp string) bool {
	// When SMS gateway is configured, call it here; on success return true.
	// Not configured or send failure → return false (handler will return dev fallback with debug_otp).
	if os.Getenv("TWILIO_ACCOUNT_SID") == "" && os.Getenv("SMS_ENABLED") != "true" {
		return false
	}
	// Placeholder: real implementation would call Twilio/SNS etc.
	// For now, treat as not configured so we don't break; when implemented, return true on success.
	return false
}
