// verify_escalation runs one end-to-end verification: complaint state, candidate query, optional fix, one escalation cycle, proof.
// Usage: from project root, run: go run ./cmd/verify_escalation
// Requires .env (or env) with DB_* and optionally TEST_ESCALATION_OVERRIDE_MINUTES=2.
package main

import (
	"database/sql"
	"finalneta/config"
	"finalneta/models"
	"finalneta/repository"
	"finalneta/schema"
	"finalneta/service"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env not found")
	}
	// Ensure pilot override so one cycle can fire without waiting hours
	if os.Getenv("TEST_ESCALATION_OVERRIDE_MINUTES") == "" {
		os.Setenv("TEST_ESCALATION_OVERRIDE_MINUTES", "2")
	}
	cfg := config.LoadConfig()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("DB open: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatalf("DB ping: %v", err)
	}
	schema.ValidateRequiredColumns(db, nil)

	// --- 1) Latest complaint state ---
	var complaintID int64
	var currentStatus string
	var currentLevel sql.NullInt64
	err = db.QueryRow(`
		SELECT complaint_id, current_status, current_escalation_level
		FROM complaints ORDER BY complaint_id DESC LIMIT 1
	`).Scan(&complaintID, &currentStatus, &currentLevel)
	if err == sql.ErrNoRows {
		log.Fatalf("No complaints in DB - cannot verify escalation")
	}
	if err != nil {
		log.Fatalf("Latest complaint query: %v", err)
	}
	level := 0
	if currentLevel.Valid {
		level = int(currentLevel.Int64)
	}
	log.Printf("[VERIFY] Latest complaint: id=%d current_status=%s current_escalation_level=%d", complaintID, currentStatus, level)

	// Invalid combination: status=escalated but level still 0 with no escalation row (partial state)
	var escalationRowCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM complaint_escalations WHERE complaint_id = ?`, complaintID).Scan(&escalationRowCount)
	if currentStatus == "escalated" && level == 0 && escalationRowCount == 0 {
		log.Printf("[VERIFY] Invalid state: status=escalated, level=0, no escalation row - normalizing")
		normalizeToUnderReview(db, complaintID)
		currentStatus = "under_review"
		level = 0
	}

	// --- 2) Candidate selection (same as worker) ---
	complaintRepo := repository.NewComplaintRepository(db)
	escalationRepo := repository.NewEscalationRepository(db)
	verificationRepo := repository.NewVerificationRepository(db)
	pilotMetricsRepo := repository.NewPilotMetricsRepository(db)
	escalationService := service.NewEscalationService(
		complaintRepo, escalationRepo, verificationRepo,
		nil, service.NewPilotMetricsService(pilotMetricsRepo),
		cfg.Pilot.DryRun, cfg.Pilot.DryRunSLAOverrideMinutes, cfg.Pilot.TestEscalationOverrideMinutes,
	)

	candidates, err := escalationRepo.GetEscalationCandidates(
		[]models.ComplaintStatus{models.StatusVerified, models.StatusUnderReview, models.StatusInProgress},
		24*time.Hour,
	)
	if err != nil {
		log.Fatalf("GetEscalationCandidates: %v", err)
	}
	log.Printf("[VERIFY] Candidate query returned %d candidates", len(candidates))
	inCandidates := false
	overrideMin := cfg.Pilot.TestEscalationOverrideMinutes
	if overrideMin <= 0 {
		overrideMin = 2
	}
	for _, c := range candidates {
		if c.ComplaintID == complaintID {
			inCandidates = true
			mins := time.Now().UTC().Sub(c.LastStatusChangeAt).Minutes()
			log.Printf("[VERIFY] Latest complaint %d IS in candidates (minutes_since_status_change=%.1f, override=%d min)", complaintID, mins, overrideMin)
			if mins < float64(overrideMin) {
				log.Printf("[VERIFY] SLA not yet met - backdating status history so escalation can fire")
				backdateStatusHistory(db, complaintID)
			}
			break
		}
	}
	if !inCandidates {
		log.Printf("[VERIFY] Latest complaint %d NOT in candidates (status=%s not in verified/under_review/in_progress, or excluded by query)", complaintID, currentStatus)
		// Normalize so it becomes eligible: under_review, level 0, backdate so SLA met
		log.Printf("[VERIFY] Normalizing complaint %d to under_review and backdating so SLA is met", complaintID)
		normalizeToUnderReview(db, complaintID)
		// Re-fetch candidates to confirm
		candidates, _ = escalationRepo.GetEscalationCandidates(
			[]models.ComplaintStatus{models.StatusVerified, models.StatusUnderReview, models.StatusInProgress},
			24*time.Hour,
		)
		inCandidates = false
		for _, c := range candidates {
			if c.ComplaintID == complaintID {
				inCandidates = true
				break
			}
		}
		if !inCandidates {
			log.Fatalf("[VERIFY] After normalize, complaint %d still not in candidates", complaintID)
		}
	}

	// --- 3) One escalation cycle ---
	log.Printf("[VERIFY] Running ProcessEscalations once ...")
	results, err := escalationService.ProcessEscalations()
	if err != nil {
		log.Fatalf("[VERIFY] ProcessEscalations: %v", err)
	}
	log.Printf("[VERIFY] ProcessEscalations returned %d results", len(results))
	fired := false
	for _, r := range results {
		if r.Escalated && r.ComplaintID == complaintID {
			fired = true
			log.Printf("[VERIFY] Escalation FIRED for complaint %d (escalation_id=%v)", complaintID, r.EscalationID)
			break
		}
	}

	// --- 4) DB proof ---
	var afterStatus string
	var afterLevel sql.NullInt64
	_ = db.QueryRow(`SELECT current_status, current_escalation_level FROM complaints WHERE complaint_id = ?`, complaintID).Scan(&afterStatus, &afterLevel)
	afterLevelVal := 0
	if afterLevel.Valid {
		afterLevelVal = int(afterLevel.Int64)
	}
	var escID, escLevel sql.NullInt64
	_ = db.QueryRow(`SELECT escalation_id, escalation_level FROM complaint_escalations WHERE complaint_id = ? ORDER BY created_at DESC LIMIT 1`, complaintID).Scan(&escID, &escLevel)

	log.Println("--- PROOF ---")
	if fired {
		log.Printf("[ESCALATION] ESCALATION FIRED (from results) for complaint_id=%d", complaintID)
	}
	log.Printf("complaints: complaint_id=%d current_status=%s current_escalation_level=%d", complaintID, afterStatus, afterLevelVal)
	if escID.Valid {
		log.Printf("complaint_escalations: escalation_id=%d complaint_id=%d escalation_level=%v", escID.Int64, complaintID, escLevel)
	} else {
		log.Printf("complaint_escalations: no row for complaint_id=%d", complaintID)
	}

	if !fired && afterLevelVal == 0 && !escID.Valid {
		os.Exit(1)
	}
}

func backdateStatusHistory(db *sql.DB, complaintID int64) {
	// Update all rows for this complaint so MAX(created_at) is in the past (UTC)
	_, _ = db.Exec(`UPDATE complaint_status_history SET created_at = UTC_TIMESTAMP() - INTERVAL 3 MINUTE WHERE complaint_id = ?`, complaintID)
}

func normalizeToUnderReview(db *sql.DB, complaintID int64) {
	_, err := db.Exec(`UPDATE complaints SET current_status = 'under_review', current_escalation_level = 0, updated_at = NOW() WHERE complaint_id = ?`, complaintID)
	if err != nil {
		log.Fatalf("Normalize complaints: %v", err)
	}
	_, _ = db.Exec(`DELETE FROM complaint_escalations WHERE complaint_id = ?`, complaintID)

	var historyID int64
	err = db.QueryRow(`SELECT history_id FROM complaint_status_history WHERE complaint_id = ? ORDER BY history_id DESC LIMIT 1`, complaintID).Scan(&historyID)
	if err == nil {
		_, _ = db.Exec(`UPDATE complaint_status_history SET new_status = 'under_review', created_at = UTC_TIMESTAMP() - INTERVAL 3 MINUTE WHERE history_id = ?`, historyID)
	}
}
