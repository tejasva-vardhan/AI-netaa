# Escalation Worker Pseudocode
## Production-Grade Implementation Guide

**Context**: Public Grievance System for Shivpuri, MP (Pincode: 473551)
**Escalation Levels**: L1 → L2 → L3 (max level; no escalation beyond L3)
**SLA Rules**: L1→L2: 72 hours, L2→L3: 120 hours
**Approach**: SLA-based, deterministic, rule-driven (NO AI)

---

## 1. Worker Structure

```go
// EscalationWorker runs periodically to process escalations
type EscalationWorker struct {
    escalationService *EscalationService
    notificationService *NotificationService  // For email events
    interval time.Duration                    // e.g., 30 minutes
    stopChan chan struct{}
    running bool
}

// Start initializes and runs the worker
func (w *EscalationWorker) Start() {
    // Start goroutine with ticker
    // Process immediately on start
    // Then process every interval
}

// Main worker loop
func (w *EscalationWorker) run() {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()
    
    // Process immediately
    w.processEscalations()
    
    for {
        select {
        case <-ticker.C:
            w.processEscalations()
        case <-w.stopChan:
            return
        }
    }
}
```

---

## 2. Main Processing Function

```go
// processEscalations is the core escalation processing logic
// IDEMPOTENT: Safe to run multiple times
func (w *EscalationWorker) processEscalations() {
    startTime := time.Now()
    log.Printf("[ESCALATION] Starting escalation processing cycle")
    
    // Step 1: Load active escalation rules
    rules, err := w.escalationService.GetActiveEscalationRules()
    if err != nil {
        log.Printf("[ERROR] Failed to load escalation rules: %v", err)
        return
    }
    
    if len(rules) == 0 {
        log.Printf("[ESCALATION] No active escalation rules configured")
        return
    }
    
    // Step 2: Find complaints that may need escalation
    candidates := w.findEscalationCandidates()
    
    // Step 3: Process each candidate
    escalatedCount := 0
    skippedCount := 0
    
    for _, candidate := range candidates {
        result, err := w.processComplaintEscalation(candidate, rules)
        if err != nil {
            log.Printf("[ERROR] Failed to process complaint %d: %v", candidate.ComplaintID, err)
            continue  // Continue with next complaint
        }
        
        if result != nil && result.Escalated {
            escalatedCount++
            log.Printf("[ESCALATION] Escalated complaint %d to level %d: %s", 
                result.ComplaintID, result.EscalationLevel, result.Reason)
        } else {
            skippedCount++
        }
    }
    
    duration := time.Since(startTime)
    log.Printf("[ESCALATION] Processing complete: %d escalated, %d skipped, took %v", 
        escalatedCount, skippedCount, duration)
}
```

---

## 3. Find Escalation Candidates Query

```go
// findEscalationCandidates queries complaints that may need escalation
func (w *EscalationWorker) findEscalationCandidates() []EscalationCandidate {
    /*
    SQL QUERY:
    SELECT 
        c.complaint_id,
        c.complaint_number,
        c.current_status,
        c.priority,
        c.assigned_department_id,
        c.assigned_officer_id,
        c.location_id,
        c.created_at,
        c.updated_at,
        -- Calculate last status change time
        COALESCE(
            (SELECT MAX(created_at) 
             FROM complaint_status_history 
             WHERE complaint_id = c.complaint_id),
            c.created_at
        ) as last_status_change_at,
        -- Get current escalation level (0 if none)
        COALESCE(
            (SELECT MAX(escalation_level) 
             FROM complaint_escalations 
             WHERE complaint_id = c.complaint_id),
            0
        ) as current_escalation_level
    FROM complaints c
    WHERE 
        -- Only active complaints (not resolved/closed/rejected)
        c.current_status NOT IN ('resolved', 'closed', 'rejected')
        -- Must have assigned department
        AND c.assigned_department_id IS NOT NULL
        -- Must have location
        AND c.location_id IS NOT NULL
    ORDER BY c.created_at ASC
    */
    
    // Execute query and return candidates
    return candidates
}
```

