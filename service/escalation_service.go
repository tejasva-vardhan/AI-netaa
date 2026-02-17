package service

import (
	"database/sql"
	"encoding/json"
	"finalneta/models"
	"finalneta/repository"
	"fmt"
	"log"
	"time"
)

// EscalationService handles escalation and reminder logic
type EscalationService struct {
	complaintRepo               *repository.ComplaintRepository
	escalationRepo              *repository.EscalationRepository
	verificationRepo            *repository.VerificationRepository
	emailShadowService          *EmailShadowService   // optional; pilot email shadow mode
	pilotMetricsService         *PilotMetricsService // optional; pilot metrics
	dryRun                      bool                 // PILOT_DRY_RUN: Enable dry-run/testing mode
	dryRunSLAOverrideMinutes    int                  // PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES: Override SLA hours with minutes (0 = disabled)
	testEscalationOverrideMinutes int                 // TEST_ESCALATION_OVERRIDE_MINUTES: Safe test-only SLA override (0 = disabled)
}

// NewEscalationService creates a new escalation service
func NewEscalationService(
	complaintRepo *repository.ComplaintRepository,
	escalationRepo *repository.EscalationRepository,
	verificationRepo *repository.VerificationRepository,
	emailShadowService *EmailShadowService,
	pilotMetricsService *PilotMetricsService,
	dryRun bool,
	dryRunSLAOverrideMinutes int,
	testEscalationOverrideMinutes int,
) *EscalationService {
	return &EscalationService{
		complaintRepo:               complaintRepo,
		escalationRepo:              escalationRepo,
		verificationRepo:            verificationRepo,
		emailShadowService:          emailShadowService,
		pilotMetricsService:         pilotMetricsService,
		dryRun:                      dryRun,
		dryRunSLAOverrideMinutes:    dryRunSLAOverrideMinutes,
		testEscalationOverrideMinutes: testEscalationOverrideMinutes,
	}
}

// ProcessEscalations processes all complaints that may need escalation
// This is the main entry point for the escalation engine
//
// Flow:
// 1. Load active escalation rules
// 2. Get escalation candidates (complaints matching criteria)
// 3. For each candidate, evaluate rules
// 4. Create escalation records if conditions met
// 5. Update complaint status via status history
// 6. Log all actions to audit_log
func (s *EscalationService) ProcessEscalations() ([]models.EscalationResult, error) {
	// DRY RUN MODE: Log mode status
	if s.dryRun {
		log.Printf("[DRY RUN] Escalation worker running in DRY RUN mode")
		if s.dryRunSLAOverrideMinutes > 0 {
			log.Printf("[DRY RUN] SLA override: %d minutes (instead of hours)", s.dryRunSLAOverrideMinutes)
		}
	}
	
	// Load active escalation rules
	rules, err := s.escalationRepo.GetActiveEscalationRules()
	if err != nil {
		return nil, fmt.Errorf("failed to load escalation rules: %w", err)
	}

	log.Printf("[ESCALATION_DEBUG] Loaded %d active rules", len(rules))
	for i, rule := range rules {
		log.Printf("[ESCALATION_DEBUG] Rule %d: escalation_level=%d", i+1, rule.EscalationLevel)
	}
	if len(rules) == 0 {
		log.Printf("[ESCALATION_DEBUG] No rules configured - skipping")
		return []models.EscalationResult{}, nil
	}

	candidates, err := s.escalationRepo.GetEscalationCandidates(
		[]models.ComplaintStatus{
			models.StatusVerified,
			models.StatusUnderReview,
			models.StatusInProgress,
		},
		24*time.Hour, // unused; candidate query no longer filters by time
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get escalation candidates: %w", err)
	}

	log.Printf("[ESCALATION_DEBUG] Fetched %d escalation candidates", len(candidates))
	now := time.Now().UTC()
	for _, c := range candidates {
		mins := now.Sub(c.LastStatusChangeAt).Minutes()
		pincode := ""
		if c.Pincode.Valid {
			pincode = c.Pincode.String
		}
		deptID := int64(0)
		if c.AssignedDepartmentID.Valid {
			deptID = c.AssignedDepartmentID.Int64
		}
		log.Printf("[ESCALATION_DEBUG] Candidate complaint_id=%d current_status=%s department_id=%d pincode=%s location_id=%d minutes_since_status_change=%.1f",
			c.ComplaintID, c.CurrentStatus, deptID, pincode, c.LocationID, mins)
	}

	var results []models.EscalationResult
	for _, candidate := range candidates {
		result, err := s.processComplaintEscalation(candidate, rules)
		if err != nil {
			log.Printf("[ESCALATION] Skipping complaint %d: %v", candidate.ComplaintID, err)
			continue
		}
		if result != nil {
			results = append(results, *result)
		}
	}

	return results, nil
}

