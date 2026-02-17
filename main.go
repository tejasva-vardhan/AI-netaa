package main

import (
	"database/sql"
	"finalneta/config"
	"finalneta/repository"
	"finalneta/routes"
	"finalneta/schema"
	"finalneta/service"
	"finalneta/worker"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	// Load configuration
	cfg := config.LoadConfig()
	
	// Log dry-run mode status if enabled
	if cfg.Pilot.DryRun {
		log.Printf("[DRY RUN] Pilot dry-run mode ENABLED")
		if cfg.Pilot.DryRunSLAOverrideMinutes > 0 {
			log.Printf("[DRY RUN] SLA override: %d minutes (instead of hours)", cfg.Pilot.DryRunSLAOverrideMinutes)
		} else {
			log.Printf("[DRY RUN] SLA override: DISABLED (using normal hours)")
		}
		log.Printf("[DRY RUN] Emails still go to shadow inbox: %s", service.PilotInboxEmail)
	}

	// Log test escalation override status
	if cfg.Pilot.TestEscalationOverrideMinutes > 0 {
		log.Printf("Test escalation override ENABLED: %d minutes", cfg.Pilot.TestEscalationOverrideMinutes)
	} else {
		log.Printf("Test escalation override DISABLED")
	}

	// Initialize database connection (UTC for consistent timestamps)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Database connection established")

	// Verify required columns exist (prevents escalation/status-history failures from schema lag)
	schema.ValidateRequiredColumns(db, nil)

	// Initialize repositories
	complaintRepo := repository.NewComplaintRepository(db)
	verificationRepo := repository.NewVerificationRepository(db)
	escalationRepo := repository.NewEscalationRepository(db)
	notificationRepo := repository.NewNotificationRepository(db)
	departmentRepo := repository.NewDepartmentRepository(db)
	userRepo := repository.NewUserRepository(db) // ISSUE 1 & 2: User repository
	authorityRepo := repository.NewAuthorityRepository(db)
	emailLogRepo := repository.NewEmailLogRepository(db)
	pilotMetricsRepo := repository.NewPilotMetricsRepository(db)
	voiceNoteRepo := repository.NewVoiceNoteRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo) // ISSUE 1 & 2: User service
	emailShadowService := service.NewEmailShadowService(emailLogRepo)
	pilotMetricsService := service.NewPilotMetricsService(pilotMetricsRepo)
	complaintService := service.NewComplaintService(complaintRepo, departmentRepo, emailShadowService, pilotMetricsService)
	verificationService := service.NewVerificationService(
		complaintRepo,
		verificationRepo,
		nil, // Use default config
	)
	escalationService := service.NewEscalationService(
		complaintRepo,
		escalationRepo,
		verificationRepo,
		emailShadowService,
		pilotMetricsService,
		cfg.Pilot.DryRun,
		cfg.Pilot.DryRunSLAOverrideMinutes,
		cfg.Pilot.TestEscalationOverrideMinutes,
	)
	notificationService := service.NewNotificationService(
		notificationRepo,
		complaintRepo,
		nil, // Use default config
	)

	// Escalation worker interval: config-driven and auto-adaptive for pilot/test
	const defaultProductionIntervalSec = 3600
	const pilotOverrideMaxIntervalSec = 30
	intervalSeconds := defaultProductionIntervalSec
	intervalReason := "production"
	if cfg.Pilot.TestEscalationOverrideMinutes > 0 {
		intervalSeconds = pilotOverrideMaxIntervalSec
		if cfg.Pilot.EscalationWorkerIntervalSeconds > 0 && cfg.Pilot.EscalationWorkerIntervalSeconds < pilotOverrideMaxIntervalSec {
			intervalSeconds = cfg.Pilot.EscalationWorkerIntervalSeconds
		}
		intervalReason = "pilot override"
	} else if cfg.Pilot.EscalationWorkerIntervalSeconds > 0 {
		intervalSeconds = cfg.Pilot.EscalationWorkerIntervalSeconds
	}
	escalationWorker := worker.NewEscalationWorker(
		escalationService,
		time.Duration(intervalSeconds)*time.Second,
	)
	log.Printf("Escalation worker interval: %d seconds", intervalSeconds)
	log.Printf("Reason: %s", intervalReason)
	escalationWorker.Start()

	notificationWorker := worker.NewNotificationWorker(
		notificationService,
		30*time.Second, // Process every 30 seconds
	)
	notificationWorker.Start()

	// Initialize abuse prevention service
	abusePreventionRepo := repository.NewAbusePreventionRepository(db)
	abusePreventionService := service.NewAbusePreventionService(abusePreventionRepo)

	// Setup routes
	router := routes.SetupRoutes(
		complaintService,
		verificationService,
		escalationService,
		escalationWorker,
		userService,
		complaintRepo,
		authorityRepo,
		abusePreventionService,
		emailShadowService,
		pilotMetricsService,
		voiceNoteRepo,
	)

	// Add CORS middleware
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
			// CRITICAL: Include Authorization header for JWT token authentication
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-Actor-Type, X-Officer-ID")
			w.Header().Set("Access-Control-Max-Age", "3600")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// Wrap router with CORS middleware
	handler := corsHandler(router)

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