---

## 4. Process Single Complaint Escalation

```go
// processComplaintEscalation processes escalation for one complaint
func (w *EscalationWorker) processComplaintEscalation(
    candidate EscalationCandidate,
    rules []EscalationRule,
) (*EscalationResult, error) {
    
    // Step 1: Get current escalation level
    currentLevel := candidate.CurrentEscalationLevel  // 0 = L1, 1 = L2, etc.
    nextLevel := currentLevel + 1
    
    // Step 2: Check if already at max level (L4 = level 3)
    if nextLevel > 3 {  // L4 is max (0-indexed: L1=0, L2=1, L3=2, L4=3)
        return nil, nil  // No further escalation possible
    }
    
    // Step 3: Find applicable escalation rules for next level
    applicableRules := w.findApplicableRules(candidate, rules, nextLevel)
    
    if len(applicableRules) == 0 {
        return nil, nil  // No rules for this level
    }
    
    // Step 4: Evaluate each rule to find matching one
    for _, rule := range applicableRules {
        // Step 4a: Parse rule conditions (JSON)
        conditions, err := parseEscalationConditions(rule.Conditions)
        if err != nil {
            continue  // Skip invalid rule
        }
        
        // Step 4b: Check if SLA deadline exceeded
        shouldEscalate, reason := w.evaluateSLA(candidate, conditions, rule)
        if !shouldEscalate {
            continue  // SLA not breached yet
        }
        
        // Step 4c: IDEMPOTENCY CHECK - Prevent double escalation
        alreadyEscalated, err := w.checkIdempotency(candidate.ComplaintID, nextLevel)
        if err != nil {
            return nil, fmt.Errorf("idempotency check failed: %w", err)
        }
        if alreadyEscalated {
            log.Printf("[ESCALATION] Complaint %d already escalated to level %d (skipping)", 
                candidate.ComplaintID, nextLevel)
            return nil, nil  // Already escalated, skip
        }
        
        // Step 4d: Execute escalation (transaction)
        return w.executeEscalation(candidate, rule, nextLevel, reason)
    }
    
    return nil, nil  // No escalation needed
}
```

---

## 5. Find Applicable Rules

```go
// findApplicableRules filters rules matching complaint's context
func (w *EscalationWorker) findApplicableRules(
    candidate EscalationCandidate,
    rules []EscalationRule,
    targetLevel int,
) []EscalationRule {
    var applicable []EscalationRule
    
    for _, rule := range rules {
        // Must match target escalation level
        if rule.EscalationLevel != targetLevel {
            continue
        }
        
        // Check department match
        if rule.FromDepartmentID.Valid {
            if !candidate.AssignedDepartmentID.Valid ||
               candidate.AssignedDepartmentID.Int64 != rule.FromDepartmentID.Int64 {
                continue  // Department mismatch
            }
        }
        
        // Check location match (pincode/district)
        if rule.FromLocationID.Valid {
            if candidate.LocationID != rule.FromLocationID.Int64 {
                continue  // Location mismatch
            }
        }
        
        // Rule matches - add to applicable list
        applicable = append(applicable, rule)
    }
    
    return applicable
}
```

---

## 6. Evaluate SLA (Time-Based Conditions)