// processComplaintEscalation processes escalation for a single complaint
func (s *EscalationService) processComplaintEscalation(
	candidate models.EscalationCandidate,
	rules []models.EscalationRule,
) (*models.EscalationResult, error) {
	// Get current escalation level (0 = L1, 1 = L2, 2 = L3)
	currentLevel, err := s.escalationRepo.GetLastEscalationLevel(candidate.ComplaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get escalation level: %w", err)
	}
	log.Printf("[ESCALATION_DEBUG] complaint_id=%d current_escalation_level=%d (rule.escalation_level must equal this)", candidate.ComplaintID, currentLevel)

	// SAFEGUARD 1: Do NOT escalate beyond L3 (max escalation level)
	// L3 = currentLevel 2 (0=L1, 1=L2, 2=L3)
	const MaxEscalationLevel = 2 // L3 is the maximum
	if currentLevel >= MaxEscalationLevel {
		log.Printf("[ESCALATION_DEBUG] skip complaint %d: max escalation level reached (currentLevel=%d)", candidate.ComplaintID, currentLevel)
		return nil, nil
	}

	// Find applicable rules: escalation_level refers to CURRENT level before escalation
	// Rule with escalation_level = 0 means "when at L1 (level 0), escalate"
	// Rule with escalation_level = 1 means "when at L2 (level 1), escalate"
	var applicableRules []models.EscalationRule
	for _, rule := range rules {
		if rule.EscalationLevel == currentLevel {
			// Check if rule matches complaint's department and location
			if s.ruleMatchesComplaint(rule, candidate) {
				applicableRules = append(applicableRules, rule)
			}
		}
	}

	if len(applicableRules) == 0 {
		log.Printf("[ESCALATION_DEBUG] skip complaint %d: no rule found (department/location mismatch or no rule for level %d)", candidate.ComplaintID, currentLevel)
		return nil, nil
	}
	log.Printf("[ESCALATION_DEBUG] complaint_id=%d applicable_rules=%d", candidate.ComplaintID, len(applicableRules))

	// Evaluate conditions for each applicable rule
	for _, rule := range applicableRules {
		conditions, err := repository.ParseEscalationConditions(rule.Conditions)
		if err != nil {
			continue // Skip rule if conditions can't be parsed
		}

		if conditions == nil {
			continue // No conditions defined
		}

		// Check if this is a reminder (not escalation)
		if conditions.IsReminder {
			reminderResult, err := s.processReminder(candidate, rule, conditions)
			if err != nil {
				continue
			}
			if reminderResult != nil && reminderResult.ReminderSent {
				// Return reminder result (but don't escalate)
				return &models.EscalationResult{
					ComplaintID: candidate.ComplaintID,
					Escalated:   false,
					Reason:      reminderResult.Reason,
					ProcessedAt: reminderResult.ProcessedAt,
				}, nil
			}
			continue // Reminder processed, check next rule
		}

		// Evaluate escalation conditions
		shouldEscalate, reason := s.evaluateEscalationConditions(candidate, conditions)
		if !shouldEscalate {
			log.Printf("[ESCALATION_DEBUG] skip complaint %d: SLA/conditions not satisfied - %s", candidate.ComplaintID, reason)
			continue
		}

		// Check idempotency - don't escalate if already escalated at this level recently
		alreadyEscalated, err := s.escalationRepo.HasExistingEscalation(
			candidate.ComplaintID,
			rule.EscalationLevel,
			1, // Within 1 hour
		)
		if err != nil {
			return nil, fmt.Errorf("failed to check idempotency: %w", err)
		}
		if alreadyEscalated {
			return nil, nil // Already escalated, skip
		}

		// Perform escalation
		return s.executeEscalation(candidate, rule, reason)
	}

	return nil, nil // No escalation needed
}

