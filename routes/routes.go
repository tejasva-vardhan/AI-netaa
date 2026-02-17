package routes

import (
	"finalneta/handler"
	"finalneta/middleware"
	"finalneta/repository"
	"finalneta/service"
	"finalneta/worker"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// SetupRoutes configures all API routes
func SetupRoutes(
	complaintService *service.ComplaintService,
	verificationService *service.VerificationService,
	escalationService *service.EscalationService,
	escalationWorker *worker.EscalationWorker,
	userService *service.UserService,
	complaintRepo *repository.ComplaintRepository,
	authorityRepo *repository.AuthorityRepository,
	abusePreventionService *service.AbusePreventionService,
	emailShadowService *service.EmailShadowService,
	pilotMetricsService *service.PilotMetricsService,
	voiceNoteRepo *repository.VoiceNoteRepository,
) *mux.Router {
	router := mux.NewRouter()

	// Initialize handlers
	complaintHandler := handler.NewComplaintHandler(complaintService, userService, abusePreventionService, complaintRepo, voiceNoteRepo)
	verificationHandler := handler.NewVerificationHandler(verificationService)
	escalationHandler := handler.NewEscalationHandler(escalationService, escalationWorker)
	phoneVerificationHandler := handler.NewPhoneVerificationHandler(userService)
	chatHandler := handler.NewChatHandler(pilotMetricsService)

	// Initialize auth middleware
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "pilot-secret-key-change-in-production" // Default for pilot
	}
	authMiddleware := middleware.NewAuthMiddleware(userService, jwtSecret)

	// Initialize authority service and handlers
	authorityService := service.NewAuthorityService(complaintRepo, authorityRepo, emailShadowService, pilotMetricsService)
	authorityHandler := handler.NewAuthorityHandler(authorityService)
	authorityAuthHandler := handler.NewAuthorityAuthHandler(authorityService)
	authorityAuthMiddleware := middleware.NewAuthorityAuthMiddleware(authorityService, jwtSecret)

	// API v1 routes
	apiV1 := router.PathPrefix("/api/v1").Subrouter()

	// Complaint routes (protected - require auth)
	complaints := apiV1.PathPrefix("/complaints").Subrouter()
	
	// GET /api/v1/complaints - Get user's complaints list (REQUIRES AUTH)
	complaints.Handle("", authMiddleware.RequireAuth(http.HandlerFunc(complaintHandler.GetUserComplaints))).Methods("GET")
	
	// POST /api/v1/complaints - Create a new complaint (REQUIRES AUTH)
	complaints.Handle("", authMiddleware.RequireAuth(http.HandlerFunc(complaintHandler.CreateComplaint))).Methods("POST")
	
	// GET /api/v1/complaints/{id} - Get complaint by ID (citizen view) (REQUIRES AUTH)
	complaints.Handle("/{id}", authMiddleware.RequireAuth(http.HandlerFunc(complaintHandler.GetComplaintByID))).Methods("GET")
	
	// GET /api/v1/complaints/{id}/timeline - Get complaint status timeline (REQUIRES AUTH)
	complaints.Handle("/{id}/timeline", authMiddleware.RequireAuth(http.HandlerFunc(complaintHandler.GetStatusTimeline))).Methods("GET")

	// POST /api/v1/complaints/{id}/voice - Upload voice note (citizen owner only; one per complaint; overwrite allowed)
	complaints.Handle("/{id}/voice", authMiddleware.RequireAuth(http.HandlerFunc(complaintHandler.UploadVoice))).Methods("POST")

	// POST /api/v1/complaints/{id}/verify - Verify a complaint (rule-based). Admin only; no public status write.
	complaints.Handle("/{id}/verify", middleware.RequireAdminAuth(http.HandlerFunc(verificationHandler.VerifyComplaint))).Methods("POST")

	// Phone verification routes (ISSUE 1 & 2: User creation during phone verification)
	users := apiV1.PathPrefix("/users").Subrouter()
	
	// POST /api/v1/users/otp/send - Send OTP to phone number
	users.HandleFunc("/otp/send", phoneVerificationHandler.SendOTP).Methods("POST")
	
	// POST /api/v1/users/otp/verify - Verify OTP and create/get user
	users.HandleFunc("/otp/verify", phoneVerificationHandler.VerifyOTP).Methods("POST")

	// POST /api/v1/users/chat/reset - Reset chat/draft (hidden "restart"); requires auth
	users.Handle("/chat/reset", authMiddleware.RequireAuth(http.HandlerFunc(chatHandler.ResetChatDraft))).Methods("POST")

	// Escalation routes (admin only; worker runs internally, no HTTP)
	escalations := apiV1.PathPrefix("/escalations").Subrouter()
	escalations.Handle("/process", middleware.RequireAdminAuth(http.HandlerFunc(escalationHandler.ProcessEscalations))).Methods("POST")

	// Authority dashboard routes (protected - require authority auth)
	authority := apiV1.PathPrefix("/authority").Subrouter()
	
	// POST /api/v1/authority/login - Authority login (email+password OR static token); returns JWT with officer_id and authority_level.
	authority.HandleFunc("/login", authorityAuthHandler.Login).Methods("POST")
	// POST /api/v1/authority/logout - Client discards token (no server-side invalidation in pilot).
	authority.HandleFunc("/logout", authorityAuthHandler.Logout).Methods("POST")
	// GET /api/v1/authority/me - Officer profile (requires authority auth).
	authority.Handle("/me", authorityAuthMiddleware.RequireAuthorityAuth(http.HandlerFunc(authorityAuthHandler.Me))).Methods("GET")

	// GET /api/v1/authority/complaints - Get complaints assigned to logged-in authority
	authority.Handle("/complaints", authorityAuthMiddleware.RequireAuthorityAuth(http.HandlerFunc(authorityHandler.GetMyComplaints))).Methods("GET")
	
	// POST /api/v1/authority/complaints/{id}/status - Update complaint status
	authority.Handle("/complaints/{id}/status", authorityAuthMiddleware.RequireAuthorityAuth(http.HandlerFunc(authorityHandler.UpdateComplaintStatus))).Methods("POST")
	
	// POST /api/v1/authority/complaints/{id}/note - Add internal note
	authority.Handle("/complaints/{id}/note", authorityAuthMiddleware.RequireAuthorityAuth(http.HandlerFunc(authorityHandler.AddNote))).Methods("POST")

	// Admin routes (env-based token; separate from citizen/authority). No UI; pilot operation only.
	adminHandler := handler.NewAdminHandler(authorityRepo, complaintRepo)
	admin := apiV1.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.RequireAdminAuth)
	admin.HandleFunc("/authorities", adminHandler.GetAuthorities).Methods("GET")
	admin.HandleFunc("/authorities", adminHandler.CreateAuthority).Methods("POST")
	admin.HandleFunc("/authorities/{officer_id}", adminHandler.UpdateAuthority).Methods("PUT")

	// Public read-only case page by complaint_number (shareable; complaint_id never exposed).
	publicHandler := handler.NewPublicHandler(complaintRepo)
	apiV1.HandleFunc("/public/complaints/by-number/{complaint_number}", publicHandler.GetPublicComplaintByNumber).Methods("GET")

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	return router
}