```go
// evaluateSLA checks if SLA deadline has been exceeded
func (w *EscalationWorker) evaluateSLA(
    candidate EscalationCandidate,
    conditions EscalationConditions,
    rule EscalationRule,
) (bool, string) {
    now := time.Now()
    
    // Step 1: Check status condition
    if len(conditions.Statuses) > 0 {
        statusMatch := false
        for _, requiredStatus := range conditions.Statuses {
            if string(candidate.CurrentStatus) == requiredStatus {
                statusMatch = true
                break
            }
        }
        if !statusMatch {
            return false, "Status condition not met"
        }
    }
    
    // Step 2: Check priority condition (if specified)
    if len(conditions.Priorities) > 0 {
        priorityMatch := false
        for _, requiredPriority := range conditions.Priorities {
            if string(candidate.Priority) == requiredPriority {
                priorityMatch = true
                break
            }
        }
        if !priorityMatch {
            return false, "Priority condition not met"
        }
    }
    
    // Step 3: Check time-based SLA conditions
    if conditions.TimeBased == nil {
        return false, "No time-based conditions defined"
    }
    
    timeBased := conditions.TimeBased
    
    // Calculate reference time (last update or last status change)
    var referenceTime time.Time
    if timeBased.UseLastStatusChange && !candidate.LastStatusChangeAt.IsZero() {
        referenceTime = candidate.LastStatusChangeAt
    } else if candidate.UpdatedAt.Valid {
        referenceTime = candidate.UpdatedAt.Time
    } else {
        referenceTime = candidate.CreatedAt
    }
    
    // Check hours since reference time
    hoursSinceReference := now.Sub(referenceTime).Hours()
    
    // Get SLA hours for this department and level
    // Query: SELECT sla_hours FROM department_sla_config 
    //        WHERE department_id = ? AND escalation_level = ?
    slaHours := w.getSLAHours(candidate.AssignedDepartmentID.Int64, rule.EscalationLevel)
    
    // Check if SLA breached
    if hoursSinceReference < float64(slaHours) {
        return false, fmt.Sprintf("SLA not breached yet (%.1f hours elapsed, %d hours required)", 
            hoursSinceReference, slaHours)
    }
    
    // SLA BREACHED - escalation required
    reason := fmt.Sprintf("SLA breached: %.1f hours elapsed since last update (SLA: %d hours)", 
        hoursSinceReference, slaHours)
    
    return true, reason
}
```

---

## 7. Idempotency Check

```go
// checkIdempotency ensures complaint not already escalated at this level
func (w *EscalationWorker) checkIdempotency(
    complaintID int64,
    escalationLevel int,
) (bool, error) {
    /*
    SQL QUERY:
    SELECT COUNT(*) 
    FROM complaint_escalations
    WHERE complaint_id = ?
      AND escalation_level = ?
      AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
    */
    
    // If count > 0, already escalated recently
    // Return true if already escalated, false otherwise
    return alreadyEscalated, nil
}
```

---

## 8. Execute Escalation (Transaction)