// ruleMatchesComplaint checks if an escalation rule matches a complaint
func (s *EscalationService) ruleMatchesComplaint(
	rule models.EscalationRule,
	candidate models.EscalationCandidate,
) bool {
	// Check department match
	if rule.FromDepartmentID.Valid {
		if !candidate.AssignedDepartmentID.Valid ||
			candidate.AssignedDepartmentID.Int64 != rule.FromDepartmentID.Int64 {
			return false
		}
	}

	// Check location match
	if rule.FromLocationID.Valid {
		if candidate.LocationID != rule.FromLocationID.Int64 {
			return false
		}
	}

	return true
}

// evaluateEscalationConditions evaluates if escalation conditions are met
func (s *EscalationService) evaluateEscalationConditions(
	candidate models.EscalationCandidate,
	conditions *models.EscalationConditions,
) (bool, string) {
	now := time.Now().UTC()

	// Check status conditions
	if len(conditions.Statuses) > 0 {
		statusMatch := false
		for _, status := range conditions.Statuses {
			if string(candidate.CurrentStatus) == status {
				statusMatch = true
				break
			}
		}
		if !statusMatch {
			return false, "Status condition not met"
		}
	}

	// Check priority conditions
	if len(conditions.Priorities) > 0 {
		priorityMatch := false
		for _, priority := range conditions.Priorities {
			if string(candidate.Priority) == priority {
				priorityMatch = true
				break
			}
		}
		if !priorityMatch {
			return false, "Priority condition not met"
		}
	}

	// Check time-based conditions
	if conditions.TimeBased != nil {
		timeBased := conditions.TimeBased

		// Check hours since last update
		if timeBased.HoursSinceLastUpdate > 0 {
			var lastUpdate time.Time
			if candidate.UpdatedAt.Valid {
				lastUpdate = candidate.UpdatedAt.Time
			} else {
				lastUpdate = candidate.CreatedAt
			}

			hoursSinceUpdate := now.Sub(lastUpdate).Hours()
			if hoursSinceUpdate < float64(timeBased.HoursSinceLastUpdate) {
				return false, fmt.Sprintf("Not enough time since last update: %.1f hours", hoursSinceUpdate)
			}
		}

		// Check SLA hours (preferred) or legacy hours_since_status_change
		slaHours := timeBased.SLAHours
		if slaHours == 0 && timeBased.HoursSinceStatusChange > 0 {
			slaHours = timeBased.HoursSinceStatusChange // Backward compatibility
		}
		if slaHours > 0 {
			// Effective SLA in MINUTES for comparison
			var effectiveSlaMinutes float64
			if s.testEscalationOverrideMinutes > 0 {
				effectiveSlaMinutes = float64(s.testEscalationOverrideMinutes)
			} else if s.dryRun && s.dryRunSLAOverrideMinutes > 0 {
				effectiveSlaMinutes = float64(s.dryRunSLAOverrideMinutes)
				log.Printf("[DRY RUN] SLA override: %d hours -> %d minutes", slaHours, s.dryRunSLAOverrideMinutes)
			} else {
				effectiveSlaMinutes = float64(slaHours) * 60
			}

			minutesSinceStatusChange := now.Sub(candidate.LastStatusChangeAt).Minutes()
			if minutesSinceStatusChange < effectiveSlaMinutes {
				if s.testEscalationOverrideMinutes > 0 {
					return false, fmt.Sprintf("SLA not breached: %.1f minutes elapsed (test override: %d minutes)", minutesSinceStatusChange, s.testEscalationOverrideMinutes)
				}
				if s.dryRun && s.dryRunSLAOverrideMinutes > 0 {
					return false, fmt.Sprintf("[DRY RUN] SLA not breached: %.1f minutes elapsed (SLA override: %d minutes)", minutesSinceStatusChange, s.dryRunSLAOverrideMinutes)
				}
				return false, fmt.Sprintf("SLA not breached: %.1f hours elapsed (SLA: %d hours)", minutesSinceStatusChange/60, slaHours)
			}
		}

		// Check hours since creation
		if timeBased.HoursSinceCreation > 0 {
			hoursSinceCreation := now.Sub(candidate.CreatedAt).Hours()
			if hoursSinceCreation < float64(timeBased.HoursSinceCreation) {
				return false, fmt.Sprintf("Not enough time since creation: %.1f hours", hoursSinceCreation)
			}
		}
	}

	return true, "All conditions met"
}