```go
// executeEscalation performs the actual escalation in a transaction
func (w *EscalationWorker) executeEscalation(
    candidate EscalationCandidate,
    rule EscalationRule,
    escalationLevel int,
    reason string,
) (*EscalationResult, error) {
    
    // BEGIN TRANSACTION
    tx, err := db.Begin()
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }
    defer tx.Rollback()  // Rollback on error
    
    // Step 1: Find next authority (officer) for target department and location
    /*
    SQL QUERY:
    SELECT officer_id, full_name, designation, email
    FROM officers
    WHERE department_id = ?
      AND location_id = ?
      AND is_active = true
    ORDER BY 
        CASE designation
            WHEN 'L1' THEN 1
            WHEN 'L2' THEN 2
            WHEN 'L3' THEN 3
            WHEN 'L4' THEN 4
        END
    LIMIT 1
    */
    nextOfficer := w.findNextAuthority(rule.ToDepartmentID, candidate.LocationID, escalationLevel)
    
    // Step 2: Create status history entry (REQUIRED - immutable audit trail)
    statusHistory := StatusHistory{
        ComplaintID: candidate.ComplaintID,
        OldStatus: candidate.CurrentStatus,
        NewStatus: "escalated",
        ChangedByType: "system",
        AssignedDepartmentID: rule.ToDepartmentID,
        AssignedOfficerID: nextOfficer.OfficerID,  // May be NULL if no officer found
        Notes: fmt.Sprintf("Auto-escalated to level %d: %s", escalationLevel, reason),
        CreatedAt: time.Now(),
    }
    
    /*
    SQL INSERT:
    INSERT INTO complaint_status_history (
        complaint_id, old_status, new_status, changed_by_type,
        assigned_department_id, assigned_officer_id, notes, created_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    */
    statusHistoryID := tx.Exec(statusHistory)
    
    // Step 3: Update complaint assignment
    /*
    SQL UPDATE:
    UPDATE complaints
    SET assigned_department_id = ?,
        assigned_officer_id = ?,
        current_status = 'escalated',
        updated_at = NOW()
    WHERE complaint_id = ?
    */
    tx.Exec(updateComplaint, rule.ToDepartmentID, nextOfficer.OfficerID, candidate.ComplaintID)
    
    // Step 4: Create escalation record
    escalation := Escalation{
        ComplaintID: candidate.ComplaintID,
        FromDepartmentID: candidate.AssignedDepartmentID,
        FromOfficerID: candidate.AssignedOfficerID,
        ToDepartmentID: rule.ToDepartmentID,
        ToOfficerID: nextOfficer.OfficerID,
        EscalationLevel: escalationLevel,
        Reason: reason,
        EscalatedByType: "system",
        StatusHistoryID: statusHistoryID,
        CreatedAt: time.Now(),
    }
    
    /*
    SQL INSERT:
    INSERT INTO complaint_escalations (
        complaint_id, from_department_id, from_officer_id,
        to_department_id, to_officer_id, escalation_level,
        reason, escalated_by_type, status_history_id, created_at
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    */
    escalationID := tx.Exec(escalation)
    
    // Step 5: Log to audit_log (REQUIRED - immutable audit trail)
    auditLog := AuditLog{
        EntityType: "complaint",
        EntityID: candidate.ComplaintID,
        Action: "escalation",
        ActionByType: "system",
        Metadata: JSON{
            "escalation_id": escalationID,
            "escalation_level": escalationLevel,
            "from_department": candidate.AssignedDepartmentID,
            "to_department": rule.ToDepartmentID,
            "reason": reason,
        },
        CreatedAt: time.Now(),
    }
    
    /*
    SQL INSERT:
    INSERT INTO audit_log (
        entity_type, entity_id, action, action_by_type, metadata, created_at
    ) VALUES (?, ?, ?, ?, ?, ?)
    */
    tx.Exec(auditLog)
    
    // COMMIT TRANSACTION
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("failed to commit escalation transaction: %w", err)
    }
    
    // Step 6: Emit email notification event (AFTER transaction commit)
    // This is non-blocking - if email fails, escalation still succeeded
    go w.emitEscalationEmail(escalation, nextOfficer, candidate)
    
    return &EscalationResult{
        ComplaintID: candidate.ComplaintID,
        Escalated: true,
        EscalationLevel: escalationLevel,
        Reason: reason,
        ProcessedAt: time.Now(),
    }, nil
}
```

---

## 9. Find Next Authority

```go
// findNextAuthority finds the appropriate officer for escalation level
func (w *EscalationWorker) findNextAuthority(
    departmentID int64,
    locationID int64,
    escalationLevel int,
) *Officer {
    /*
    SQL QUERY:
    SELECT officer_id, full_name, designation, email
    FROM officers
    WHERE department_id = ?
      AND location_id = ?
      AND is_active = true
      -- Match level based on designation pattern or level field
      AND (
          (escalation_level = ?) OR
          (designation LIKE '%L?%')  -- Pattern matching for L1, L2, L3, L4
      )
    ORDER BY escalation_level ASC
    LIMIT 1
    */
    
    // If no officer found, return nil (complaint assigned to department only)
    return officer
}
```

---

## 10. Get SLA Hours

```go
// getSLAHours retrieves SLA hours for department and escalation level
func (w *EscalationWorker) getSLAHours(
    departmentID int64,
    escalationLevel int,
) int {
    /*
    SQL QUERY:
    SELECT sla_hours
    FROM department_sla_config
    WHERE department_id = ?
      AND escalation_level = ?
      AND is_active = true
    LIMIT 1
    */
    
    // Default SLA if not configured:
    // L1 → L2: 48 hours
    // L2 → L3: 72 hours
    // L3 → L4: 96 hours
    
    defaultSLAs := map[int]int{
        1: 48,  // L1 → L2
        2: 72,  // L2 → L3
        3: 96,  // L3 → L4
    }
    
    if slaHours == 0 {
        return defaultSLAs[escalationLevel]
    }
    
    return slaHours
}
```

---

## 11. Emit Email Notification Event

```go
// emitEscalationEmail emits event for email notification (non-blocking)
func (w *EscalationWorker) emitEscalationEmail(
    escalation Escalation,
    nextOfficer *Officer,
    candidate EscalationCandidate,
) {
    // Get complaint details for email
    complaint := w.getComplaintDetails(escalation.ComplaintID)
    
    // Build email notification event
    emailEvent := EmailNotificationEvent{
        Type: "escalation",
        ToEmail: nextOfficer.Email,
        ToName: nextOfficer.FullName,
        Template: "complaint_escalated",
        TemplateData: map[string]interface{}{
            "complaint_id": complaint.ComplaintNumber,
            "department_name": nextOfficer.DepartmentName,
            "issue_summary": complaint.Title,
            "location": complaint.LocationName,
            "submitted_at": complaint.CreatedAt,
            "escalation_level": escalation.EscalationLevel,
            "authority_name": nextOfficer.FullName,
            "reason": escalation.Reason,
        },
    }
    
    // Send to notification queue (non-blocking)
    // Notification service handles retry logic
    w.notificationService.QueueEmail(emailEvent)
    
    log.Printf("[ESCALATION] Email notification queued for complaint %d", escalation.ComplaintID)
}
```

---

## 12. Error Handling Strategy

```go
// Error handling principles:

// 1. Transaction Safety
//    - Always use transactions for multi-step operations
//    - Rollback on any error
//    - Commit only after all steps succeed

// 2. Idempotency
//    - Check for existing escalation before creating new one
//    - Use time window (e.g., 1 hour) to prevent duplicates
//    - Safe to re-run worker multiple times

// 3. Partial Failures
//    - If one complaint fails, continue with others
//    - Log errors but don't stop entire batch
//    - Email failures don't block escalation (async)

// 4. Logging
//    - Log all escalations with reason
//    - Log errors with context (complaint_id, level)
//    - Log skipped complaints with reason

// 5. Recovery
//    - Worker can be restarted safely
//    - Idempotency prevents double-processing
//    - Audit log provides full trail
```

---

## 13. Key SQL Queries Summary

```sql
-- Query 1: Find escalation candidates
SELECT c.*, 
       COALESCE(MAX(csh.created_at), c.created_at) as last_status_change_at,
       COALESCE(MAX(ce.escalation_level), 0) as current_escalation_level
FROM complaints c
LEFT JOIN complaint_status_history csh ON c.complaint_id = csh.complaint_id
LEFT JOIN complaint_escalations ce ON c.complaint_id = ce.complaint_id
WHERE c.current_status NOT IN ('resolved', 'closed', 'rejected')
  AND c.assigned_department_id IS NOT NULL
GROUP BY c.complaint_id;

-- Query 2: Check idempotency
SELECT COUNT(*) 
FROM complaint_escalations
WHERE complaint_id = ?
  AND escalation_level = ?
  AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR);

-- Query 3: Get SLA hours
SELECT sla_hours
FROM department_sla_config
WHERE department_id = ?
  AND escalation_level = ?
  AND is_active = true;

-- Query 4: Find next authority
SELECT officer_id, full_name, designation, email
FROM officers
WHERE department_id = ?
  AND location_id = ?
  AND is_active = true
ORDER BY escalation_level ASC
LIMIT 1;
```