// executeEscalation performs the actual escalation
// Escalation reassigns authority (department), not personnel (officers)
func (s *EscalationService) executeEscalation(
	candidate models.EscalationCandidate,
	rule models.EscalationRule,
	reason string,
) (*models.EscalationResult, error) {
	// Determine target department: if rule.ToDepartmentID is NULL, escalate within same department
	var targetDepartmentID int64
	if !rule.ToDepartmentID.Valid {
		// NULL means escalate within same department hierarchy
		if !candidate.AssignedDepartmentID.Valid {
			log.Printf("[ESCALATION] Warning: complaint %d has no assigned department, cannot escalate", candidate.ComplaintID)
			return nil, nil
		}
		targetDepartmentID = candidate.AssignedDepartmentID.Int64
	} else {
		targetDepartmentID = rule.ToDepartmentID.Int64
	}

	// Determine target location (default to same location)
	toLocationID := candidate.LocationID
	if rule.ToLocationID.Valid {
		toLocationID = rule.ToLocationID.Int64
	}

	// Authority lookup: department_id + location_id + current escalation level
	targetLevel := rule.EscalationLevel + 1
	log.Printf("[ESCALATION_DEBUG] FindAuthorityByDepartmentPincodeLevel inputs: department_id=%d location_id=%d current_level=%d (target L%d)",
		targetDepartmentID, toLocationID, rule.EscalationLevel, targetLevel)

	authorityID, err := s.escalationRepo.FindAuthorityByDepartmentPincodeLevel(
		targetDepartmentID,
		toLocationID,
		rule.EscalationLevel,
	)
	if err != nil {
		log.Printf("[ESCALATION_DEBUG] skip complaint %d: authority lookup failed - %v", candidate.ComplaintID, err)
		return nil, nil
	}

	var toOfficerID *int64
	if authorityID != nil {
		toOfficerID = authorityID
		log.Printf("[ESCALATION_DEBUG] authority lookup returned officer_id=%d", *authorityID)
	} else {
		log.Printf("[ESCALATION_DEBUG] authority lookup returned 0 rows (no officer for dept=%d location=%d level=%d)", targetDepartmentID, toLocationID, rule.EscalationLevel)
		// [PILOT] Escalate anyway when no authority found (level increment, department unchanged, officer unassigned).
		log.Printf("[ESCALATION] WARNING [PILOT] No authority found for complaint %d - escalating anyway (level increment, department unchanged, officer unassigned)", candidate.ComplaintID)
		toOfficerID = nil
	}

	// Update complaint status to "escalated" via status history
	// Escalation reassigns to target department (authority), not individual officer
	err = s.complaintRepo.UpdateComplaintStatus(
		candidate.ComplaintID,
		models.StatusEscalated,
		&targetDepartmentID,
		toOfficerID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update complaint status: %w", err)
	}

	// Create status history entry (REQUIRED - escalation audit: system, no actor_id, reason)
	reasonNote := fmt.Sprintf("Escalated to level %d: %s", rule.EscalationLevel, reason)
	if s.dryRun {
		reasonNote = fmt.Sprintf("[DRY RUN] Escalated to level %d: %s", rule.EscalationLevel, reason)
	}
	statusHistory := &models.ComplaintStatusHistory{
		ComplaintID:   candidate.ComplaintID,
		OldStatus:     sql.NullString{String: string(candidate.CurrentStatus), Valid: true},
		NewStatus:     models.StatusEscalated,
		ChangedByType: models.ActorSystem,
		ActorType:     sql.NullString{String: string(models.StatusHistoryActorSystem), Valid: true},
		ActorID:       sql.NullInt64{Valid: false}, // system has no actor_id
		Reason:        sql.NullString{String: reasonNote, Valid: true},
		Notes:         sql.NullString{String: reasonNote, Valid: true},
	}

	// Set new assignment (authority reassignment)
	statusHistory.AssignedDepartmentID = sql.NullInt64{Int64: targetDepartmentID, Valid: true}
	if toOfficerID != nil {
		statusHistory.AssignedOfficerID = sql.NullInt64{Int64: *toOfficerID, Valid: true}
	}

	err = s.complaintRepo.CreateStatusHistory(statusHistory)
	if err != nil {
		return nil, fmt.Errorf("failed to create status history: %w", err)
	}

	// Create escalation record (with link to status history)
	// Escalation level stored is the CURRENT level before escalation (rule.EscalationLevel)
	escalation := &models.ComplaintEscalation{
		ComplaintID:        candidate.ComplaintID,
		ToDepartmentID:     targetDepartmentID,
		EscalationLevel:    rule.EscalationLevel, // Current level before escalation
		EscalatedByType:    models.ActorSystem,
		Reason:             sql.NullString{String: reason, Valid: true},
		StatusHistoryID:    sql.NullInt64{Int64: statusHistory.HistoryID, Valid: true},
	}

	if candidate.AssignedDepartmentID.Valid {
		escalation.FromDepartmentID = candidate.AssignedDepartmentID
	}
	if candidate.AssignedOfficerID.Valid {
		escalation.FromOfficerID = candidate.AssignedOfficerID
	}
	if toOfficerID != nil {
		escalation.ToOfficerID = sql.NullInt64{Int64: *toOfficerID, Valid: true}
	}

	err = s.escalationRepo.CreateEscalation(escalation)
	if err != nil {
		return nil, fmt.Errorf("failed to create escalation: %w", err)
	}
	newLevel := rule.EscalationLevel + 1
	if err = s.complaintRepo.UpdateComplaintEscalationLevel(candidate.ComplaintID, newLevel); err != nil {
		// Log but don't fail - column may not exist in all envs
		log.Printf("[ESCALATION] Warning: could not update complaints.current_escalation_level: %v", err)
	}
	log.Printf("[ESCALATION] ESCALATION FIRED complaint_id=%d new_escalation_level=%d (from %d) escalation_id=%d", candidate.ComplaintID, newLevel, rule.EscalationLevel, escalation.EscalationID)

	// Log to audit_log (REQUIRED)
	auditData := map[string]interface{}{
		"escalation_id":     escalation.EscalationID,
		"escalation_level":  rule.EscalationLevel, // Current level before escalation
		"from_department":  candidate.AssignedDepartmentID,
		"to_department":     targetDepartmentID,
		"reason":           reason,
	}
	if s.dryRun {
		auditData["dry_run"] = true
		auditData["dry_run_sla_override_minutes"] = s.dryRunSLAOverrideMinutes
		log.Printf("[DRY RUN] Escalation executed for complaint %d: L%d -> L%d", candidate.ComplaintID, rule.EscalationLevel, rule.EscalationLevel+1)
	}
	err = s.logEscalationAction(
		candidate.ComplaintID,
		"escalation",
		auditData,
	)
	if err != nil {
		// Log error but don't fail - audit logging should be resilient
	}

	// Emit pilot metrics: escalation_triggered
	if s.pilotMetricsService != nil {
		// Get user_id from complaint
		complaint, err := s.complaintRepo.GetComplaintByID(candidate.ComplaintID)
		userID := int64(0)
		if err == nil {
			userID = complaint.UserID
		}
		metadata := map[string]interface{}{
			"escalation_level": rule.EscalationLevel,
			"target_level":     rule.EscalationLevel + 1,
			"from_department":  targetDepartmentID,
			"to_department":     targetDepartmentID,
			"reason":           reason,
		}
		s.pilotMetricsService.EmitEscalationTriggered(candidate.ComplaintID, userID, rule.EscalationLevel, metadata)
	}

	// Pilot: send escalation email to shadow inbox only (async, non-blocking)
	// Authority abstraction: department_id + level, not officer-based
	if s.emailShadowService != nil {
		deptID := targetDepartmentID
		deptName := fmt.Sprintf("Department %d", deptID)
		s.emailShadowService.SendEscalationEmailAsync(
			candidate.ComplaintID,
			candidate.ComplaintNumber,
			rule.EscalationLevel+1, // Target level (L2=1, L3=2)
			deptID,
			deptName,
			reason,
		)
	}

	return &models.EscalationResult{
		ComplaintID:  candidate.ComplaintID,
		Escalated:    true,
		EscalationID: &escalation.EscalationID,
		NewStatus:    stringPtr(string(models.StatusEscalated)),
		Reason:       reason,
		ProcessedAt:  time.Now().UTC(),
	}, nil
}