---

## 14. Idempotency Notes

```
IDEMPOTENCY GUARANTEES:

1. Time Window Check
   - Check if escalation exists within last 1 hour
   - Prevents duplicate escalations if worker runs multiple times

2. Level Check
   - Only escalate to next level (currentLevel + 1)
   - Cannot skip levels or go backwards

3. Status Check
   - Only process active complaints
   - Skip resolved/closed/rejected complaints

4. Transaction Isolation
   - All DB operations in single transaction
   - Atomic: all succeed or all fail
   - Prevents partial escalations

5. Safe Re-run
   - Worker can be stopped and restarted
   - Previous escalations are preserved
   - Only new escalations are processed
```

---

## 15. Escalation Flow Diagram (Pseudocode)

```
START: Worker runs every X minutes
  ↓
Load active escalation rules
  ↓
Query complaints (status NOT IN resolved/closed, has department)
  ↓
FOR EACH complaint:
  │
  ├─ Get current escalation level (0 = L1, 1 = L2, etc.)
  │
  ├─ Calculate next level (currentLevel + 1)
  │
  ├─ IF nextLevel > 3: SKIP (max level reached)
  │
  ├─ Find applicable rules for next level
  │
  ├─ FOR EACH applicable rule:
  │   │
  │   ├─ Parse rule conditions (JSON)
  │   │
  │   ├─ Evaluate SLA:
  │   │   ├─ Check status match
  │   │   ├─ Check priority match
  │   │   ├─ Calculate hours since last update
  │   │   └─ Compare with SLA hours
  │   │
  │   ├─ IF SLA not breached: CONTINUE (next rule)
  │   │
  │   ├─ Check idempotency (already escalated?)
  │   │
  │   ├─ IF already escalated: SKIP
  │   │
  │   └─ EXECUTE ESCALATION (transaction):
  │       ├─ Find next authority
  │       ├─ Create status_history entry
  │       ├─ Update complaint assignment
  │       ├─ Create escalation record
  │       ├─ Log to audit_log
  │       ├─ COMMIT transaction
  │       └─ Emit email event (async)
  │
  └─ NEXT complaint
  ↓
Log summary (escalated count, skipped count, duration)
END
```

---

## 16. Configuration Example

```go
// Example escalation rule configuration (JSON in escalation_rules.conditions):
{
    "statuses": ["under_review", "in_progress"],
    "priorities": ["medium", "high", "urgent"],
    "time_based": {
        "hours_since_last_update": 48,
        "use_last_status_change": true
    },
    "is_reminder": false
}

// Example SLA configuration (department_sla_config table):
// department_id=1, escalation_level=1, sla_hours=48  (PWD: L1→L2 in 48h)
// department_id=1, escalation_level=2, sla_hours=72  (PWD: L2→L3 in 72h)
// department_id=1, escalation_level=3, sla_hours=96  (PWD: L3→L4 in 96h)
```

---

## 17. Testing Considerations

```go
// Test scenarios:

// 1. Normal escalation (SLA breached)
//    - Complaint in "under_review" for 50 hours
//    - Expected: Escalate to L2

// 2. Idempotency (already escalated)
//    - Complaint escalated 30 minutes ago
//    - Expected: Skip (within 1-hour window)

// 3. Max level reached
//    - Complaint already at L4
//    - Expected: Skip (no further escalation)

// 4. No matching rule
//    - Complaint department has no rule for next level
//    - Expected: Skip

// 5. SLA not breached
//    - Complaint updated 10 hours ago, SLA is 48 hours
//    - Expected: Skip

// 6. Transaction rollback
//    - Simulate DB error during escalation
//    - Expected: Rollback, no partial updates

// 7. Email failure
//    - Email service unavailable
//    - Expected: Escalation succeeds, email queued for retry
```

---

## END OF PSEUDOCODE