// processReminder processes a reminder (not escalation)
func (s *EscalationService) processReminder(
	candidate models.EscalationCandidate,
	rule models.EscalationRule,
	conditions *models.EscalationConditions,
) (*models.ReminderResult, error) {
	// Check if reminder interval has passed
	if conditions.ReminderIntervalHours == nil || *conditions.ReminderIntervalHours == 0 {
		return nil, nil // No reminder interval configured
	}

	// Get last reminder time
	lastReminder, err := s.escalationRepo.GetLastReminderTime(candidate.ComplaintID)
	if err != nil {
		return nil, fmt.Errorf("failed to get last reminder time: %w", err)
	}

	now := time.Now().UTC()
	shouldSendReminder := false
	reason := ""

	if lastReminder == nil {
		// Never sent reminder, check if conditions are met
		shouldEscalate, reason := s.evaluateEscalationConditions(candidate, conditions)
		if shouldEscalate {
			shouldSendReminder = true
			reason = "First reminder: " + reason
		}
	} else {
		// Check if reminder interval has passed
		hoursSinceLastReminder := now.Sub(*lastReminder).Hours()
		if hoursSinceLastReminder >= float64(*conditions.ReminderIntervalHours) {
			shouldSendReminder = true
			reason = fmt.Sprintf("Reminder sent (last reminder %.1f hours ago)", hoursSinceLastReminder)
		}
	}

	if !shouldSendReminder {
		return nil, nil
	}

	// Log reminder to audit_log
	err = s.logEscalationAction(
		candidate.ComplaintID,
		"reminder",
		map[string]interface{}{
			"reminder_reason": reason,
			"rule_id":        rule.RuleID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to log reminder: %w", err)
	}

	return &models.ReminderResult{
		ComplaintID:  candidate.ComplaintID,
		ReminderSent: true,
		Reason:      reason,
		ProcessedAt:  now,
	}, nil
}

// logEscalationAction logs escalation/reminder actions to audit_log
func (s *EscalationService) logEscalationAction(
	complaintID int64,
	action string,
	metadata map[string]interface{},
) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	auditLog := &models.AuditLog{
		EntityType:   "complaint",
		EntityID:     complaintID,
		Action:       action,
		ActionByType: models.ActorSystem,
		Metadata:     sql.NullString{String: string(metadataJSON), Valid: true},
	}

	err = s.complaintRepo.CreateAuditLog(auditLog)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
